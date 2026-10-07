// Package jira reads your Jira work through acli, Atlassian's CLI: what's
// assigned to you, what you watch, what you reported, and what you finished
// lately.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// Service is Jira, through acli.
type Service struct {
	Run work.Runner

	mu sync.Mutex
	// statuses are the statuses seen per project on the last dashboard,
	// offered when moving a work item.
	statuses map[string][]string
}

// New reads Jira through the acli on PATH.
func New() *Service { return &Service{Run: work.Exec} }

func (s *Service) Name() string      { return "Jira" }
func (s *Service) CLI() string       { return "acli" }
func (s *Service) Package() string   { return "atlassian/acli/acli" }
func (s *Service) Connect() []string { return []string{"acli", "jira", "auth", "login", "--web"} }

// signedOutWords are what acli says when nobody is signed in.
var signedOutWords = []string{"not logged in", "not authenticated", "no account", "unauthorized", "auth login", "please log in", "please login"}

func signedOut(text string) bool {
	text = strings.ToLower(text)
	return slices.ContainsFunc(signedOutWords, func(w string) bool { return strings.Contains(text, w) })
}

// Check reads `acli jira auth status`, which names the site and account.
func (s *Service) Check(ctx context.Context) (work.Status, string, error) {
	out, err := s.Run(ctx, "acli", "jira", "auth", "status")
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return work.NotInstalled, "", nil
	case err != nil && signedOut(err.Error()+string(out)):
		return work.SignedOut, "", nil
	case err != nil:
		return work.Unknown, "", err
	case signedOut(string(out)):
		return work.SignedOut, "", nil
	}
	var site, user string
	for line := range strings.SplitSeq(string(out), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch {
		case strings.Contains(k, "site") && site == "":
			site = strings.TrimPrefix(strings.TrimPrefix(v, "https://"), "http://")
		case (strings.Contains(k, "email") || strings.Contains(k, "user") || strings.Contains(k, "account")) && user == "":
			user = v
		}
	}
	switch {
	case user != "" && site != "":
		return work.Connected, user + " on " + site, nil
	case site != "":
		return work.Connected, site, nil
	}
	return work.Connected, "signed in", nil
}

// jiraTime reads Jira's timestamps, like 2026-10-01T10:00:00.000+0200.
type jiraTime struct{ time.Time }

func (t *jiraTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil || s == "" {
		return nil // no time: zero
	}
	for _, layout := range []string{"2006-01-02T15:04:05.000-0700", time.RFC3339Nano, "2006-01-02"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("jira: unreadable time %q", s)
}

type person struct{ DisplayName string }

func (p *person) name() string {
	if p == nil {
		return ""
	}
	return p.DisplayName
}

type status struct {
	Name           string
	StatusCategory struct{ Key string }
}

type issue struct {
	Key    string
	Self   string
	Fields struct {
		Summary   string
		Updated   jiraTime
		Created   jiraTime
		Status    *status
		Priority  *struct{ Name string }
		IssueType *struct{ Name string } `json:"issuetype"`
		Assignee  *person
		Reporter  *person
		Project   *struct{ Key, Name string }
		Labels    []string
		// Detail only.
		Description json.RawMessage
		Comment     *struct {
			Comments []struct {
				Author  *person
				Created jiraTime
				Body    json.RawMessage
			}
		}
		Components []struct{ Name string }
		Parent     *struct {
			Key    string
			Fields struct{ Summary string }
		}
		Subtasks []struct {
			Key    string
			Fields struct {
				Summary string
				Status  *status
			}
		}
		IssueLinks []struct {
			Type         struct{ Inward, Outward string }
			InwardIssue  *struct{ Key string }
			OutwardIssue *struct{ Key string }
		} `json:"issuelinks"`
	}
}

// project is the work item's project key: PROJ for PROJ-7.
func (is issue) project() string {
	if p := is.Fields.Project; p != nil && p.Key != "" {
		return p.Key
	}
	key, _, _ := strings.Cut(is.Key, "-")
	return key
}

// url links to the work item on its site, from its API address.
func (is issue) url() string {
	base, _, ok := strings.Cut(is.Self, "/rest/")
	if !ok {
		return ""
	}
	return base + "/browse/" + is.Key
}

const searchFields = "key,summary,status,priority,issuetype,assignee,reporter,updated,project,labels"

// search runs a JQL query. acli prints a list of work items; an object
// holding them under "issues" reads as well.
func (s *Service) search(ctx context.Context, jql string) ([]issue, error) {
	out, err := s.Run(ctx, "acli", "jira", "workitem", "search", "--jql", jql, "--fields", searchFields, "--limit", "50", "--json")
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)
	var issues []issue
	if bytes.HasPrefix(out, []byte("{")) {
		var wrapped struct{ Issues []issue }
		err = json.Unmarshal(out, &wrapped)
		issues = wrapped.Issues
	} else if len(out) > 0 {
		err = json.Unmarshal(out, &issues)
	}
	if err != nil {
		return nil, fmt.Errorf("acli printed something other than the JSON expected: %w", err)
	}
	return issues, nil
}

// sections are the dashboard's lists, each a JQL query.
var sections = []struct {
	key, title, jql, empty string
}{
	{"Mine", "Assigned to you, not done", "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC",
		"Nothing open is assigned to you."},
	{"Watching", "Watched, updated this week", "watcher = currentUser() AND (assignee != currentUser() OR assignee IS EMPTY) AND updated >= -7d ORDER BY updated DESC",
		"Nothing you watch changed this week."},
	{"Reported", "Reported by you, still open", "reporter = currentUser() AND statusCategory != Done ORDER BY updated DESC",
		"Everything you reported is done."},
	{"Done", "Finished in the last two weeks", "assignee = currentUser() AND statusCategory = Done AND updated >= -14d ORDER BY updated DESC",
		"Nothing finished in the last two weeks."},
}

// Dashboard runs the sections' searches side by side.
func (s *Service) Dashboard(ctx context.Context) (work.Dashboard, error) {
	results := make([][]issue, len(sections))
	errs := make([]error, len(sections))
	var wg sync.WaitGroup
	for i, sec := range sections {
		wg.Go(func() { results[i], errs[i] = s.search(ctx, sec.jql) })
	}
	wg.Wait()
	if errs[0] != nil {
		return work.Dashboard{}, errs[0]
	}

	dash := work.Dashboard{Fetched: time.Now()}
	seen := map[string][]string{}
	for i, sec := range sections {
		section := work.Section{Key: sec.key, Title: sec.title, Empty: sec.empty}
		if errs[i] != nil {
			section.Empty = "Couldn't search: " + errs[i].Error()
		}
		for _, is := range results[i] {
			section.Items = append(section.Items, item(is, sec.key))
			if st := is.Fields.Status; st != nil && !slices.Contains(seen[is.project()], st.Name) {
				seen[is.project()] = append(seen[is.project()], st.Name)
			}
		}
		section.Total = len(section.Items)
		dash.Sections = append(dash.Sections, section)
	}
	s.mu.Lock()
	s.statuses = seen
	s.mu.Unlock()
	return dash, nil
}

var urgentPriorities = []string{"Highest", "High", "Blocker", "Critical"}

func categoryTone(category string) work.Tone {
	switch category {
	case "indeterminate":
		return work.Info
	case "done":
		return work.Good
	}
	return work.Dim
}

// item turns a work item into an item of the given section, with what to
// do about it.
func item(is issue, section string) work.Item {
	f := is.Fields
	it := work.Item{
		Kind:    work.WorkItem,
		ID:      is.Key,
		Ref:     is.Key,
		Title:   f.Summary,
		Where:   is.project(),
		Author:  f.Reporter.name(),
		URL:     is.url(),
		Updated: f.Updated.Time,
	}
	if f.Project != nil && f.Project.Name != "" {
		it.Where = f.Project.Name
	}
	category := ""
	if f.Status != nil {
		category = f.Status.StatusCategory.Key
		it.Badges = append(it.Badges, work.Badge{Text: f.Status.Name, Tone: categoryTone(category)})
	}
	urgent := f.Priority != nil && slices.Contains(urgentPriorities, f.Priority.Name)
	if urgent {
		it.Badges = append(it.Badges, work.Badge{Text: f.Priority.Name + " priority", Tone: work.Warn})
	}
	if f.IssueType != nil {
		it.Badges = append(it.Badges, work.Badge{Text: f.IssueType.Name, Tone: work.Dim})
	}
	switch section {
	case "Mine":
		switch {
		case urgent:
			it.Next, it.NextTone, it.Urgency = "high priority", work.Warn, 60
		case category == "indeterminate":
			it.Next, it.NextTone, it.Urgency = "in progress", work.Info, 55
		default:
			it.Next, it.NextTone, it.Urgency = "to do", work.Dim, 35
		}
	case "Watching":
		it.Next, it.NextTone, it.Urgency = "updated", work.Info, 30
	}
	return it
}
