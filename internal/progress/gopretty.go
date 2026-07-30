package progress

import (
	"io"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	pkgprogress "github.com/padiazg/go-crap/pkg/progress"
)

type GoPrettyReporter struct {
	pw      progress.Writer
	current *progress.Tracker
	baseMsg string
	mu      sync.Mutex
	done    bool
}

func NewGoPrettyReporter(w io.Writer) *GoPrettyReporter {
	pw := progress.NewWriter()
	pw.SetOutputWriter(w)
	pw.SetAutoStop(false)
	pw.SetUpdateFrequency(100 * time.Millisecond)
	pw.SetSortBy(progress.SortByIndex)

	s := pw.Style()
	s.Visibility.ETA = false
	s.Visibility.ETAOverall = false
	s.Visibility.Percentage = false
	s.Visibility.Speed = false
	s.Visibility.SpeedOverall = false
	s.Visibility.TrackerOverall = false
	s.Visibility.Pinned = false
	s.Visibility.Time = true

	s.Options.DoneString = "done"

	return &GoPrettyReporter{pw: pw}
}

func (r *GoPrettyReporter) Render() {
	r.pw.Render()
}

func (r *GoPrettyReporter) StartPhase(name pkgprogress.Phase, total int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.baseMsg = string(name)
	t := &progress.Tracker{
		Message: r.baseMsg,
	}
	if total > 0 {
		t.Total = int64(total)
	}
	r.pw.AppendTracker(t)
	r.current = t
}

func (r *GoPrettyReporter) Advance(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil {
		r.current.Increment(int64(n))
	}
}

func (r *GoPrettyReporter) SetDetail(detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil && detail != "" {
		r.current.UpdateMessage(r.baseMsg + "  " + detail)
	}
}

func (r *GoPrettyReporter) SetTotal(total int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil {
		r.current.UpdateTotal(int64(total))
	}
}

func (r *GoPrettyReporter) FinishPhase() {
	r.mu.Lock()
	if r.current != nil {
		r.current.MarkAsDone()
		r.current = nil
		r.baseMsg = ""
	}
	r.mu.Unlock()
}

func (r *GoPrettyReporter) Done() {
	r.mu.Lock()
	if r.done {
		r.mu.Unlock()
		return
	}
	r.done = true
	r.mu.Unlock()
	r.pw.Stop()
}

func (r *GoPrettyReporter) Errored() {
	r.mu.Lock()
	if r.current != nil {
		r.current.MarkAsErrored()
		r.current = nil
		r.baseMsg = ""
	}
	r.mu.Unlock()
}
