package gitlab

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

// fakeGlab answers glab commands that start with one of answers' keys.
func fakeGlab(t *testing.T, answers map[string]string) *Service {
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
	return &Service{Run: run, Host: DefaultHost}
}

const api = "glab api --hostname gitlab.com "

const dashboardJSON = `{"data": {"currentUser": {
  "username": "me",
  "authored": {"count": 3, "nodes": [
    {"iid": "1", "title": "Red pipeline", "webUrl": "https://gitlab.com/g/app/-/merge_requests/1", "updatedAt": "2026-10-01T10:00:00Z",
     "sourceBranch": "red", "author": {"username": "me"}, "project": {"fullPath": "g/app"},
     "headPipeline": {"status": "FAILED"}, "approved": false, "detailedMergeStatus": "CI_MUST_PASS"},
    {"iid": "2", "title": "Good to go", "webUrl": "https://gitlab.com/g/app/-/merge_requests/2", "updatedAt": "2026-10-02T10:00:00Z",
     "sourceBranch": "green", "author": {"username": "me"}, "project": {"fullPath": "g/app"},
     "headPipeline": {"status": "SUCCESS"}, "approved": true, "detailedMergeStatus": "MERGEABLE"},
    {"iid": "3", "title": "Behind", "webUrl": "https://gitlab.com/g/sub/lib/-/merge_requests/3", "draft": true, "updatedAt": "2026-10-03T10:00:00Z",
     "sourceBranch": "old", "author": {"username": "me"}, "project": {"fullPath": "g/sub/lib"},
     "headPipeline": null, "approved": false, "detailedMergeStatus": "NEED_REBASE"}
  ]},
  "review": {"count": 1, "nodes": [
    {"iid": "7", "title": "Please review", "webUrl": "https://gitlab.com/g/svc/-/merge_requests/7", "updatedAt": "2026-09-30T10:00:00Z",
     "sourceBranch": "feat", "author": {"username": "pat"}, "project": {"fullPath": "g/svc"}}
  ]},
  "todos": {"nodes": [
    {"action": "review_requested", "targetType": "MERGEREQUEST", "createdAt": "2026-10-04T10:00:00Z", "author": {"username": "pat"},
     "project": {"fullPath": "g/svc"}, "target": {"webUrl": "https://gitlab.com/g/svc/-/merge_requests/7", "iid": "7", "title": "Please review"}},
    {"action": "mentioned", "targetType": "ISSUE", "createdAt": "2026-10-04T11:00:00Z", "author": {"username": "sam"},
     "project": {"fullPath": "g/app"}, "target": {"webUrl": "https://gitlab.com/g/app/-/issues/4", "iid": "4", "title": "Who knows?"}},
    {"action": "marked", "targetType": "EPIC", "createdAt": "2026-10-04T12:00:00Z", "author": {"username": "me"},
     "project": null, "target": {"webUrl": "https://gitlab.com/groups/g/-/epics/1"}}
  ]}
}}}`

const issuesJSON = `[{"iid": 8, "title": "Broken", "web_url": "https://gitlab.com/g/app/-/issues/8", "updated_at": "2026-09-29T10:00:00Z",
  "author": {"username": "sam"}, "labels": ["bug", "p1", "ui", "extra"], "references": {"full": "g/app#8"}}]`

func TestDashboard(t *testing.T) {
	s := fakeGlab(t, map[string]string{api + "graphql": dashboardJSON, api + "issues?": issuesJSON})
	d, err := s.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Account != "me on gitlab.com" {
		t.Errorf("account %q", d.Account)
	}
	var keys []string
	for _, sec := range d.Sections {
		keys = append(keys, fmt.Sprintf("%s %d/%d", sec.Key, len(sec.Items), sec.Total))
	}
	if want := []string{"Reviews 1/1", "MRs 3/3", "Issues 1/1", "To-dos 3/3"}; !slices.Equal(keys, want) {
		t.Errorf("sections %v, want %v", keys, want)
	}

	mrs := d.Sections[1].Items
	if mrs[0].ID != "g/app!1" || mrs[0].Ref != "!1" || mrs[0].Branch != "red" {
		t.Errorf("first MR %+v", mrs[0])
	}
	var labels []string
	for _, b := range d.Sections[2].Items[0].Badges {
		labels = append(labels, b.Text)
	}
	if got := strings.Join(labels, " "); got != "bug p1 ui" {
		t.Errorf("issue labels %q, want the first three", got)
	}
	todos := d.Sections[3].Items
	if todos[1].Kind != work.Issue || todos[1].ID != "g/app#4" {
		t.Errorf("a to-do on an issue should be that issue: %+v", todos[1])
	}
	if todos[2].Kind != work.Other || todos[2].Next != "to-do" {
		t.Errorf("an epic to-do is a plain to-do: %+v", todos[2])
	}

	var order []string
	for _, it := range d.FollowUps() {
		order = append(order, it.ID+" "+it.Next)
	}
	want := []string{
		"g/svc!7 review requested", // a review and a to-do, listed once
		"g/app!1 pipeline failed",
		"g/sub/lib!3 need rebase",
		"g/app!2 ready to merge",
		"g/app#4 mentioned",
		"g/app#8 assigned to you",
		"https://gitlab.com/groups/g/-/epics/1 to-do",
	}
	if !slices.Equal(order, want) {
		t.Errorf("follow-ups\n got %q\nwant %q", order, want)
	}
}

func TestDashboardGraphQLError(t *testing.T) {
	s := fakeGlab(t, map[string]string{
		api + "graphql": `{"errors": [{"message": "Field 'x' doesn't exist"}]}`,
		api + "issues?": "[]",
	})
	if _, err := s.Dashboard(context.Background()); err == nil || !strings.Contains(err.Error(), "doesn't exist") {
		t.Errorf("a GraphQL error should fail the dashboard, got %v", err)
	}
}

func TestCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		answer  string
		status  work.Status
		account string
		err     bool
	}{
		"signed in":   {`{"username": "me"}`, work.Connected, "me on gitlab.com", false},
		"signed out":  {"error:glab: 401 Unauthorized (HTTP 401)", work.SignedOut, "", false},
		"unreachable": {"error:glab: dial tcp: lookup gitlab.com: no such host", work.Unknown, "", true},
	} {
		t.Run(name, func(t *testing.T) {
			s := fakeGlab(t, map[string]string{api + "user": tc.answer})
			status, account, err := s.Check(context.Background())
			if status != tc.status || account != tc.account || (err != nil) != tc.err {
				t.Errorf("Check = %v, %q, %v", status, account, err)
			}
		})
	}
	missing := &Service{Host: DefaultHost, Run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("glab: %w", exec.ErrNotFound)
	}}
	if status, _, err := missing.Check(context.Background()); status != work.NotInstalled || err != nil {
		t.Errorf("without glab: %v, %v", status, err)
	}
}

func TestMergeRequestDetail(t *testing.T) {
	s := fakeGlab(t, map[string]string{api + `graphql -f query=query { project(fullPath: "g/sub/lib") { mergeRequest(iid: "3")`: `{"data": {"project": {"mergeRequest": {
	  "state": "opened", "draft": true, "sourceBranch": "old", "targetBranch": "main", "description": "Rebase me",
	  "approved": false, "detailedMergeStatus": "NEED_REBASE", "author": {"username": "me"},
	  "approvedBy": {"nodes": [{"username": "pat"}]}, "labels": {"nodes": []},
	  "diffStatsSummary": {"additions": 3, "deletions": 4, "fileCount": 2},
	  "headPipeline": {"status": "FAILED", "jobs": {"nodes": [
	    {"name": "lint", "status": "SUCCESS", "stage": {"name": "test"}},
	    {"name": "unit", "status": "FAILED", "stage": {"name": "test"}},
	    {"name": "deploy", "status": "MANUAL", "stage": {"name": "ship"}}
	  ]}},
	  "notes": {"nodes": [
	    {"author": {"username": "pat"}, "body": "approved this merge request", "createdAt": "2026-10-03T12:00:00Z", "system": true},
	    {"author": {"username": "sam"}, "body": "Looks off", "createdAt": "2026-10-03T11:00:00Z", "system": false}
	  ]}
	}}}}`})
	d, err := s.Detail(context.Background(), work.Item{Kind: work.PullRequest, ID: "g/sub/lib!3"})
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	for _, f := range d.Fields {
		fields[f.Label] = f.Value
	}
	for label, want := range map[string]string{
		"State": "opened, draft", "Branch": "old → main", "Changes": "+3 −4 in 2 files",
		"Approved by": "pat", "Merge": "need rebase", "Pipeline": "failed",
	} {
		if fields[label] != want {
			t.Errorf("%s = %q, want %q", label, fields[label], want)
		}
	}
	var checks []string
	for _, c := range d.Checks {
		checks = append(checks, c.Name+" "+c.State)
	}
	if want := []string{"test › unit failed", "test › lint success", "ship › deploy manual"}; !slices.Equal(checks, want) {
		t.Errorf("checks %q, want %q", checks, want)
	}
	if len(d.Activity) != 2 || d.Activity[0].What != "commented" || d.Activity[1].What != "approved this merge request" {
		t.Errorf("activity %+v, want oldest first, GitLab's own notes as what happened", d.Activity)
	}
}

func TestMissingMergeRequest(t *testing.T) {
	s := fakeGlab(t, map[string]string{api + "graphql": `{"data": {"project": {"mergeRequest": null}}}`})
	if _, err := s.Detail(context.Background(), work.Item{Kind: work.PullRequest, ID: "g/app!99"}); err == nil {
		t.Error("a merge request GitLab doesn't return should be an error")
	}
}

func TestActions(t *testing.T) {
	s := &Service{Host: DefaultHost, Clones: func() work.Clones { return work.Clones{"gitlab.com/g/app": "/src/app"} }}
	keys := func(it work.Item) string {
		var out []string
		for _, a := range s.Actions(it) {
			out = append(out, a.Key)
		}
		return strings.Join(out, " ")
	}
	mr := work.Item{Kind: work.PullRequest, ID: "g/app!1", Ref: "!1", Branch: "red"}
	if got := keys(mr); got != "d c v C m b" {
		t.Errorf("merge request actions %q", got)
	}
	for _, a := range s.Actions(mr) {
		if !slices.Contains(a.Args, "https://gitlab.com/g/app") && a.Key != "b" {
			t.Errorf("%s should name the project by URL: %v", a.Help, a.Args)
		}
		if a.Key == "v" && a.Confirm == "" {
			t.Error("glab approves without asking, so dotui should ask")
		}
		if a.Key == "c" && (a.Pause || !slices.Contains(a.Args, "red")) {
			t.Errorf("the pipeline view is a screen of its own, for the branch: %+v", a)
		}
	}
	if got := keys(work.Item{Kind: work.PullRequest, ID: "g/other!2"}); got != "d v C m" {
		t.Errorf("without a branch or a clone: %q", got)
	}
	if got := keys(work.Item{Kind: work.Issue, ID: "g/app#8"}); got != "C" {
		t.Errorf("issue actions %q", got)
	}
}
