package core

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/ui"
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

func (ee *ERowExec) RunAsync(fn func(context.Context, io.Writer) error) {
	// Note: textarea w.close() (textareawriter) could lock if run() is not on own goroutine, if w.close waits for UI goroutine to finish and run() is currently occupying it (w.close called after a run(), no local locks involved, just that the UI goroutine is not getting released).
	// Note: commented since the w.close() is currently not blocking the UI goroutine, just ensures it gets queued)

	//	go ee.run(fn)
	//}

	//func (ee *ERowExec) run(fn func(context.Context, io.Writer) error) {

	ee.mu.Lock()
	defer ee.mu.Unlock()

	// cancel and wait for previous if any
	ee.mu.cancel()
	ee.mu.fnWait.Wait()

	// new context
	ctx, cancel := context.WithCancel(ee.erow.ctx)
	ee.mu.cancel = cancel

	w := ee.erow.TextAreaWriter() // needs to be closed in the end

	ee.mu.fnWait.Add(1)
	go func() {
		defer ee.mu.fnWait.Done()

		// indicate the row is running
		ee.erow.Ed.UI.RunOnUIGoRoutine(func() {
			ee.erow.Row.SetState(ui.RowStateExecuting, true)
			ee.erow.Row.TextArea.SetStrClearHistory("")
			ee.erow.Row.TextArea.ClearPos()
		})

		err := fn(ctx, w)
		if err != nil {
			fmt.Fprintf(w, "# error: %v\n", err)
		}

		// clear cancel resources
		cancel()

		if err := w.Close(); err != nil {
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
