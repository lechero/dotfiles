package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// fakeACLI answers acli commands whose text contains one of answers' keys.
func fakeACLI(t *testing.T, answers map[string]string) *Service {
	t.Helper()
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		for part, out := range answers {
			if strings.Contains(cmd, part) {
				if strings.HasPrefix(out, "error:") {
					return nil, errors.New(strings.TrimPrefix(out, "error:"))
				}
				return []byte(out), nil
			}
		}
		t.Errorf("unexpected command %q", cmd)
		return nil, errors.New("unexpected command")
	}
	return &Service{Run: run}
}

func issueJSON(key, summary, status, category, priority, updated string) string {
	return fmt.Sprintf(`{"key": %q, "self": "https://acme.atlassian.net/rest/api/3/issue/1%s",
	  "fields": {"summary": %q, "updated": %q, "status": {"name": %q, "statusCategory": {"key": %q}},
	  "priority": {"name": %q}, "issuetype": {"name": "Task"}, "reporter": {"displayName": "Pat"},
	  "project": {"key": "WEB", "name": "Website"}}}`, key, key[4:], summary, updated, status, category, priority)
}

func TestDashboard(t *testing.T) {
	mine := "[" + strings.Join([]string{
		issueJSON("WEB-1", "Fix login", "In Progress", "indeterminate", "Medium", "2026-10-06T09:00:00.000+0200"),
		issueJSON("WEB-2", "Outage page", "To Do", "new", "Highest", "2026-10-05T09:00:00.000+0200"),
		issueJSON("WEB-3", "Tidy footer", "To Do", "new", "Low", "2026-10-04T09:00:00.000+0200"),
	}, ",") + "]"
	// acli may wrap results in an object; both read.
	watching := `{"issues": [` + issueJSON("WEB-9", "Design review", "In Review", "indeterminate", "Medium", "2026-10-06T12:00:00.000Z") + `]}`
	s := fakeACLI(t, map[string]string{
		"assignee = currentUser() AND statusCategory != Done": mine,
		"watcher = currentUser()":                             watching,
		"reporter = currentUser()":                            "[]",
		"statusCategory = Done":                               "error:acli: rate limited",
	})
	d, err := s.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, sec := range d.Sections {
		keys = append(keys, fmt.Sprintf("%s %d", sec.Key, len(sec.Items)))
	}
	if want := []string{"Mine 3", "Watching 1", "Reported 0", "Done 0"}; !slices.Equal(keys, want) {
		t.Errorf("sections %v, want %v", keys, want)
	}
	if !strings.Contains(d.Sections[3].Empty, "rate limited") {
		t.Errorf("a failed search should say why it's empty: %q", d.Sections[3].Empty)
	}

	first := d.Sections[0].Items[0]
	if first.URL != "https://acme.atlassian.net/browse/WEB-1" || first.Where != "Website" || first.Author != "Pat" {
		t.Errorf("first item %+v", first)
	}
	if want := time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC); !first.Updated.Equal(want) {
		t.Errorf("updated %v, want %v", first.Updated, want)
	}

	var order []string
	for _, it := range d.FollowUps() {
		order = append(order, it.ID+" "+it.Next)
	}
	want := []string{"WEB-2 high priority", "WEB-1 in progress", "WEB-3 to do", "WEB-9 updated"}
	if !slices.Equal(order, want) {
		t.Errorf("follow-ups\n got %q\nwant %q", order, want)
	}

	// Moving offers the statuses the dashboard showed for that project.
	move := s.Actions(work.Item{Kind: work.WorkItem, ID: "WEB-3"})[0]
	if want := []string{"In Progress", "To Do", "In Review"}; !slices.Equal(move.Choices, want) {
		t.Errorf("move choices %q, want %q", move.Choices, want)
	}
	if !slices.Contains(move.Args, work.ChoiceArg) || move.Confirm != "" {
		t.Errorf("move %+v: the status is picked, and acli asks itself", move)
	}
	other := s.Actions(work.Item{Kind: work.WorkItem, ID: "OPS-4"})[0]
	if !slices.Equal(other.Choices, fallbackStatuses) {
		t.Errorf("a project not seen yet gets the usual statuses, got %q", other.Choices)
	}
}

func TestCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		answer  string
		status  work.Status
		account string
		err     bool
	}{
		"signed in":        {"✓ Authenticated\n  Site: acme.atlassian.net\n  Email: me@example.com\n", work.Connected, "me@example.com on acme.atlassian.net", false},
		"site only":        {"Site: https://acme.atlassian.net\n", work.Connected, "acme.atlassian.net", false},
		"says signed out":  {"You are not logged in. Run acli jira auth login.\n", work.SignedOut, "", false},
		"fails signed out": {"error:acli: unauthorized: run `acli jira auth login`", work.SignedOut, "", false},
		"unreachable":      {"error:acli: dial tcp: i/o timeout", work.Unknown, "", true},
	} {
		t.Run(name, func(t *testing.T) {
			s := fakeACLI(t, map[string]string{"auth status": tc.answer})
			status, account, err := s.Check(context.Background())
			if status != tc.status || account != tc.account || (err != nil) != tc.err {
				t.Errorf("Check = %v, %q, %v", status, account, err)
			}
		})
	}
	missing := &Service{Run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("acli: %w", exec.ErrNotFound)
	}}
	if status, _, err := missing.Check(context.Background()); status != work.NotInstalled || err != nil {
		t.Errorf("without acli: %v, %v", status, err)
	}
}

func TestDetail(t *testing.T) {
	s := fakeACLI(t, map[string]string{"workitem view WEB-1": `{"key": "WEB-1", "fields": {
	  "summary": "Fix login", "status": {"name": "In Progress", "statusCategory": {"key": "indeterminate"}},
	  "priority": {"name": "High"}, "issuetype": {"name": "Bug"}, "assignee": null, "reporter": {"displayName": "Pat"},
	  "labels": ["auth"], "components": [{"name": "Web"}],
	  "parent": {"key": "WEB-100", "fields": {"summary": "Accounts epic"}},
	  "issuelinks": [{"type": {"inward": "is blocked by", "outward": "blocks"}, "outwardIssue": {"key": "WEB-7"}}],
	  "subtasks": [{"key": "WEB-11", "fields": {"summary": "Write test", "status": {"name": "Done", "statusCategory": {"key": "done"}}}}],
	  "description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Users get "}, {"type": "text", "text": "logged out", "marks": [{"type": "strong"}]}]}]},
	  "comment": {"comments": [{"author": {"displayName": "Sam"}, "created": "2026-10-06T10:00:00.000+0000",
	    "body": {"type": "doc", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "On it"}]}]}}]}
	}}`})
	d, err := s.Detail(context.Background(), work.Item{Kind: work.WorkItem, ID: "WEB-1"})
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	for _, f := range d.Fields {
		fields[f.Label] = f.Value
	}
	for label, want := range map[string]string{
		"Status": "In Progress", "Type": "Bug", "Priority": "High", "Assignee": "nobody", "Reporter": "Pat",
		"Parent": "WEB-100 Accounts epic", "Labels": "auth", "Components": "Web", "Links": "blocks WEB-7",
	} {
		if fields[label] != want {
			t.Errorf("%s = %q, want %q", label, fields[label], want)
		}
	}
	if d.Body != "Users get **logged out**" {
		t.Errorf("body %q", d.Body)
	}
	if d.ChecksTitle != "Subtasks" || len(d.Checks) != 1 || d.Checks[0].Name != "WEB-11 Write test" || d.Checks[0].Tone != work.Good {
		t.Errorf("subtasks %q %+v", d.ChecksTitle, d.Checks)
	}
	if len(d.Activity) != 1 || d.Activity[0].Author != "Sam" || d.Activity[0].Text != "On it" {
		t.Errorf("activity %+v", d.Activity)
	}
}

func TestRichText(t *testing.T) {
	doc := `{"type": "doc", "content": [
	  {"type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Steps"}]},
	  {"type": "orderedList", "content": [
	    {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Open "}, {"type": "text", "text": "/login", "marks": [{"type": "code"}]}]}]},
	    {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Ask "}, {"type": "mention", "attrs": {"text": "@Pat"}}]}]}
	  ]},
	  {"type": "paragraph", "content": [{"type": "text", "text": "docs", "marks": [{"type": "link", "attrs": {"href": "https://example.com"}}]}, {"type": "hardBreak"}, {"type": "text", "text": "next line"}]},
	  {"type": "codeBlock", "attrs": {"language": "go"}, "content": [{"type": "text", "text": "x := 1"}]},
	  {"type": "table", "content": [
	    {"type": "tableRow", "content": [{"type": "tableHeader", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "A"}]}]}, {"type": "tableHeader", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "B"}]}]}]},
	    {"type": "tableRow", "content": [{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "1"}]}]}, {"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "2"}]}]}]}
	  ]},
	  {"type": "mediaSingle", "content": [{"type": "media"}]}
	]}`
	want := "## Steps\n\n" +
		"1. Open `/login`\n2. Ask @Pat\n\n" +
		"[docs](https://example.com)  \nnext line\n\n" +
		"```go\nx := 1\n```\n\n" +
		"| A | B |\n| --- | --- |\n| 1 | 2 |\n\n" +
		"*(attachment)*"
	if got := richText(json.RawMessage(doc)); got != want {
		t.Errorf("richText\n got %q\nwant %q", got, want)
	}
	if got := richText(json.RawMessage(`"plain text, from an older API"`)); got != "plain text, from an older API" {
		t.Errorf("a plain string body: %q", got)
	}
	if got := richText(json.RawMessage(`null`)); got != "" {
		t.Errorf("no body: %q", got)
	}
}
