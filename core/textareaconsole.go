package core

import (
	"io"
	"sync"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/util/fontutil"
)

// implement [termemu.ConsoleConn] interface
type TextAreaConsole struct {
	erow *ERow
	rwc  io.ReadWriteCloser
	temu *termemu.Emu

	paint struct {
		sync.Mutex
		on bool
	}
}

func newTextAreaConsole(erow *ERow, rwc io.ReadWriteCloser) *TextAreaConsole {
	tac := &TextAreaConsole{erow: erow, rwc: rwc}
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
	return tac.rwc.Close()
}

//----------

func (tac *TextAreaConsole) Error(err error) {
	tac.erow.Ed.Error(err)
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
	scr := tac.temu.Snapshot()
	//b := scr.Bytes(false, true)
	b := scr.Bytes(true, true) // full border
	tac.erow.Row.TextArea.SetBytesClearHistory(b)
	tac.erow.Row.TextArea.SetCursorIndex(0)

	//erow.Row.TextArea.MarkNeedsPaint()
	//erow.Row.TextArea.AppendBytesClearHistory(buf.Bytes())
}

//----------

func (tac *TextAreaConsole) SetSize(w, h int) {
	tac.erow.termOpts.W, tac.erow.termOpts.H = w, h
	updateConsoleFontSize(tac.erow)
}

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

	//h += 1000 // TESTING

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
