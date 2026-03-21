// Package progress provides a terminal progress bar for translation tracking.
package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Bar tracks translation progress with a visual progress bar.
type Bar struct {
	mu        sync.Mutex
	total     int
	completed int
	startTime time.Time
	label     string
}

// New creates a new progress bar.
func New(total int, label string) *Bar {
	return &Bar{
		total:     total,
		completed: 0,
		startTime: time.Now(),
		label:     label,
	}
}

// Increment increases the completed count and redraws the bar.
func (b *Bar) Increment() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.completed++
	b.draw()
}

// Done prints a final newline after the bar is complete.
func (b *Bar) Done() {
	fmt.Println()
}

func (b *Bar) draw() {
	width := 30
	pct := float64(b.completed) / float64(b.total)
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	elapsed := time.Since(b.startTime)
	eta := ""
	if b.completed > 0 && b.completed < b.total {
		perItem := elapsed / time.Duration(b.completed)
		remaining := perItem * time.Duration(b.total-b.completed)
		eta = fmt.Sprintf(" ETA %s", formatDuration(remaining))
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	fmt.Printf("\r  %s [%s] %3.0f%% (%d/%d)%s", b.label, bar, pct*100, b.completed, b.total, eta)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}
