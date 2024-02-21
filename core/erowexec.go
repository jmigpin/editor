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

func (ee *ERowExec) RunAsync(fn func(context.Context, io.ReadWriter) error) {
	// Note: textarea w.close() (textareawriter) could deadlock if runasync() is not on own goroutine. If w.close waits for UI goroutine to finish and runasync() is currently occupying it (w.close called after a runasync(), just that the UI goroutine is not getting released). Launching in a goroutine allows RunAsync() itself to be called from a uigoroutine since this func will return immediately
	go ee.runAsync2(nil, nil, fn)
}
func (ee *ERowExec) RunAsyncWithCancel(fn func(context.Context, io.ReadWriter) error) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ee.erow.ctx)
	go ee.runAsync2(ctx, cancel, fn)
	return ctx, cancel
}
func (ee *ERowExec) runAsync2(optCtx context.Context, optCancel context.CancelFunc, fn func(context.Context, io.ReadWriter) error) {
	ee.c.Lock()
	defer ee.c.Unlock()

	ee.c.q++
	id := ee.c.q

	// cancel and wait for previous if any
	ee.c.cancel()
	for ee.c.running {
		ee.c.cond.Wait()
	}

	// there is another request after this one, don't start since the next one would cancel this one
	if id != ee.c.q {
		return
	}

	// new context
	ctx := (context.Context)(nil)
	cancel := (context.CancelFunc)(nil)
	if optCtx != nil {
		ctx, cancel = optCtx, optCancel
	} else {
		ctx, cancel = context.WithCancel(ee.erow.ctx)
	}
	ee.c.cancel = cancel

	rwc := ee.erow.TextAreaReadWriteCloser()

	ee.c.running = true
	go func() {
		defer func() {
			ee.c.Lock()
			defer ee.c.Unlock()
			ee.c.running = false
			ee.c.cond.Broadcast()
		}()

		// indicate the row is running
		ee.erow.Ed.UI.RunOnUIGoRoutine(func() {
			ee.erow.Row.SetState(ui.RowStateExecuting, true)
			ee.erow.Row.TextArea.SetStrClearHistory("")
			ee.erow.Row.TextArea.ClearPos()
		})

		err := fn(ctx, rwc)
		if err != nil {
			fmt.Fprintf(rwc, "# error: %v\n", err)
		}

		// clear cancel resources
		cancel()

		if err := rwc.Close(); err != nil {
			ee.erow.Ed.Error(err)
		}

		ee.erow.Ed.UI.RunOnUIGoRoutine(func() {
			ee.erow.Row.SetState(ui.RowStateExecuting, false)
		})
	}()
}

//----------

func (ee *ERowExec) Stop() {
	ee.c.Lock()
	defer ee.c.Unlock()

	ee.c.q++ // if this was issued after another cmd, that cmd is not going to start

	if ee.c.cancel != nil {
		ee.c.cancel()
	}
}
