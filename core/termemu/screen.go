package termemu

import (
	"bytes"
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strings"

	"golang.org/x/image/colornames"
)

type Screen struct {
	W, H int

	Grid  *[][]Cell // current grid ptr
	grid1 [][]Cell
	grid2 [][]Cell // alternate screen buffer

	bounds struct {
		top, bot, left, right int // inclusive
	}

	//scrollBack []byte // TODO: support for both grids?

	cursor   Cursor
	curAttr  Attr
	wrapNext bool // autowrap support

	pmodes   PrivModes
	graphics Graphics
	tabStops []bool // len==W; true where a tab stop exists

	csiSaveCursor        SaveCursor
	escSaveCursorAndAttr struct {
		SaveCursor
		attr Attr
	}
}

func NewScreen(w, h int) *Screen {
	s := &Screen{}
	s.resize(w, h)
	s.pmodes = *newPrivModes()
	s.graphics = *newGraphics()
	return s
}

func (s *Screen) resize(w, h int) {
	s.W, s.H = w, h
	s.resetBounds()
	s.newGrids()
	s.boundsClamp(&s.cursor)
	s.initTabStops()
}

func (s *Screen) Clone() *Screen {
	s2 := *s // copy
	s2.grid1 = cloneGrid(s.grid1)
	s2.grid2 = cloneGrid(s.grid2)
	s2.pmodes = *s.pmodes.clone()
	s2.graphics = *s.graphics.clone()
	return &s2
}

//----------

func (s *Screen) newGrid() {
	*s.Grid = newGrid(s.W, s.H)
}

func (s *Screen) newGrids() {
	on2 := s.Grid == &s.grid2

	s.grid1 = newGrid(s.W, s.H)
	s.grid2 = newGrid(s.W, s.H)

	s.setGrid2(on2)
}

func (s *Screen) setGrid2(on bool) {
	if on {
		s.Grid = &s.grid2
	} else {
		s.Grid = &s.grid1
	}
}

//----------

func (s *Screen) resetBounds() {
	s.bounds.top, s.bounds.bot = 0, s.H-1
}

func (s *Screen) boundsClamp(c *Cursor) {
	c.x = clamp(c.x, 0, s.W-1)
	c.y = clamp(c.y, 0, s.H-1)
}

func (s *Screen) boundsClampScrollingRegion(c *Cursor) {
	if s.pmodes.leftRightMargin() {
		c.x = clamp(c.x, s.bounds.left, s.bounds.right)
	}
	c.y = clamp(c.y, s.bounds.top, s.bounds.bot)
}

func (s *Screen) boundsScrollMin(c Cursor) XY {
	a, _ := s.boundsScrollEdges(c)
	return a
}
func (s *Screen) boundsScrollMax(c Cursor) XY {
	_, b := s.boundsScrollEdges(c)
	return b
}

func (s *Screen) boundsScrollEdges(c Cursor) (XY, XY) {
	a := XY{0, 0}
	b := XY{s.W - 1, s.H - 1} // inclusive
	if s.boundsInTopBottom(c) {
		a.y = s.bounds.top
		b.y = s.bounds.bot
	}
	if s.pmodes.leftRightMargin() && s.boundsInLeftRight(c) {
		a.x = s.bounds.left
		b.x = s.bounds.right
	}
	return a, b
}
func (s *Screen) boundsInScrollRegion(c Cursor) bool {
	return s.boundsInTopBottom(c) &&
		(!s.pmodes.leftRightMargin() || s.boundsInLeftRight(c))
}
func (s *Screen) boundsInLeftRight(c Cursor) bool {
	return inside(c.x, s.bounds.left, s.bounds.right)
}
func (s *Screen) boundsInTopBottom(c Cursor) bool {
	return inside(c.y, s.bounds.top, s.bounds.bot)
}

//----------

func (s *Screen) copySubGrid(dst XY, a, b XY) {
	w := [][]Cell{}
	// copy to tmp first to allow correct overwriting
	for y := a.y; y < b.y; y++ {
		w = append(w, cloneCells((*s.Grid)[y][a.x:b.x]))
	}
	// copy to the destination
	for k, u := range w {
		copy((*s.Grid)[dst.y+k][dst.x:], u)
	}
}

func (s *Screen) clearSubGrid(a, b XY) {
	for y := a.y; y < b.y; y++ {
		s.clearCells((*s.Grid)[y][a.x:b.x])
	}
}
func (s *Screen) clearCells(w []Cell) {
	for i := range w {
		//w[i] = Cell{}
		w[i] = Cell{A: s.curAttr}
	}
}

//----------

func (s *Screen) copyRange(dst XY, x1, x2 int) {
	s.copySubGrid(dst, XY{x1, dst.y}, XY{x2, dst.y + 1})
}
func (s *Screen) clearRange(dst XY, n int) {
	s.clearCells((*s.Grid)[dst.y][dst.x : dst.x+n])
}

//----------

func (s *Screen) clearLine(y int) {
	s.clearLines(y, 1)
}
func (s *Screen) clearLines(y, n int) {
	a, b := 0, s.W-1
	s.clearSubGrid(XY{a, y}, XY{b + 1, y + n})
}

//----------

func (s *Screen) cancelWrap() {
	s.wrapNext = false
}

//----------

func (s *Screen) IsCursor(x, y int) bool {
	return s.cursor.x == x && s.cursor.y == y
}

//----------
//----------

func (s *Screen) putRune(r rune) {
	if s.graphics.isSpecial() {
		r = mapDecSpecial(r)
	}

	if s.pmodes.insert() {
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

	(*s.Grid)[s.cursor.y][s.cursor.x] = Cell{R: r, A: s.curAttr}

	if !s.pmodes.insert() {
		if s.cursor.x == s.boundsScrollMax(s.cursor).x {
			if s.pmodes.autoWrap() {
				// do not move now; set wrap for the *next* printable
				s.wrapNext = true
			} // else: stay at last column, overwrite subsequent prints
		} else {
			s.cursor.x++
		}
	}
}

//----------

func (s *Screen) carriageReturn() {
	s.cancelWrap()
	s.cursor.x = s.boundsScrollMin(s.cursor).x
}

func (s *Screen) lineFeed() {
	s.cancelWrap()

	if s.pmodes.LineFeedNewline() {
		s.carriageReturn()
	}

	if s.cursor.y == s.boundsScrollMax(s.cursor).y {
		s.scrollUpRegion(1)
	} else {
		s.cursor.y++
	}
}

func (s *Screen) backspace() {
	s.cancelWrap()
	s.cursor.x--
	s.boundsClamp(&s.cursor)
}

//----------

func (s *Screen) csiVpa_moveToRow(row1 int) { // 1-based
	s.cancelWrap()
	s.cursor.y = row1 - 1
	s.boundsClamp(&s.cursor)
}

//----------

func (s *Screen) setScrollRegion(top1, bot1 int) {
	s.bounds.top = top1 - 1
	s.bounds.bot = bot1 - 1
	// set cursor to home
	s.cursor = XY{0, 0}
	if s.pmodes.origin() {
		s.cursor.y = s.bounds.top
	}
	if s.pmodes.leftRightMargin() {
		s.cursor.x = s.bounds.left
	}
}

// shifts up, blanks bottom
func (s *Screen) scrollUpRegion(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	a, b := s.boundsScrollEdges(s.cursor)

	// move rows [top+1..bot] up by 1
	s.copySubGrid(a, XY{a.x, a.y + 1}, XY{b.x + 1, b.y + 1})

	// clear bottom row
	s.clearSubGrid(XY{a.x, b.y}, XY{b.x + 1, b.y + 1})
}

// shift down, blanks top
func (s *Screen) scrollDownRegion(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	a, b := s.boundsScrollEdges(s.cursor)

	// move rows [top..bot-1] down by 1
	s.copySubGrid(XY{a.x, a.y + 1}, XY{a.x, a.y}, XY{b.x + 1, b.y})

	// clear top row
	s.clearSubGrid(XY{a.x, a.y}, XY{b.x + 1, a.y + 1})
}

//----------

func (s *Screen) initTabStops() {
	s.tabStops = make([]bool, s.W)
	for x := 8; x < s.W; x += 8 { // every 8 cols
		s.tabStops[x] = true
	}
}

func (s *Screen) nextTabX(c Cursor) int {
	maxX := s.boundsScrollMax(c).x
	for i := c.x + 1; i < maxX; i++ {
		if s.tabStops[i] {
			return i
		}
	}
	return maxX
}
func (s *Screen) prevTabX(c Cursor) int {
	minX := s.boundsScrollMin(c).x
	for i := c.x - 1; i >= minX; i-- {
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
	s.bounds.left = clamp(left1-1, 0, s.W-1)
	s.bounds.right = clamp(right1-1, 0, s.W-1)

}

//----------
//----------

func (s *Screen) csiCup_cursorPosition(row1, col1 int) {
	s.cancelWrap()
	row, col := row1-1, col1-1
	c := Cursor{x: col, y: row}

	ok := false
	if s.pmodes.leftRightMargin() {
		ok = true
		c.x += s.bounds.left
	}
	if s.pmodes.origin() {
		ok = true
		c.y += s.bounds.top
	}
	if ok {
		s.boundsClampScrollingRegion(&c)
	} else {
		s.boundsClamp(&c)
	}

	s.cursor = c
}

func (s *Screen) csiCuu_cursorUp(v int) {
	s.cancelWrap()
	s.cursor.y -= v
	s.boundsClamp(&s.cursor)
}
func (s *Screen) csiCud_cursorDown(v int) {
	s.cancelWrap()
	s.cursor.y += v
	s.boundsClamp(&s.cursor)
}
func (s *Screen) csiCuf_cursorForward(v int) {
	s.cancelWrap()
	s.cursor.x += v
	s.boundsClamp(&s.cursor)
}
func (s *Screen) csiCub_cursorBackward(v int) {
	s.cancelWrap()
	s.cursor.x -= v
	s.boundsClamp(&s.cursor)
}

func (s *Screen) csiEd_eraseInDisplay(mode int) {
	s.cancelWrap()
	switch mode {
	case 0: // cursor to end
		s.csiEl_eraseInLine(0)
		for y := s.cursor.y + 1; y < s.H; y++ {
			s.clearLine(y)
		}
	case 1: // home to cursor
		for y := 0; y < s.cursor.y; y++ {
			s.clearLine(y)
		}
		s.csiEl_eraseInLine(1)
	default: // 2 or others: entire screen
		for y := 0; y < s.H; y++ {
			s.clearLine(y)
		}
	}
}

func (s *Screen) csiEl_eraseInLine(mode int) {
	y := s.cursor.y
	switch mode {
	case 0: // cursor to end
		n := s.W - s.cursor.x
		s.clearRange(s.cursor, n)
	case 1: // start to cursor
		n := s.cursor.x + 1
		s.clearRange(XY{0, s.cursor.y}, n)
	default: // 2: whole line
		s.clearLine(y)
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
			s.curAttr.Reverse = true
		case p == 27:
			s.curAttr.NoReverse = false
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

	y, x := s.cursor.y, s.cursor.x

	maxX := s.W - 1
	ins := maxX - x + 1
	if n > ins {
		n = ins
	}
	shift := ins - n

	s.copyRange(XY{x + n, y}, x, x+shift) // shift right

	s.clearRange(XY{x, y}, n) // clear left
}

func (s *Screen) csiDch_deleteChars(n int) {
	s.cancelWrap()

	y, x := s.cursor.y, s.cursor.x

	maxX := s.W - 1
	rem := maxX - x + 1
	if n > rem {
		n = rem
	}
	shift := rem - n

	s.copyRange(XY{x, y}, x+n, x+n+shift) // shift left

	s.clearRange(XY{maxX - n + 1, y}, n) // clear right
}

func (s *Screen) csiEch_eraseChars(n int) {
	s.cancelWrap()
	s.clearRange(s.cursor, n)
}

func (s *Screen) csiCpr_cursorPositionReport() (int, int) {
	row1 := s.cursor.y + 1
	col1 := s.cursor.x + 1
	return row1, col1
}

// insert n blank lines at cursor row within [sTop..sBot].
func (s *Screen) csiIl_insertLines(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	y := s.cursor.y
	a, b := s.boundsScrollEdges(s.cursor)
	maxY := b.y

	ins := maxY - y + 1
	if n > ins {
		n = ins
	}

	// shift down [y..sBot-n] → [y+n..sBot]
	s.copySubGrid(XY{a.x, y + n}, XY{a.x, y}, XY{b.x + 1, maxY - n + 1})

	// clear inserted top lines
	s.clearLines(y, n)
}

// delete n lines at cursor row within [sTop..sBot].
func (s *Screen) csiDl_deleteLines(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	y := s.cursor.y
	a, b := s.boundsScrollEdges(s.cursor)
	maxY := b.y

	del := maxY - y + 1
	if n > del {
		n = del
	}

	// shift up [y+n..sBot] → [y..sBot-n]
	s.copySubGrid(XY{a.x, y}, XY{a.x, y + n}, XY{b.x + 1, maxY + 1})

	// clear vacated bottom lines
	s.clearLines(maxY-n+1, n)
}

func (s *Screen) csiSu_scrollUp(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	a, b := s.boundsScrollEdges(s.cursor)
	h := b.y - a.y + 1
	if n > h {
		n = h
	}

	s.scrollUpRegion(n)
}

func (s *Screen) csiSd_scrollDown(n int) {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	a, b := s.boundsScrollEdges(s.cursor)
	h := b.y - a.y + 1
	if n > h {
		n = h
	}

	s.scrollDownRegion(n)
}

func (s *Screen) csiCht_cursorHorizontalTabulation(n int) {
	s.escHt_tab(n)
}
func (s *Screen) csiCha_cursorHorizontalAbsolute(col1 int) {
	s.cursor.x = col1 - 1
	s.boundsClamp(&s.cursor)
}
func (s *Screen) csiCbt_cursorBackwardTab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.x = s.prevTabX(s.cursor)
	}
}

func (s *Screen) csiTbc_tabClear(ps int) {
	switch ps {
	case 0: // at cursor
		x := s.cursor.x
		if 0 <= x && x < s.W {
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

func (s *Screen) csiColm_column132Mode() bool {
	cols := 80
	if s.pmodes.column132() {
		cols = 132
	}
	if s.W != cols {
		s.resize(cols, s.H)
		return true
	}
	return false
}

//----------
//----------

func (s *Screen) escInd_index() {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	s.cancelWrap()

	if s.cursor.y == s.boundsScrollMax(s.cursor).y {
		s.scrollUpRegion(1)
	} else {
		s.cursor.y++
		s.boundsClamp(&s.cursor)
	}
}

func (s *Screen) escRi_reverseIndex() {
	if !s.boundsInScrollRegion(s.cursor) {
		return
	}

	s.cancelWrap()

	if s.cursor.y == s.boundsScrollMin(s.cursor).y {
		s.scrollDownRegion(1)
	} else {
		s.cursor.y--
		s.boundsClamp(&s.cursor)
	}
}

func (s *Screen) escHt_tab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.x = s.nextTabX(s.cursor)
	}
}

func (s *Screen) escHts_horizontalTabSet() {
	x := s.cursor.x
	if 0 <= x && x < s.W {
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
	s.wrapNext = false
	s.cursor = Cursor{}
	s.resetBounds()
	s.pmodes = *newPrivModes()
	s.graphics = *newGraphics()
	s.initTabStops()
	if hard {
		s.newGrids()
	}
}

func (s *Screen) escAln_screenAlignment() {
	s.cancelWrap()
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			(*s.Grid)[y][x] = Cell{R: 'E', A: s.curAttr}
		}
	}
	s.cursor = Cursor{}
}

//----------
//----------

func (scr *Screen) Print() {
	fmt.Println(string(scr.Bytes(true, false)))
}
func (scr *Screen) PrintWithCursor() {
	fmt.Println(string(scr.Bytes(true, true)))
}

func (scr *Screen) Bytes(border, cursor bool) []byte {
	buf := &bytes.Buffer{}
	pr := func(s string) { buf.WriteString(s) }
	br := func(s string) {
		if border {
			pr(s)
		}
	}

	width := len((*scr.Grid)[0])
	br("┌")
	br(strings.Repeat("─", width))
	br("┐\n")

	for y, line := range *scr.Grid {
		br("│")
		for x, cell := range line {
			if cursor && scr.IsCursor(x, y) {
				pr("◙")
				continue
			}
			if cell.R == 0 {
				pr(" ")
				continue
			}
			pr(string(cell.R))
		}
		br("│")
		pr("\n")
	}

	br("└")
	br(strings.Repeat("─", width))
	//br("┘\n")
	br("┘")

	return buf.Bytes()
}

//----------
//----------
//----------

type XY struct {
	x, y int
}

//func (u XY) addX(v int) XY { u2 := u; u2.x += v; return u2 }
//func (u XY) addY(v int) XY { u2 := u; u2.y += v; return u2 }

//----------

type Cursor = XY // 0-based

//----------

type Cell struct {
	R rune
	A Attr
}

//----------

type Attr struct {
	Fg                 *AttrColor
	Bg                 *AttrColor
	Bold               bool
	Reverse, NoReverse bool // reverse video
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
	c  Cursor
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
	s.boundsClamp(&s.cursor)
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

func (m *PrivModes) AppCursorKeys() bool   { return m.isOn("?1") }
func (m *PrivModes) AnsiNotVT52() bool     { return m.isOn("?2") }
func (m *PrivModes) column132() bool       { return m.isOn("?3") }
func (m *PrivModes) softScroll() bool      { return m.isOn("?4") }
func (m *PrivModes) reverseVideo() bool    { return m.isOn("?5") }
func (m *PrivModes) origin() bool          { return m.isOn("?6") }
func (m *PrivModes) autoWrap() bool        { return m.isOn("?7") }
func (m *PrivModes) autoRepeat() bool      { return m.isOn("?8") }
func (m *PrivModes) showCursor() bool      { return m.isOn("?25") }
func (m *PrivModes) leftRightMargin() bool { return m.isOn("?69") }
func (m *PrivModes) BracketedPaste() bool  { return m.isOn("?2004") }

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

//type Buffer struct {
//	grid       [][]Cell
//	scrollBack []Cell
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
func newGrid(w, h int) [][]Cell {
	out := make([][]Cell, h)
	for i := range out {
		out[i] = make([]Cell, w)
	}
	return out
}

//----------

func clamp(v, lo, hi int) int {
	return min(hi, max(lo, v))
}
func inside(v, lo, hi int) bool {
	return v >= lo && v <= hi
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
