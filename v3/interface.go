package pb

import (
	"errors"
	"sync"
	"time"
)

// Progressable is an interface for objects that can report their progress.
// Implementations should return consistent values for Total() and provide
// updates to Value() as progress is made. FinishedChan() should close when
// the operation is complete.
type Progressable interface {
	// Total returns the total amount of work to be done.
	Total() int64

	// Value returns the current progress value.
	Value() int64

	// FinishedChan returns a channel that closes when the operation is complete.
	FinishedChan() <-chan struct{}
}

// TitleProgressable is an interface for Progressable objects that have an optional title.
// This extends the Progressable interface by adding a Title() method. If a Progressable
// also implements TitleProgressable, RegisterProgressable will automatically format the
// progress bar to display the title.
type TitleProgressable interface {
	Progressable
	// Title returns the title for this progress operation. Return an empty string
	// if no title is desired.
	Title() string
}

// RegisterProgressable creates and configures a ProgressBar for the given Progressable.
// It automatically detects if the Progressable implements TitleProgressable and configures
// the progress bar template accordingly. The removeFunc is called when the progress is
// complete to allow for cleanup. RegisterProgressable returns an error if pr is nil.
//
// The function starts a goroutine that synchronizes the Progressable's progress to the
// ProgressBar at regular intervals (100ms). The caller should not call Start() on the
// returned ProgressBar - instead, add it to a Pool which manages display timing.
func RegisterProgressable(pr Progressable, removeFunc func(*ProgressBar)) (*ProgressBar, error) {
	if pr == nil {
		return nil, errors.New("RegisterProgressable: pr is nil")
	}

	bar := New64(pr.Total())

	hasTitle(pr, bar)

	// Pool will configure NotPrint and ManualUpdate when we Add it
	go progressWorker(pr, bar, removeFunc)
	return bar, nil
}

// hasTitle checks if pr implements TitleProgressable and configures the progress bar
// template with the title if present and non-empty.
func hasTitle(pr Progressable, bar *ProgressBar) {
	if pr == nil || bar == nil {
		return
	}

	if tp, ok := pr.(TitleProgressable); ok {
		title := tp.Title()
		if title == "" {
			return
		}

		// Set the title in the bar's data
		bar.Set("title", title)
		// Use a template that includes the title
		bar.SetTemplateString(`{{string . "title"}} {{counters . }} {{bar . }} {{percent . }} {{speed . }}`)
	}
}
func progressWorker(pr Progressable, bar *ProgressBar, removeFunc func(*ProgressBar)) {
	fc := pr.FinishedChan()
	ticker := time.NewTicker(100 * time.Millisecond) // Update more frequently for smoother display
	defer ticker.Stop()
	defer removeFunc(bar)
	// Don't call bar.Finish() - let the pool handle final display
	for {
		select {
		case <-ticker.C:
			// Just update values, pool will handle display
			bar.SetTotal(pr.Total())
			bar.SetCurrent(pr.Value())
		case _, ok := <-fc:
			if !ok {
				// Update one final time before finishing
				bar.SetTotal(pr.Total())
				bar.SetCurrent(pr.Value())
				return
			}
		}
	}
}

// PoolProgressFactory creates and manages progress bars for a pb.Pool.
// It wraps a Pool and a WaitGroup to simplify adding multiple Progressables
// to a pool while tracking their completion.
type PoolProgressFactory struct {
	// Pool is the progress bar pool to which progressables are added.
	Pool *Pool
	// Wg is a WaitGroup that tracks the completion of all registered progressables.
	Wg *sync.WaitGroup
}

// NewPoolProgressFactory creates a new PoolProgressFactory for the given pool.
// The factory creates its own internal WaitGroup for tracking progress completion.
// Returns a pointer to the initialized PoolProgressFactory.
func NewPoolProgressFactory(pool *Pool) *PoolProgressFactory {
	return &PoolProgressFactory{
		Pool: pool,
		Wg:   &sync.WaitGroup{},
	}
}

// Register adds a Progressable to the factory's pool and tracks its completion.
// The Progressable's progress is displayed using the pool's synchronized display mechanism.
// Returns an error if p is nil. This method increments the factory's WaitGroup and
// ensures it is properly decremented when the progress completes.
// The caller should use factory.Wg.Wait() to block until all registered progressables complete.
func (f *PoolProgressFactory) Register(p Progressable) error {
	if p == nil {
		return errors.New("Register: progressable is nil")
	}

	f.Wg.Add(1)
	removeFunc := func(pb *ProgressBar) {
		// Note: root pb.Pool doesn't have Remove() method
		// Progress bar will be cleaned up when pool stops
		f.Wg.Done()
	}

	bar, err := RegisterProgressable(p, removeFunc)
	if err != nil {
		f.Wg.Done()
		return err
	}
	// Don't call Start() - the pool manages display
	f.Pool.Add(bar)
	return nil
}
