package scan

import (
	"strconv"
	"strings"
)

// Reasons a directory was not read.
const (
	SkipPrivacy     = "macOS privacy"
	SkipNoAccess    = "no access"
	SkipOtherVolume = "other volume"
	SkipNoResponse  = "no response — macOS may be asking for permission"
)

// promptingRel are places under $HOME that macOS guards with a consent dialog
// ("… would like to access files in your Documents folder", "… data from other
// apps"). Touching them without Full Disk Access pops a dialog and blocks until
// someone answers, so a scan skips them unless Options.IncludePrivate is set.
// Library/Containers is skipped whole: even Docker's (where its VM disk lives)
// asked for permission when run from a terminal on 2026-09-28.
var promptingRel = []string{
	"Desktop", "Documents", "Downloads", "Pictures", "Movies", "Music",
	"Library/Containers", "Library/Group Containers", "Library/Mobile Documents",
	"Library/CloudStorage", "Library/Calendars", "Library/Reminders", "Library/Photos",
	"Library/HomeKit", "Library/Application Support/AddressBook",
	"Library/Application Support/FileProvider", "Library/Application Support/CloudDocs",
	// Media & Apple Music permission: both hung a terminal scan on 2026-09-28.
	"Library/Caches/com.apple.Music", "Library/Caches/com.apple.TV",
	"Library/Caches/com.apple.podcasts", "Library/Caches/com.apple.iTunes",
}

// fdaOnlyRel only Full Disk Access can read. They fail with EPERM rather than
// prompting, so skipping them only saves noise.
var fdaOnlyRel = []string{
	".Trash", "Library/Mail", "Library/Messages", "Library/Safari", "Library/Cookies",
	"Library/Suggestions", "Library/Biome", "Library/Accounts", "Library/Daemon Containers",
	"Library/IdentityServices", "Library/Sharing", "Library/Trial", "Library/Metadata",
	"Library/PersonalizationPortrait", "Library/DuetExpertCenter", "Library/Autosave Information",
	"Library/Shortcuts", "Library/News", "Library/Stocks", "Library/Weather", "Library/Passes",
	"Library/StatusKit", "Library/Keychains", "Library/Assistant", "Library/ContainerManager",
	"Library/CoreFollowUp", "Library/Staging", "Library/Intelligence", "Library/IntelligencePlatform",
	"Library/Application Support/CallHistoryDB", "Library/Application Support/CallHistoryTransactions",
	"Library/Application Support/com.apple.TCC", "Library/Application Support/Knowledge",
	"Library/Application Support/MobileSync", "Library/Application Support/com.apple.sharedfilelist",
}

var (
	promptingSet = toSet(promptingRel)
	fdaOnlySet   = toSet(fdaOnlyRel)
)

// PrivacyReason says why rel (a path relative to $HOME) should not be read,
// or "" when it can be.
func PrivacyReason(rel string, includePrivate bool) string {
	if !includePrivate && (promptingSet[rel] || fdaOnlySet[rel]) {
		return SkipPrivacy
	}
	return ""
}

// Prompting reports whether macOS would ask before reading rel — the skipped
// places worth telling people they can include (the rest only Full Disk Access opens).
func Prompting(rel string) bool { return promptingSet[rel] }

// InsidePrompting reports whether rel is a consent-guarded place or inside one.
func InsidePrompting(rel string) bool {
	for p := rel; p != ""; {
		if promptingSet[p] {
			return true
		}
		i := strings.LastIndexByte(p, '/')
		if i < 0 {
			break
		}
		p = p[:i]
	}
	return false
}

// SummarizePrivate names the skipped places macOS would ask about: top-level
// ones by name, app containers as such, and the rest as a count.
func SummarizePrivate(rels []string) []string {
	var out []string
	deeper := 0
	for _, r := range rels {
		switch {
		case !Prompting(r):
		case r == "Library/Containers":
			out = append(out, "app containers (Docker's VM disk too)")
		case !strings.Contains(r, "/"):
			out = append(out, r)
		default:
			deeper++
		}
	}
	if deeper > 0 {
		out = append(out, strconv.Itoa(deeper)+" more in Library")
	}
	return out
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
