package core

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/ui"
)

type ERowExec struct {
	erow   *ERow
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	runW   io.WriteCloser
}

func (eexec *ERowExec) Start() context.Context {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()

	// clear old context if exists
	if eexec.ctx != nil {
		eexec.clear2()
	}

	// indicate the row is running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, true)
	})

	// new context
	eexec.ctx, eexec.cancel = context.WithCancel(context.Background())

	return eexec.ctx
}

func (eexec *ERowExec) Stop() {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()
	if eexec.cancel != nil {
		eexec.cancel()
	}
}

// Clears state if the ctx matches. Fn is called before clear if in context.
func (eexec *ERowExec) Clear(ctx context.Context, fn func()) {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()

	// stop current ctx if arg is nil
	if ctx == nil {
		ctx = eexec.ctx
	}

	if ctx == eexec.ctx {
		// run function since still running in the requested context
		if fn != nil {
			fn()
		}

		eexec.clear2()
	}
}

func (eexec *ERowExec) clear2() {
	// clear resources
	eexec.cancel()
	eexec.cancel = nil
	eexec.ctx = nil

	// clear run resources
	if eexec.runW != nil {
		eexec.runW.Close()
		eexec.runW = nil
	}

	// indicate the row is not running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, false)
	})
}

//----------

func (eexec *ERowExec) Run(fexec func(context.Context, io.Writer) error) {
	ctx := eexec.Start() // will cancel previous existent ctx
	go func() {
		w := eexec.erow.TextAreaWriter()
		defer w.Close()

		// keep w to ensure early close on clear
		eexec.runW = w

		err := fexec(ctx, w)
		eexec.erow.Exec.Clear(ctx, func() {
			if err != nil {
				fmt.Fprintf(w, "# error: %v", err)
			}
		})
	}()
}
