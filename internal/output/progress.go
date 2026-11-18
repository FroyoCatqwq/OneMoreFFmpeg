package output

import (
	"errors"
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

// Progress reports byte-transfer progress. The terminal states are distinct:
// a failed or aborted transfer must never render as 100%/complete. All methods
// are safe to call concurrently and terminal transitions are idempotent.
type Progress interface {
	Update(n int64)
	Complete()
	Fail(err error)
	Abort()
	// Close releases renderer resources only. It never marks success and is
	// intended for deferred cleanup on paths that did not reach a terminal state.
	Close()
}

type noopProgress struct{}

func (noopProgress) Update(int64) {}
func (noopProgress) Complete()    {}
func (noopProgress) Fail(error)   {}
func (noopProgress) Abort()       {}
func (noopProgress) Close()       {}

// terminalState guards idempotent terminal transitions.
type terminalState struct {
	once  sync.Once
	mu    sync.RWMutex
	state string // "", "completed", "failed", "aborted"
}

func (t *terminalState) set(state string, fn func()) bool {
	applied := false
	t.once.Do(func() {
		t.mu.Lock()
		t.state = state
		t.mu.Unlock()
		fn()
		applied = true
	})
	return applied
}

func (t *terminalState) current() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// stepProgress renders plain stage lines for non-interactive output.
type stepProgress struct {
	w       io.Writer
	label   string
	total   int64
	current int64
	mu      sync.Mutex
	lastPct int
	term    terminalState
}

func (s *stepProgress) Update(n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n > s.current {
		s.current = n
	}
	if s.total <= 0 {
		return // unknown total: never invent a percentage
	}
	pct := int(n * 100 / s.total)
	// Only 25/50/75 are progress milestones. 100% is a SUCCESS terminal state
	// printed exclusively by Complete(): receiving all bytes does not mean the
	// checksum/commit succeeded, so Update must never render 100%.
	if pct >= 100 {
		return
	}
	milestone := (pct / 25) * 25
	if milestone > s.lastPct {
		s.lastPct = milestone
		_, _ = fmt.Fprintf(s.w, "%s: %d%%\n", s.label, milestone)
	}
}

func (s *stepProgress) terminalLine(status string, withErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	received := humanBytes(s.current)
	if s.total > 0 {
		if status == "completed" {
			_, _ = fmt.Fprintf(s.w, "%s: completed 100%% (%s)\n", s.label, humanBytes(s.total))
		} else if withErr != nil {
			_, _ = fmt.Fprintf(s.w, "%s: %s after %s/%s (%v)\n", s.label, status, received, humanBytes(s.total), withErr)
		} else {
			_, _ = fmt.Fprintf(s.w, "%s: %s at %s/%s\n", s.label, status, received, humanBytes(s.total))
		}
		return
	}
	// Unknown total: report bytes received without a fabricated percentage.
	if withErr != nil {
		_, _ = fmt.Fprintf(s.w, "%s: %s after %s (%v)\n", s.label, status, received, withErr)
	} else {
		_, _ = fmt.Fprintf(s.w, "%s: %s (%s received)\n", s.label, status, received)
	}
}

func (s *stepProgress) Complete() {
	s.term.set("completed", func() { s.terminalLine("completed", nil) })
}

func (s *stepProgress) Fail(err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	s.term.set("failed", func() { s.terminalLine("failed", err) })
}

func (s *stepProgress) Abort() {
	s.term.set("aborted", func() { s.terminalLine("cancelled", nil) })
}

func (s *stepProgress) Close() { /* no renderer resources */ }

// barProgress renders an animated bar on a TTY via go-pretty.
type barProgress struct {
	writer  progress.Writer
	tracker *progress.Tracker
	term    terminalState
	done    chan struct{}
	close   sync.Once
}

func (b *barProgress) Update(n int64) {
	// With AutoStopDisabled the bar never auto-completes when value reaches
	// total. Clamp below total so a failed checksum after a full byte stream
	// cannot show 100%; only Complete() sets the final value.
	if b.tracker.Total > 1 && n >= b.tracker.Total {
		n = b.tracker.Total - 1
	}
	b.tracker.SetValue(n)
}

func (b *barProgress) finishRenderer() {
	select {
	case <-b.done:
	case <-time.After(2 * time.Second):
	}
}

func (b *barProgress) Complete() {
	b.term.set("completed", func() {
		if b.tracker.Total > 0 {
			b.tracker.SetValue(b.tracker.Total)
		}
		b.tracker.MarkAsDone()
		b.finishRenderer()
	})
}

func (b *barProgress) Fail(err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	b.term.set("failed", func() {
		// MarkAsErrored keeps the last real value and never renders completion.
		b.tracker.MarkAsErrored()
		b.tracker.UpdateMessage(fmt.Sprintf("%s (failed: %v)", b.tracker.Message, err))
		b.finishRenderer()
	})
}

func (b *barProgress) Abort() {
	b.term.set("aborted", func() {
		b.tracker.MarkAsErrored()
		b.tracker.UpdateMessage(b.tracker.Message + " (cancelled)")
		b.finishRenderer()
	})
}

func (b *barProgress) Close() {
	b.close.Do(func() {
		// Only stop the renderer; never mark success.
		if b.term.current() == "" {
			b.tracker.MarkAsErrored()
			b.tracker.UpdateMessage(b.tracker.Message + " (stopped)")
		}
		b.finishRenderer()
	})
}

// interactive reports whether animated progress is appropriate for w.
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

// NewProgress returns a Progress appropriate for the destination.
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
	pw.SetStyle(progress.StyleDefault)
	pw.SetUpdateFrequency(100 * time.Millisecond)

	tracker := &progress.Tracker{
		Message: label, Total: total, Units: progress.UnitsBytes,
		AutoStopDisabled: true, // only Complete() may mark success
	}
	pw.AppendTracker(tracker)
	p := &barProgress{writer: pw, tracker: tracker, done: make(chan struct{})}
	go func() {
		pw.Render()
		close(p.done)
	}()
	return p
}
