package core

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/ui"
)

type ERowExec struct {
	erow *ERow
	c    struct { // count
		sync.Mutex
		q       int
		running bool
		cond    *sync.Cond
		cancel  context.CancelFunc
	}
}

func NewERowExec(erow *ERow) *ERowExec {
	ee := &ERowExec{erow: erow}
	ee.c.cancel = func() {}
	ee.c.cond = sync.NewCond(&ee.c)
	return ee
}

//----------

func (ee *ERowExec) RunAsync(fn func(context.Context, io.ReadWriter) error) (context.Context, context.CancelFunc) {
	return ee.startRunAsync(func(ctx context.Context) {
		tarwc := newERowTaReadWriteCloser(ee.erow)
		err := fn(ctx, tarwc)
		if err != nil {
			fmt.Fprintf(tarwc, "# error: %v\n", err)
		}
		if err := tarwc.Close(); err != nil {
			ee.erow.Ed.Error(err)
		}
	})
}

//----------

func (ee *ERowExec) startRunAsync(fn func(context.Context)) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ee.erow.ctx)
	go func() {
		err := ee.startRunAsync2(ctx, cancel, fn)
		if err != nil { // start error
			ee.erow.Ed.Error(err)
		}
	}()
	return ctx, cancel
}

// should start on its own goroutine to avoid deadlocks
func (ee *ERowExec) startRunAsync2(ctx context.Context, cancel context.CancelFunc, fn func(context.Context)) error {
	ee.c.Lock()
	defer ee.c.Unlock()

	ee.c.q++
	id := ee.c.q

	// cancel and wait for previous if any
	ee.c.cancel()
	for ee.c.running {
		ee.c.cond.Wait()
	}

	// there is another request after this one (or a stop), don't start since the next one would cancel this one
	if id != ee.c.q {
		cancel() // clear resources
		return fmt.Errorf("not running, a later request was done")
	}

	// new
	ee.c.cancel = cancel

	ee.c.running = true
	ee.showRunning(true)

	// start
	go func() {
		defer func() {
			cancel() // clear resources
			ee.c.Lock()
			ee.c.running = false
			ee.showRunning(false)
			ee.c.Unlock()
			ee.c.cond.Broadcast()
		}()

		fn(ctx) // run
	}()

	return nil
}

//----------

func (ee *ERowExec) Stop() {
	ee.c.Lock()
	defer ee.c.Unlock()
	ee.c.q++      // if issued after another cmd, that cmd is not going to start
	ee.c.cancel() // always set
}

//---------

func (ee *ERowExec) showRunning(on bool) {
	ee.erow.Ed.UI.RunOnUIGoRoutine(func() {
		ee.erow.Row.SetState(ui.RowStateExecuting, on)
		if on {
			//ee.erow.uiResetTermRunState()
			ee.erow.Row.TextArea.SetStrClearHistory("")
			ee.erow.Row.TextArea.ClearPos()
		}
	})
}
