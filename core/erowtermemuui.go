package core

import (
	"fmt"
	"image/color"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
)

// implements [termemu.Tui] interface
type ERowTermEmuUI struct {
	temu *ERowTermEmu
	sp   *termemu.ScreenPrinter
	dec  []*drawer4.Decoration

	render struct {
		useGrayscale bool
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

	ta := tui.temu.erow.Row.TextArea
	defaultFg := ta.TreeThemePaletteColor("text_fg")
	defaultBg := ta.TreeThemePaletteColor("text_bg")

	useGrayscale := tui.render.useGrayscale
	isLightTheme := isLightColor(defaultBg)

	addColor0 := func(offset int, fg, bg termemu.TermColor, inverse bool) {
		if fg.IsDefault() && bg.IsDefault() && !inverse {
			return
		}
		fg2, bg2, bgExplicit := resolveTermCellColors(fg, bg, inverse, defaultFg, defaultBg)
		if useGrayscale {
			fg2 = grayscaleColor(fg2)
			if bgExplicit {
				bg2 = grayscaleColor(bg2)
			}
		}
		if isLightTheme {
			fg2 = ensureContrastColor(fg2, bg2)
		}
		if !bgExplicit {
			bg2 = nil
		}
		dop := &D4COp{Offset: offset, Fg: fg2, Bg: bg2}
		dop2 := &D4COp{Offset: offset + 1, SetNil: true} // reset
		dops = append(dops, dop, dop2)
	}

	addSep0 := func(offset int) {
		decs = append(decs, &drawer4.Decoration{
			Offset: offset,
			Kind:   drawer4.DecorationHorizontalRule,
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

type D4COp = drawer4.ColorizeOp

//----------
//----------
//----------

func resolveTermCellColors(fg, bg termemu.TermColor, inverse bool, defaultFg, defaultBg color.Color) (_, _ color.Color, _ bool) {
	bgExplicit := !bg.IsDefault()
	fg2 := resolveTermColor(fg, defaultFg)
	bg2 := resolveTermColor(bg, defaultBg)
	if inverse {
		fg2, bg2 = bg2, fg2
		bgExplicit = !bgExplicit
	}
	return fg2, bg2, bgExplicit
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

func ensureContrastColor(fg, bg color.Color) color.Color {
	if fg == nil || bg == nil {
		return fg
	}
	fl := colorLuma8(fg)
	bl := colorLuma8(bg)
	if absInt(fl-bl) >= 125 {
		return fg
	}
	if bl >= 128 {
		return scaleColorRGB(fg, minFloat64(0.4, 0.72*(255.0/maxFloat64(1, float64(fl)))))
	}
	return tintColorRGB(fg, 0.55)
}

func isLightColor(c color.Color) bool {
	if c == nil {
		return false
	}
	r, g, b, _ := c.RGBA()
	lum := (299*r + 587*g + 114*b + 500) / 1000
	return lum >= 0x8000
}

func colorLuma8(c color.Color) int {
	r, g, b, _ := c.RGBA()
	return int((299*r + 587*g + 114*b + 500) / 1000 >> 8)
}

func scaleColorRGB(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(clampInt(int(float64(r>>8)*factor), 0, 255)),
		G: uint8(clampInt(int(float64(g>>8)*factor), 0, 255)),
		B: uint8(clampInt(int(float64(b>>8)*factor), 0, 255)),
		A: uint8(a >> 8),
	}
}

func tintColorRGB(c color.Color, amount float64) color.Color {
	r, g, b, a := c.RGBA()
	mix := func(v uint32) uint8 {
		u := float64(v >> 8)
		u += (255 - u) * amount
		return uint8(clampInt(int(u), 0, 255))
	}
	return color.RGBA{mix(r), mix(g), mix(b), uint8(a >> 8)}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
