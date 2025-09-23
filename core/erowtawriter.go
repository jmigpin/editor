package core

import (
	"io"

	"github.com/jmigpin/editor/util/iout"
)

type ERowTaWriteCloser struct {
	io.WriteCloser
}

func newERowTaWriteCloser(erow *ERow) *ERowTaWriteCloser {
	tawc := &ERowTaWriteCloser{}

	// synced writer to slow down memory usage
	w := iout.FnWriter(func(b []byte) (int, error) {
		var err error
		erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
			err = erow.AppendBytesClearHistory2(b)
		})
		return len(b), err
	})

	// buffered for performance, which needs timed output (auto-flush)
	tawc.WriteCloser = iout.NewAutoBufWriter(w, 4096*2)

	// DEBUG: no buffer
	//tawc.WriteCloser := &iout.RWC{nil, w, iout.FnCloser(func() error { return nil })}

	return tawc
}
