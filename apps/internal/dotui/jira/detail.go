package jira

import (
	"context"
	"slices"
	"strings"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

const viewFields = "summary,status,priority,issuetype,assignee,reporter,created,updated,description,comment," +
	"labels,components,parent,subtasks,issuelinks,project"

// Detail reads a work item with `acli jira workitem view`.
func (s *Service) Detail(ctx context.Context, it work.Item) (work.Detail, error) {
	var is issue
	if err := work.RunJSON(ctx, s.Run, &is, "acli", "jira", "workitem", "view", it.ID, "--fields", viewFields, "--json"); err != nil {
		return work.Detail{}, err
	}
	f := is.Fields
	d := work.Detail{Body: richText(f.Description), ChecksTitle: "Subtasks"}
	add := func(label, value string, tone work.Tone) {
		if value != "" {
			d.Fields = append(d.Fields, work.Field{Label: label, Value: value, Tone: tone})
		}
	}
	if f.Status != nil {
		add("Status", f.Status.Name, categoryTone(f.Status.StatusCategory.Key))
	}
	if f.IssueType != nil {
		add("Type", f.IssueType.Name, work.Dim)
	}
	if f.Priority != nil {
		tone := work.Dim
		if slices.Contains(urgentPriorities, f.Priority.Name) {
			tone = work.Warn
		}
		add("Priority", f.Priority.Name, tone)
	}
	assignee := f.Assignee.name()
	if assignee == "" {
		assignee = "nobody"
	}
	add("Assignee", assignee, work.Dim)
	add("Reporter", f.Reporter.name(), work.Dim)
	if p := f.Parent; p != nil {
		add("Parent", p.Key+" "+p.Fields.Summary, work.Info)
	}
	add("Labels", strings.Join(f.Labels, ", "), work.Info)
	components := make([]string, len(f.Components))
	for i, c := range f.Components {
		components[i] = c.Name
	}
	add("Components", strings.Join(components, ", "), work.Dim)
	var links []string
	for _, l := range f.IssueLinks {
		switch {
		case l.OutwardIssue != nil:
			links = append(links, l.Type.Outward+" "+l.OutwardIssue.Key)
		case l.InwardIssue != nil:
			links = append(links, l.Type.Inward+" "+l.InwardIssue.Key)
		}
	}
	add("Links", strings.Join(links, ", "), work.Dim)

	for _, sub := range f.Subtasks {
		c := work.Check{Name: sub.Key + " " + sub.Fields.Summary}
		if st := sub.Fields.Status; st != nil {
			c.State, c.Tone = st.Name, categoryTone(st.StatusCategory.Key)
		}
		d.Checks = append(d.Checks, c)
	}
	if f.Comment != nil {
		for _, c := range f.Comment.Comments {
			d.Activity = append(d.Activity, work.Note{Author: c.Author.name(), When: c.Created.Time, What: "commented", Text: richText(c.Body)})
		}
	}
	return d, nil
}

// fallbackStatuses are offered for a project the dashboard hasn't shown.
var fallbackStatuses = []string{"To Do", "In Progress", "Done"}

// Actions hand the terminal to acli, which asks before it moves or assigns
// anything. A comment is written in $EDITOR first.
func (s *Service) Actions(it work.Item) []work.Action {
	if it.Kind != work.WorkItem || it.ID == "" {
		return nil
	}
	project, _, _ := strings.Cut(it.ID, "-")
	s.mu.Lock()
	choices := s.statuses[project]
	s.mu.Unlock()
	if len(choices) == 0 {
		choices = fallbackStatuses
	}
	key := it.ID
	return []work.Action{
		{Key: "t", Help: "move", Choices: choices, Pause: true,
			Args: []string{"acli", "jira", "workitem", "transition", "--key", key, "--status", work.ChoiceArg}},
		{Key: "a", Help: "assign to me", Pause: true,
			Args: []string{"acli", "jira", "workitem", "assign", "--key", key, "--assignee", "@me"}},
		{Key: "C", Help: "comment", Compose: true, Pause: true,
			Args: []string{"acli", "jira", "workitem", "comment", "create", "--key", key, "--body", work.BodyArg}},
	}
}
