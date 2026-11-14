package core

import (
	"io"

	"github.com/jmigpin/editor/util/iout"
)

type ERowTaReadWriteCloser struct {
	io.ReadWriteCloser
	MsgW io.Writer
}

func newERowTaReadWriteCloser(erow *ERow, lnm bool) *ERowTaReadWriteCloser {
	tarwc := &ERowTaReadWriteCloser{}

	tarc := newERowTaReadCloser(erow)
	tawc := newERowTaWriteCloser(erow)

	cl := iout.FnCloser(func() error {
		_ = tarc.Close()
		return tawc.Close()
	})

	tarwc.ReadWriteCloser = iout.RWC{tarc, tawc, cl}
	tarwc.MsgW = tarwc

	if erow.termOpts.emuOpts.Mode.On() {
		temu := newERowTermEmu(erow, tarwc.ReadWriteCloser, lnm)
		tarwc.ReadWriteCloser = temu
		tarwc.MsgW = iout.FnWriter(temu.emu.WriteMessage)
	}

	return tarwc
}
