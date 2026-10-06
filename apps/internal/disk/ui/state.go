package ui

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// What the interface remembers between runs, next to the run history in
// ~/Library/Application Support/manage-disk.

// scanStats is the last full scan, so the next one can show a real progress bar.
type scanStats struct {
	Files int64         `json:"files"`
	Bytes int64         `json:"bytes"`
	Took  time.Duration `json:"took"`
	At    time.Time     `json:"at"`
}

func loadScanStats(dir string) scanStats {
	var s scanStats
	if b, err := os.ReadFile(filepath.Join(dir, "scan.json")); err == nil {
		// A damaged file means no previous scan, not part of one.
		if json.Unmarshal(b, &s) != nil {
			return scanStats{}
		}
	}
	return s
}

func saveScanStats(dir string, s scanStats) {
	if dir == "" || os.MkdirAll(dir, 0o755) != nil {
		return
	}
	b, _ := json.Marshal(s)
	// Best effort: without it, the next scan has no time to compare with.
	_ = os.WriteFile(filepath.Join(dir, "scan.json"), b, 0o644)
}

// freeSample is the free space seen at one moment; together they draw the trend.
type freeSample struct {
	At   time.Time `json:"at"`
	Free int64     `json:"free"`
}

const (
	sampleEvery = 10 * time.Minute // restarting often doesn't flood the trend
	keepSamples = 200
)

// recordFree adds a sample (unless the last is recent) and returns the trend.
func recordFree(dir string, free int64, now time.Time) []freeSample {
	samples := loadFree(dir)
	if n := len(samples); n > 0 && now.Sub(samples[n-1].At) < sampleEvery {
		samples[n-1].Free = free // keep the newest reading without adding a point
	} else {
		samples = append(samples, freeSample{At: now, Free: free})
	}
	if len(samples) > keepSamples {
		samples = samples[len(samples)-keepSamples:]
	}
	if dir != "" && os.MkdirAll(dir, 0o755) == nil {
		var b strings.Builder
		for _, s := range samples {
			line, _ := json.Marshal(s)
			b.Write(line)
			b.WriteByte('\n')
		}
		// Best effort: without it, the free-space sparkline starts over.
		_ = os.WriteFile(filepath.Join(dir, "free.jsonl"), []byte(b.String()), 0o644)
	}
	return samples
}

func loadFree(dir string) []freeSample {
	f, err := os.Open(filepath.Join(dir, "free.jsonl"))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }() // read-only
	var out []freeSample
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var s freeSample
		if json.Unmarshal(sc.Bytes(), &s) == nil && !s.At.IsZero() {
			out = append(out, s)
		}
	}
	return out
}

var sparks = []rune("▁▂▃▄▅▆▇█")

// sparkline draws the last width samples, low to high between their own min and max.
func sparkline(samples []freeSample, width int) string {
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	if len(samples) == 0 {
		return ""
	}
	lo, hi := samples[0].Free, samples[0].Free
	for _, s := range samples {
		lo, hi = min(lo, s.Free), max(hi, s.Free)
	}
	out := make([]rune, len(samples))
	for i, s := range samples {
		level := len(sparks) / 2
		if hi > lo {
			level = int(float64(s.Free-lo) / float64(hi-lo) * float64(len(sparks)-1))
		}
		out[i] = sparks[level]
	}
	return string(out)
}
