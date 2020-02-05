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
	mu   struct {
		sync.Mutex
		ctx    context.Context
		cancel context.CancelFunc
		w      io.WriteCloser
	}
}

func NewERowExec(erow *ERow) *ERowExec {
	return &ERowExec{erow: erow}
}

//----------

func (eexec *ERowExec) Start(fexec func(context.Context, io.Writer) error) {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()

	// cancel old context if exists
	if eexec.mu.cancel != nil {
		eexec.clear()
	}

	// indicate the row is running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, true)
	})

	// new context
	ctx, cancel := context.WithCancel(eexec.erow.ctx)
	eexec.mu.ctx, eexec.mu.cancel = ctx, cancel

	// writer
	w := eexec.erow.TextAreaWriter() // needs to be closed in the end
	eexec.mu.w = w                   // keep w to ensure early close on clear

	go func() {
		err := fexec(ctx, w)

		eexec.mu.Lock()
		defer eexec.mu.Unlock()

		// show error if the context matches, if it doesn't match then the previous instance was canceled already
		if eexec.mu.ctx == ctx {
			if err != nil {
				fmt.Fprintf(w, "# error: %v", err)
			}
			eexec.clear()
		}
	}()
}

func (eexec *ERowExec) clear() {
	// clear resources
	eexec.mu.cancel()
	eexec.mu.cancel = nil
	eexec.mu.w.Close()
	eexec.mu.w = nil

	// indicate the row is not running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, false)
	})
}

//----------

func (eexec *ERowExec) Stop() {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()
	if eexec.mu.cancel != nil {
		eexec.mu.cancel()
	}
}
