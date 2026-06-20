package core

import (
	"image"
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/osutil"
)

type ERowTermEmu struct {
	io.ReadWriteCloser // emu provides this
	emu                *termemu.Emu
	tui                *ERowTermEmuUI

	erow    *ERow
	userRwc io.ReadWriteCloser

	opsBuf []*TextColorOp

	optPtyCmd *osutil.PtyCmd
}

func newERowTermEmu(erow *ERow, rwc io.ReadWriteCloser) *ERowTermEmu {
	temu := &ERowTermEmu{erow: erow}
	temu.userRwc = rwc

	temu.tui = newERowTermEmuUI(temu)
	temu.emu = termemu.NewEmu(temu.userRwc, temu.tui, erow.termOpts.emuOpts)
	// Enable LNM initially so editor prints (like pid prints or errors) and non-pty outputs avoid staircase formatting.
	temu.emu.SetLNM(true)
	temu.ReadWriteCloser = temu.emu

	// Publish only after emu is ready; layout callbacks can re-enter during row creation.
	erow.optTemu = temu

	erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
		erow.uiCalcAndSetTermSize()
	})

	return temu
}

func (temu *ERowTermEmu) Close() error {
	defer func() {
		temu.erow.optTemu = nil

		// Has to wait in sync because otherwise it could clash with another row being created and setting optTemu. This close is called from a detached goroutine (runasync) and so not currently inside a ui goroutine.
		temu.erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
			temu.erow.uiCalcAndSetTermSize()
		})

		temu.userRwc.Close()
	}()

	temu.tui.Close()

	return temu.ReadWriteCloser.Close()
}

//----------

func (temu *ERowTermEmu) setSize(cr, px image.Point) {
	if cr2, changed := temu.emu.SetSize(cr); changed {
		// align PTY with emu size after possible clamp
		if temu.optPtyCmd != nil {
			if err := temu.setPtySize(cr2, px); err != nil {
				temu.tui.Error(err)
			}
		}
	}
}

func (temu *ERowTermEmu) setPty(ptyCmd *osutil.PtyCmd) {
	temu.optPtyCmd = ptyCmd
}

func (temu *ERowTermEmu) setPtySize(cr, psize P) error {
	return temu.optPtyCmd.SetSize(cr.X, cr.Y, psize.X, psize.Y)
}

func (temu *ERowTermEmu) onPtyStart() error {
	// Disable LNM when PTY starts to let raw mode / TUI programs handle newlines according to spec.
	temu.emu.SetLNM(false)
	cr := temu.emu.GetSize()
	psize := P{1, 1}
	return temu.setPtySize(cr, psize)
}

//----------
//----------
//----------

type P = termemu.P
