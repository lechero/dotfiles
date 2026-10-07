package work

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Cache keeps each service's last dashboard, so its tab has something to
// show the moment dotui opens, while a fresh one loads.
type Cache struct {
	Dir string // "" keeps nothing
}

// DefaultCache keeps dashboards in dotui's folder of the user cache
// directory: ~/Library/Caches/dotui on macOS.
func DefaultCache() Cache {
	dir, err := os.UserCacheDir()
	if err != nil {
		return Cache{}
	}
	return Cache{Dir: filepath.Join(dir, "dotui")}
}

func (c Cache) path(service string) string {
	return filepath.Join(c.Dir, strings.ToLower(service)+".json")
}

// Load returns the dashboard saved for service, if there is one.
func (c Cache) Load(service string) (Dashboard, bool) {
	var d Dashboard
	if c.Dir == "" {
		return d, false
	}
	data, err := os.ReadFile(c.path(service))
	if err != nil || json.Unmarshal(data, &d) != nil {
		return Dashboard{}, false
	}
	return d, true
}

// Save keeps d for service. It's only a head start for the next run, so a
// failure to write is ignored. Titles of private work are in it, so only
// the user can read it.
func (c Cache) Save(service string, d Dashboard) {
	if c.Dir == "" {
		return
	}
	data, err := json.Marshal(d)
	if err != nil {
		return
	}
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(c.path(service), data, 0o600) // best effort, see above
}
