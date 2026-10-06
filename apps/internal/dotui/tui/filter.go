package tui

import (
	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
)

// visiblePackages returns the rows to show: packages up to maxPrio, and only
// the missing ones if missingOnly. Without an inventory nothing is known to
// be missing yet, so missingOnly has no effect.
func visiblePackages(c catalog.Catalog, inv *brew.Inventory, maxPrio int, missingOnly bool) []pkgItem {
	var items []pkgItem
	for _, p := range c.UpTo(maxPrio) {
		it := pkgItem{pkg: p}
		if inv != nil {
			it.state, it.known = inv.State(p), true
		}
		if missingOnly && it.known && it.state != brew.Missing {
			continue
		}
		items = append(items, it)
	}
	return items
}

// tally counts installed and total packages for one priority.
type tally struct {
	installed, total int
}

// prioTallies counts packages per priority, indexed by priority.
func prioTallies(c catalog.Catalog, inv *brew.Inventory) [catalog.MaxPrio + 1]tally {
	var t [catalog.MaxPrio + 1]tally
	for _, p := range c.Packages {
		t[p.Prio].total++
		if inv != nil && inv.State(p) != brew.Missing {
			t[p.Prio].installed++
		}
	}
	return t
}
