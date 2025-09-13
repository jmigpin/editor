package syncutil

import (
	"sync"
	"time"
)

// runs fn only one at a time without overlaps, and always runs one last time.
type Throttler struct {
	sync.Mutex
	pending   bool
	running   bool
	lastStart time.Time

	Interval time.Duration
	Fn       func(done func())
}

func NewThrottler() *Throttler {
	thr := &Throttler{}
	thr.Interval = time.Second / 10
	return thr
}

func (thr *Throttler) Call() {
	thr.Lock()
	defer thr.Unlock()
	if thr.pending {
		return
	}
	thr.pending = true
	if !thr.running {
		thr.schedule()
	}
}

func (thr *Throttler) run() {
	thr.Lock()
	thr.lastStart = time.Now()
	thr.pending = false
	thr.running = true
	thr.Unlock()

	done := func() {
		thr.Lock()
		defer thr.Unlock()
		thr.running = false
		if thr.pending {
			thr.schedule()
		}
	}

	thr.Fn(done)
}

func (thr *Throttler) schedule() {
	d := thr.durationToNext()

	//_ = time.AfterFunc(d, thr.run)

	go func() {
		if d > 0 {
			time.Sleep(d)
		}
		thr.run()
	}()
}

func (thr *Throttler) durationToNext() time.Duration {
	d := time.Since(thr.lastStart)
	return max(thr.Interval-d, 0)
}
