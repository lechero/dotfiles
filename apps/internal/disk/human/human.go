// Package human formats sizes and times for people.
package human

import (
	"fmt"
	"time"
)

// Bytes formats n in IEC units (KiB, MiB, GiB), the units `df -h` and `du -h` use.
func Bytes(n int64) string {
	if n < 0 {
		return "-" + Bytes(-n)
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit && exp < 4; m /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	suffix := [...]string{"KiB", "MiB", "GiB", "TiB", "PiB"}[exp]
	if v >= 100 {
		return fmt.Sprintf("%.0f %s", v, suffix)
	}
	return fmt.Sprintf("%.1f %s", v, suffix)
}

// Ago says how long before now t was, coarsely: "just now", "3 hours ago", "yesterday".
func Ago(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute") + " ago"
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour") + " ago"
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

// Duration rounds d to what a person cares about: "850ms", "42s", "3m10s".
func Duration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(time.Second).String()
}

// Count formats n with thousands separators: 1234567 → "1,234,567".
func Count(n int64) string {
	if n < 0 {
		return "-" + Count(-n)
	}
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
