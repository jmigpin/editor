package core

import (
	"errors"
	"fmt"
	"image/color"
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/osutil"
)

type ERowTermEmu struct {
	io.ReadWriteCloser // emu provides this
	emu                *termemu.Emu
	tui                *ERowTermEmuUI

	erow    *ERow
	userRwc io.ReadWriteCloser

	reg    *evreg.Regist
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

	temu.erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
		temu.calcAndSetTermSize()
	})

	// textarea layout for console
	temu.reg = erow.Row.TextArea.EvReg.Add(ui.TextAreaBoundsEventId, func(ev0 any) {
		//ev := ev0.(*ui.TextAreaBoundsEvent)
		temu.calcAndSetTermSize()
	})

	return temu
}

func (temu *ERowTermEmu) Close() error {
	defer temu.userRwc.Close()
	defer func() { temu.erow.optTemu = nil }()

	temu.reg.Unregister()

	// TODO: emuplain freezing/locking

	temu.tui.Close()
	temu.erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
		temu.erow.Row.TextArea.SetThemeFontFace(temu.erow.runOpts.ffaceRestore)
	})

	return temu.ReadWriteCloser.Close()
}

//----------

// runs inside ui goroutine to get textarea pixel size
func (temu *ERowTermEmu) calcAndSetTermSize() {
	fface := temu.erow.runOpts.fface

	cr, psize := temu.termSize(fface) // cr=cols/rows, area pixel size

	// DISABLED: can be annoying at times
	//// support col132 mode, but ends allowing dynamic font size when the screen rows/cols are lower then the minimum required
	//if cr2 := temu.emu.ClampSize(cr); cr2 != cr {
	//	if fface2, ok := temu.termFontFace(cr2, psize, fface); ok {
	//		cr3, psize2 := temu.termSize(fface2)
	//		cr = temu.emu.ClampSize(cr3)
	//		fface = fface2
	//		psize = psize2 // usually the same
	//	}
	//}

	fface0 := temu.erow.Row.TextArea.TreeThemeFontFace()
	if fface != fface0 {
		temu.erow.Row.TextArea.SetThemeFontFace(fface)
	}

	if temu.tui.sp.UseGrayscale != temu.erow.runOpts.useGrayscale {
		temu.tui.sp.UseGrayscale = temu.erow.runOpts.useGrayscale
		// Force repaint in case size did not change. Call this through emu to be managed and avoid flicker
		temu.emu.NeedsPaint()
	}

	// UX-ADAPTATION: skip resize if window is too small (e.g. collapsed) to avoid pushing to scrollback, as well as avoid certain programs to recalc their contents when columns go directly to zero (ex: from 80->0) due to the textarea not being visible (ex: some other row got its space)
	if cr.X > 1 && cr.Y > 1 {
		if cr2, changed := temu.emu.SetSize(cr); changed {
			// align PTY with emu size after possible clamp
			if err := temu.setPtySize(cr2, psize); err != nil {
				//temu.tui.Error(err) // commented: pty cmd can be nil while the exec is starting, gives errors
			}
		}
	}
}

// triggered by a term sequence that changes cols/rows
func (temu *ERowTermEmu) uiCalcAndSetTermSize() {
	temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		temu.calcAndSetTermSize()
	})
}

//----------

func (temu *ERowTermEmu) setPty(ptyCmd *osutil.PtyCmd) {
	temu.optPtyCmd = ptyCmd
}

func (temu *ERowTermEmu) setPtySize(cr, psize P) error {
	if temu.optPtyCmd == nil {
		return errors.New("opt pty cmd is nil")
	}
	return temu.optPtyCmd.SetSize(cr.X, cr.Y, psize.X, psize.Y)
}

func (temu *ERowTermEmu) onPtyStart() error {
	cr := temu.emu.GetSize()
	psize := P{1, 1}
	return temu.setPtySize(cr, psize)
}

//----------

func (temu *ERowTermEmu) termSize(fface *fontutil.FontFace) (_, _ termemu.P) {
	runeW, _ := fface.Face.GlyphAdvance('W')
	rw := runeW.Ceil()
	lh := fface.LineHeightInt()

	sx, sy := temu.taPixelSize()
	sx -= rw // newline
	sx, sy = max(sx, 0), max(sy, 0)
	pixs := P{sx, sy}

	// max cols/rows at wanted font
	cols := pixs.X / rw
	rows := pixs.Y / lh

	// Use fixed rows if specified
	if temu.erow.runOpts.fixedRows > 0 {
		rows = temu.erow.runOpts.fixedRows
	}

	cr := P{cols, rows}

	return cr, pixs // columns/rows, available area pixel size
}

func (temu *ERowTermEmu) taPixelSize() (int, int) {
	ta := temu.erow.Row.TextArea
	b := ta.Bounds
	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		// handle extra space on the left side used inside the drawer
		b = d.InnerBounds()
	}
	return b.Dx(), b.Dy()
}

//----------

func (temu *ERowTermEmu) termFontFace(cr, pixs P, origFace *fontutil.FontFace) (*fontutil.FontFace, bool) {

	faceAtSize := func(v float64) *fontutil.FontFace {
		fopts2 := origFace.Opts // copy
		fopts2.SetSize(v)
		return origFace.Font.FontFace(fopts2)
	}

	runeSize := func(p float64) (int, int, bool) {
		face2 := faceAtSize(p)
		adv, ok := face2.Face.GlyphAdvance('W')
		if !ok {
			return 0, 0, false
		}
		return adv.Ceil(), face2.LineHeightInt(), true
	}

	_, _, p, ok := fitRuneSizeF(pixs.X, pixs.Y, cr.X, cr.Y, runeSize)
	if !ok {
		return nil, false
	}

	maxFSize := origFace.Opts.Size()
	if p > maxFSize {
		p = maxFSize
	}

	return faceAtSize(p), true
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
	tui.temu.uiCalcAndSetTermSize()
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

//----------
//----------
//----------

// finds the largest p (float) such that w×x <= sx and h×y <= sy, where (x,y) = runeSize(p) are pixel dims (monotonic non-decreasing in p). Returns the chosen (x,y,p).
func fitRuneSizeF(sx, sy, w, h int, runeSize func(p float64) (int, int, bool)) (int, int, float64, bool) {

	// TODO: font increaments of 0.25?

	if sx <= 0 || sy <= 0 || w <= 0 || h <= 0 {
		return 0, 0, 0, false
	}

	fits := func(p float64) bool {
		x, y, ok := runeSize(p)
		if !ok {
			return false
		}
		_ = y
		//return w*x <= sx && h*y <= sy
		//return w*x < sx && h*y < sy
		return w*x < sx
	}

	const (
		minP  = 1e-1
		maxP  = 1e3
		eps   = 1e-1 // binary search tolerance on p
		maxIt = 3    // cap iterations
	)

	// Find some fitting p (shrink if needed).
	lo, hi := 0.0, 1.0
	for !fits(hi) && hi > minP {
		hi *= 0.5
	}
	if !fits(hi) {
		return 0, 0, 0, false
	}
	lo = hi

	// Exponentially grow to bracket the first non-fitting p.
	for fits(hi) && hi < maxP {
		lo = hi
		hi *= 2
	}

	// Binary search in [lo,hi] for max fitting p.
	best := lo
	for it := 0; it < maxIt && hi-lo > eps; it++ {
		mid := (lo + hi) / 2
		if fits(mid) {
			best = mid
			lo = mid
		} else {
			hi = mid
		}
	}

	x, y, _ := runeSize(best)
	return x, y, best, true
}
