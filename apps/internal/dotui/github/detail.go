package github

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

type login struct{ Login string }

func (l *login) name() string {
	if l == nil || l.Login == "" {
		return "ghost"
	}
	return l.Login
}

type comment struct {
	Author    *login
	CreatedAt time.Time
	Body      string
}

type labels []struct{ Name string }

func (ls labels) String() string {
	names := make([]string, len(ls))
	for i, l := range ls {
		names[i] = l.Name
	}
	return strings.Join(names, ", ")
}

// Detail reads a pull request or an issue with gh's view command.
func (s *Service) Detail(ctx context.Context, it work.Item) (work.Detail, error) {
	repo, num, ok := splitID(it.ID)
	switch {
	case !ok:
		return work.Detail{Body: "There's nothing more to show here. Press o to open it on GitHub."}, nil
	case it.Kind == work.PullRequest:
		return s.prDetail(ctx, repo, num)
	default:
		return s.issueDetail(ctx, repo, num)
	}
}

func (s *Service) prDetail(ctx context.Context, repo, num string) (work.Detail, error) {
	var pr struct {
		State, HeadRefName, BaseRefName    string
		ReviewDecision, Mergeable, Body    string
		IsDraft                            bool
		Author                             *login
		Additions, Deletions, ChangedFiles int
		Labels                             labels
		CreatedAt                          time.Time
		Comments                           []comment
		Reviews                            []struct {
			Author      *login
			State       string
			SubmittedAt time.Time
			Body        string
		}
		StatusCheckRollup []struct {
			Name, Status, Conclusion string // check runs
			Context, State           string // commit statuses
		}
	}
	fields := "state,headRefName,baseRefName,reviewDecision,mergeable,body,isDraft,author,additions,deletions," +
		"changedFiles,labels,createdAt,comments,reviews,statusCheckRollup"
	if err := work.RunJSON(ctx, s.Run, &pr, "gh", "pr", "view", num, "-R", repo, "--json", fields); err != nil {
		return work.Detail{}, err
	}

	state := work.Field{Label: "State", Value: strings.ToLower(pr.State), Tone: stateTone(pr.State)}
	if pr.IsDraft {
		state.Value += ", draft"
	}
	d := work.Detail{
		Body: pr.Body,
		Fields: []work.Field{
			state,
			{Label: "Branch", Value: pr.HeadRefName + " → " + pr.BaseRefName},
			{Label: "Author", Value: pr.Author.name()},
			{Label: "Changes", Value: fmt.Sprintf("+%d −%d in %d %s", pr.Additions, pr.Deletions, pr.ChangedFiles, plural(pr.ChangedFiles, "file", "files"))},
		},
	}
	if pr.ReviewDecision != "" {
		tone := map[string]work.Tone{"APPROVED": work.Good, "CHANGES_REQUESTED": work.Bad}[pr.ReviewDecision]
		d.Fields = append(d.Fields, work.Field{Label: "Review", Value: words(pr.ReviewDecision), Tone: tone})
	}
	if pr.Mergeable == "CONFLICTING" {
		d.Fields = append(d.Fields, work.Field{Label: "Merge", Value: "has conflicts with " + pr.BaseRefName, Tone: work.Bad})
	}
	if len(pr.Labels) > 0 {
		d.Fields = append(d.Fields, work.Field{Label: "Labels", Value: pr.Labels.String(), Tone: work.Info})
	}

	for _, c := range pr.StatusCheckRollup {
		if c.Context != "" { // a commit status
			d.Checks = append(d.Checks, work.Check{Name: c.Context, State: strings.ToLower(c.State), Tone: checkTone(c.State)})
			continue
		}
		state := c.Conclusion
		if c.Status != "COMPLETED" {
			state = c.Status
		}
		d.Checks = append(d.Checks, work.Check{Name: c.Name, State: words(state), Tone: checkTone(state)})
	}
	slices.SortStableFunc(d.Checks, func(a, b work.Check) int { return cmp.Compare(b.Tone, a.Tone) }) // failures first

	for _, r := range pr.Reviews {
		if r.State == "COMMENTED" && r.Body == "" {
			continue // the comments of a review are in the diff, not here
		}
		d.Activity = append(d.Activity, work.Note{Author: r.Author.name(), When: r.SubmittedAt, What: words(r.State), Text: r.Body})
	}
	for _, c := range pr.Comments {
		d.Activity = append(d.Activity, work.Note{Author: c.Author.name(), When: c.CreatedAt, What: "commented", Text: c.Body})
	}
	sortNotes(d.Activity)
	return d, nil
}

func (s *Service) issueDetail(ctx context.Context, repo, num string) (work.Detail, error) {
	var is struct {
		State, Body string
		Author      *login
		Assignees   []login
		Labels      labels
		Milestone   *struct{ Title string }
		CreatedAt   time.Time
		Comments    []comment
	}
	fields := "state,body,author,assignees,labels,milestone,createdAt,comments"
	if err := work.RunJSON(ctx, s.Run, &is, "gh", "issue", "view", num, "-R", repo, "--json", fields); err != nil {
		return work.Detail{}, err
	}
	d := work.Detail{
		Body: is.Body,
		Fields: []work.Field{
			{Label: "State", Value: strings.ToLower(is.State), Tone: stateTone(is.State)},
			{Label: "Author", Value: is.Author.name()},
		},
	}
	if len(is.Assignees) > 0 {
		names := make([]string, len(is.Assignees))
		for i, a := range is.Assignees {
			names[i] = a.Login
		}
		d.Fields = append(d.Fields, work.Field{Label: "Assignees", Value: strings.Join(names, ", ")})
	}
	if len(is.Labels) > 0 {
		d.Fields = append(d.Fields, work.Field{Label: "Labels", Value: is.Labels.String(), Tone: work.Info})
	}
	if is.Milestone != nil {
		d.Fields = append(d.Fields, work.Field{Label: "Milestone", Value: is.Milestone.Title})
	}
	for _, c := range is.Comments {
		d.Activity = append(d.Activity, work.Note{Author: c.Author.name(), When: c.CreatedAt, What: "commented", Text: c.Body})
	}
	sortNotes(d.Activity)
	return d, nil
}

// Actions hand the terminal to gh. review, merge and checks ask or show
// their own questions; a comment is written in $EDITOR first.
func (s *Service) Actions(it work.Item) []work.Action {
	repo, num, ok := splitID(it.ID)
	if !ok {
		return nil
	}
	switch it.Kind {
	case work.PullRequest:
		actions := []work.Action{
			{Key: "d", Help: "diff", Args: []string{"gh", "pr", "diff", num, "-R", repo}, Pause: true},
			{Key: "c", Help: "checks", Args: []string{"gh", "pr", "checks", num, "-R", repo, "--watch"}, Pause: true},
			{Key: "v", Help: "review", Args: []string{"gh", "pr", "review", num, "-R", repo}, Pause: true},
			{Key: "C", Help: "comment", Args: []string{"gh", "pr", "comment", num, "-R", repo, "--body", work.BodyArg}, Compose: true, Pause: true},
			{Key: "m", Help: "merge", Args: []string{"gh", "pr", "merge", num, "-R", repo}, Pause: true},
		}
		if s.Clones != nil {
			if dir, ok := s.Clones().Find(host, repo); ok {
				actions = append(actions, work.Action{
					Key: "b", Help: "check out", Args: []string{"gh", "pr", "checkout", num}, Dir: dir, Pause: true,
					Confirm: fmt.Sprintf("Check out %s in %s? That clone switches to its branch.", it.Ref, dir),
				})
			}
		}
		return actions
	case work.Issue:
		return []work.Action{
			{Key: "C", Help: "comment", Args: []string{"gh", "issue", "comment", num, "-R", repo, "--body", work.BodyArg}, Compose: true, Pause: true},
		}
	}
	return nil
}

func stateTone(state string) work.Tone {
	switch state {
	case "OPEN":
		return work.Good
	case "MERGED":
		return work.Info
	}
	return work.Dim
}

func checkTone(state string) work.Tone {
	switch state {
	case "SUCCESS":
		return work.Good
	case "FAILURE", "ERROR", "TIMED_OUT", "STARTUP_FAILURE", "ACTION_REQUIRED":
		return work.Bad
	case "PENDING", "EXPECTED", "IN_PROGRESS", "QUEUED", "WAITING", "REQUESTED":
		return work.Warn
	}
	return work.Dim // skipped, neutral, cancelled
}

// words turns CHANGES_REQUESTED into "changes requested".
func words(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", " ")) }

func sortNotes(notes []work.Note) {
	slices.SortStableFunc(notes, func(a, b work.Note) int { return a.When.Compare(b.When) })
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
