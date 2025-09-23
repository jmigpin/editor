package termemu

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"maps"
	"slices"

	"golang.org/x/image/colornames"
)

type Screen struct {
	bounds      R // min=0
	region      R // top/bottom scroll region + left/right margins
	regionLeft  int
	regionRight int

	//sizeInPixels P // TODO: sixel support?

	Grid  *[][]Cell // current grid ptr
	grid1 [][]Cell
	grid2 [][]Cell // alternate screen buffer

	ScrollBack  *[]byte
	scrollBack1 []byte // grid1
	//scrollBack2 []byte

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

	onSizeChange func()

	testing bool
}

func NewScreen() *Screen {
	s := &Screen{}
	s.privModes = *newPrivModes()
	s.graphics = *newGraphics()
	s.setGrid2(false)
	s.setSize(P{1, 1}, false) // usual terminal defaults: 80x24
	return s
}

//----------

func (s *Screen) size() P {
	return P{s.bounds.Dx(), s.bounds.Dy()}
}
func (s *Screen) updateSize() { // ex: csi 132 column mode
	s.setSize(s.size(), true)
}
func (s *Screen) setSize(size P, triggerOnChange bool) {
	size = s.clampSize(size)
	if size == s.size() {
		return
	}

	s.bounds.Min = P{}
	s.bounds.Max = size
	s.updateRegion()
	clampInR(&s.cursor, s.bounds)
	s.resizeGrids()
	s.initTabStops()

	if triggerOnChange && s.onSizeChange != nil {
		s.onSizeChange()
	}
}
func (s *Screen) clampSize(size P) P {
	if s.privModes.column132() {
		size.X = 132
	}
	if s.testing {
		size.X = max(size.X, 1)
		size.Y = max(size.Y, 1)
	} else {
		size.X = max(size.X, 50)
		size.Y = max(size.Y, 10)

		//// TESTING
		//size.X = 80
		//size.Y = 24
	}
	return size
}

//----------

func (s *Screen) updateRegion() {
	s.region = s.bounds
	s.updateRegionX()
}

func (s *Screen) Clone() *Screen {
	s2 := *s // copy

	s2.grid1 = cloneGrid(s.grid1)
	s2.grid2 = cloneGrid(s.grid2)
	if s2.Grid == &s.grid1 {
		s2.Grid = &s2.grid1
	} else {
		s2.Grid = &s2.grid2
	}

	s2.scrollBack1 = slices.Clone(s.scrollBack1)
	if s2.ScrollBack == &s.scrollBack1 {
		s2.ScrollBack = &s2.scrollBack1
	} else {
		//s2.ScrollBack = &s2.scrollBack2
		s2.ScrollBack = nil
	}

	s2.privModes = *s.privModes.clone()
	s2.graphics = *s.graphics.clone()
	return &s2
}

//----------

func (s *Screen) clearGrids() {
	s.grid1 = newGrid(s.bounds.Max)
	s.grid2 = newGrid(s.bounds.Max)
}
func (s *Screen) clearGrid() {
	*s.Grid = newGrid(s.bounds.Max)
}

//----------

func (s *Screen) resizeGrids() {
	s.grid1 = resizeGrid(s.grid1, s.bounds.Max)
	s.grid2 = resizeGrid(s.grid2, s.bounds.Max)
}

func (s *Screen) setGrid2(on bool) {
	if on {
		s.Grid = &s.grid2
		//s.ScrollBack = &s.scrollBack2
		s.ScrollBack = nil
	} else {
		s.Grid = &s.grid1
		s.ScrollBack = &s.scrollBack1
	}
}

//----------

func (s *Screen) clampRegionY() {
	clampInY(&s.region.Min.Y, s.bounds)
	clampInYInclusive(&s.region.Max.Y, s.bounds)
}

func (s *Screen) clampRegionLeftRight() {
	clampInX(&s.regionLeft, s.bounds)
	clampInXInclusive(&s.regionRight, s.bounds)
}

func (s *Screen) updateRegionX() {
	if s.privModes.leftRightMargin() {
		s.region.Min.X = s.regionLeft
		s.region.Max.X = s.regionRight
	} else {
		s.region.Min.X = s.bounds.Min.X
		s.region.Max.X = s.bounds.Max.X
	}
}

//----------

// dynamic: depends on p; if inside the region then region, otherwise full size
func (s *Screen) dynBounds(p P) R {
	if p.In(s.region) {
		return s.region
	}
	return s.bounds
}

//----------

func (s *Screen) copyR(dst P, r R) {
	w := [][]Cell{}
	// copy to tmp first to allow correct overwriting
	for y := r.Min.Y; y < r.Max.Y; y++ {
		w = append(w, cloneCells((*s.Grid)[y][r.Min.X:r.Max.X]))
	}
	// copy to the destination
	for k, u := range w {
		copy((*s.Grid)[dst.Y+k][dst.X:], u)
	}
}

func (s *Screen) copyRangeX(dst P, minX, maxX int) {
	s.copyR(dst, R{P{minX, dst.Y}, P{maxX, dst.Y + 1}})
}

//----------

func (s *Screen) clearR(r R) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		s.clearCells((*s.Grid)[y][r.Min.X:r.Max.X])
	}
}

func (s *Screen) clearCells(w []Cell) {
	for i := range w {
		w[i] = Cell{A: s.curAttr}
	}
}

func (s *Screen) clearRangeX(dst P, n int) {
	n = min(n, s.bounds.Dx()-dst.X)
	s.clearCells((*s.Grid)[dst.Y][dst.X : dst.X+n])
}

func (s *Screen) clearLineInBounds(y int) {
	s.clearLinesInBounds(y, 1)
}
func (s *Screen) clearLinesInBounds(y, n int) {
	r := s.bounds
	r.Min.Y = y
	r.Max.Y = y + n
	s.clearR(r)
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

func (s *Screen) putRune(r rune) {
	if s.graphics.isSpecial() {
		r = mapDecSpecial(r)
	}

	if s.privModes.insert() {
		s.cancelWrap()
		s.csiIch_insertChars(1)
	} else {
		// apply pending wrap first
		if s.wrapNext {
			s.cancelWrap()
			s.carriageReturn()
			s.lineFeed()
		}
	}

	(*s.Grid)[s.cursor.Y][s.cursor.X] = Cell{R: r, A: s.curAttr}

	if s.privModes.insert() {
		s.cursor.X++
		clampInX(&s.cursor.X, s.bounds)
	} else {
		if s.cursor.X == s.dynBounds(s.cursor).Max.X-1 {
			if s.privModes.autoWrap() {
				// do not move now; set wrap for the *next* printable
				s.wrapNext = true
			} // else: stay at last column, overwrite subsequent prints
		} else {
			s.cursor.X++
			//clampInX(&s.cursor.X, s.bounds)
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
		s.scrollUpR(r, 1)
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
	clampInR(&s.cursor, s.bounds)
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

// shifts up, blanks bottom
func (s *Screen) scrollUpR(r0 R, n int) {

	n = clamp(n, 0, r0.Dy())

	//----------

	// keep scrollback
	if s.ScrollBack != nil && r0.Min == s.bounds.Min && r0.Max.X == s.bounds.Max.X {
		sb := &s.ScrollBack
		for i := range n {
			for _, c := range (*s.Grid)[i] {
				ru := c.R
				if ru == 0 {
					ru = ' '
				}
				**sb = appendRune(**sb, ru)
			}
			**sb = bytes.TrimRight(**sb, "\n")
			**sb = appendRune(**sb, '\n')
		}
	}

	//----------

	// move rows [top+n..bot] up by 1
	dst := r0.Min
	r1 := r0
	r1.Min.Y += n
	s.copyR(dst, r1)

	// clear bottom rows
	r2 := r0
	r2.Min.Y = r0.Max.Y - n
	s.clearR(r2)

}

// shift down, blanks top
func (s *Screen) scrollDownR(r0 R, n int) {
	n = clamp(n, 0, r0.Dy())

	// move rows [top..bot-n] down by 1
	dst := r0.Min.Add(P{0, n})
	r1 := r0
	r1.Max.Y -= n
	s.copyR(dst, r1)

	// clear top rows
	r2 := r0
	r2.Max.Y = r0.Min.Y + n
	s.clearR(r2)
}

//----------

func (s *Screen) initTabStops() {
	w := s.bounds.Max.X
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

	clampInR(&p, s.bounds)

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

		b := s.bounds // copy
		b.Min.Y = s.cursor.Y + 1
		s.clearR(b)
	case 1: // home to cursor
		b := s.bounds // copy
		b.Max.Y = s.cursor.Y
		s.clearR(b)

		s.csiEl_eraseInLine(1)
	//case 2: // entire screen
	//case 3: // entire screen and the scrollback buffer
	default:
		// TODO: mark as clear, but don't print yet and wait for the next content

		s.clearR(s.bounds)
	}
}

func (s *Screen) csiEl_eraseInLine(mode int) {
	s.cancelWrap()
	switch mode {
	case 0: // cursor to end
		n := s.bounds.Max.X - s.cursor.X
		s.clearRangeX(s.cursor, n)
	case 1: // start to cursor
		n := s.cursor.X + 1
		s.clearRangeX(P{0, s.cursor.Y}, n)
	default: // 2: whole line
		s.clearLineInBounds(s.cursor.Y)
	}
}

func (s *Screen) csiSgr_selectGraphicRendition(params []int) {
	if len(params) == 0 {
		s.curAttr = Attr{}
		return
	}
	for _, p := range params {
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
			u := AttrColor(p - 30)
			s.curAttr.Fg = &u
		case p == 39:
			s.curAttr.Fg = nil
		case 40 <= p && p <= 47:
			u := AttrColor(p - 40)
			s.curAttr.Bg = &u
		case p == 49:
			s.curAttr.Bg = nil
		}
	}
}

func (s *Screen) csiIch_insertChars(n int) {
	s.cancelWrap()

	r0 := s.bounds

	n = clamp(n, 0, r0.Max.X-s.cursor.X)

	// shift right
	dst := s.cursor.Add(P{n, 0})
	s.copyRangeX(dst, s.cursor.X, r0.Max.X-n)

	// clear left
	s.clearRangeX(s.cursor, n)
}

func (s *Screen) csiDch_deleteChars(n int) {
	s.cancelWrap()

	r0 := s.bounds

	n = clamp(n, 0, r0.Max.X-s.cursor.X)

	// shift left
	dst := s.cursor
	s.copyRangeX(dst, s.cursor.X+n, r0.Max.X)

	// clear right
	dst2 := s.cursor
	dst2.X = r0.Max.X - n
	s.clearRangeX(dst2, n)
}

func (s *Screen) csiEch_eraseChars(n int) {
	s.cancelWrap()
	s.clearRangeX(s.cursor, n)
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
	s.scrollDownR(r, n)
}

// region only: delete n lines at cursor row within region
func (s *Screen) csiDl_deleteLines(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}

	r.Min.Y = s.cursor.Y
	s.scrollUpR(r, n)
}

// region only
func (s *Screen) csiSu_scrollUp(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}
	n = clamp(n, 1, r.Dy())
	s.scrollUpR(r, n)
}

// region only
func (s *Screen) csiSd_scrollDown(n int) {
	r := s.dynBounds(s.cursor)
	if !s.cursor.In(r) {
		return
	}
	n = clamp(n, 1, r.Dy())
	s.scrollDownR(r, n)
}

//----------

func (s *Screen) csiCht_cursorHorizontalTabulation(n int) {
	s.escHt_tab(n)
}
func (s *Screen) csiCha_cursorHorizontalAbsolute(col1 int) {
	s.cursor.X = col1 - 1
	clampInR(&s.cursor, s.bounds)
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
		if 0 <= x && x < s.bounds.Max.X {
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
		s.scrollUpR(r, 1)
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
		s.scrollDownR(r, 1)
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
	if 0 <= x && x < s.bounds.Max.X {
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
	s.cursor = P{}
	s.updateRegion()
	s.initTabStops()
	if hard {
		s.clearGrids()
	}
}

func (s *Screen) escAln_screenAlignment() {
	s.cancelWrap()
	for y := 0; y < s.bounds.Max.Y; y++ {
		for x := 0; x < s.bounds.Max.X; x++ {
			(*s.Grid)[y][x] = Cell{R: 'E', A: s.curAttr}
		}
	}
	s.cursor = P{}
}

//----------
//----------

// useful for debug
func (scr *Screen) Print() {
	fmt.Println(string(scr.Bprint(true, false)))
}
func (scr *Screen) PrintWithCursor() {
	fmt.Println(string(scr.Bprint(true, true)))
}
func (scr *Screen) Bprint(border, cursor bool) []byte {
	sp := NewScreenPrinter()
	sp.Border = border
	if cursor {
		sp.CursorRune = '◙'
	}
	return sp.Bprint(scr)
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

type Cell struct {
	R rune
	A Attr
}

//----------

type Attr struct {
	Fg      *AttrColor
	Bg      *AttrColor
	Bold    bool
	Inverse bool // inverse fg/bg
}

type AttrColor int

func (ac *AttrColor) Color() color.Color {
	if ac == nil {
		return nil
	}
	switch *ac {
	case 0:
		return colornames.Black
	case 1:
		return colornames.Red
	case 2:
		return colornames.Green
	case 3:
		return colornames.Yellow
	case 4:
		return colornames.Blue
	case 5:
		return colornames.Magenta
	case 6:
		return colornames.Cyan
	case 7:
		return colornames.White
	default:
		panic("!")
	}
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
	clampInR(&s.cursor, s.bounds)
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

//type ScrollBack struct {
//	toMove []byte // scrolled but not yet displayed
//	scroll []byte // read only, already displayed
//}

//----------
//----------
//----------

func cloneCells(r []Cell) []Cell {
	return slices.Clone(r)
}
func cloneGrid(g [][]Cell) [][]Cell {
	out := make([][]Cell, len(g))
	for i := range g {
		out[i] = cloneCells(g[i])
	}
	return out
}

func newGrid(size P) [][]Cell {
	out := make([][]Cell, size.Y)
	for i := range out {
		out[i] = make([]Cell, size.X)
	}
	return out
}

func resizeGrid(grid [][]Cell, size P) [][]Cell {
	if d := size.Y - len(grid); d > 0 {
		// TODO: scrollback

		//grid = append([][]Cell(nil), grid[d:]...) // keep lower lines
		grid = append(grid, make([][]Cell, d)...)
	} else {
		grid = grid[:size.Y] // keep top lines
	}
	for i := range grid {
		//row := grid[i]
		if d := size.X - len(grid[i]); d > 0 {
			grid[i] = append(grid[i], make([]Cell, d)...)
		} else {
			// TODO: check wraps
			// TODO: check unwraps
			// reduce size
			grid[i] = grid[i][:size.X]
		}
	}
	return grid
}

//----------

var decSpec = map[rune]rune{
	'j': '┘', 'k': '┐', 'l': '┌', 'm': '└',
	'n': '┼', 'q': '─', 'x': '│',
	't': '├', 'u': '┤', 'v': '┴', 'w': '┬',
	'y': '≤', 'z': '≥', '{': 'π', '|': '≠', '}': '£', '~': '·',
}

func mapDecSpecial(r rune) rune {
	if v, ok := decSpec[r]; ok {
		return v
	}
	return r
}

//----------
