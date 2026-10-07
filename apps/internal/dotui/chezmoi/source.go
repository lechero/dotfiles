package chezmoi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Remote is how origin's copy of the source's branch compares with it.
type Remote int

const (
	RemoteUnknown Remote = iota // couldn't ask origin
	RemoteSame
	RemoteNewer // origin has commits this clone doesn't: `chezmoi update` gets them
	RemoteOlder // this clone has commits origin doesn't: they aren't pushed
)

// Source is the state of chezmoi's source repository.
type Source struct {
	Dir     string
	Branch  string
	Head    string // short commit
	Subject string
	When    time.Time
	// Dirty counts files changed and not committed.
	Dirty  int
	Remote Remote
	// RemoteErr says why origin couldn't be asked, like being offline.
	RemoteErr error
}

// runGit runs git in dir; tests replace it.
var runGit = func(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s", args[0], msg)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}

// SourceDir is source, or chezmoi's own source directory when it's "".
func SourceDir(ctx context.Context, source string) (string, error) {
	if source != "" {
		return source, nil
	}
	out, err := exec.CommandContext(ctx, "chezmoi", "source-path").Output()
	if err != nil {
		return "", fmt.Errorf("chezmoi source-path: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SourceState reads the source repository, and asks origin about its
// branch with `git ls-remote`, which changes nothing locally.
func SourceState(ctx context.Context, source string) (Source, error) {
	dir, err := SourceDir(ctx, source)
	if err != nil {
		return Source{}, err
	}
	s := Source{Dir: dir}
	status, err := runGit(ctx, dir, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return s, err
	}
	s.Branch, s.Dirty = parseStatus(status)
	log, err := runGit(ctx, dir, "log", "-1", "--format=%h%x00%ct%x00%s")
	if err != nil {
		return s, err
	}
	if parts := strings.SplitN(strings.TrimSpace(log), "\x00", 3); len(parts) == 3 {
		s.Head, s.Subject = parts[0], parts[2]
		if secs, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			s.When = time.Unix(secs, 0)
		}
	}
	s.Remote, s.RemoteErr = remoteState(ctx, dir, s.Branch)
	return s, nil
}

// parseStatus reads `git status --porcelain=v2 --branch`: the branch, and
// how many files changed.
func parseStatus(out string) (branch string, dirty int) {
	for line := range strings.SplitSeq(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			branch = strings.TrimPrefix(line, "# branch.head ")
		case line != "" && !strings.HasPrefix(line, "#"):
			dirty++
		}
	}
	return branch, dirty
}

func remoteState(ctx context.Context, dir, branch string) (Remote, error) {
	if branch == "" || branch == "(detached)" {
		return RemoteUnknown, errors.New("the source isn't on a branch")
	}
	out, err := runGit(ctx, dir, "ls-remote", "origin", "refs/heads/"+branch)
	if err != nil {
		return RemoteUnknown, err
	}
	remote, _, _ := strings.Cut(strings.TrimSpace(out), "\t")
	if remote == "" {
		return RemoteUnknown, fmt.Errorf("origin has no branch %s", branch)
	}
	head, err := runGit(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return RemoteUnknown, err
	}
	if strings.TrimSpace(head) == remote {
		return RemoteSame, nil
	}
	// origin's commit in this clone's history means this clone is ahead;
	// anything else, including a commit never fetched, means origin moved on.
	if _, err := runGit(ctx, dir, "merge-base", "--is-ancestor", remote, "HEAD"); err == nil {
		return RemoteOlder, nil
	}
	return RemoteNewer, nil
}
