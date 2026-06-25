package core

import (
	"fmt"
	"image/color"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/drawutil"
)

// implements [termemu.Tui] interface
type ERowTermEmuUI struct {
	temu *ERowTermEmu
	sp   *termemu.ScreenPrinter
	dec  []*drawutil.Decoration

	render struct {
		useGrayscale bool
	}

	restore struct {
		syntaxHighlight bool
		lineWrap        bool
		//scrollBarX      bool
		//scrollBarY      bool
		//bg color.Color
	}
}

func newERowTermEmuUI(temu *ERowTermEmu) *ERowTermEmuUI {
	tui := &ERowTermEmuUI{temu: temu}

	tui.sp = termemu.NewScreenPrinter()
	tui.render.useGrayscale = temu.erow.colorizeOpts.termGrayscale

	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		ta := tui.temu.erow.Row.TextArea

		ta.EnableTerminalColors(true)
		ta.EnableTerminalDecorations(true)
		ta.SetTerminalColorOps(nil)
		ta.SetTerminalDecorations(nil)

		// keep
		tui.restore.syntaxHighlight = ta.SyntaxHighlight()
		ta.EnableSyntaxHighlight(false)

		opt := ta.Drawer.TextDrawerOptions()
		tui.restore.lineWrap = opt.LineWrap.On
		if tui.temu.erow.termOpts.emuOpts.Mode.IsGrid() {
			opt.LineWrap.On = false
		}

		//sa := tui.temu.erow.Row.ScrollArea.XBar
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

		opt := ta.Drawer.TextDrawerOptions()
		opt.LineWrap.On = tui.restore.lineWrap

		//tui.temu.erow.Row.SetThemePaletteColor("toolbar_text_bg", tui.restore.bg)

		//tui.temu.erow.Row.ScrollArea.SetBars(tui.restore.scrollBarX, tui.restore.scrollBarY)

		//ta.SetBytesClearHistory(nil)	// commented: clearing hides output of temporary cmds (ex: ls)
	})
	return nil
}

//----------

func (tui *ERowTermEmuUI) OnColumnModeChange() {
	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tui.temu.erow.uiUpdateFontAndTermSize()
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

func (tui *ERowTermEmuUI) SyncScreen() {
	tui.temu.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tui.screenSync2()
	})
}
func (tui *ERowTermEmuUI) screenSync2() {
	scr := tui.temu.emu.Snapshot()
	ops, bs := tui.buildScreenOpsAndBytes(scr)

	ta := tui.temu.erow.Row.TextArea
	erow := tui.temu.erow
	ta.SetTerminalColorOps(ops)
	ta.SetTerminalDecorations(tui.dec)
	erow.OverwriteBytesClearHistory(0, ta.RW().Max(), bs)
}

//----------

func (tui *ERowTermEmuUI) buildScreenOpsAndBytes(scr *termemu.Screen) ([]*TextColorOp, []byte) {
	dops := []*TextColorOp{}
	decs := []*drawutil.Decoration{}

	ta := tui.temu.erow.Row.TextArea
	defaultFg := ta.TreeThemePaletteColor("text_fg")
	defaultBg := ta.TreeThemePaletteColor("text_bg")

	useGrayscale := tui.render.useGrayscale

	addColor0 := func(offset int, fg, bg termemu.TermColor, inverse bool) {
		fg2, bg2, setBg, ok := termCellColors(fg, bg, inverse, defaultFg, defaultBg, useGrayscale)
		if !ok {
			return
		}
		dop := &TextColorOp{Offset: offset, Fg: fg2}
		if setBg {
			dop.Bg = bg2
		}
		dop2 := &TextColorOp{Offset: offset + 1, SetNil: true} // reset
		dops = append(dops, dop, dop2)
	}

	addSep0 := func(offset int) {
		decs = append(decs, &drawutil.Decoration{
			Offset: offset,
			Kind:   drawutil.DecorationHorizontalRule,
			Fg:     defaultFg,
		})
	}

	tui.sp.ColorFn = addColor0
	tui.sp.SepFn = addSep0
	bs := tui.sp.Bprint(scr)
	tui.dec = decs

	return dops, bs
}

//----------
//----------
//----------

type TextColorOp = drawutil.ColorizeOp

//----------
//----------
//----------

func termCellColors(fg, bg termemu.TermColor, inverse bool, defaultFg, defaultBg color.Color, useGrayscale bool) (_, _ color.Color, _ bool, _ bool) {
	if fg.IsDefault() && bg.IsDefault() && !inverse {
		return nil, nil, false, false
	}
	fg2, bg2, _, bgExplicit := resolveTermCellColors(fg, bg, inverse, defaultFg, defaultBg)
	if useGrayscale {
		fg2 = grayscaleColor(fg2)
		if bgExplicit {
			bg2 = grayscaleColor(bg2)
		}
	}
	return fg2, bg2, bgExplicit || inverse, true
}

func resolveTermCellColors(fg, bg termemu.TermColor, inverse bool, defaultFg, defaultBg color.Color) (_, _ color.Color, _ bool, _ bool) {
	fgExplicit := !fg.IsDefault()
	bgExplicit := !bg.IsDefault()
	fg2 := resolveTermColor(fg, defaultFg)
	bg2 := resolveTermColor(bg, defaultBg)
	if inverse {
		fg2, bg2 = bg2, fg2
		fgExplicit, bgExplicit = bgExplicit, fgExplicit
	}
	return fg2, bg2, fgExplicit, bgExplicit
}

func resolveTermColor(tc termemu.TermColor, defaultColor color.Color) color.Color {
	switch tc.Kind() {
	case termemu.TermColorDefault:
		return defaultColor
	case termemu.TermColorIndexed:
		return termemu.XTerm256Color(tc.Index())
	case termemu.TermColorRGB:
		return tc.RGBA()
	default:
		panic("unexpected term color")
	}
}

func grayscaleColor(c color.Color) color.Color {
	if c == nil {
		return nil
	}
	r, g, b, a := c.RGBA()
	y := uint16((299*r + 587*g + 114*b + 500) / 1000)
	return color.RGBA64{y, y, y, uint16(a)}
}
