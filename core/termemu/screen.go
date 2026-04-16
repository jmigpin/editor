package termemu

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"maps"
	"slices"
)

type Screen struct {
	//bounds      R   // min=0
	region      R   // top/bottom scroll region + left/right margins
	regionLeft  int // x region active on privmode
	regionRight int // x region active on privmode

	//sizeInPixels P // TODO: sixel support?

	grid  *Grid
	grid1 *Grid
	grid2 *Grid // alternate screen buffer

	cursor   P
	curAttr  Attr
	wrapNext bool // autowrap support

	privModes PrivModes
	graphics  Graphics
	tabStops  []bool // len==W; true where a tab stop exists

	csiSaveCursor        SaveCursor
	escSaveCursorAndAttr struct {
		SaveCursor
		attr Attr
	}

	col132Mode         bool
	onColumnModeChange func()

	testing bool
}

func NewScreen() *Screen {
	s := &Screen{}
	s.privModes = *newPrivModes()
	s.graphics = *newGraphics()

	s.grid1 = newGrid(s)
	s.grid1.hasScrollBack = true

	s.grid2 = newGrid(s)
	//s.grid2.hasScrollBack = true

	s.setGrid2(false)

	_, _ = s.setSize(P{1, 1})

	return s
}

//----------

func (s *Screen) setSize(size P) (_ P, changed bool) {
	size = s.clampSize(size)

	if size == s.grid.size {
		return size, false
	}

	s.grid1.resize(size)
	s.grid2.resize(size)

	s.updateRegion()

	clampInR(&s.cursor, s.grid.bounds())

	s.initTabStops()

	return size, true
}

func (s *Screen) clampSize(p P) P {
	p.X = max(p.X, 1)
	p.Y = max(p.Y, 1)
	if s.privModes.column132() {
		p.X = 132
	}
	return p
}

//----------

func (s *Screen) updateColumnMode() {
	if m := s.privModes.column132(); m == s.col132Mode {
		return
	} else {
		s.col132Mode = m
		_, _ = s.setSize(s.grid.size)
		if s.onColumnModeChange != nil {
			s.onColumnModeChange()
		}
	}
}

//----------

func (s *Screen) updateRegion() {
	s.region = s.grid.bounds()
	s.updateRegionX()
}

func (s *Screen) updateRegionX() {
	if s.privModes.leftRightMargin() {
		s.region.Min.X = s.regionLeft
		s.region.Max.X = s.regionRight
	} else {
		s.region.Min.X = 0
		s.region.Max.X = s.grid.size.X
	}
}

func (s *Screen) Clone() *Screen {
	s2 := *s // copy

	s2.grid1 = s.grid1.clone()
	s2.grid2 = s.grid2.clone()
	// fix pointers to point to the new screen
	s2.grid1.scr = &s2
	s2.grid2.scr = &s2

	s2.setGrid2(s.grid == s.grid2)

	s2.privModes = *s.privModes.clone()
	s2.graphics = *s.graphics.clone()
	return &s2
}

//----------

func (s *Screen) setGrid2(on bool) {
	if on {
		s.grid = s.grid2
	} else {
		s.grid = s.grid1
	}
}

//----------

func (s *Screen) clampRegionY() {
	clampInY(&s.region.Min.Y, s.grid.bounds())
	clampInYInclusive(&s.region.Max.Y, s.grid.bounds())
}

func (s *Screen) clampRegionLeftRight() {
	clampInX(&s.regionLeft, s.grid.bounds())
	clampInXInclusive(&s.regionRight, s.grid.bounds())
}

//----------

// dynamic: depends on p; if inside the region then region, otherwise full size
func (s *Screen) dynBounds(p P) R {
	if p.In(s.region) {
		return s.region
	}
	return s.grid.bounds()
}

//----------

func (s *Screen) clearLineInBounds(y int) {
	s.clearLinesInBounds(y, 1)
}
func (s *Screen) clearLinesInBounds(y, n int) {
	r := s.grid.bounds()
	r.Min.Y = y
	r.Max.Y = y + n
	s.grid.clearR(r)
}

//----------

func (s *Screen) cancelWrap() {
	s.wrapNext = false
}

//----------

func (s *Screen) IsCursor(x, y int) bool {
	return P{x, y} == s.cursor
}

//----------
//----------

func (s *Screen) putRune(ru rune) {
	if s.graphics.isSpecial() {
		ru = mapDecSpecial(ru)
	}

	if s.privModes.insert() {
		s.cancelWrap()
		s.csiIch_insertChars(1)
	} else {
		// apply pending wrap first
		if s.wrapNext {
			s.cancelWrap()
			// UX-ADAPTATION: mark previous line as wrapped; ScreenPrinter will omit the \n, allowing the editor to perform visual wrapping and indentation.
			line := s.grid.line(s.cursor.Y)
			line.Wrapped = true
			// UX-ADAPTATION: truncate any old off-screen data to ensure clean visual wrap in the editor.
			line.cells = line.cells[:s.grid.size.X]

			s.carriageReturn()
			s.lineFeed()
		}
	}

	*s.grid.cell(s.cursor) = Cell{R: ru, A: s.curAttr}

	if s.privModes.insert() {
		s.cursor.X++
		clampInX(&s.cursor.X, s.grid.bounds())
	} else {
		if s.cursor.X >= s.dynBounds(s.cursor).Max.X-1 {
			if s.privModes.autoWrap() {
				// do not move now; set wrap for the *next* printable
				s.wrapNext = true
			} // else: stay at last column, overwrite
		} else {
			s.cursor.X++
		}
	}
}

//----------

func (s *Screen) carriageReturn() {
	s.cancelWrap()
	s.cursor.X = s.dynBounds(s.cursor).Min.X
}

func (s *Screen) lineFeed() {
	s.cancelWrap()

	if s.privModes.LineFeedNewline() {
		s.carriageReturn()
	}

	r := s.dynBounds(s.cursor)
	if s.cursor.Y == r.Max.Y-1 {
		s.grid.scrollUpR(r, 1)
	} else {
		s.cursor.Y++
	}
}

func (s *Screen) backspace() {
	s.cancelWrap()
	r := s.dynBounds(s.cursor)
	s.cursor.X--
	clampInR(&s.cursor, r)
}

//----------

func (s *Screen) csiVpa_moveToRow(row1 int) { // 1-based
	s.cancelWrap()
	s.cursor.Y = row1 - 1
	clampInR(&s.cursor, s.grid.bounds())
}

//----------

func (s *Screen) setScrollRegion(top1, bot1 int) {
	s.region.Min.Y = top1 - 1
	s.region.Max.Y = bot1 - 1 + 1
	s.clampRegionY()

	// set cursor to home
	s.cursor = P{0, 0}
	if s.privModes.origin() {
		s.cursor.Y = s.region.Min.Y
	}
	if s.privModes.leftRightMargin() {
		s.cursor.X = s.region.Min.X
	}
}

//----------

func (s *Screen) initTabStops() {
	w := s.grid.size.X
	s.tabStops = make([]bool, w)
	for x := 8; x < w; x += 8 { // every 8 cols
		s.tabStops[x] = true
	}
}

func (s *Screen) nextTabX(p P) int {
	maxX := s.dynBounds(p).Max.X - 1
	for i := p.X + 1; i < maxX; i++ {
		if s.tabStops[i] {
			return i
		}
	}
	return maxX
}
func (s *Screen) prevTabX(p P) int {
	minX := s.dynBounds(p).Min.X
	for i := p.X - 1; i >= minX; i-- {
		if s.tabStops[i] {
			return i
		}
	}
	return minX
}

//----------
//----------

func (s *Screen) csiSlrm_setLeftRightMargins(left1, right1 int) {
	s.cancelWrap()
	s.regionLeft = left1 - 1
	s.regionRight = right1 - 1 + 1
	s.clampRegionLeftRight()
	s.updateRegionX()
}

//----------
//----------

func (s *Screen) csiCup_cursorPosition(row1, col1 int) {
	s.cancelWrap()
	row, col := row1-1, col1-1
	p := P{X: col, Y: row}

	clampInR(&p, s.grid.bounds())

	if s.privModes.leftRightMargin() {
		p.X += s.region.Min.X
		clampInX(&p.X, s.region)
	}
	if s.privModes.origin() {
		p.Y += s.region.Min.Y
		clampInY(&p.Y, s.region)
	}

	s.cursor = p
}

func (s *Screen) csiCuu_cursorUp(v int) {
	s.cancelWrap()
	r := s.dynBounds(s.cursor)
	s.cursor.Y -= v
	clampInR(&s.cursor, r)
}
func (s *Screen) csiCud_cursorDown(v int) {
	s.cancelWrap()
	r := s.dynBounds(s.cursor)
	s.cursor.Y += v
	clampInR(&s.cursor, r)
}
func (s *Screen) csiCuf_cursorForward(v int) {
	s.cancelWrap()
	r := s.dynBounds(s.cursor)
	s.cursor.X += v
	clampInR(&s.cursor, r)
}
func (s *Screen) csiCub_cursorBackward(v int) {
	s.cancelWrap()
	r := s.dynBounds(s.cursor)
	s.cursor.X -= v
	clampInR(&s.cursor, r)
}

func (s *Screen) csiEd_eraseInDisplay(mode int) {
	s.cancelWrap()
	switch mode {
	case 0: // cursor to end
		s.csiEl_eraseInLine(0)

		b := s.grid.bounds() // copy
		b.Min.Y = s.cursor.Y + 1
		s.grid.clearR(b)
	case 1: // home to cursor
		b := s.grid.bounds() // copy
		b.Max.Y = s.cursor.Y
		s.grid.clearR(b)

		s.csiEl_eraseInLine(1)
	case 2: // entire screen
		s.grid.clearBounds()
	case 3: // entire screen and the scrollback buffer
		s.grid.clearBounds()
		s.grid.scrollBack = nil
	default:
		// ignore unknown ED modes
	}
}

func (s *Screen) csiEl_eraseInLine(mode int) {
	s.cancelWrap()
	switch mode {
	case 0: // cursor to end
		n := s.grid.size.X - s.cursor.X
		s.grid.clearRangeX(s.cursor, n)
	case 1: // start to cursor
		n := s.cursor.X + 1
		s.grid.clearRangeX(P{0, s.cursor.Y}, n)
	default: // 2: whole line
		s.clearLineInBounds(s.cursor.Y)
	}
}

func (s *Screen) csiSgr_selectGraphicRendition(params []int) {
	if len(params) == 0 {
		s.curAttr = Attr{}
		return
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		switch {
		case p == 0:
			s.curAttr = Attr{}
		case p == 1:
			s.curAttr.Bold = true
		case p == 7:
			s.curAttr.Inverse = true
		case p == 22:
			s.curAttr.Bold = false // also faint=false
		case p == 27:
			s.curAttr.Inverse = false

		case 30 <= p && p <= 37:
			s.curAttr.Fg = xterm256Color(p - 30)
		case p == 39:
			s.curAttr.Fg = nil

		case 40 <= p && p <= 47:
			s.curAttr.Bg = xterm256Color(p - 40)
		case p == 49:
			s.curAttr.Bg = nil

		// bright options
		case 90 <= p && p <= 97:
			s.curAttr.Fg = xterm256Color(8 + p - 90)
		case 100 <= p && p <= 107:
			s.curAttr.Bg = xterm256Color(8 + p - 100)

		// 256 colors + rgb colors
		case p == 38 || p == 48:
			if i+2 < len(params) && params[i+1] == 5 {
				n := params[i+2]
				if 0 <= n && n <= 255 {
					if p == 38 {
						s.curAttr.Fg = xterm256Color(n)
					} else {
						s.curAttr.Bg = xterm256Color(n)
					}
				}
				i += 2
			} else if i+4 < len(params) && params[i+1] == 2 {
				r, g, b := params[i+2], params[i+3], params[i+4]
				if 0 <= r && r <= 255 && 0 <= g && g <= 255 && 0 <= b && b <= 255 {
					c := color.RGBA{uint8(r), uint8(g), uint8(b), 255}
					if p == 38 {
						s.curAttr.Fg = c
					} else {
						s.curAttr.Bg = c
					}
				}
				i += 4
			}
		}
	}
}

func (s *Screen) csiIch_insertChars(n int) {
	s.cancelWrap()

	r0 := s.grid.bounds()

	n = clamp(n, 0, r0.Max.X-s.cursor.X)

	// shift right
	dst := s.cursor.Add(P{n, 0})
	s.grid.copyRangeX(dst, s.cursor.X, r0.Max.X-n)

	// clear left
	s.grid.clearRangeX(s.cursor, n)
}

func (s *Screen) csiDch_deleteChars(n int) {
	s.cancelWrap()

	r0 := s.grid.bounds()

	n = clamp(n, 0, r0.Max.X-s.cursor.X)

	// shift left
	dst := s.cursor
	s.grid.copyRangeX(dst, s.cursor.X+n, r0.Max.X)

	// clear right
	dst2 := s.cursor
	dst2.X = r0.Max.X - n
	s.grid.clearRangeX(dst2, n)
}

func (s *Screen) csiEch_eraseChars(n int) {
	s.cancelWrap()
	s.grid.clearRangeX(s.cursor, n)
}

func (s *Screen) csiCpr_cursorPositionReport() (int, int) {
	row1 := s.cursor.Y + 1
	col1 := s.cursor.X + 1
	return row1, col1
}

//----------

// region only: insert n blank lines at cursor row within region
func (s *Screen) csiIl_insertLines(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}

	r.Min.Y = s.cursor.Y
	s.grid.scrollDownR(r, n)
}

// region only: delete n lines at cursor row within region
func (s *Screen) csiDl_deleteLines(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}

	r.Min.Y = s.cursor.Y
	s.grid.scrollUpR(r, n)
}

// region only
func (s *Screen) csiSu_scrollUp(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}
	n = clamp(n, 1, r.Dy())
	s.grid.scrollUpR(r, n)
}

// region only
func (s *Screen) csiSd_scrollDown(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}
	n = clamp(n, 1, r.Dy())
	s.grid.scrollDownR(r, n)
}

//----------

func (s *Screen) csiCht_cursorHorizontalTabulation(n int) {
	s.escHt_tab(n)
}
func (s *Screen) csiCha_cursorHorizontalAbsolute(col1 int) {
	s.cursor.X = col1 - 1
	clampInR(&s.cursor, s.grid.bounds())
}
func (s *Screen) csiCbt_cursorBackwardTab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.X = s.prevTabX(s.cursor)
	}
}

func (s *Screen) csiTbc_tabClear(ps int) {
	switch ps {
	case 0: // at cursor
		x := s.cursor.X
		if 0 <= x && x < s.grid.size.X {
			s.tabStops[x] = false
		}
	case 3: // all
		for i := range s.tabStops {
			s.tabStops[i] = false
		}
	default:
		// TBC 1/2 are rarely implemented; safely ignore.
	}
}

func (s *Screen) csiScp_saveCursorPos() {
	s.csiSaveCursor.save(s)
}
func (s *Screen) csiRcp_restoreCursorPos() {
	s.csiSaveCursor.restore(s)
}

func (s *Screen) csiCnl_cursorNextLine(n int) {
	for i := 0; i < n; i++ {
		s.carriageReturn()
		s.escInd_index() // down 1, scroll inside T/B margins
	}
}

func (s *Screen) csiCpl_cursorPreviousLine(n int) {
	for i := 0; i < n; i++ {
		s.carriageReturn()
		s.escRi_reverseIndex() // up 1, scroll inside T/B margins
	}
}

//----------
//----------

func (s *Screen) escInd_index() {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}

	s.cancelWrap()

	if s.cursor.Y == r.Max.Y-1 {
		s.grid.scrollUpR(r, 1)
	} else {
		s.cursor.Y++
	}
}

func (s *Screen) escRi_reverseIndex() {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}

	s.cancelWrap()

	if s.cursor.Y == r.Min.Y {
		s.grid.scrollDownR(r, 1)
	} else {
		s.cursor.Y--
	}
}

//----------

func (s *Screen) escHt_tab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.X = s.nextTabX(s.cursor)
	}
}

func (s *Screen) escHts_horizontalTabSet() {
	x := s.cursor.X
	if 0 <= x && x < s.grid.size.X {
		s.tabStops[x] = true
	}
}

func (s *Screen) escSc_saveCursor() {
	s.escSaveCursorAndAttr.save(s)
	s.escSaveCursorAndAttr.attr = s.curAttr
}
func (s *Screen) escRc_restoreCursor() {
	s.escSaveCursorAndAttr.restore(s)
	s.curAttr = s.escSaveCursorAndAttr.attr
}

func (s *Screen) escNel_nextLine() {
	s.cancelWrap()
	s.carriageReturn()
	s.lineFeed()
}

func (s *Screen) escRis_reset(hard bool) {
	s.privModes = *newPrivModes()
	s.graphics = *newGraphics()
	s.wrapNext = false
	s.updateRegion()
	s.initTabStops()
	if hard {
		s.cursor = P{}
		s.grid1.clearBounds()
		s.grid2.clearBounds()
		s.grid1.scrollBack = nil
		s.grid2.scrollBack = nil
	}
}

func (s *Screen) escAln_screenAlignment() {
	s.cancelWrap()
	for y := 0; y < s.grid.size.Y; y++ {
		for x := 0; x < s.grid.size.X; x++ {
			s.grid.lines[y].cells[x] = Cell{R: 'E', A: s.curAttr}
		}
	}
	s.cursor = P{}
}

//----------

//----------
//----------

// useful for debug
func (scr *Screen) Print() {
	fmt.Println(scr.Sprint(false))
}
func (scr *Screen) PrintWithCursor() {
	fmt.Println(scr.Sprint(true))
}

// bytes print
func (scr *Screen) Bprint(cursor bool) []byte {
	sp := NewScreenPrinter()
	sp.testing = scr.testing
	if cursor {
		sp.CursorRune = '◙'
	}
	return sp.Bprint(scr)
}

func (scr *Screen) Sprint(cursor bool) string {
	return string(scr.Bprint(cursor))
}

// quoted print
func (scr *Screen) Qprint(cursor bool) string {
	return fmt.Sprintf("%q", scr.Bprint(cursor))
}

//----------
//----------
//----------

type P = image.Point     // 0-based
type R = image.Rectangle // r.Max exclusive

//----------

func clampInR(p *P, r R) {
	clampInX(&p.X, r)
	clampInY(&p.Y, r)
}

func clampInX(v *int, r R) {
	*v = clamp(*v, r.Min.X, r.Max.X-1)
}
func clampInY(v *int, r R) {
	*v = clamp(*v, r.Min.Y, r.Max.Y-1)
}

func clampInXInclusive(v *int, r R) {
	*v = clamp(*v, r.Min.X, r.Max.X)
}
func clampInYInclusive(v *int, r R) {
	*v = clamp(*v, r.Min.Y, r.Max.Y)
}

func clamp(v, lo, hi int) int {
	return min(hi, max(lo, v))
}

//----------

type Grid struct {
	size  P
	lines []GridLine

	hasScrollBack bool
	scrollBack    []byte

	scr *Screen
}

func newGrid(scr *Screen) *Grid {
	return &Grid{scr: scr}
}

func (g *Grid) bounds() R {
	return R{Max: g.size}
}

func (g *Grid) line(y int) *GridLine {
	return &g.lines[y]
}
func (g *Grid) cell(p P) *Cell {
	return g.line(p.Y).cell(p.X)
}

//----------

func (g *Grid) copyR(dst P, r R) {
	w := []GridLine{}
	// copy to tmp first to allow correct overwriting
	for y := r.Min.Y; y < r.Max.Y; y++ {
		w = append(w, g.line(y).Clone())
	}

	// copy to the destination
	for k, gl := range w {
		line := g.line(dst.Y + k)
		if dst.X == 0 && r.Min.X == 0 && r.Max.X == g.size.X {
			line.cells = gl.cells
			line.Wrapped = gl.Wrapped
		} else {
			copy(line.cells[dst.X:], gl.cells[r.Min.X:r.Max.X])
		}
	}
}
func (g *Grid) copyRangeX(dst P, minX, maxX int) {
	g.copyR(dst, R{P{minX, dst.Y}, P{maxX, dst.Y + 1}})
}

//----------

func (g *Grid) clearBounds() {
	g.clearR(g.bounds())
}

func (g *Grid) clearR(r R) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		g.clearLineCells(y, r.Min.X, r.Max.X)
	}
}
func (g *Grid) clearRangeX(dst P, n int) {
	n = min(n, g.size.X-dst.X)
	g.clearLineCells(dst.Y, dst.X, dst.X+n)
}

func (g *Grid) clearLineCells(y int, x0, x1 int) {
	line := g.line(y)

	for x := x0; x < x1; x++ {
		*line.cell(x) = Cell{A: g.scr.curAttr}
	}

	if x0 == 0 && x1 == g.size.X {
		line.Wrapped = false
		line.cells = line.cells[:g.size.X]
	}
}

//----------

// shifts up, blanks bottom
func (g *Grid) scrollUpR(r0 R, n int) {
	n = clamp(n, 0, r0.Dy())

	//----------

	// keep scrollback
	if g.hasScrollBack &&
		r0.Min == (P{0, 0}) &&
		r0.Max.X == g.size.X {

		sb := &g.scrollBack
		for i := range n {
			line := g.line(i)
			for _, c := range line.cells {
				*sb = appendRune(*sb, c.printableRune())
			}

			// clean end of line to avoid space wraps
			*sb = bytes.TrimRight(*sb, " \t")
			if !line.Wrapped {
				*sb = appendRune(*sb, '\n')
			}
		}
	}

	//----------

	// move rows [top+n..bot] up by n
	dst := r0.Min
	r1 := r0
	r1.Min.Y += n
	g.copyR(dst, r1)

	// clear bottom rows
	r2 := r0
	r2.Min.Y = r0.Max.Y - n
	g.clearR(r2)
}

// shift down, blanks top
func (g *Grid) scrollDownR(r0 R, n int) {
	n = clamp(n, 0, r0.Dy())

	// move rows [top..bot-n] down by n
	dst := r0.Min.Add(P{0, n})
	r1 := r0
	r1.Max.Y -= n
	g.copyR(dst, r1)

	// clear top rows
	r2 := r0
	r2.Max.Y = r0.Min.Y + n
	g.clearR(r2)
}

//----------

func (g *Grid) clone() *Grid {
	g2 := *g
	g2.lines = make([]GridLine, g.size.Y)
	for i := range g.size.Y {
		g2.lines[i] = g.lines[i].Clone()
	}
	g2.scrollBack = slices.Clone(g.scrollBack)
	return &g2
}

func (g *Grid) resize(size P) {
	if d := size.Y - g.size.Y; d > 0 {
		g.lines = append(g.lines, newGridLines(P{size.X, d})...)
	} else if d < 0 {
		d = -d
		// In shrinking height we scroll only the minimal number of rows needed to keep the cursor visible in the new viewport, because scrolling by all removed rows ("d") can over-scroll content, duplicate entries in scrollback, and desynchronize cursor position/ reporting after resize; therefore "need" is computed as cursorY-(newHeight-1) and clamped to [0,d].
		need := g.scr.cursor.Y - (size.Y - 1)
		need = clamp(need, 0, d)
		if need > 0 {
			g.scrollUpR(g.bounds(), need)
			g.scr.cursor.Y -= need
		}
		g.lines = g.lines[:size.Y] // truncate
	}

	for i := range g.lines {
		line := &g.lines[i]
		if d := size.X - len(line.cells); d > 0 {
			line.cells = append(line.cells, make([]Cell, d)...)
		} else if d < 0 {
			// UX-ADAPTATION: keep logical content (cells beyond physical width) even if physical width decreased, preserving existing output during resize.
		}
	}

	g.size = size
}

//----------

type GridLine struct {
	cells   []Cell // logical line
	Wrapped bool   // if true, next line is logical continuation of this one
}

func newGridLine(x int) GridLine {
	return GridLine{cells: make([]Cell, x)}
}
func newGridLines(size P) []GridLine {
	w := make([]GridLine, size.Y)
	for i := range w {
		w[i] = newGridLine(size.X)
	}
	return w
}

func (gl *GridLine) cell(x int) *Cell {
	return &gl.cells[x]
}

func (gl *GridLine) Clone() GridLine {
	gl2 := *gl
	gl2.cells = slices.Clone(gl.cells)
	return gl2
}

//----------

type Cell struct {
	R rune
	A Attr
}

func (c *Cell) printableRune() rune {
	if c.R == 0 {
		return ' '
	}
	return c.R
}

//----------

type Attr struct {
	Fg      color.Color
	Bg      color.Color
	Bold    bool
	Inverse bool // inverse fg/bg
}

//----------
//----------
//----------

type SaveCursor struct {
	ok bool
	c  P    // cursor
	wn bool // wrapnext
	//aw bool // autowrap
}

func (c *SaveCursor) save(s *Screen) {
	c.ok = true
	c.c = s.cursor
	c.wn = s.wrapNext
	//c.aw = s.pmodes.autoWrap()
}
func (c *SaveCursor) restore(s *Screen) {
	if !c.ok {
		return
	}
	s.cursor = c.c
	clampInR(&s.cursor, s.grid.bounds())
	s.wrapNext = c.wn
	//s.pmodes.set("?7", c.aw)
}

//----------
//----------
//----------

// PrivModes keeps DEC private modes (?n).
type PrivModes struct {
	m map[string]bool
}

func newPrivModes() *PrivModes {
	m := &PrivModes{m: make(map[string]bool)}
	// defaults
	m.set("20", true)  // line feed newline
	m.set("?2", true)  // ansi
	m.set("?7", true)  // auto wrap
	m.set("?25", true) // show cursor
	return m
}

func (m *PrivModes) set(idx string, on bool) { m.m[idx] = on }
func (m *PrivModes) isOn(idx string) bool    { return m.m[idx] }

func (m *PrivModes) clone() *PrivModes {
	m2 := *m // copy
	m2.m = maps.Clone(m.m)
	return &m2
}

//----------

func (m *PrivModes) insert() bool          { return m.isOn("4") }
func (m *PrivModes) LineFeedNewline() bool { return m.isOn("20") }

func (m *PrivModes) AppCursorKeys() bool      { return m.isOn("?1") }
func (m *PrivModes) AnsiNotVT52() bool        { return m.isOn("?2") }
func (m *PrivModes) column132() bool          { return m.isOn("?3") }
func (m *PrivModes) softScroll() bool         { return m.isOn("?4") }
func (m *PrivModes) reverseVideo() bool       { return m.isOn("?5") }
func (m *PrivModes) origin() bool             { return m.isOn("?6") }
func (m *PrivModes) autoWrap() bool           { return m.isOn("?7") }
func (m *PrivModes) autoRepeat() bool         { return m.isOn("?8") }
func (m *PrivModes) showCursor() bool         { return m.isOn("?25") }
func (m *PrivModes) leftRightMargin() bool    { return m.isOn("?69") }
func (m *PrivModes) AlternateBuffer() bool    { return m.isOn("?1049") }
func (m *PrivModes) BracketedPaste() bool     { return m.isOn("?2004") }
func (m *PrivModes) SynchronizedOutput() bool { return m.isOn("?2026") }

//----------
//----------
//----------

type Graphics struct {
	sel  string
	bank map[string]string
}

func newGraphics() *Graphics {
	gr := &Graphics{}
	gr.bank = map[string]string{}
	// defaults
	gr.sel = "g0"
	gr.bank["g0"] = "ascii"
	gr.bank["g1"] = "special"
	return gr
}

func (gr *Graphics) set(kind, typ string) {
	if typ != "" {
		gr.bank[kind] = typ // designate
	} else {
		gr.sel = kind // select
	}
}
func (gr *Graphics) isSpecial() bool {
	if u, ok := gr.bank[gr.sel]; ok {
		return u == "special"
	}
	return false
}

func (gr *Graphics) clone() *Graphics {
	gr2 := *gr // copy
	gr2.bank = maps.Clone(gr.bank)
	return &gr2
}

//----------
//----------
//----------

func mapDecSpecial(r rune) rune {
	var decSpec = map[rune]rune{
		'j': '┘', 'k': '┐', 'l': '┌', 'm': '└',
		'n': '┼', 'q': '─', 'x': '│',
		't': '├', 'u': '┤', 'v': '┴', 'w': '┬',
		'y': '≤', 'z': '≥', '{': 'π', '|': '≠', '}': '£', '~': '·',
	}
	if v, ok := decSpec[r]; ok {
		return v
	}
	return r
}

//----------

func grayscaleColor(c color.Color) color.Color {
	if c == nil {
		return nil
	}
	r, g, b, a := c.RGBA()
	// luma approximation on 16-bit channels
	y := uint16((299*r + 587*g + 114*b + 500) / 1000)
	return color.RGBA64{y, y, y, uint16(a)}
}

func xterm256Color(n int) color.Color {
	switch {
	case 0 <= n && n <= 15:
		ansi16 := [16]color.RGBA{
			{0, 0, 0, 255},       // 0
			{205, 0, 0, 255},     // 1
			{0, 205, 0, 255},     // 2
			{205, 205, 0, 255},   // 3
			{0, 0, 238, 255},     // 4
			{205, 0, 205, 255},   // 5
			{0, 205, 205, 255},   // 6
			{229, 229, 229, 255}, // 7
			{127, 127, 127, 255}, // 8
			{255, 0, 0, 255},     // 9
			{0, 255, 0, 255},     // 10
			{255, 255, 0, 255},   // 11
			{92, 92, 255, 255},   // 12
			{255, 0, 255, 255},   // 13
			{0, 255, 255, 255},   // 14
			{255, 255, 255, 255}, // 15
		}
		return ansi16[n]
	case 16 <= n && n <= 231:
		k := n - 16
		levels := [6]uint8{0, 95, 135, 175, 215, 255}
		r := levels[k/36]
		g := levels[(k/6)%6]
		b := levels[k%6]
		return color.RGBA{r, g, b, 255}
	case 232 <= n && n <= 255:
		v := uint8(8 + (n-232)*10)
		return color.RGBA{v, v, v, 255}
	default:
		panic("!")
	}
}
