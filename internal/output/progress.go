package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/mattn/go-isatty"
)

func humanBytes(n int64) string { return humanize.Bytes(uint64(n)) }

// Progress reports byte-transfer progress. Callers in the business layer
// depend only on this interface; animation and environment detection live
// entirely inside this package.
type Progress interface {
	Update(n int64)
	Finish()
}

type noopProgress struct{}

func (noopProgress) Update(int64) {}
func (noopProgress) Finish()      {}

// stepProgress is used when the output is not an interactive terminal:
// plain stage lines with no animation / escape characters (CI, pipes,
// redirected stderr).
type stepProgress struct {
	w       io.Writer
	label   string
	total   int64
	mu      sync.Mutex
	lastPct int
	once    sync.Once
}

func (s *stepProgress) Update(n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.total <= 0 {
		return
	}
	pct := int(n * 100 / s.total)
	// Log coarse milestones only: 0, 25, 50, 75, 100%.
	milestone := (pct / 25) * 25
	if milestone > s.lastPct {
		s.lastPct = milestone
		_, _ = fmt.Fprintf(s.w, "%s: %d%%\n", s.label, milestone)
	}
}

func (s *stepProgress) Finish() {
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.total > 0 {
			_, _ = fmt.Fprintf(s.w, "%s: 100%% (%s)\n", s.label, humanBytes(s.total))
		} else {
			_, _ = fmt.Fprintf(s.w, "%s: done\n", s.label)
		}
	})
}

// barProgress is the interactive TTY renderer backed by go-pretty.
type barProgress struct {
	writer  progress.Writer
	tracker *progress.Tracker
	once    sync.Once
	done    chan struct{}
}

func (b *barProgress) Update(n int64) { b.tracker.SetValue(n) }

func (b *barProgress) Finish() {
	b.once.Do(func() {
		b.tracker.MarkAsDone()
		select {
		case <-b.done:
		case <-time.After(2 * time.Second):
		}
	})
}

// interactive reports whether animated progress is appropriate for w.
// Animation is disabled for non-files, non-TTYs, CI environments and when
// the user asks for reduced motion / no color.
func interactive(w io.Writer) bool {
	if w == nil {
		return false
	}
	for _, key := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		if v := os.Getenv(key); v != "" && v != "false" && v != "0" {
			return false
		}
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") || os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, ok := w.(*os.File)
	return ok && (isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd()))
}

// NewProgress returns a Progress appropriate for the destination. TTY output
// gets an animated bar with percentage and ETA; CI/pipes get plain stage
// lines; a nil writer yields a no-op (used for --json where stdout must stay
// pure JSON).
func NewProgress(w io.Writer, label string, total int64) Progress {
	if w == nil {
		return noopProgress{}
	}
	if !interactive(w) {
		return &stepProgress{w: w, label: label, total: total}
	}

	pw := progress.NewWriter()
	pw.SetOutputWriter(w)
	pw.SetAutoStop(true)
	pw.SetTrackerLength(25)
	pw.SetMessageLength(24)
	pw.SetNumTrackersExpected(1)
	pw.SetStyle(progress.StyleDefault) // default style includes ETA + percentage
	pw.SetUpdateFrequency(100 * time.Millisecond)

	tracker := &progress.Tracker{Message: label, Total: total, Units: progress.UnitsBytes}
	pw.AppendTracker(tracker)
	p := &barProgress{writer: pw, tracker: tracker, done: make(chan struct{})}
	go func() {
		pw.Render()
		close(p.done)
	}()
	return p
}
