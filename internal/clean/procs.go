package clean

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Proc is one running process, as far as an unprivileged user can see it.
type Proc struct {
	PID     int
	Exe     string // executable path, from `ps -o comm`
	Command string // full command line
	Cwd     string // working directory, when lsof could tell
}

// Procs is a snapshot of running processes, taken before discovering what to
// clean so nothing gets deleted out from under a program that is using it.
type Procs []Proc

// SnapshotProcs reads the process table with ps and working directories with
// lsof. It leaves out this process itself.
func SnapshotProcs(ctx context.Context) (Procs, error) {
	exes, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,comm=").Output()
	if err != nil {
		return nil, err
	}
	cmds, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	// lsof exits 1 when it cannot read some processes; what it printed is still good.
	cwds, _ := exec.CommandContext(ctx, "lsof", "-a", "-d", "cwd", "-Fpn").Output()
	ps := parseProcs(exes, cmds, cwds)
	self := os.Getpid()
	out := ps[:0]
	for _, p := range ps {
		if p.PID != self {
			out = append(out, p)
		}
	}
	return out, nil
}

func parseProcs(exes, cmds, cwds []byte) Procs {
	byPID := map[int]*Proc{}
	var order []int
	get := func(pid int) *Proc {
		if p, ok := byPID[pid]; ok {
			return p
		}
		p := &Proc{PID: pid}
		byPID[pid] = p
		order = append(order, pid)
		return p
	}
	eachPIDLine(exes, func(pid int, rest string) { get(pid).Exe = rest })
	eachPIDLine(cmds, func(pid int, rest string) { get(pid).Command = rest })

	pid := -1
	sc := bufio.NewScanner(bytes.NewReader(cwds))
	for sc.Scan() {
		line := sc.Text()
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			n, err := strconv.Atoi(line[1:])
			if err != nil {
				pid = -1
				continue
			}
			pid = n
		case 'n':
			if p, ok := byPID[pid]; ok {
				p.Cwd = line[1:]
			}
		}
	}

	out := make(Procs, 0, len(order))
	for _, id := range order {
		out = append(out, *byPID[id])
	}
	return out
}

// eachPIDLine calls fn for every "  <pid> <rest>" line of ps output.
func eachPIDLine(b []byte, fn func(pid int, rest string)) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<10), 4<<20) // command lines can be very long
	for sc.Scan() {
		line := strings.TrimLeft(sc.Text(), " ")
		idx := strings.IndexByte(line, ' ')
		if idx <= 0 {
			continue
		}
		pid, err := strconv.Atoi(line[:idx])
		if err != nil {
			continue
		}
		fn(pid, strings.TrimLeft(line[idx+1:], " "))
	}
}

// App returns a running process whose executable ends in suffix
// ("MacOS/Google Chrome"), or nil.
func (ps Procs) App(suffix string) *Proc {
	for i := range ps {
		if strings.HasSuffix(ps[i].Exe, suffix) {
			return &ps[i]
		}
	}
	return nil
}

// Matching returns the processes whose command line contains any of subs.
func (ps Procs) Matching(subs ...string) Procs {
	var out Procs
	for _, p := range ps {
		for _, s := range subs {
			if strings.Contains(p.Command, s) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// Program returns the processes whose executable is called name ("go"), so a
// shell that merely mentions "go build" in its command line does not count.
func (ps Procs) Program(name string) Procs {
	var out Procs
	for _, p := range ps {
		if filepath.Base(p.Exe) == name {
			out = append(out, p)
		}
	}
	return out
}

// Using returns a process that works inside dir or runs something from it, or nil.
func (ps Procs) Using(dir string) *Proc {
	for i := range ps {
		p := &ps[i]
		if within(p.Cwd, dir) || strings.Contains(p.Command, dir+"/") || strings.HasSuffix(p.Command, dir) {
			return p
		}
	}
	return nil
}

// Name is a short name for messages: "node", "Google Chrome".
func (p *Proc) Name() string {
	if p.Exe == "" {
		return "pid " + strconv.Itoa(p.PID)
	}
	name := filepath.Base(p.Exe)
	// Claude.app starts its sessions through a "disclaimer" launcher; name what it launched.
	if name == "disclaimer" {
		if _, rest, ok := strings.Cut(p.Command, " -- "); ok {
			if i := strings.Index(rest, "/Contents/MacOS/"); i >= 0 {
				prog, _, _ := strings.Cut(rest[i+len("/Contents/MacOS/"):], " ")
				return prog
			}
			if f := strings.Fields(rest); len(f) > 0 {
				return filepath.Base(f[0])
			}
		}
	}
	return name
}

// within reports whether path is dir or inside it.
func within(path, dir string) bool {
	return path != "" && (path == dir || strings.HasPrefix(path, dir+"/"))
}
