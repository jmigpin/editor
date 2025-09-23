package core

import (
	"io"

	"github.com/jmigpin/editor/util/iout"
)

type ERowTaReadWriteCloser struct {
	io.ReadWriteCloser

	optTemu *ERowTermEmu
}

func newERowTaReadWriteCloser(erow *ERow) *ERowTaReadWriteCloser {
	tarwc := &ERowTaReadWriteCloser{}

	tarc := newERowTaReadCloser(erow)
	tawc := newERowTaWriteCloser(erow)

	cl := iout.FnCloser(func() error {
		_ = tarc.Close()
		return tawc.Close()
	})

	tarwc.ReadWriteCloser = iout.RWC{tarc, tawc, cl}

	if erow.runOpts.emuOpts.Mode.On() {
		temu := newERowTermEmu(erow, tarwc.ReadWriteCloser)
		tarwc.optTemu = temu
		tarwc.ReadWriteCloser = temu
		tarc.optTemu = temu
	}

	return tarwc
}
