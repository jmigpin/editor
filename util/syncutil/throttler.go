package syncutil

import (
	"sync"
	"sync/atomic"
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

// ensure scheduled
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

//----------
//----------
//----------

// ThrottledTrigger calls a function after a period of inactivity (idle), or
// unconditionally once the max duration is reached — whichever comes first.
// Typical use: coalescing rapid updates (e.g. terminal redraws) into a single
// call, avoiding intermediate frames.
type ThrottledTrigger struct {
	fn        func()
	idle      time.Duration
	max       time.Duration
	dirty     atomic.Bool
	idleTimer *time.Timer
	maxTimer  *time.Timer
}

// NewThrottledTrigger creates a ThrottledTrigger that calls fn when triggered.
// idle: how long to wait after the last Trigger() call before firing.
//
//	Each new Trigger() resets this window. Use a small value (e.g. 2ms)
//	to coalesce bursts of updates into a single paint.
//
// max:  the maximum time to wait before firing, even if Trigger() keeps
//
//	being called continuously. Prevents starvation during sustained
//	output (e.g. cat largefile). Use ~16ms for ~60fps.
func NewThrottledTrigger(fn func(), idle, max time.Duration) *ThrottledTrigger {
	t := &ThrottledTrigger{
		fn:   fn,
		idle: idle,
		max:  max,
	}
	t.idleTimer = time.AfterFunc(idle, t.fire)
	t.maxTimer = time.AfterFunc(max, t.fire)
	t.idleTimer.Stop()
	t.maxTimer.Stop()
	return t
}

// Trigger marca como dirty e arma os timers.
func (t *ThrottledTrigger) Trigger() {
	firstDirty := !t.dirty.Swap(true)
	t.idleTimer.Stop()
	t.idleTimer.Reset(t.idle)
	if firstDirty {
		t.maxTimer.Reset(t.max)
	}
}

func (t *ThrottledTrigger) fire() {
	if !t.dirty.Swap(false) {
		return
	}
	t.idleTimer.Stop()
	t.maxTimer.Stop()
	t.fn()
}
