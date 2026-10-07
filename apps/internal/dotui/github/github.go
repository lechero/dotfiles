// Package github reads your GitHub work through gh: the pull requests you
// opened, the ones waiting for your review, the issues assigned to you and
// your unread notifications.
package github

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

const host = "github.com"

// Service is GitHub, through gh.
type Service struct {
	Run work.Runner
	// Clones finds local clones, for checking out a pull request. Nil
	// leaves checkouts out.
	Clones func() work.Clones
}

// New reads GitHub through the gh on PATH.
func New(clones func() work.Clones) *Service {
	return &Service{Run: work.Exec, Clones: clones}
}

func (s *Service) Name() string      { return "GitHub" }
func (s *Service) CLI() string       { return "gh" }
func (s *Service) Package() string   { return "gh" }
func (s *Service) Connect() []string { return []string{"gh", "auth", "login", "--hostname", host} }

// Check asks gh who it's signed in as. With --json, gh exits 0 whatever
// the state, and says per account whether the token worked.
func (s *Service) Check(ctx context.Context) (work.Status, string, error) {
	var st struct {
		Hosts map[string][]struct {
			State  string
			Active bool
			Login  string
		}
	}
	err := work.RunJSON(ctx, s.Run, &st, "gh", "auth", "status", "--json", "hosts", "--hostname", host)
	if errors.Is(err, exec.ErrNotFound) {
		return work.NotInstalled, "", nil
	}
	if err != nil {
		return work.Unknown, "", err
	}
	for _, a := range st.Hosts[host] {
		if !a.Active {
			continue
		}
		switch a.State {
		case "success":
			return work.Connected, a.Login + " on " + host, nil
		case "timeout":
			return work.Unknown, "", fmt.Errorf("gh couldn't reach %s", host)
		}
	}
	return work.SignedOut, "", nil
}

// dashboardQuery is one GraphQL request for everything but notifications.
const dashboardQuery = `query {
  viewer { login }
  mine: search(query: "is:pr is:open author:@me archived:false sort:updated-desc", type: ISSUE, first: 50) {
    issueCount
    nodes { ... on PullRequest { number title url isDraft updatedAt repository { nameWithOwner } author { login }
      reviewDecision mergeable commits(last: 1) { nodes { commit { statusCheckRollup { state } } } } } }
  }
  review: search(query: "is:pr is:open review-requested:@me archived:false sort:updated-desc", type: ISSUE, first: 50) {
    issueCount
    nodes { ... on PullRequest { number title url isDraft updatedAt repository { nameWithOwner } author { login } } }
  }
  issues: search(query: "is:issue is:open assignee:@me archived:false sort:updated-desc", type: ISSUE, first: 50) {
    issueCount
    nodes { ... on Issue { number title url updatedAt repository { nameWithOwner } author { login } labels(first: 3) { nodes { name } } } }
  }
}`

type node struct {
	Number     int
	Title      string
	URL        string
	IsDraft    bool
	UpdatedAt  time.Time
	Repository struct{ NameWithOwner string }
	Author     *struct{ Login string }
	// Pull requests only.
	ReviewDecision string
	Mergeable      string
	Commits        struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct{ State string }
			}
		}
	}
	// Issues only.
	Labels struct{ Nodes []struct{ Name string } }
}

type search struct {
	IssueCount int
	Nodes      []node
}

type notification struct {
	Reason    string
	UpdatedAt time.Time `json:"updated_at"`
	Subject   struct {
		Title string
		URL   string
		Type  string
	}
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}
}

// Dashboard runs the search query and reads notifications side by side.
func (s *Service) Dashboard(ctx context.Context) (work.Dashboard, error) {
	var (
		data struct {
			Data struct {
				Viewer               struct{ Login string }
				Mine, Review, Issues search
			}
		}
		notes            []notification
		dataErr, noteErr error
		wg               sync.WaitGroup
	)
	wg.Go(func() {
		dataErr = work.RunJSON(ctx, s.Run, &data, "gh", "api", "graphql", "-f", "query="+dashboardQuery)
	})
	wg.Go(func() {
		noteErr = work.RunJSON(ctx, s.Run, &notes, "gh", "api", "notifications?per_page=50")
	})
	wg.Wait()
	if dataErr != nil {
		return work.Dashboard{}, dataErr
	}
	d := data.Data
	dash := work.Dashboard{
		Account: d.Viewer.Login + " on " + host,
		Fetched: time.Now(),
		Sections: []work.Section{
			{Key: "Reviews", Title: "Waiting for your review", Total: d.Review.IssueCount, Items: reviewItems(d.Review.Nodes),
				Empty: "Nobody is waiting for your review."},
			{Key: "PRs", Title: "Your open pull requests", Total: d.Mine.IssueCount, Items: prItems(d.Mine.Nodes),
				Empty: "You have no open pull requests."},
			{Key: "Issues", Title: "Issues assigned to you", Total: d.Issues.IssueCount, Items: issueItems(d.Issues.Nodes),
				Empty: "No open issues are assigned to you."},
		},
	}
	inbox := work.Section{Key: "Inbox", Title: "Unread notifications", Empty: "No unread notifications."}
	if noteErr != nil {
		inbox.Empty = "Couldn't read notifications: " + noteErr.Error()
	} else {
		inbox.Items = notificationItems(notes)
		inbox.Total = len(inbox.Items)
	}
	dash.Sections = append(dash.Sections, inbox)
	return dash, nil
}

func baseItem(n node, kind work.Kind) work.Item {
	it := work.Item{
		Kind:    kind,
		ID:      fmt.Sprintf("%s#%d", n.Repository.NameWithOwner, n.Number),
		Ref:     "#" + strconv.Itoa(n.Number),
		Title:   n.Title,
		Where:   n.Repository.NameWithOwner,
		URL:     n.URL,
		Updated: n.UpdatedAt,
	}
	if n.Author != nil {
		it.Author = n.Author.Login
	}
	return it
}

func reviewItems(nodes []node) []work.Item {
	var items []work.Item
	for _, n := range nodes {
		if n.Number == 0 {
			continue
		}
		it := baseItem(n, work.PullRequest)
		it.Next, it.NextTone, it.Urgency = "review requested", work.Warn, 90
		if n.IsDraft {
			it.Badges = append(it.Badges, work.Badge{Text: "draft", Tone: work.Dim})
		}
		items = append(items, it)
	}
	return items
}

// prItems are your pull requests, each with its CI and review state, and
// what to do next: fix what failed, or merge what's ready.
func prItems(nodes []node) []work.Item {
	var items []work.Item
	for _, n := range nodes {
		if n.Number == 0 {
			continue
		}
		it := baseItem(n, work.PullRequest)
		checks := ""
		if c := n.Commits.Nodes; len(c) > 0 && c[0].Commit.StatusCheckRollup != nil {
			checks = c[0].Commit.StatusCheckRollup.State
		}
		if n.IsDraft {
			it.Badges = append(it.Badges, work.Badge{Text: "draft", Tone: work.Dim})
		}
		switch checks {
		case "SUCCESS":
			it.Badges = append(it.Badges, work.Badge{Text: "checks pass", Tone: work.Good})
		case "FAILURE", "ERROR":
			it.Badges = append(it.Badges, work.Badge{Text: "checks failing", Tone: work.Bad})
		case "PENDING", "EXPECTED":
			it.Badges = append(it.Badges, work.Badge{Text: "checks running", Tone: work.Warn})
		}
		switch n.ReviewDecision {
		case "APPROVED":
			it.Badges = append(it.Badges, work.Badge{Text: "approved", Tone: work.Good})
		case "CHANGES_REQUESTED":
			it.Badges = append(it.Badges, work.Badge{Text: "changes requested", Tone: work.Bad})
		case "REVIEW_REQUIRED":
			it.Badges = append(it.Badges, work.Badge{Text: "awaiting review", Tone: work.Dim})
		}
		conflicting := n.Mergeable == "CONFLICTING"
		if conflicting {
			it.Badges = append(it.Badges, work.Badge{Text: "conflicts", Tone: work.Bad})
		}
		switch {
		case n.ReviewDecision == "CHANGES_REQUESTED":
			it.Next, it.NextTone, it.Urgency = "changes requested", work.Bad, 85
		case checks == "FAILURE" || checks == "ERROR":
			it.Next, it.NextTone, it.Urgency = "checks failing", work.Bad, 80
		case conflicting:
			it.Next, it.NextTone, it.Urgency = "merge conflict", work.Bad, 75
		case !n.IsDraft && n.ReviewDecision == "APPROVED" && (checks == "SUCCESS" || checks == ""):
			it.Next, it.NextTone, it.Urgency = "ready to merge", work.Good, 70
		}
		items = append(items, it)
	}
	return items
}

func issueItems(nodes []node) []work.Item {
	var items []work.Item
	for _, n := range nodes {
		if n.Number == 0 {
			continue
		}
		it := baseItem(n, work.Issue)
		it.Next, it.NextTone, it.Urgency = "assigned to you", work.Info, 40
		for _, l := range n.Labels.Nodes {
			it.Badges = append(it.Badges, work.Badge{Text: l.Name, Tone: work.Info})
		}
		items = append(items, it)
	}
	return items
}

// notificationItems turns notifications about pull requests and issues into
// those items, so they open and act like the rest.
func notificationItems(notes []notification) []work.Item {
	var items []work.Item
	for _, n := range notes {
		it := work.Item{
			Kind:    work.Other,
			ID:      n.Subject.URL,
			Title:   n.Subject.Title,
			Where:   n.Repository.FullName,
			URL:     n.Repository.HTMLURL,
			Updated: n.UpdatedAt,
			Badges:  []work.Badge{{Text: strings.ReplaceAll(n.Reason, "_", " "), Tone: work.Dim}},
		}
		if kind, num, ok := subject(n.Subject.URL); ok {
			it.Kind = kind
			it.ID = fmt.Sprintf("%s#%s", n.Repository.FullName, num)
			it.Ref = "#" + num
			path := "issues"
			if kind == work.PullRequest {
				path = "pull"
			}
			it.URL = fmt.Sprintf("%s/%s/%s", n.Repository.HTMLURL, path, num)
		}
		switch n.Reason {
		case "review_requested":
			it.Next, it.NextTone, it.Urgency = "review requested", work.Warn, 90
		case "mention", "team_mention":
			it.Next, it.NextTone, it.Urgency = "mentioned", work.Info, 60
		case "assign":
			it.Next, it.NextTone, it.Urgency = "assigned to you", work.Info, 40
		}
		items = append(items, it)
	}
	return items
}

// subject reads a notification's API URL, like
// https://api.github.com/repos/o/r/pulls/12, as a pull request or issue.
func subject(apiURL string) (work.Kind, string, bool) {
	parts := strings.Split(apiURL, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	num := parts[len(parts)-1]
	if _, err := strconv.Atoi(num); err != nil {
		return "", "", false
	}
	switch parts[len(parts)-2] {
	case "pulls":
		return work.PullRequest, num, true
	case "issues":
		return work.Issue, num, true
	}
	return "", "", false
}

// splitID reads "owner/repo#12".
func splitID(id string) (repo, num string, ok bool) {
	repo, num, ok = strings.Cut(id, "#")
	return repo, num, ok && strings.Contains(repo, "/") && num != ""
}
