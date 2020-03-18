package syncutil

import (
	"fmt"
	"sync"
	"time"
)

// Continously usable timeout, instantiated once for many get()/set() calls. Fails if get() is not ready when set() is called.
type GetTimeout struct {
	d struct {
		sync.Mutex
		get struct {
			ready bool
		}
		set struct {
			gotV bool
			v    interface{}
		}
	}
	cond *sync.Cond // signals from set() or timeout()
}

func NewGetTimeout() *GetTimeout {
	t := &GetTimeout{}
	t.cond = sync.NewCond(&t.d)
	return t
}

// Waits with a timeout for the set() value.
func (t *GetTimeout) Get(timeout time.Duration, readyFn func() error) (interface{}, error) {
	timer := time.AfterFunc(timeout, t.cond.Signal)
	defer timer.Stop()

	t.d.Lock()
	if t.d.get.ready {
		panic("gettimeout: a get is currently running")
	}
	t.d.get.ready = true
	if err := readyFn(); err != nil {
		t.d.Unlock()
		return nil, err
	}
	t.cond.Wait() // wait for signal from set() or time.afterfunc()
	defer t.d.Unlock()
	t.d.get.ready = false
	if t.d.set.gotV {
		defer func() { t.d.set.gotV = false }() // reset for next run
		return t.d.set.v, nil
	}
	return nil, fmt.Errorf("gettimeout: get timeout")
}

// Fails if not able to set while get() is ready.
func (t *GetTimeout) Set(v interface{}) error {
	t.d.Lock()
	defer t.d.Unlock()
	if !t.d.get.ready {
		return fmt.Errorf("gettimeout: get is not ready")
	}
	t.d.set.gotV = true
	t.d.set.v = v
	t.cond.Signal()
	return nil
}

//----------

//// Only usable once.
//type GetOneTimeout struct {
//	d struct {
//		sync.Mutex
//		get struct {
//			ready   bool
//			timeout bool
//			done    bool
//		}
//		set struct {
//			gotV bool
//			v    interface{}
//		}
//	}
//	getCond *sync.Cond // d.get.ready
//	valCond *sync.Cond // d.set.gotV
//}

//func NewGetOneTimeout() *GetOneTimeout {
//	t := &GetOneTimeout{}
//	t.getCond = sync.NewCond(&t.d)
//	t.valCond = sync.NewCond(&t.d)
//	return t
//}

//// Waits for set() to send a value, or fails with timeout.
//func (t *GetOneTimeout) Get(timeout time.Duration) (interface{}, error) {
//	timer := time.AfterFunc(timeout, t.valCond.Signal)
//	defer timer.Stop()

//	t.d.Lock()
//	if t.d.get.ready {
//		panic("gettimeout: get was already called")
//	}
//	t.d.get.ready = true
//	t.getCond.Signal()
//	t.valCond.Wait() // wait for signal from set() or time.afterfunc()
//	defer t.d.Unlock()
//	if t.d.set.gotV {
//		t.d.get.done = true
//		return t.d.set.v, nil
//	}
//	t.d.get.timeout = true
//	return nil, fmt.Errorf("gettimeout: get timeout")
//}

//// Waits for get() to be ready (can be launched before get()). Fails if not able to set in time or if get() exited.
//func (t *GetOneTimeout) Set(v interface{}) error {
//	t.d.Lock()
//	for !t.d.get.ready {
//		t.getCond.Wait()
//	}
//	defer t.d.Unlock()
//	if t.d.get.timeout {
//		return fmt.Errorf("gettimeout: set not in time")
//	}
//	if t.d.get.done {
//		return fmt.Errorf("gettimeout: set was already called")
//	}
//	t.d.set.gotV = true
//	t.d.set.v = v
//	t.valCond.Signal()
//	return nil
//}
