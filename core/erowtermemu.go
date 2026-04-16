package core

import (
	"fmt"
	"image/color"
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
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
	erow.optTemu = temu
	temu.userRwc = rwc

	temu.tui = newERowTermEmuUI(temu)
	temu.emu = termemu.NewEmu(temu.userRwc, temu.tui, erow.runOpts.emuOpts)
	temu.ReadWriteCloser = temu.emu

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

	if temu.tui.sp.UseGrayscale != temu.erow.runOpts.useGrayscale {
		temu.tui.sp.UseGrayscale = temu.erow.runOpts.useGrayscale
		// Force repaint in case size did not change. Call this through emu to be managed and avoid flicker
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

// implements [termemu.Tui] interface
type ERowTermEmuUI struct {
	temu *ERowTermEmu
	sp   *termemu.ScreenPrinter
	dec  []*drawer4.Decoration

	defaultColors struct {
		text struct {
			fg, bg color.Color
		}
	}

	restore struct {
		syntaxHighlight bool
		//scrollBarX      bool
		//scrollBarY      bool
		//bg color.Color
	}
}

func newERowTermEmuUI(temu *ERowTermEmu) *ERowTermEmuUI {
	tui := &ERowTermEmuUI{temu: temu}

	tui.sp = termemu.NewScreenPrinter()

	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		ta := tui.temu.erow.Row.TextArea

		// defaults colors for inverse video
		tui.defaultColors.text.fg = ta.TreeThemePaletteColor("text_fg")
		tui.defaultColors.text.bg = ta.TreeThemePaletteColor("text_bg")

		ta.EnableTerminalColors(true)
		ta.EnableTerminalDecorations(true)
		ta.SetTerminalColorOps(nil)
		ta.SetTerminalDecorations(nil)

		// keep
		tui.restore.syntaxHighlight = ta.SyntaxHighlight()
		ta.EnableSyntaxHighlight(false)

		//sa := tui.temu.erow.Row.ScrollArea
		//tui.restore.scrollBarX = sa.XBar != nil
		//tui.restore.scrollBarY = sa.YBar != nil
		//tui.restore.textBg = ta.TreeThemePaletteColor("text_bg")
		//tui.restore.bg = tui.temu.erow.Row.TreeThemePaletteColor("toolbar_text_bg")
	})
	return tui
}

func (tui *ERowTermEmuUI) Close() error {
	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		ta := tui.temu.erow.Row.TextArea
		tui.dec = nil
		tui.sp.SepFn = func(int) {}

		ta.EnableTerminalColors(false)
		ta.EnableTerminalDecorations(false)
		ta.SetTerminalColorOps(nil) // clear to avoid wrong place coloring upon re-enable. Ex: another cmd usage in the same textarea
		ta.SetTerminalDecorations(nil)

		ta.EnableSyntaxHighlight(tui.restore.syntaxHighlight)

		//tui.temu.erow.Row.SetThemePaletteColor("toolbar_text_bg", tui.restore.bg)

		//tui.temu.erow.Row.ScrollArea.SetBars(tui.restore.scrollBarX, tui.restore.scrollBarY)

		//ta.SetBytesClearHistory(nil)	// commented: clearing hides output of temporary cmds (ex: ls)
	})
	return nil
}

//----------

func (tui *ERowTermEmuUI) OnColumnModeChange() {
	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tui.temu.erow.uiCalcAndSetTermSize()
	})
}

//----------

func (tui *ERowTermEmuUI) Error(err error) {
	tui.temu.erow.Ed.Error(err)
}
func (tui *ERowTermEmuUI) Print(v any) {
	tui.temu.erow.Ed.Message(fmt.Sprint(v))
}

//----------

func (tui *ERowTermEmuUI) Paint() {
	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tui.paint2()
	})
}
func (tui *ERowTermEmuUI) paint2() {
	scr := tui.temu.emu.Snapshot()
	ops, bs := tui.paintOpsBytes(scr)
	ta := tui.temu.erow.Row.TextArea
	ta.SetTerminalColorOps(ops)
	ta.SetTerminalDecorations(tui.dec)
	tui.temu.erow.OverwriteBytesClearHistory(0, ta.RW().Max(), bs)
}

//// darken bg color
//cint := func(c int) color.RGBA {
//	return imageutil.RgbaFromInt(c)
//}
//_ = cint
//tbg := ta.TreeThemePaletteColor("text_bg")
//tbg2 := imageutil.TintOrShade(tui.restore.bg, 0.30)
//tbg2 := cint(0xff0000)
//tbg2 := cint(0xdddddd)
//tbg2 := cint(0xdddddd)
//ta.SetThemePaletteColor("text_bg", tbg2)
//tui.temu.erow.Row.SetThemePaletteColor("toolbar_text_bg", tbg2)

//----------

func (tui *ERowTermEmuUI) paintOpsBytes(scr *termemu.Screen) ([]*D4COp, []byte) {
	dops := []*D4COp{}
	decs := []*drawer4.Decoration{}

	// defaults colors for inverse video
	defColors := func(fg, bg color.Color) (_, _ color.Color) {
		if fg == nil {
			fg = tui.defaultColors.text.fg
		}
		if bg == nil {
			bg = tui.defaultColors.text.bg
		}
		return fg, bg
	}

	addColor0 := func(offset int, fg, bg color.Color, inverse bool) {
		if fg == nil && bg == nil && !inverse {
			return
		}
		if inverse {
			fg, bg = defColors(fg, bg)
			fg, bg = bg, fg
		}
		dop := &D4COp{Offset: offset, Fg: fg, Bg: bg}
		dop2 := &D4COp{Offset: offset + 1, SetNil: true} // reset
		dops = append(dops, dop, dop2)
	}

	addSep0 := func(offset int) {
		decs = append(decs, &drawer4.Decoration{
			Offset: offset,
			Kind:   drawer4.DecorationHorizontalRule,
			Fg:     tui.defaultColors.text.fg,
		})
	}

	//----------

	tui.sp.ColorFn = addColor0
	tui.sp.SepFn = addSep0
	bs := tui.sp.Bprint(scr)
	tui.dec = decs

	return dops, bs
}

//----------
//----------
//----------

type D4COp = drawer4.ColorizeOp

//----------

type P = termemu.P
