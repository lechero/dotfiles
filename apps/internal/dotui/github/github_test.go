package github

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// fakeGH answers gh commands that start with one of answers' keys.
func fakeGH(t *testing.T, answers map[string]string) *Service {
	t.Helper()
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		for prefix, out := range answers {
			if strings.HasPrefix(cmd, prefix) {
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

const dashboardJSON = `{"data": {
  "viewer": {"login": "me"},
  "mine": {"issueCount": 4, "nodes": [
    {"number": 1, "title": "Failing", "url": "https://github.com/me/app/pull/1", "updatedAt": "2026-10-01T10:00:00Z",
     "repository": {"nameWithOwner": "me/app"}, "author": {"login": "me"}, "reviewDecision": "REVIEW_REQUIRED",
     "mergeable": "MERGEABLE", "commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "FAILURE"}}}]}},
    {"number": 2, "title": "Ready", "url": "https://github.com/me/app/pull/2", "updatedAt": "2026-10-02T10:00:00Z",
     "repository": {"nameWithOwner": "me/app"}, "author": {"login": "me"}, "reviewDecision": "APPROVED",
     "mergeable": "MERGEABLE", "commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "SUCCESS"}}}]}},
    {"number": 3, "title": "Asked to change", "url": "https://github.com/me/lib/pull/3", "updatedAt": "2026-10-03T10:00:00Z",
     "repository": {"nameWithOwner": "me/lib"}, "author": {"login": "me"}, "reviewDecision": "CHANGES_REQUESTED",
     "mergeable": "CONFLICTING", "commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "PENDING"}}}]}},
    {"number": 4, "title": "Draft", "url": "https://github.com/me/lib/pull/4", "isDraft": true, "updatedAt": "2026-10-04T10:00:00Z",
     "repository": {"nameWithOwner": "me/lib"}, "author": {"login": "me"}, "reviewDecision": "APPROVED",
     "mergeable": "MERGEABLE", "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}},
    {}
  ]},
  "review": {"issueCount": 1, "nodes": [
    {"number": 9, "title": "Please look", "url": "https://github.com/team/svc/pull/9", "updatedAt": "2026-09-30T10:00:00Z",
     "repository": {"nameWithOwner": "team/svc"}, "author": {"login": "pat"}}
  ]},
  "issues": {"issueCount": 1, "nodes": [
    {"number": 5, "title": "Crash on start", "url": "https://github.com/me/app/issues/5", "updatedAt": "2026-09-29T10:00:00Z",
     "repository": {"nameWithOwner": "me/app"}, "author": {"login": "sam"}, "labels": {"nodes": [{"name": "bug"}]}}
  ]}
}}`

const notificationsJSON = `[
  {"reason": "review_requested", "updated_at": "2026-10-05T10:00:00Z",
   "subject": {"title": "Please look", "url": "https://api.github.com/repos/team/svc/pulls/9", "type": "PullRequest"},
   "repository": {"full_name": "team/svc", "html_url": "https://github.com/team/svc"}},
  {"reason": "mention", "updated_at": "2026-10-05T11:00:00Z",
   "subject": {"title": "Question for you", "url": "https://api.github.com/repos/team/svc/issues/12", "type": "Issue"},
   "repository": {"full_name": "team/svc", "html_url": "https://github.com/team/svc"}},
  {"reason": "subscribed", "updated_at": "2026-10-05T12:00:00Z",
   "subject": {"title": "v2.0.0", "url": "https://api.github.com/repos/team/svc/releases/77", "type": "Release"},
   "repository": {"full_name": "team/svc", "html_url": "https://github.com/team/svc"}}
]`

func TestDashboard(t *testing.T) {
	s := fakeGH(t, map[string]string{"gh api graphql": dashboardJSON, "gh api notifications": notificationsJSON})
	d, err := s.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Account != "me on github.com" {
		t.Errorf("account %q", d.Account)
	}
	var keys []string
	for _, sec := range d.Sections {
		keys = append(keys, fmt.Sprintf("%s %d/%d", sec.Key, len(sec.Items), sec.Total))
	}
	if want := []string{"Reviews 1/1", "PRs 4/4", "Issues 1/1", "Inbox 3/3"}; !slices.Equal(keys, want) {
		t.Errorf("sections %v, want %v", keys, want)
	}

	next := map[string]string{}
	badges := map[string][]string{}
	for _, it := range d.Sections[1].Items {
		next[it.ID] = it.Next
		for _, b := range it.Badges {
			badges[it.ID] = append(badges[it.ID], b.Text)
		}
	}
	for id, want := range map[string]string{
		"me/app#1": "checks failing", "me/app#2": "ready to merge", "me/lib#3": "changes requested", "me/lib#4": "",
	} {
		if next[id] != want {
			t.Errorf("%s next = %q, want %q (badges %v)", id, next[id], want, badges[id])
		}
	}
	if got := strings.Join(badges["me/lib#3"], ", "); got != "checks running, changes requested, conflicts" {
		t.Errorf("me/lib#3 badges %q", got)
	}

	inbox := d.Sections[3].Items
	if inbox[1].ID != "team/svc#12" || inbox[1].Kind != work.Issue || inbox[1].URL != "https://github.com/team/svc/issues/12" {
		t.Errorf("a mention should be its issue: %+v", inbox[1])
	}
	if inbox[2].Kind != work.Other || inbox[2].URL != "https://github.com/team/svc" || inbox[2].Next != "" {
		t.Errorf("a release notification opens the repository and needs nothing: %+v", inbox[2])
	}

	var order []string
	for _, it := range d.FollowUps() {
		order = append(order, it.ID+" "+it.Next)
	}
	want := []string{
		"team/svc#9 review requested", // in Reviews and in the inbox, listed once
		"me/lib#3 changes requested",
		"me/app#1 checks failing",
		"me/app#2 ready to merge",
		"team/svc#12 mentioned",
		"me/app#5 assigned to you",
	}
	if !slices.Equal(order, want) {
		t.Errorf("follow-ups\n got %q\nwant %q", order, want)
	}
}

func TestDashboardWithoutNotifications(t *testing.T) {
	s := fakeGH(t, map[string]string{"gh api graphql": dashboardJSON, "gh api notifications": "error:gh: HTTP 403"})
	d, err := s.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if inbox := d.Sections[3]; len(inbox.Items) != 0 || !strings.Contains(inbox.Empty, "HTTP 403") {
		t.Errorf("the inbox should say why it's empty: %+v", inbox)
	}
}

func TestCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		answer  string
		status  work.Status
		account string
		err     bool
	}{
		"signed in":   {`{"hosts": {"github.com": [{"state": "success", "active": true, "login": "me"}]}}`, work.Connected, "me on github.com", false},
		"bad token":   {`{"hosts": {"github.com": [{"state": "error", "active": true, "login": "me"}]}}`, work.SignedOut, "", false},
		"no account":  {`{"hosts": {}}`, work.SignedOut, "", false},
		"unreachable": {`{"hosts": {"github.com": [{"state": "timeout", "active": true, "login": "me"}]}}`, work.Unknown, "", true},
	} {
		t.Run(name, func(t *testing.T) {
			s := fakeGH(t, map[string]string{"gh auth status": tc.answer})
			status, account, err := s.Check(context.Background())
			if status != tc.status || account != tc.account || (err != nil) != tc.err {
				t.Errorf("Check = %v, %q, %v", status, account, err)
			}
		})
	}

	missing := &Service{Run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("gh: %w", exec.ErrNotFound)
	}}
	if status, _, err := missing.Check(context.Background()); status != work.NotInstalled || err != nil {
		t.Errorf("without gh: %v, %v", status, err)
	}
}

func TestPullRequestDetail(t *testing.T) {
	s := fakeGH(t, map[string]string{"gh pr view 3 -R me/lib --json": `{
	  "state": "OPEN", "headRefName": "fix", "baseRefName": "main", "reviewDecision": "CHANGES_REQUESTED",
	  "mergeable": "CONFLICTING", "body": "Fixes **it**.", "author": {"login": "me"},
	  "additions": 10, "deletions": 2, "changedFiles": 1, "labels": [{"name": "bug"}],
	  "comments": [{"author": {"login": "pat"}, "createdAt": "2026-10-03T12:00:00Z", "body": "Second"}],
	  "reviews": [
	    {"author": {"login": "pat"}, "state": "CHANGES_REQUESTED", "submittedAt": "2026-10-03T11:00:00Z", "body": "First"},
	    {"author": {"login": "sam"}, "state": "COMMENTED", "submittedAt": "2026-10-03T13:00:00Z", "body": ""}
	  ],
	  "statusCheckRollup": [
	    {"name": "lint", "status": "COMPLETED", "conclusion": "SUCCESS"},
	    {"name": "test", "status": "COMPLETED", "conclusion": "FAILURE"},
	    {"context": "deploy", "state": "PENDING"}
	  ]}`})
	d, err := s.Detail(context.Background(), work.Item{Kind: work.PullRequest, ID: "me/lib#3"})
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	for _, f := range d.Fields {
		fields[f.Label] = f.Value
	}
	for label, want := range map[string]string{
		"Branch": "fix → main", "Changes": "+10 −2 in 1 file", "Review": "changes requested",
		"Merge": "has conflicts with main", "Labels": "bug",
	} {
		if fields[label] != want {
			t.Errorf("%s = %q, want %q", label, fields[label], want)
		}
	}
	var checks []string
	for _, c := range d.Checks {
		checks = append(checks, c.Name+" "+c.State)
	}
	if want := []string{"test failure", "deploy pending", "lint success"}; !slices.Equal(checks, want) {
		t.Errorf("checks %q, want failures first: %q", checks, want)
	}
	var notes []string
	for _, n := range d.Activity {
		notes = append(notes, n.Author+" "+n.What)
	}
	if want := []string{"pat changes requested", "pat commented"}; !slices.Equal(notes, want) {
		t.Errorf("activity %q, want %q: oldest first, without empty review comments", notes, want)
	}
}

func TestActions(t *testing.T) {
	s := &Service{Clones: func() work.Clones { return work.Clones{"github.com/me/app": "/src/app"} }}
	keys := func(it work.Item) string {
		var out []string
		for _, a := range s.Actions(it) {
			out = append(out, a.Key)
		}
		return strings.Join(out, " ")
	}
	pr := work.Item{Kind: work.PullRequest, ID: "me/app#1", Ref: "#1"}
	if got := keys(pr); got != "d c v C m b" {
		t.Errorf("pull request actions %q", got)
	}
	actions := s.Actions(pr)
	checkout := actions[len(actions)-1]
	if checkout.Dir != "/src/app" || checkout.Confirm == "" || !slices.Equal(checkout.Args, []string{"gh", "pr", "checkout", "1"}) {
		t.Errorf("checkout %+v: should run in the clone, after asking", checkout)
	}
	comment := actions[3]
	if !comment.Compose || !slices.Contains(comment.Args, work.BodyArg) {
		t.Errorf("comment %+v: should be written first", comment)
	}
	if got := keys(work.Item{Kind: work.PullRequest, ID: "other/repo#2"}); got != "d c v C m" {
		t.Errorf("without a clone, no checkout: %q", got)
	}
	if got := keys(work.Item{Kind: work.Issue, ID: "me/app#5"}); got != "C" {
		t.Errorf("issue actions %q", got)
	}
	if got := keys(work.Item{Kind: work.Other, ID: "https://api.github.com/repos/x/y/releases/1"}); got != "" {
		t.Errorf("a release has no actions: %q", got)
	}
}
