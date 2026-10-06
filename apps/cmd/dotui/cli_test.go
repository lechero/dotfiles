package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPackages = `
packages:
  - {name: fish, prio: 1, note: Shell}
  - {name: tmux, prio: 1, note: Multiplexer}
  - {name: charmbracelet/tap/crush, prio: 2, note: AI}
  - {name: rancher, cask: true, prio: 2}
  - {name: stow, prio: 4}
`

// fakeTools puts fake brew and chezmoi on PATH. brew reports fish and the
// rancher cask as installed and logs every call to the returned file.
func fakeTools(t *testing.T) (packages, brewLog string) {
	t.Helper()
	dir := t.TempDir()
	brewLog = filepath.Join(dir, "brew.log")
	brew := `#!/bin/sh
echo "$@" >> "` + brewLog + `"
case "$1 $2" in
  "list --formula") printf 'fish\n' ;;
  "list --cask") printf 'rancher\n' ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "brew"), []byte(brew), 0o755); err != nil {
		t.Fatal(err)
	}
	packages = filepath.Join(dir, "packages.yaml")
	if err := os.WriteFile(packages, []byte(testPackages), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin:/bin")
	t.Setenv("DOTUI_PACKAGES", "")
	return packages, brewLog
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, strings.NewReader("\n"), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestList(t *testing.T) {
	packages, _ := fakeTools(t)
	code, out, _ := runCLI("list", "--file", packages, "--missing")
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"tmux", "charmbracelet/tap/crush", "stow"} {
		if !strings.Contains(out, want) {
			t.Errorf("list --missing is missing %s:\n%s", want, out)
		}
	}
	for _, installed := range []string{"fish ", "rancher"} {
		if strings.Contains(out, installed) {
			t.Errorf("list --missing shows installed %s:\n%s", installed, out)
		}
	}
}

func TestCheck(t *testing.T) {
	packages, _ := fakeTools(t)
	code, out, _ := runCLI("check", "--file", packages, "--prio", "1")
	if code != exitFailure || !strings.Contains(out, "1 of 2 packages up to priority 1 is missing") || !strings.Contains(out, "tmux") {
		t.Errorf("check --prio 1: exit %d\n%s", code, out)
	}

	if err := os.WriteFile(packages, []byte("packages:\n  - {name: fish, prio: 1}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runCLI("check", "--file", packages)
	if code != exitOK || !strings.Contains(out, "All 1 package up to priority 2 is installed") {
		t.Errorf("check with everything installed: exit %d\n%s", code, out)
	}
}

func TestInstallRunsBrewForWhatsMissing(t *testing.T) {
	packages, brewLog := fakeTools(t)
	code, out, errOut := runCLI("install", "--file", packages)
	if code != exitOK {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	log, err := os.ReadFile(brewLog)
	if err != nil {
		t.Fatal(err)
	}
	want := "tap charmbracelet/tap\ninstall tmux charmbracelet/tap/crush\n"
	if got := strings.SplitN(string(log), "list --cask -1\n", 2)[1]; got != want {
		t.Errorf("brew calls after listing:\n%s\nwant\n%s", got, want)
	}
}

func TestInstallNamed(t *testing.T) {
	packages, brewLog := fakeTools(t)
	if code, _, errOut := runCLI("install", "--file", packages, "stow", "fish"); code != exitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	log, _ := os.ReadFile(brewLog)
	if !strings.HasSuffix(string(log), "install stow\n") {
		t.Errorf("brew log:\n%s\nwant it to install only stow, since fish is installed", log)
	}

	code, _, errOut := runCLI("install", "--file", packages, "nope")
	if code != exitUsage || !strings.Contains(errOut, "nope isn't in the package list") {
		t.Errorf("unknown name: exit %d: %s", code, errOut)
	}
}

func TestUsageErrors(t *testing.T) {
	packages, _ := fakeTools(t)
	tests := map[string][]string{
		"unknown command":    {"frobnicate"},
		"bad prio":           {"check", "--file", packages, "--prio", "9"},
		"unknown flag":       {"list", "--colour"},
		"_exec with nothing": {"_exec"},
	}
	for name, args := range tests {
		if code, _, _ := runCLI(args...); code != exitUsage {
			t.Errorf("%s: exit %d, want %d", name, code, exitUsage)
		}
	}
	if code, out, _ := runCLI("help"); code != exitOK || !strings.Contains(out, "Usage:") {
		t.Errorf("help: exit %d\n%s", code, out)
	}
}

func TestExecThenWait(t *testing.T) {
	fakeTools(t)
	code, out, _ := runCLI("_exec", "sh", "-c", "echo hi")
	if code != exitOK || !strings.Contains(out, "hi") || !strings.Contains(out, "Press Enter") {
		t.Errorf("exit %d\n%s", code, out)
	}
	if code, _, errOut := runCLI("_exec", "sh", "-c", "exit 3"); code != exitFailure || !strings.Contains(errOut, "sh failed") {
		t.Errorf("failing command: exit %d: %s", code, errOut)
	}
}
