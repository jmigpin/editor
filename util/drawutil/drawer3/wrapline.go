package drawer3

import (
	"image/color"
	"unicode"

	"github.com/jmigpin/editor/util/mathutil"
)

var WrapLineRune = rune(8594) // positioned at the start of wrapped line (left)

type WrapLine struct {
	EExt
	Opt  WrapLineOpt
	line *Line
	d    Drawer // needed by SetOn

	// setup values
	data WrapLineData
}

func WrapLine1(line *Line, d Drawer) WrapLine {
	return WrapLine{line: line, d: d}
}

func (wl *WrapLine) SetOn(v bool) {
	if v != wl.EExt.On() {
		wl.d.SetNeedMeasure(true)
	}
	wl.EExt.SetOn(v)
}

func (wl *WrapLine) resetData() {
	// keep previous instance maxX value (needs to be explicitly set on measure)
	wl.data = WrapLineData{maxX: wl.data.maxX}
}

func (wl *WrapLine) Start(r *ExtRunner) {
	wl.resetData()
}

func (wl *WrapLine) Iterate(r *ExtRunner) {
	// don't act if other clones are active (ex: annotations)
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	wl.data.state = WLStateNormal

	penXAdv := r.RR.Pen.X + r.RR.Advance

	// keep track of indentation for wrapped lines
	if !wl.data.NotStartingSpaces {
		if unicode.IsSpace(r.RR.Ru) {
			wl.data.Indent = penXAdv
		} else {
			wl.data.NotStartingSpaces = true
		}
	}

	// wrap line
	// pen.x>0 forces at least one rune per line
	// ri>0 covers the FirstLineOffsetX case
	if penXAdv > wl.data.maxX && r.RR.Pen.X > 0 && r.RR.Ri > 0 {
		if !wl.newline(r) {
			return
		}
	}
	if !r.NextExt() {
		return
	}

	// reset data on newline
	if r.RR.Ru == '\n' {
		wl.resetData()
	}
}

func (wl *WrapLine) newline(r *ExtRunner) bool {
	// wrap line rune advance
	wlrAdv0, _ := r.D.Face().GlyphAdvance(WrapLineRune)
	wlrAdv := mathutil.Intf2(wlrAdv0)

	// wrap line margin-to-left-border minimum
	margin := wlrAdv
	wAdv0, _ := r.D.Face().GlyphAdvance('W')
	wAdv := mathutil.Intf2(wAdv0)
	margin = wlrAdv + 8*wAdv

	// helper vars
	runeAdv := r.RR.Advance
	runeAdv1 := wl.data.maxX - r.RR.Pen.X
	if runeAdv1 < 0 {
		runeAdv1 = 0
	}

	origRu := r.RR.Ru
	r.RR.PushRiClone()

	// bg close to the border
	wl.data.state = WLStateLine1Bg
	r.RR.Ru = 0
	r.RR.Advance = runeAdv1
	if !r.NextExt() {
		return false
	}

	// newline
	wl.line.NewLine(r)
	r.RR.Pen.X = wl.data.Indent

	//// wraplinerune before the indent
	//if r.RR.Pen.X-wlrAdv > 0 {
	//	r.RR.Pen.X -= wlrAdv
	//}

	// make wrap line rune always visible
	if r.RR.Pen.X >= wl.data.maxX-margin {
		r.RR.Pen.X = wl.data.maxX - margin
		if r.RR.Pen.X < 0 {
			r.RR.Pen.X = 0
		}
	}

	startPenX := r.RR.Pen.X

	// bg on start of newline
	wl.data.state = WLStateLine2Bg
	r.RR.Ru = 0
	r.RR.Pen.X = startPenX
	bgAdv := wlrAdv // fixed size
	r.RR.Advance = bgAdv
	if !r.NextExt() {
		return false
	}

	// wraplinerune
	wl.data.state = WLStateLine2Rune
	r.RR.Ru = WrapLineRune
	r.RR.Pen.X = startPenX
	r.RR.Advance = wlrAdv
	if !r.NextExt() {
		return false
	}

	// original rune
	wl.data.state = WLStateNormal
	r.RR.PopRiClone()
	r.RR.Ru = origRu
	r.RR.Pen.X = startPenX + bgAdv
	r.RR.Advance = runeAdv

	return true
}

//----------

// Implements PosDataKeeper
func (wl *WrapLine) KeepPosData() interface{} {
	return wl.data
}

// Implements PosDataKeeper
func (wl *WrapLine) RestorePosData(data interface{}) {
	wl.data = data.(WrapLineData)
}

//----------

type WrapLineData struct {
	state             WLState
	NotStartingSpaces bool // is after first non space char
	Indent            mathutil.Intf
	maxX              mathutil.Intf
}

//----------

type WLState int

const (
	WLStateNormal WLState = iota
	WLStateLine1Bg
	WLStateLine2Bg
	WLStateLine2Rune
)

//----------

type WrapLineOpt struct {
	Fg, Bg color.Color
}

//----------

type WrapLineColor struct {
	EExt
	cc    *CurColors
	wline *WrapLine
}

func WrapLineColor1(wline *WrapLine, cc *CurColors) WrapLineColor {
	return WrapLineColor{wline: wline, cc: cc}
}

func (wlinec *WrapLineColor) Iterate(r *ExtRunner) {
	if !wlinec.wline.On() {
		r.NextExt()
		return
	}

	switch wlinec.wline.data.state {
	case WLStateLine1Bg, WLStateLine2Bg, WLStateLine2Rune:
		if wlinec.wline.Opt.Fg != nil {
			wlinec.cc.Fg = wlinec.wline.Opt.Fg
		}
		if wlinec.wline.Opt.Bg != nil {
			wlinec.cc.Bg = wlinec.wline.Opt.Bg
		}
	}
	r.NextExt()
}
