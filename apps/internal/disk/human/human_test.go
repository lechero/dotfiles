package human

import (
	"testing"
	"time"
)

func TestBytes(t *testing.T) {
	cases := map[int64]string{
		0:                      "0 B",
		1023:                   "1023 B",
		1536:                   "1.5 KiB",
		5 << 30:                "5.0 GiB",
		107 << 30:              "107 GiB",
		-(3 << 20):             "-3.0 MiB",
		int64(1.5 * (1 << 40)): "1.5 TiB",
	}
	for in, want := range cases {
		if got := Bytes(in); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	cases := map[time.Duration]string{
		10 * time.Second: "just now",
		time.Minute:      "1 minute ago",
		3 * time.Hour:    "3 hours ago",
		30 * time.Hour:   "yesterday",
		72 * time.Hour:   "3 days ago",
	}
	for d, want := range cases {
		if got := Ago(now, now.Add(-d)); got != want {
			t.Errorf("Ago(-%v) = %q, want %q", d, got, want)
		}
	}
}

func TestCount(t *testing.T) {
	cases := map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567"}
	for in, want := range cases {
		if got := Count(in); got != want {
			t.Errorf("Count(%d) = %q, want %q", in, got, want)
		}
	}
}
