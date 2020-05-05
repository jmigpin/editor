package core

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/v2/ui"
)

////godebug:annotatefile

type ERowExec struct {
	erow *ERow
	mu   struct {
		sync.Mutex
		cancel context.CancelFunc
		fnWait sync.WaitGroup // added while locked
	}
}

func NewERowExec(erow *ERow) *ERowExec {
	ee := &ERowExec{erow: erow}
	ee.mu.cancel = func() {}
	return ee
}

//----------

func (ee *ERowExec) RunAsync(fn func(context.Context, io.ReadWriter) error) {
	// Note: textarea w.close() (textareawriter) could deadlock if runasync() is not on own goroutine. If w.close waits for UI goroutine to finish and runasync() is currently occupying it (w.close called after a runasync(), just that the UI goroutine is not getting released).
	// launching in a goroutine allows RunAsync() itself to be called from a uigoroutine since this func will return immediately
	go ee.runAsync2(fn)
}

func (ee *ERowExec) runAsync2(fn func(context.Context, io.ReadWriter) error) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	// cancel and wait for previous if any
	ee.mu.cancel()
	ee.mu.fnWait.Wait()

	// new context
	ctx, cancel := context.WithCancel(ee.erow.ctx)
	ee.mu.cancel = cancel

	rwc := ee.erow.TextAreaReadWriteCloser()

	ee.mu.fnWait.Add(1)
	go func() {
		defer ee.mu.fnWait.Done()

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
	ee.mu.Lock()
	defer ee.mu.Unlock()
	if ee.mu.cancel != nil {
		ee.mu.cancel()
	}
}
