package syncutil

import (
	"fmt"
	"sync"
	"time"
)

// Continously usable, instantiated once for many wait()/set() calls. Fails if wait() is not ready when set() is called.
// Usage:
//
//	w:=NewWaitForSet()
//	w.Start(5*time.Second)
//	...
//	// sync/async call to w.Set()
//	...
//	v,err := w.WaitForSet()
//	if err!=nil {
//	}
type WaitForSet struct {
	d struct {
		sync.Mutex
		get struct {
			timer   *time.Timer
			waiting bool
		}
		set struct {
			gotV bool
			v    interface{}
		}
	}
	cond *sync.Cond // signals from set() or timeout()
}

func NewWaitForSet() *WaitForSet {
	w := &WaitForSet{}
	w.cond = sync.NewCond(&w.d)
	return w
}

//----------

func (w *WaitForSet) Start(timeout time.Duration) {
	w.d.Lock()
	defer w.d.Unlock()
	if w.d.get.timer != nil {
		panic("waitforset: timer!=nil")
	}
	w.d.get.timer = time.AfterFunc(timeout, w.cond.Signal)
}

func (w *WaitForSet) WaitForSet() (interface{}, error) {
	w.d.Lock()
	defer w.d.Unlock()
	defer w.clearTimer()
	if w.d.get.timer == nil {
		panic("waitforset: not started")
	}
	if w.d.get.waiting {
		panic("waitforset: already waiting")
	}
	w.d.get.waiting = true
	defer func() { w.d.get.waiting = false }()

	// wait for signal if the value was not set yet
	if !w.d.set.gotV {
		w.cond.Wait() // wait for signal from set() or timeout start()
	}

	if w.d.set.gotV {
		defer func() { w.d.set.gotV = false }() // reset for next run
		return w.d.set.v, nil
	}
	return nil, fmt.Errorf("waitforset: timeout")
}

//----------

// In case waitforset() is not going to be called.
func (w *WaitForSet) Cancel() {
	w.d.Lock()
	defer w.d.Unlock()
	w.clearTimer()
}

func (w *WaitForSet) clearTimer() {
	w.d.get.timer.Stop()
	w.d.get.timer = nil
}

//----------

// Fails if not able to set while get() is ready.
func (w *WaitForSet) Set(v interface{}) error {
	w.d.Lock()
	defer w.d.Unlock()
	if w.d.get.timer == nil {
		return fmt.Errorf("waitforset: not waiting for set")
	}
	w.d.set.gotV = true
	w.d.set.v = v
	w.cond.Signal()
	return nil
}
