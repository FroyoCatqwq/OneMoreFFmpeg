package output

import (
	"io"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

// Progress is a thin wrapper over go-pretty's progress writer. The download
// layer reports bytes through Update; this package owns rendering and never
// exposes the third-party type to callers.
type Progress struct {
	writer  progress.Writer
	tracker *progress.Tracker
	once    sync.Once
	done    chan struct{}
}

// NewProgress starts rendering a progress bar for total bytes to w.
// Enable it only when the caller wants human output (not --json).
func NewProgress(w io.Writer, label string, total int64) *Progress {
	pw := progress.NewWriter()
	pw.SetOutputWriter(w)
	pw.SetAutoStop(true)
	pw.SetTrackerLength(25)
	pw.SetMessageLength(24)
	pw.SetNumTrackersExpected(1)
	pw.SetStyle(progress.StyleDefault)
	pw.SetUpdateFrequency(100 * time.Millisecond)

	tracker := &progress.Tracker{
		Message: label,
		Total:   total,
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(tracker)
	p := &Progress{writer: pw, tracker: tracker, done: make(chan struct{})}
	go func() {
		pw.Render()
		close(p.done)
	}()
	return p
}

// Update sets the number of bytes transferred so far.
func (p *Progress) Update(n int64) {
	if p == nil {
		return
	}
	p.tracker.SetValue(n)
}

// Finish marks the transfer complete and waits for the final render.
func (p *Progress) Finish() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		// MarkAsDone is the thread-safe way to finish a tracker: combined with
		// SetAutoStop(true) at construction, Render() sees all trackers done
		// and returns on its own. Calling SetAutoStop here would race with the
		// render goroutine reading the same field.
		p.tracker.MarkAsDone()
		// Bound the wait: go-pretty stops shortly after all trackers finish,
		// but never let a renderer bug hang an otherwise successful download.
		select {
		case <-p.done:
		case <-time.After(2 * time.Second):
		}
	})
}
