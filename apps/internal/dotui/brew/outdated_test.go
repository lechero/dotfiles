package brew

import (
	"slices"
	"testing"
)

func TestParseOutdated(t *testing.T) {
	updates, err := ParseOutdated([]byte(`{
	  "formulae": [
	    {"name": "glab", "installed_versions": ["1.120.0"], "current_version": "1.121.0", "pinned": false},
	    {"name": "node", "installed_versions": ["24.1.0", "24.2.0"], "current_version": "24.3.0", "pinned": true}
	  ],
	  "casks": [{"name": "kitty", "installed_versions": "0.44.0", "current_version": "0.45.0"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Update{
		{Name: "glab", Installed: "1.120.0", Latest: "1.121.0"},
		{Name: "node", Installed: "24.1.0, 24.2.0", Latest: "24.3.0", Pinned: true},
		{Name: "kitty", Installed: "0.44.0", Latest: "0.45.0", Cask: true},
	}
	if !slices.Equal(updates, want) {
		t.Errorf("ParseOutdated\n got %+v\nwant %+v", updates, want)
	}
	if _, err := ParseOutdated([]byte("Error: no network")); err == nil {
		t.Error("text that isn't JSON should be an error")
	}
}
