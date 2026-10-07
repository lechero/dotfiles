// Package machine reads what dotui's dashboard says about this Mac: its
// name, macOS version, uptime, load and battery.
package machine

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Info is this Mac, as far as it could be read. Anything unread is zero.
type Info struct {
	Name     string // "Miguel's MacBook Pro"
	User     string // the full name of whoever is signed in
	MacOS    string // "26.5.2"
	Booted   time.Time
	Load     float64 // over the last minute
	CPUs     int
	Battery  int // percent; -1 without a battery
	Charging bool
	OnPower  bool
}

// Runner runs a command and returns its output; tests replace it.
type Runner func(ctx context.Context, name string, args ...string) (string, error)

func execRun(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// Read reads this Mac.
func Read(ctx context.Context) Info { return read(ctx, execRun) }

func read(ctx context.Context, run Runner) Info {
	info := Info{Battery: -1}
	get := func(name string, args ...string) string {
		out, err := run(ctx, name, args...)
		if err != nil {
			return ""
		}
		return out
	}
	info.Name = get("scutil", "--get", "ComputerName")
	info.User = get("id", "-F")
	info.MacOS = get("sw_vers", "-productVersion")
	info.Booted = parseBoottime(get("sysctl", "-n", "kern.boottime"))
	info.Load = parseLoad(get("sysctl", "-n", "vm.loadavg"))
	info.CPUs, _ = strconv.Atoi(get("sysctl", "-n", "hw.logicalcpu"))
	info.Battery, info.Charging, info.OnPower = parseBattery(get("pmset", "-g", "batt"))
	return info
}

var boottime = regexp.MustCompile(`sec = (\d+)`)

// parseBoottime reads `sysctl -n kern.boottime`: { sec = 1791297031, usec = 945617 } Tue Oct  6 16:30:31 2026
func parseBoottime(s string) time.Time {
	m := boottime.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}
	}
	secs, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(secs, 0)
}

// parseLoad reads `sysctl -n vm.loadavg`: { 13.17 9.14 8.29 }
func parseLoad(s string) float64 {
	fields := strings.Fields(strings.Trim(s, "{} "))
	if len(fields) == 0 {
		return 0
	}
	load, _ := strconv.ParseFloat(fields[0], 64)
	return load
}

var batteryPercent = regexp.MustCompile(`(\d+)%;\s*([^;]+)`)

// parseBattery reads `pmset -g batt`:
//
//	Now drawing from 'AC Power'
//	 -InternalBattery-0 (id=23986275)	80%; AC attached; not charging present: true
func parseBattery(s string) (percent int, charging, onPower bool) {
	onPower = strings.Contains(s, "'AC Power'")
	m := batteryPercent.FindStringSubmatch(s)
	if m == nil {
		return -1, false, onPower
	}
	percent, _ = strconv.Atoi(m[1])
	state := strings.TrimSpace(m[2])
	charging = state == "charging" || strings.HasPrefix(state, "charging")
	return percent, charging, onPower
}

// FirstName is the first word of the user's full name.
func (i Info) FirstName() string {
	first, _, _ := strings.Cut(strings.TrimSpace(i.User), " ")
	return first
}
