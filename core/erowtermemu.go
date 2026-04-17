package core

import (
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

	opsBuf []*D4COp

	optPtyCmd *osutil.PtyCmd
}

func newERowTermEmu(erow *ERow, rwc io.ReadWriteCloser) *ERowTermEmu {
	temu := &ERowTermEmu{erow: erow}
	temu.userRwc = rwc

	temu.tui = newERowTermEmuUI(temu)
	temu.emu = termemu.NewEmu(temu.userRwc, temu.tui, erow.runOpts.emuOpts)
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

func (temu *ERowTermEmu) onRecalcSetSize() {
	fface := temu.erow.Row.TextArea.TreeThemeFontFace()

	cr, psize := temu.erow.termSize(fface)
	if temu.tui.render.useGrayscale != temu.erow.runOpts.useGrayscale {
		temu.tui.render.useGrayscale = temu.erow.runOpts.useGrayscale
		temu.emu.NeedsPaint()
	}

	// UX-ADAPTATION: skip resize if window is too small (e.g. collapsed) to avoid pushing to scrollback, as well as avoid certain programs to recalc their contents when columns go directly to zero (ex: from 80->0) due to the textarea not being visible (ex: some other row got its space)
	if cr.X > 1 && cr.Y > 1 {
		if cr2, changed := temu.emu.SetSize(cr); changed {
			// align PTY with emu size after possible clamp
			if temu.optPtyCmd != nil {
				if err := temu.setPtySize(cr2, psize); err != nil {
					temu.tui.Error(err)
				}
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
	cr := temu.emu.GetSize()
	psize := P{1, 1}
	return temu.setPtySize(cr, psize)
}

//----------
//----------
//----------

type P = termemu.P
