package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"manage-disk/internal/clean"
	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

func (a *app) overviewKey(key string) tea.Cmd {
	switch key {
	case "r":
		return tea.Batch(a.startScan(), a.discoverAll())
	case "p":
		a.opts.IncludePrivate = !a.opts.IncludePrivate
		if a.opts.IncludePrivate {
			a.flash = "Rescanning with Desktop, Documents, Downloads… — macOS may ask for permission"
		}
		return a.startScan()
	}
	return nil
}

func (a *app) scanStatus() string {
	if a.scanning {
		p := a.prog
		line := fmt.Sprintf("%s Scanning… %s files · %s · %s", a.spin.View(),
			human.Count(p.Files()), human.Bytes(p.Bytes()), human.Duration(time.Since(a.scanStart)))
		if cur := p.Current(); cur != "" {
			line += "  " + sDim.Render(truncLeft(tildePath(a.home, cur), max(10, a.w-60)))
		}
		if wait := p.Waiting(); wait != "" {
			line += "\n" + sYellow.Render("Waiting on "+truncLeft(tildePath(a.home, wait), max(10, a.w-90))+
				" — if macOS shows a permission dialog, either answer is fine; the scan moves on after 15s.")
		}
		return line
	}
	if a.scanErr != nil {
		return sRed.Render("Scan failed: " + a.scanErr.Error())
	}
	if a.res == nil {
		return ""
	}
	return sDim.Render(fmt.Sprintf("Scanned %s files (%s) in %s", human.Count(a.res.Root.Files),
		human.Bytes(a.res.Root.Size), human.Duration(a.res.Took)))
}

func (a *app) overviewView(h int) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	line(a.scanStatus())
	line("")
	if a.res == nil {
		line(sDim.Render("Measuring your home folder. Big homes take a minute; everything else works meanwhile."))
		line("")
		a.writeCleanable(line)
		return b.String()
	}

	total := a.res.Root.Size
	nameW, sizeW := 24, 10
	barW := max(8, min(40, a.w-nameW-sizeW-10))
	line(sHeading.Render("Where your space goes"))
	for _, c := range a.cats {
		name := c.Name
		if c.Partial {
			name += "*"
		}
		if c.Size < 1<<20 {
			if c.Partial {
				line(fmt.Sprintf(" %s %s  %s", pad(name, nameW), padLeft("—", sizeW), sDim.Render("not scanned, see * below")))
			}
			continue
		}
		frac := float64(c.Size) / float64(max(1, total))
		line(fmt.Sprintf(" %s %s  %s %s", pad(name, nameW), padLeft(human.Bytes(c.Size), sizeW),
			bar(frac, barW, sAccent), sDim.Render(fmt.Sprintf("%3.0f%%", frac*100))))
	}
	line("")

	// Whatever room is left goes to the biggest spots.
	used := strings.Count(b.String(), "\n")
	room := h - used - 9
	if room > 2 && len(a.hot) > 0 {
		line(sHeading.Render("Biggest spots"))
		tags := a.cleanTags()
		for i, n := range a.hot {
			if i >= room {
				break
			}
			path := tildePath(a.home, n.Path())
			tag, tagW := "", 0
			if t, ok := tags[n.Path()]; ok {
				tag, tagW = "  "+sGreen.Render(t), len(t)+2
			}
			line(fmt.Sprintf(" %s  %s%s", padLeft(human.Bytes(n.Size), sizeW),
				truncLeft(path, max(10, a.w-sizeW-4-tagW)), tag))
		}
		line("")
	}
	a.writeCleanable(line)
	if nr := a.res.NoResponse; len(nr) > 0 {
		var names []string
		for _, p := range nr {
			names = append(names, tildePath(a.home, p))
		}
		line("")
		line(sYellow.Render("! Never answered, so not measured (macOS may have asked for permission): " + strings.Join(names, ", ")))
	}
	if len(a.res.Private) > 0 && !a.opts.IncludePrivate {
		line("")
		line(sDim.Render("* Not scanned, macOS guards them: " + strings.Join(scan.SummarizePrivate(a.res.Private), ", ") +
			". Press p to include them (macOS may ask for permission)."))
	}
	return b.String()
}

func (a *app) writeCleanable(line func(string)) {
	var t1, t2 int64
	var n1, n2, blocked1 int
	loading := false
	for _, ts := range a.tasks {
		loading = loading || ts.loading || ts.waiting
		size, n := ts.ready()
		if ts.task.Tier == clean.Tier1 {
			if ts.enabled {
				t1 += size
				n1 += n
				blocked1 += ts.blocked()
			}
		} else {
			t2 += size
			n2 += n
		}
	}
	head := sHeading.Render("Cleanable now")
	if loading {
		head += "  " + a.spin.View() + sDim.Render(" still measuring")
	}
	line(head)
	t1s := fmt.Sprintf(" Tier 1  %s in %s", sGreen.Render(human.Bytes(t1)), plural(n1, "item"))
	if blocked1 > 0 {
		t1s += sYellow.Render(fmt.Sprintf("  (%d blocked by running apps)", blocked1))
	}
	line(t1s + sDim.Render("  — caches, on by default"))
	line(fmt.Sprintf(" Tier 2  %s in %s", sYellow.Render(human.Bytes(t2)), plural(n2, "item")) +
		sDim.Render("  — costs a rebuild or re-download, off by default"))
	line(sDim.Render(" Press 3 to review and clean."))
}

func tildePath(home, p string) string {
	if p == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(p, home+"/"); ok {
		return "~/" + rel
	}
	return p
}

// cleanTags maps paths that a cleanup would delete to a short label.
func (a *app) cleanTags() map[string]string {
	tags := map[string]string{}
	for _, ts := range a.tasks {
		if ts.task.Run != nil {
			continue // commands decide what they delete
		}
		for _, it := range ts.items {
			for _, p := range it.Paths {
				tags[p] = fmt.Sprintf("T%d %s", ts.task.Tier, ts.task.Title)
			}
		}
	}
	return tags
}
