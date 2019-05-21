package iout

import (
	"io"
	"sync"
)

type MultiClose struct {
	sync.Mutex
	closers map[io.Closer]bool
}

func NewMultiClose() *MultiClose {
	return &MultiClose{closers: map[io.Closer]bool{}}
}

func (mc *MultiClose) Add(closer io.Closer) {
	mc.Lock()
	defer mc.Unlock()
	mc.closers[closer] = false
}

//----------

func (mc *MultiClose) close(closer io.Closer) error {
	if mc.done(closer) {
		return nil
	}
	return closer.Close() // run unlocked
}

func (mc *MultiClose) done(closer io.Closer) bool {
	mc.Lock()
	defer mc.Unlock()
	done, ok := mc.closers[closer]
	if !ok {
		panic("unknown closer")
	}
	if done {
		return true // was done already
	}
	mc.closers[closer] = true
	return false // was not done
}

//----------

func (mc *MultiClose) CloseCalled(closer io.Closer) bool {
	mc.Lock()
	defer mc.Unlock()
	done, ok := mc.closers[closer]
	if !ok {
		panic("unknown closer")
	}
	return done
}

// Does not call closer, just the others. Run inside a Close().
func (mc *MultiClose) CloseRest(closer io.Closer) error {
	_ = mc.done(closer)
	return mc.CloseAll()
}

func (mc *MultiClose) CloseAll() error {
	// avoid data race since calling each close will be done unlocked
	mc.Lock()
	var w []io.Closer
	for closer, done := range mc.closers {
		if !done {
			w = append(w, closer)
		}
	}
	mc.Unlock()

	var me MultiError
	for _, closer := range w {
		me.Add(mc.close(closer))
	}
	return me.Result()
}
