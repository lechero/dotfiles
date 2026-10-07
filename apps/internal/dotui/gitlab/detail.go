package gitlab

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

type note struct {
	Author    *user
	Body      string
	CreatedAt time.Time
	System    bool
}

type titles struct{ Nodes []struct{ Title string } }

func (t titles) String() string {
	names := make([]string, len(t.Nodes))
	for i, n := range t.Nodes {
		names[i] = n.Title
	}
	return strings.Join(names, ", ")
}

// gql quotes s for a GraphQL query. JSON's string syntax is GraphQL's.
func gql(s string) string {
	b, _ := json.Marshal(s) // a string always marshals
	return string(b)
}

// Detail reads a merge request or an issue with one GraphQL query.
func (s *Service) Detail(ctx context.Context, it work.Item) (work.Detail, error) {
	project, iid, kind, ok := splitID(it.ID)
	switch {
	case !ok:
		return work.Detail{Body: "There's nothing more to show here. Press o to open it on GitLab."}, nil
	case kind == work.PullRequest:
		return s.mrDetail(ctx, project, iid)
	default:
		return s.issueDetail(ctx, project, iid)
	}
}

func (s *Service) mrDetail(ctx context.Context, project, iid string) (work.Detail, error) {
	query := fmt.Sprintf(`query { project(fullPath: %s) { mergeRequest(iid: %s) {
  state draft sourceBranch targetBranch description approved detailedMergeStatus
  author { username } approvedBy { nodes { username } } labels { nodes { title } }
  diffStatsSummary { additions deletions fileCount }
  headPipeline { status jobs(retried: false, first: 100) { nodes { name status stage { name } } } }
  notes(first: 100) { nodes { author { username } body createdAt system } }
} } }`, gql(project), gql(iid))
	var data struct {
		Project *struct {
			MergeRequest *struct {
				State, SourceBranch, TargetBranch, Description, DetailedMergeStatus string
				Draft, Approved                                                     bool
				Author                                                              *user
				ApprovedBy                                                          struct{ Nodes []user }
				Labels                                                              titles
				DiffStatsSummary                                                    *struct{ Additions, Deletions, FileCount int }
				HeadPipeline                                                        *struct {
					Status string
					Jobs   struct {
						Nodes []struct {
							Name, Status string
							Stage        *struct{ Name string }
						}
					}
				}
				Notes struct{ Nodes []note }
			}
		}
	}
	if err := s.graphql(ctx, &data, query); err != nil {
		return work.Detail{}, err
	}
	if data.Project == nil || data.Project.MergeRequest == nil {
		return work.Detail{}, fmt.Errorf("GitLab has no merge request !%s in %s, or you can't see it", iid, project)
	}
	m := data.Project.MergeRequest

	state := work.Field{Label: "State", Value: m.State, Tone: stateTone(m.State)}
	if m.Draft {
		state.Value += ", draft"
	}
	d := work.Detail{
		Body: m.Description,
		Fields: []work.Field{
			state,
			{Label: "Branch", Value: m.SourceBranch + " → " + m.TargetBranch},
			{Label: "Author", Value: m.Author.name()},
		},
	}
	if st := m.DiffStatsSummary; st != nil {
		d.Fields = append(d.Fields, work.Field{Label: "Changes", Value: fmt.Sprintf("+%d −%d in %d %s",
			st.Additions, st.Deletions, st.FileCount, plural(st.FileCount, "file", "files"))})
	}
	if names := m.ApprovedBy.Nodes; len(names) > 0 {
		approvers := make([]string, len(names))
		for i, u := range names {
			approvers[i] = u.Username
		}
		d.Fields = append(d.Fields, work.Field{Label: "Approved by", Value: strings.Join(approvers, ", "), Tone: work.Good})
	}
	if m.DetailedMergeStatus != "" {
		tone := map[string]work.Tone{"MERGEABLE": work.Good, "CONFLICT": work.Bad, "NEED_REBASE": work.Bad,
			"REQUESTED_CHANGES": work.Bad, "DISCUSSIONS_NOT_RESOLVED": work.Warn}[m.DetailedMergeStatus]
		d.Fields = append(d.Fields, work.Field{Label: "Merge", Value: words(m.DetailedMergeStatus), Tone: tone})
	}
	if len(m.Labels.Nodes) > 0 {
		d.Fields = append(d.Fields, work.Field{Label: "Labels", Value: m.Labels.String(), Tone: work.Info})
	}
	if p := m.HeadPipeline; p != nil {
		d.Fields = append(d.Fields, work.Field{Label: "Pipeline", Value: words(p.Status), Tone: pipelineTone(p.Status)})
		for _, j := range p.Jobs.Nodes {
			name := j.Name
			if j.Stage != nil {
				name = j.Stage.Name + " › " + j.Name
			}
			d.Checks = append(d.Checks, work.Check{Name: name, State: words(j.Status), Tone: pipelineTone(j.Status)})
		}
		slices.SortStableFunc(d.Checks, func(a, b work.Check) int { return cmp.Compare(b.Tone, a.Tone) }) // failures first
	}
	d.Activity = notes(m.Notes.Nodes)
	return d, nil
}

func (s *Service) issueDetail(ctx context.Context, project, iid string) (work.Detail, error) {
	query := fmt.Sprintf(`query { project(fullPath: %s) { issue(iid: %s) {
  state description author { username } assignees { nodes { username } }
  labels { nodes { title } } milestone { title }
  notes(first: 100) { nodes { author { username } body createdAt system } }
} } }`, gql(project), gql(iid))
	var data struct {
		Project *struct {
			Issue *struct {
				State, Description string
				Author             *user
				Assignees          struct{ Nodes []user }
				Labels             titles
				Milestone          *struct{ Title string }
				Notes              struct{ Nodes []note }
			}
		}
	}
	if err := s.graphql(ctx, &data, query); err != nil {
		return work.Detail{}, err
	}
	if data.Project == nil || data.Project.Issue == nil {
		return work.Detail{}, fmt.Errorf("GitLab has no issue #%s in %s, or you can't see it", iid, project)
	}
	is := data.Project.Issue
	d := work.Detail{
		Body: is.Description,
		Fields: []work.Field{
			{Label: "State", Value: is.State, Tone: stateTone(is.State)},
			{Label: "Author", Value: is.Author.name()},
		},
	}
	if a := is.Assignees.Nodes; len(a) > 0 {
		names := make([]string, len(a))
		for i, u := range a {
			names[i] = u.Username
		}
		d.Fields = append(d.Fields, work.Field{Label: "Assignees", Value: strings.Join(names, ", ")})
	}
	if len(is.Labels.Nodes) > 0 {
		d.Fields = append(d.Fields, work.Field{Label: "Labels", Value: is.Labels.String(), Tone: work.Info})
	}
	if is.Milestone != nil {
		d.Fields = append(d.Fields, work.Field{Label: "Milestone", Value: is.Milestone.Title})
	}
	d.Activity = notes(is.Notes.Nodes)
	return d, nil
}

// notes are comments, and GitLab's own notes on what happened, like
// "approved this merge request", oldest first.
func notes(nodes []note) []work.Note {
	out := make([]work.Note, 0, len(nodes))
	for _, n := range nodes {
		if n.System {
			out = append(out, work.Note{Author: n.Author.name(), When: n.CreatedAt, What: n.Body})
			continue
		}
		out = append(out, work.Note{Author: n.Author.name(), When: n.CreatedAt, What: "commented", Text: n.Body})
	}
	slices.SortStableFunc(out, func(a, b work.Note) int { return a.When.Compare(b.When) })
	return out
}

// Actions hand the terminal to glab. Merging asks for itself; approving
// doesn't, so dotui asks first. A comment is written in $EDITOR first.
func (s *Service) Actions(it work.Item) []work.Action {
	project, iid, kind, ok := splitID(it.ID)
	if !ok {
		return nil
	}
	repo := []string{"-R", "https://" + s.Host + "/" + project}
	if kind == work.Issue {
		return []work.Action{
			{Key: "C", Help: "comment", Args: append([]string{"glab", "issue", "note", iid, "-m", work.BodyArg}, repo...), Compose: true, Pause: true},
		}
	}
	actions := []work.Action{
		{Key: "d", Help: "diff", Args: append([]string{"glab", "mr", "diff", iid}, repo...), Pause: true},
	}
	if it.Branch != "" {
		actions = append(actions, work.Action{Key: "c", Help: "pipeline", Args: append([]string{"glab", "ci", "view", "-b", it.Branch}, repo...)})
	}
	actions = append(actions,
		work.Action{Key: "v", Help: "approve", Args: append([]string{"glab", "mr", "approve", iid}, repo...), Pause: true,
			Confirm: fmt.Sprintf("Approve %s in %s?", it.Ref, project)},
		work.Action{Key: "C", Help: "comment", Args: append([]string{"glab", "mr", "note", "create", iid, "-m", work.BodyArg}, repo...), Compose: true, Pause: true},
		work.Action{Key: "m", Help: "merge", Args: append([]string{"glab", "mr", "merge", iid}, repo...), Pause: true},
	)
	if s.Clones != nil {
		if dir, ok := s.Clones().Find(s.Host, project); ok {
			actions = append(actions, work.Action{
				Key: "b", Help: "check out", Args: []string{"glab", "mr", "checkout", iid}, Dir: dir, Pause: true,
				Confirm: fmt.Sprintf("Check out %s in %s? That clone switches to its branch.", it.Ref, dir),
			})
		}
	}
	return actions
}

func stateTone(state string) work.Tone {
	switch state {
	case "opened":
		return work.Good
	case "merged":
		return work.Info
	}
	return work.Dim
}

func pipelineTone(status string) work.Tone {
	switch status {
	case "SUCCESS":
		return work.Good
	case "FAILED":
		return work.Bad
	case "RUNNING", "PENDING", "CREATED", "PREPARING", "WAITING_FOR_RESOURCE", "SCHEDULED":
		return work.Warn
	}
	return work.Dim // skipped, canceled, manual
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
