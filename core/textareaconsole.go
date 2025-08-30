package core

import (
	"fmt"
	"image/color"
	"io"
	"sync"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/fontutil"
)

// implements [termemu.ConsoleConn] interface
// user side in (user<->emulator<->exec)
type TextAreaConsole struct {
	erow *ERow
	rwc  io.ReadWriteCloser
	temu *termemu.Emu

	paint struct {
		sync.Mutex
		on bool
	}

	sp *termemu.ScreenPrinter
}

func newTextAreaConsole(erow *ERow, rwc io.ReadWriteCloser) *TextAreaConsole {
	tac := &TextAreaConsole{erow: erow, rwc: rwc}

	tac.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tac.erow.Row.TextArea.EnableTerminalColors(true)
	})

	tac.sp = termemu.NewScreenPrinter()

	return tac
}

//----------

func (tac *TextAreaConsole) Read(p []byte) (int, error) {
	return tac.rwc.Read(p)
}
func (tac *TextAreaConsole) Write(p []byte) (int, error) {
	return tac.rwc.Write(p)
}
func (tac *TextAreaConsole) Close() error {
	defer func() {
		tac.erow.Ed.UI.RunOnUIGoRoutine(func() {
			ta := tac.erow.Row.TextArea
			ta.EnableSyntaxHighlight(true) // TODO: always on or set to previous?
			ta.EnableTerminalColors(false)
			ta.SetTerminalColorOps(nil) // clear to avoid bad caloring upon re-enable

			//tac.erow.Row.ScrollArea.SetBars(false, true)

			//ta.SetBytesClearHistory(nil)	// commented: clearing hides output of temporary cmds (ex: ls)
		})
	}()
	return tac.rwc.Close()
}

//----------

func (tac *TextAreaConsole) Error(err error) {
	tac.erow.Ed.Error(err)
}
func (tac *TextAreaConsole) Print(v any) {
	tac.erow.Ed.Message(fmt.Sprint(v))
}

//----------

func (tac *TextAreaConsole) SetSize(w, h int) {
	tac.erow.termOpts.W, tac.erow.termOpts.H = w, h
	updateConsoleFontSize(tac.erow)
}

//----------

func (tac *TextAreaConsole) Repaint() {
	if tac.temu == nil {
		return
	}

	// performance: avoid calling paint too many times
	tac.paint.Lock()
	defer tac.paint.Unlock()
	if !tac.paint.on {
		tac.paint.on = true
		tac.erow.Ed.UI.RunOnUIGoRoutine(func() {
			tac.paint.Lock()
			tac.paint.on = false
			tac.paint.Unlock()
			tac.paintNow()
		})
	} // else a paint call is already on the stack
}
func (tac *TextAreaConsole) paintNow() {
	ops, b := tac.paintOpsBytes()
	ta := tac.erow.Row.TextArea
	ta.EnableSyntaxHighlight(false)
	ta.SetTerminalColorOps(ops)

	//ta.SetBytesClearHistory(b) // issue: no scroll mode

	//ta.SetBytesClearHistory(nil)        // issue: sets cursor to zero
	//tac.erow.AppendBytesClearHistory(b)

	tac.erow.OverwriteBytesClearHistory(0, ta.RW().Max(), b)

	// TODO
	//ta.
	//imageutil.FillRectangle(bgf.d.st.drawR.img, r, st.lineBg)

	//tac.erow.Row.ScrollArea.SetBars(false, false)
}
func (tac *TextAreaConsole) paintOpsBytes() ([]*D4COp, []byte) {
	scr := tac.temu.Snapshot()

	dops := []*D4COp{}

	// defaults colors for inverse video
	tfg := tac.erow.Row.TextArea.TreeThemePaletteColor("text_fg")
	tbg := tac.erow.Row.TextArea.TreeThemePaletteColor("text_bg")
	defColors := func(fg, bg color.Color) (_, _ color.Color) {
		if fg == nil {
			fg = tfg
		}
		if bg == nil {
			bg = tbg
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

	//----------

	tac.sp.Border = true
	tac.sp.Cursor = true
	tac.sp.ColorFn = addColor0
	bs := tac.sp.Bprint(scr)

	return dops, bs
}

//----------
//----------
//----------

type D4COp = drawer4.ColorizeOp

//----------
//----------
//----------

func updateConsoleFontSize(erow *ERow) {
	if erow.termOpts.Mode == termemu.ModeUI {
		setConsoleFontSize(erow)
	}
}

func setConsoleFontSize(erow *ERow) {
	w := max(80, erow.termOpts.W)
	h := max(24, erow.termOpts.H)

	// TODO: get this from the emu screen
	// extra border drawn around the snapshot
	w += 2 + 1 // +1 is the extra space set at start on the left side
	h += 2

	ta := erow.Row.TextArea

	// use mono font
	origFace := erow.termOpts.origFace
	face := origFace
	if !face.TestIsMono() {
		face = fontutil.DefaultMonoFont().FontFace(face.Opts)
	}

	// TODO: max font size

	faceAtSize := func(v float64) *fontutil.FontFace {
		fopts2 := face.Opts // copy
		fopts2.SetSize(float64(v))
		return face.Font.FontFace(fopts2)
	}

	runeSize := func(p float64) (int, int) {
		face2 := faceAtSize(p)
		adv, ok := face2.Face.GlyphAdvance('W')
		if !ok {
			return 0, 0
		}
		return adv.Ceil(), face2.LineHeightInt()
	}

	sx, sy := ta.Bounds.Dx(), ta.Bounds.Dy()
	rx, ry, p := fitRuneSizeF(sx, sy, w, h, runeSize)

	if rx == 0 && ry == 0 && p == 0 {
		ta.SetThemeFontFace(origFace)
		return
	}

	maxFSize := origFace.Opts.Size()
	if p > maxFSize {
		p = maxFSize
	}

	face2 := faceAtSize(p)
	ta.SetThemeFontFace(face2)
}

// finds the largest p (float) such that w×x <= sx and h×y <= sy,
// where (x,y) = runeSize(p) are pixel dims (monotonic non-decreasing in p).
// Returns the chosen (x,y,p). If impossible, returns zeros.
func fitRuneSizeF(sx, sy, w, h int, runeSize func(p float64) (int, int)) (int, int, float64) {
	if sx <= 0 || sy <= 0 || w <= 0 || h <= 0 {
		return 0, 0, 0
	}
	fits := func(p float64) bool {
		x, y := runeSize(p)
		//return x > 0 && y > 0 && w*x <= sx && h*y <= sy
		//return x > 0 && y > 0 && w*x < sx && h*y < sy
		return x > 0 && y > 0 && w*x < sx
	}

	const (
		minP  = 1e-1
		maxP  = 1e3
		eps   = 1e-1 // binary search tolerance on p
		maxIt = 5    // cap iterations
	)

	// Find some fitting p (shrink if needed).
	lo, hi := 0.0, 1.0
	for !fits(hi) && hi > minP {
		hi *= 0.5
	}
	if !fits(hi) {
		return 0, 0, 0
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

	x, y := runeSize(best)
	return x, y, best
}
