// Package work is what dotui's GitHub, GitLab and Jira tabs show: a
// dashboard of what needs following up, the detail of one item, and the
// commands that act on it. Each service gets them from its own CLI.
package work

import (
	"cmp"
	"context"
	"slices"
	"time"
)

// Tone colours a badge or a field.
type Tone int

// Tones, from neutral to alarming.
const (
	Dim Tone = iota
	Info
	Good
	Warn
	Bad
)

// Badge is a short state shown next to an item, like "checks failing".
type Badge struct {
	Text string
	Tone Tone
}

// Kind is what an item is.
type Kind string

// Kinds of item.
const (
	PullRequest Kind = "pull request" // a merge request too
	Issue       Kind = "issue"
	WorkItem    Kind = "work item" // Jira
	Other       Kind = "other"     // a notification about a commit, release…
)

// Item is one thing to follow up.
type Item struct {
	Kind Kind
	// ID is what the service's commands take, like "owner/repo#12" or
	// "PROJ-7".
	ID      string
	Ref     string // how it's shown: "#12", "!9", "PROJ-7"
	Title   string
	Where   string // repository, project or Jira project
	Branch  string // a pull request's source branch
	Author  string
	URL     string
	Updated time.Time
	Badges  []Badge
	// Next says why it needs you, like "review requested". Items with one
	// make the dashboard's follow-up list.
	Next     string
	NextTone Tone
	// Urgency orders the follow-up list, highest first.
	Urgency int
}

// Section is one list on a service's tab, like "Your pull requests".
type Section struct {
	Key   string // short name for the sub-tab: "PRs"
	Title string
	Items []Item
	// Total is how many the service has, which can be more than Items.
	Total int
	Empty string // what to say when there are none
}

// Dashboard is everything a service's tab shows before opening an item.
type Dashboard struct {
	Account  string // "lechero on github.com"
	Sections []Section
	Fetched  time.Time
}

// FollowUps lists the items that need you, most urgent first, then most
// recently updated, then in the order the service listed them. An item in
// two sections is listed once, for the more urgent reason.
func (d Dashboard) FollowUps() []Item {
	var out []Item
	at := map[string]int{} // where each ID is in out
	for _, s := range d.Sections {
		for _, it := range s.Items {
			if it.Next == "" {
				continue
			}
			i, seen := at[it.ID]
			switch {
			case !seen:
				at[it.ID] = len(out)
				out = append(out, it)
			case it.Urgency > out[i].Urgency:
				out[i] = it
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Item) int {
		if c := cmp.Compare(b.Urgency, a.Urgency); c != 0 {
			return c
		}
		return b.Updated.Compare(a.Updated)
	})
	return out
}

// Detail is what an item's page shows.
type Detail struct {
	Fields []Field
	Checks []Check
	// ChecksTitle heads the checks, when they aren't CI: "Subtasks".
	ChecksTitle string
	Body        string // Markdown
	Activity    []Note // reviews and comments, oldest first
}

// Field is one line of an item's facts, like "Branch  feature → main".
type Field struct {
	Label, Value string
	Tone         Tone
}

// Check is one CI job or status.
type Check struct {
	Name, State string
	Tone        Tone
}

// Note is a comment, a review or a change of state.
type Note struct {
	Author string
	When   time.Time
	What   string // "commented", "approved"…
	Text   string // Markdown
}

// Action is something done to an item, always by handing the terminal to
// the service's CLI, so it can ask its own questions and show its output.
type Action struct {
	Key  string // "v"
	Help string // "review"
	Args []string
	// Dir is where to run it, for commands that need a local clone.
	Dir string
	// Confirm is asked in dotui first, for a command that changes
	// something without asking itself. Empty when the command asks.
	Confirm string
	// Pause waits for Enter afterwards, so the output can be read. Off for
	// commands with a screen of their own, like a pager.
	Pause bool
	// Compose opens $EDITOR first, for a comment. The text replaces the
	// argument BodyArg; an empty text cancels.
	Compose bool
	// Choices are picked from first, like the status to move to. The pick
	// replaces ChoiceArg in Args.
	Choices []string
}

// Placeholders in an action's Args.
const (
	BodyArg   = "{body}"
	ChoiceArg = "{choice}"
)

// Status is whether a service's CLI can be used.
type Status int

// Statuses.
const (
	Unknown Status = iota // before the first check
	NotInstalled
	SignedOut
	Connected
)

// Service is a source of work: GitHub, GitLab or Jira.
type Service interface {
	// Name is the tab's name: "GitHub".
	Name() string
	// CLI is the command the service is read through: "gh".
	CLI() string
	// Package is the CLI's name in the package list, for `dotui install`.
	Package() string
	// Check reports whether the CLI is installed and signed in, and as whom.
	// An error means it couldn't tell, like when the service is unreachable.
	Check(ctx context.Context) (Status, string, error)
	// Connect is the command that signs in, run in the terminal.
	Connect() []string
	Dashboard(ctx context.Context) (Dashboard, error)
	Detail(ctx context.Context, it Item) (Detail, error)
	// Actions are what can be done to it, besides opening it in the
	// browser and copying its link, which work on every item.
	Actions(it Item) []Action
}
