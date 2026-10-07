// Package gitlab reads your GitLab work through glab: the merge requests you
// opened, the ones waiting for your review, the issues assigned to you and
// your to-do list.
package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// DefaultHost is the GitLab dotui reads when no other is given.
const DefaultHost = "gitlab.com"

// Service is one GitLab host, through glab.
type Service struct {
	Run  work.Runner
	Host string
	// Clones finds local clones, for checking out a merge request. Nil
	// leaves checkouts out.
	Clones func() work.Clones
}

// New reads host through the glab on PATH.
func New(host string, clones func() work.Clones) *Service {
	return &Service{Run: work.Exec, Host: host, Clones: clones}
}

func (s *Service) Name() string      { return "GitLab" }
func (s *Service) CLI() string       { return "glab" }
func (s *Service) Package() string   { return "glab" }
func (s *Service) Connect() []string { return []string{"glab", "auth", "login", "--hostname", s.Host} }

// api runs `glab api` against the service's host.
func (s *Service) api(ctx context.Context, v any, args ...string) error {
	return work.RunJSON(ctx, s.Run, v, "glab", append([]string{"api", "--hostname", s.Host}, args...)...)
}

// graphql runs a GraphQL query and decodes its data into v.
func (s *Service) graphql(ctx context.Context, v any, query string) error {
	var res struct {
		Data   json.RawMessage
		Errors []struct{ Message string }
	}
	if err := s.api(ctx, &res, "graphql", "-f", "query="+query); err != nil {
		return err
	}
	if len(res.Errors) > 0 {
		return fmt.Errorf("GitLab: %s", res.Errors[0].Message)
	}
	return json.Unmarshal(res.Data, v)
}

// Check asks GitLab who glab is signed in as. A 401 means signed out;
// any other failure means it couldn't tell.
func (s *Service) Check(ctx context.Context) (work.Status, string, error) {
	var user struct{ Username string }
	err := s.api(ctx, &user, "user")
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return work.NotInstalled, "", nil
	case err != nil && strings.Contains(err.Error(), "401"):
		return work.SignedOut, "", nil
	case err != nil:
		return work.Unknown, "", err
	}
	return work.Connected, user.Username + " on " + s.Host, nil
}

const dashboardQuery = `query {
  currentUser {
    username
    authored: authoredMergeRequests(state: opened, first: 50, sort: UPDATED_DESC) {
      count nodes { ...mr headPipeline { status } approved detailedMergeStatus }
    }
    review: reviewRequestedMergeRequests(state: opened, first: 50, sort: UPDATED_DESC) { count nodes { ...mr } }
    todos(state: pending, first: 50) {
      nodes { action targetType createdAt author { username } project { fullPath }
        target { webUrl ... on MergeRequest { iid title } ... on Issue { iid title } } }
    }
  }
}
fragment mr on MergeRequest { iid title webUrl draft updatedAt sourceBranch author { username } project { fullPath } }`

type user struct{ Username string }

func (u *user) name() string {
	if u == nil {
		return "ghost"
	}
	return u.Username
}

type mr struct {
	IID                 string
	Title, WebURL       string
	Draft               bool
	UpdatedAt           time.Time
	SourceBranch        string
	Author              *user
	Project             struct{ FullPath string }
	HeadPipeline        *struct{ Status string }
	Approved            bool
	DetailedMergeStatus string
}

type mrs struct {
	Count int
	Nodes []mr
}

type todo struct {
	Action     string
	TargetType string
	CreatedAt  time.Time
	Author     *user
	Project    *struct{ FullPath string }
	Target     struct {
		WebURL string
		IID    string
		Title  string
	}
}

type issue struct {
	IID        int
	Title      string
	WebURL     string    `json:"web_url"`
	UpdatedAt  time.Time `json:"updated_at"`
	Author     *user
	Labels     []string
	References struct{ Full string }
}

// Dashboard runs the GraphQL query and lists assigned issues side by side.
func (s *Service) Dashboard(ctx context.Context) (work.Dashboard, error) {
	var (
		data struct {
			CurrentUser struct {
				Username         string
				Authored, Review mrs
				Todos            struct{ Nodes []todo }
			}
		}
		issues            []issue
		dataErr, issueErr error
		wg                sync.WaitGroup
	)
	wg.Go(func() { dataErr = s.graphql(ctx, &data, dashboardQuery) })
	wg.Go(func() {
		issueErr = s.api(ctx, &issues, "issues?scope=assigned_to_me&state=opened&order_by=updated_at&per_page=50")
	})
	wg.Wait()
	if dataErr != nil {
		return work.Dashboard{}, dataErr
	}
	u := data.CurrentUser
	dash := work.Dashboard{
		Account: u.Username + " on " + s.Host,
		Fetched: time.Now(),
		Sections: []work.Section{
			{Key: "Reviews", Title: "Waiting for your review", Total: u.Review.Count, Items: reviewItems(u.Review.Nodes),
				Empty: "Nobody is waiting for your review."},
			{Key: "MRs", Title: "Your open merge requests", Total: u.Authored.Count, Items: mrItems(u.Authored.Nodes),
				Empty: "You have no open merge requests."},
		},
	}
	assigned := work.Section{Key: "Issues", Title: "Issues assigned to you", Empty: "No open issues are assigned to you."}
	if issueErr != nil {
		assigned.Empty = "Couldn't list issues: " + issueErr.Error()
	} else {
		assigned.Items = issueItems(issues)
		assigned.Total = len(assigned.Items)
	}
	todos := todoItems(u.Todos.Nodes)
	dash.Sections = append(dash.Sections, assigned,
		work.Section{Key: "To-dos", Title: "Your to-do list", Items: todos, Total: len(todos), Empty: "Your to-do list is empty."})
	return dash, nil
}

func baseItem(m mr) work.Item {
	return work.Item{
		Kind:    work.PullRequest,
		ID:      m.Project.FullPath + "!" + m.IID,
		Ref:     "!" + m.IID,
		Title:   m.Title,
		Where:   m.Project.FullPath,
		Branch:  m.SourceBranch,
		Author:  m.Author.name(),
		URL:     m.WebURL,
		Updated: m.UpdatedAt,
	}
}

func reviewItems(nodes []mr) []work.Item {
	items := make([]work.Item, 0, len(nodes))
	for _, m := range nodes {
		it := baseItem(m)
		it.Next, it.NextTone, it.Urgency = "review requested", work.Warn, 90
		if m.Draft {
			it.Badges = append(it.Badges, work.Badge{Text: "draft", Tone: work.Dim})
		}
		items = append(items, it)
	}
	return items
}

// mrItems are your merge requests, each with its pipeline and approval, and
// what to do next.
func mrItems(nodes []mr) []work.Item {
	items := make([]work.Item, 0, len(nodes))
	for _, m := range nodes {
		it := baseItem(m)
		pipeline := ""
		if m.HeadPipeline != nil {
			pipeline = m.HeadPipeline.Status
		}
		if m.Draft {
			it.Badges = append(it.Badges, work.Badge{Text: "draft", Tone: work.Dim})
		}
		switch pipeline {
		case "SUCCESS":
			it.Badges = append(it.Badges, work.Badge{Text: "pipeline passed", Tone: work.Good})
		case "FAILED":
			it.Badges = append(it.Badges, work.Badge{Text: "pipeline failed", Tone: work.Bad})
		case "RUNNING", "PENDING", "CREATED", "PREPARING", "WAITING_FOR_RESOURCE":
			it.Badges = append(it.Badges, work.Badge{Text: "pipeline running", Tone: work.Warn})
		}
		if m.Approved {
			it.Badges = append(it.Badges, work.Badge{Text: "approved", Tone: work.Good})
		}
		status := m.DetailedMergeStatus
		switch status {
		case "CONFLICT", "NEED_REBASE":
			it.Badges = append(it.Badges, work.Badge{Text: words(status), Tone: work.Bad})
		case "DISCUSSIONS_NOT_RESOLVED":
			it.Badges = append(it.Badges, work.Badge{Text: "unresolved threads", Tone: work.Warn})
		case "NOT_APPROVED":
			it.Badges = append(it.Badges, work.Badge{Text: "awaiting approval", Tone: work.Dim})
		case "REQUESTED_CHANGES":
			it.Badges = append(it.Badges, work.Badge{Text: "changes requested", Tone: work.Bad})
		}
		switch {
		case status == "REQUESTED_CHANGES":
			it.Next, it.NextTone, it.Urgency = "changes requested", work.Bad, 85
		case pipeline == "FAILED":
			it.Next, it.NextTone, it.Urgency = "pipeline failed", work.Bad, 80
		case status == "CONFLICT" || status == "NEED_REBASE":
			it.Next, it.NextTone, it.Urgency = words(status), work.Bad, 75
		case status == "MERGEABLE" && !m.Draft:
			it.Next, it.NextTone, it.Urgency = "ready to merge", work.Good, 70
		case status == "DISCUSSIONS_NOT_RESOLVED":
			it.Next, it.NextTone, it.Urgency = "unresolved threads", work.Warn, 65
		}
		items = append(items, it)
	}
	return items
}

func issueItems(issues []issue) []work.Item {
	items := make([]work.Item, 0, len(issues))
	for _, is := range issues {
		project, _, _ := strings.Cut(is.References.Full, "#")
		it := work.Item{
			Kind:     work.Issue,
			ID:       is.References.Full,
			Ref:      fmt.Sprintf("#%d", is.IID),
			Title:    is.Title,
			Where:    project,
			Author:   is.Author.name(),
			URL:      is.WebURL,
			Updated:  is.UpdatedAt,
			Next:     "assigned to you",
			NextTone: work.Info,
			Urgency:  40,
		}
		for _, l := range is.Labels[:min(3, len(is.Labels))] {
			it.Badges = append(it.Badges, work.Badge{Text: l, Tone: work.Info})
		}
		items = append(items, it)
	}
	return items
}

// todoNext is why each kind of to-do needs you, and how urgently.
var todoNext = map[string]struct {
	next    string
	tone    work.Tone
	urgency int
}{
	"review_requested":   {"review requested", work.Warn, 90},
	"approval_required":  {"approval required", work.Warn, 85},
	"build_failed":       {"pipeline failed", work.Bad, 80},
	"unmergeable":        {"can't be merged", work.Bad, 75},
	"mentioned":          {"mentioned", work.Info, 60},
	"directly_addressed": {"mentioned", work.Info, 60},
	"assigned":           {"assigned to you", work.Info, 40},
}

func todoItems(todos []todo) []work.Item {
	items := make([]work.Item, 0, len(todos))
	for _, t := range todos {
		it := work.Item{
			Kind:     work.Other,
			ID:       t.Target.WebURL,
			Title:    t.Target.Title,
			URL:      t.Target.WebURL,
			Author:   t.Author.name(),
			Updated:  t.CreatedAt,
			Badges:   []work.Badge{{Text: words(t.Action), Tone: work.Dim}},
			Next:     "to-do",
			NextTone: work.Dim,
			Urgency:  30,
		}
		if t.Project != nil {
			it.Where = t.Project.FullPath
			switch {
			case t.Target.IID != "" && t.TargetType == "MERGEREQUEST":
				it.Kind, it.ID, it.Ref = work.PullRequest, it.Where+"!"+t.Target.IID, "!"+t.Target.IID
			case t.Target.IID != "" && t.TargetType == "ISSUE":
				it.Kind, it.ID, it.Ref = work.Issue, it.Where+"#"+t.Target.IID, "#"+t.Target.IID
			}
		}
		if n, ok := todoNext[t.Action]; ok {
			it.Next, it.NextTone, it.Urgency = n.next, n.tone, n.urgency
		}
		items = append(items, it)
	}
	return items
}

// words turns NEED_REBASE into "need rebase".
func words(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", " ")) }

// splitID reads "group/project!9" or "group/project#12".
func splitID(id string) (project, iid string, kind work.Kind, ok bool) {
	if p, n, found := strings.Cut(id, "!"); found {
		return p, n, work.PullRequest, p != "" && n != ""
	}
	if p, n, found := strings.Cut(id, "#"); found {
		return p, n, work.Issue, p != "" && n != ""
	}
	return "", "", "", false
}
