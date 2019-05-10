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
	w      io.WriteCloser
}

func NewERowExec(erow *ERow) *ERowExec {
	return &ERowExec{erow: erow}
}

//----------

func (eexec *ERowExec) Start(fexec func(context.Context, io.Writer) error) {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()

	// cancel old context if exists
	if eexec.cancel != nil {
		eexec.clear2()
	}

	// indicate the row is running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, true)
	})

	// new context
	ctx, cancel := context.WithCancel(eexec.erow.ctx)
	eexec.ctx, eexec.cancel = ctx, cancel

	// writer
	w := eexec.erow.TextAreaWriter()
	eexec.w = w // keep w to ensure early close on clear

	go func() {
		err := fexec(ctx, w)

		eexec.mu.Lock()
		defer eexec.mu.Unlock()

		// show error if the context matches, if it doesn't match then the previous instance was canceled already
		if eexec.ctx == ctx {
			if err != nil {
				fmt.Fprintf(w, "# error: %v", err)
			}
			eexec.clear2()
		}
	}()
}

func (eexec *ERowExec) clear2() {
	// clear resources
	eexec.cancel()
	eexec.cancel = nil
	eexec.w.Close()

	// indicate the row is not running
	eexec.erow.Ed.UI.RunOnUIGoRoutine(func() {
		eexec.erow.Row.SetState(ui.RowStateExecuting, false)
	})
}

//----------

func (eexec *ERowExec) Stop() {
	eexec.mu.Lock()
	defer eexec.mu.Unlock()
	if eexec.cancel != nil {
		eexec.cancel()
	}
}
