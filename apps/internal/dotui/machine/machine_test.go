package machine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	answers := map[string]string{
		"scutil --get ComputerName": "Miguel’s MacBook Pro",
		"id -F":                     "Miguel Fuentes",
		"sw_vers -productVersion":   "26.5.2",
		"sysctl -n kern.boottime":   "{ sec = 1791297031, usec = 945617 } Tue Oct  6 16:30:31 2026",
		"sysctl -n vm.loadavg":      "{ 13.17 9.14 8.29 }",
		"sysctl -n hw.logicalcpu":   "12",
		"pmset -g batt":             "Now drawing from 'AC Power'\n -InternalBattery-0 (id=23986275)\t80%; AC attached; not charging present: true",
	}
	run := func(_ context.Context, name string, args ...string) (string, error) {
		if out, ok := answers[name+" "+strings.Join(args, " ")]; ok {
			return out, nil
		}
		return "", errors.New("no such command")
	}
	info := read(context.Background(), run)
	want := Info{
		Name: "Miguel’s MacBook Pro", User: "Miguel Fuentes", MacOS: "26.5.2",
		Booted: time.Unix(1791297031, 0), Load: 13.17, CPUs: 12,
		Battery: 80, Charging: false, OnPower: true,
	}
	if info != want {
		t.Errorf("read\n got %+v\nwant %+v", info, want)
	}
	if info.FirstName() != "Miguel" {
		t.Errorf("first name %q", info.FirstName())
	}
}

func TestReadWithNothingAnswering(t *testing.T) {
	info := read(context.Background(), func(context.Context, string, ...string) (string, error) {
		return "", errors.New("not here")
	})
	if info != (Info{Battery: -1}) {
		t.Errorf("nothing readable should leave everything unknown, got %+v", info)
	}
}

func TestParseBattery(t *testing.T) {
	for in, want := range map[string]struct {
		percent           int
		charging, onPower bool
	}{
		"Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1)\t54%; discharging; 3:10 remaining present: true": {54, false, false},
		"Now drawing from 'AC Power'\n -InternalBattery-0 (id=1)\t71%; charging; 0:45 remaining present: true":         {71, true, true},
		"Now drawing from 'AC Power'": {-1, false, true}, // a Mac without a battery
	} {
		p, c, o := parseBattery(in)
		if p != want.percent || c != want.charging || o != want.onPower {
			t.Errorf("parseBattery(%q) = %d, %v, %v; want %+v", in, p, c, o, want)
		}
	}
}
