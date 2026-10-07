package chezmoi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeGit answers git commands by their arguments; a missing answer fails.
func fakeGit(t *testing.T, answers map[string]string) {
	t.Helper()
	old := runGit
	t.Cleanup(func() { runGit = old })
	runGit = func(_ context.Context, _ string, args ...string) (string, error) {
		out, ok := answers[strings.Join(args, " ")]
		if !ok || strings.HasPrefix(out, "error:") {
			return "", errors.New("git " + args[0] + " failed")
		}
		return out, nil
	}
}

const status = `# branch.oid 177c1e6bd06c
# branch.head main
# branch.upstream origin/main
# branch.ab +0 -0
1 .M N... 100644 100644 100644 aaa bbb README.md
? notes.txt
`

func TestSourceState(t *testing.T) {
	head := "177c1e6bd06c0000000000000000000000000000"
	for name, tc := range map[string]struct {
		remote, ancestor string
		want             Remote
	}{
		"same":  {head + "\trefs/heads/main\n", "error:", RemoteSame},
		"newer": {"999aaa0000000000000000000000000000000000\trefs/heads/main\n", "error:", RemoteNewer},
		"older": {"555bbb0000000000000000000000000000000000\trefs/heads/main\n", "", RemoteOlder},
	} {
		t.Run(name, func(t *testing.T) {
			remoteSHA, _, _ := strings.Cut(tc.remote, "\t")
			fakeGit(t, map[string]string{
				"status --porcelain=v2 --branch":                  status,
				"log -1 --format=%h%x00%ct%x00%s":                 "177c1e6\x001791300000\x00fix(dotui): read Jira\n",
				"ls-remote origin refs/heads/main":                tc.remote,
				"rev-parse HEAD":                                  head + "\n",
				"merge-base --is-ancestor " + remoteSHA + " HEAD": tc.ancestor,
			})
			s, err := SourceState(context.Background(), "/src")
			if err != nil {
				t.Fatal(err)
			}
			if s.Branch != "main" || s.Dirty != 2 || s.Head != "177c1e6" || s.Subject != "fix(dotui): read Jira" || !s.When.Equal(time.Unix(1791300000, 0)) {
				t.Errorf("source %+v", s)
			}
			if s.Remote != tc.want || s.RemoteErr != nil {
				t.Errorf("remote %v (%v), want %v", s.Remote, s.RemoteErr, tc.want)
			}
		})
	}
}

func TestSourceStateOffline(t *testing.T) {
	fakeGit(t, map[string]string{
		"status --porcelain=v2 --branch":   status,
		"log -1 --format=%h%x00%ct%x00%s":  "177c1e6\x001791300000\x00fix\n",
		"ls-remote origin refs/heads/main": "error:",
	})
	s, err := SourceState(context.Background(), "/src")
	if err != nil {
		t.Fatal(err)
	}
	if s.Remote != RemoteUnknown || s.RemoteErr == nil {
		t.Errorf("offline, origin is unknown with a reason: %v, %v", s.Remote, s.RemoteErr)
	}
}
