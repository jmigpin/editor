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

	scrollBack []byte

	cursor   Cursor
	curAttr  Attr
	wrapNext bool // autowrap support

	pmodes   PrivModes
	graphics Graphics
	tabStops []bool // len==W; true where a tab stop exists

	gbX GridBounds
	gbY GridBounds

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
	s.Grid = &s.grid1
	return s
}

func (s *Screen) Clone() *Screen {
	s2 := *s // copy
	//s2.Grid = cloneGrid(s.Grid)
	s2.grid1 = cloneGrid(s.grid1)
	s2.grid2 = cloneGrid(s.grid2)
	s2.pmodes = *s.pmodes.clone()
	s2.graphics = *s.graphics.clone()
	return &s2
}

func (s *Screen) resize(w, h int) {
	s.W, s.H = w, h

	// make new grid
	s.newGrids()

	// TODO: loses current region settings.. review this
	s.gbX = *newGridBounds(0, s.W-1)
	s.gbY = *newGridBounds(0, s.H-1)

	s.setCursor(s.cursor, true) // clamp cursor

	s.initTabStops()
}

//----------

func (s *Screen) newGrids() {
	s.grid1 = newGrid(s.W, s.H)
	s.grid2 = newGrid(s.W, s.H)
}
func (s *Screen) newGrid() {
	*s.Grid = newGrid(s.W, s.H)
}
func (s *Screen) setGrid2(on bool) {
	if on {
		s.Grid = &s.grid2
	} else {
		s.Grid = &s.grid1
	}
}

//----------

func (s *Screen) copySubGrid(dstX, dstY, x1, y1, x2, y2 int) {
	w := [][]Cell{}
	// copy to tmp first to allow correct overwriting
	for y := y1; y < y2; y++ {
		w = append(w, cloneCells((*s.Grid)[y][x1:x2]))
	}
	// copy to the destination
	for k, u := range w {
		copy((*s.Grid)[dstY+k][dstX:], u)
	}
}
func (s *Screen) clearSubGrid(x1, y1, x2, y2 int) {
	for y := y1; y < y2; y++ {
		s.clearCells((*s.Grid)[y][x1:x2])
	}
}
func (s *Screen) clearCells(w []Cell) {
	for i := range w {
		//w[i] = Cell{}
		w[i] = Cell{A: s.curAttr}
	}
}

//----------

func (s *Screen) copyRange(y, dstX, x1, x2 int) {
	s.copySubGrid(dstX, y, x1, y, x2, y+1)
}
func (s *Screen) clearRange(y, x, n int) {
	a, b := s.gbX.AB()
	x = max(a, x)
	xn := min(b+1, x+n)
	s.clearCells((*s.Grid)[y][x:xn])
}

//----------

func (s *Screen) copyLines(dstY, y1, y2 int) {
	a, b := s.gbX.AB()
	s.copySubGrid(a, dstY, a, y1, b+1, y2)
}

func (s *Screen) clearLines(y, n int) {
	a, b := s.gbX.AB()
	s.clearSubGrid(a, y, b+1, y+n)
}

func (s *Screen) clearLine(y int) {
	s.clearLines(y, 1)
}

//----------

func (s *Screen) cancelWrap() {
	s.wrapNext = false
}

//----------
//----------

func (s *Screen) putRune(r rune) {
	if s.graphics.isSpecial() {
		r = mapDecSpecial(r)
	}

	// apply pending wrap first
	if s.wrapNext {
		s.cancelWrap()
		s.carriageReturn()
		s.lineFeed()
	}

	(*s.Grid)[s.cursor.y][s.cursor.x] = Cell{R: r, A: s.curAttr}

	if s.cursor.x == s.gbX.B() {
		if s.pmodes.autoWrap() {
			// do not move now; set wrap for the *next* printable
			s.wrapNext = true
		} // else: stay at last column, overwrite subsequent prints
	} else {
		s.setCursorX(s.cursor.x+1, true)
	}
}

//----------

func (s *Screen) carriageReturn() {
	s.cancelWrap()
	s.cursor.x = s.gbX.A()
}

func (s *Screen) lineFeed() {
	s.cancelWrap()

	if s.pmodes.LineFeedNewline() {
		s.carriageReturn()
	}

	if s.cursor.y == s.gbY.iB { // need review
		s.scrollUpRegion()
	} else {
		s.setCursorY(s.cursor.y+1, true)
	}
}

func (s *Screen) backspace() {
	s.moveRel(-1, 0)
}

//----------

func (s *Screen) moveTo(row1, col1 int) { // 1-based
	s.cancelWrap()
	s.moveToRow(row1)
	s.moveToCol(col1)
}
func (s *Screen) moveToRow(row1 int) { // 1-based
	s.cancelWrap()
	s.setCursorY(row1-1, false)
}
func (s *Screen) moveToCol(col1 int) { // 1-based
	s.cancelWrap()
	s.setCursorX(col1-1, false)
}
func (s *Screen) moveToOrigin() {
	s.moveTo(1, 1)
}

//----------

func (s *Screen) moveRel(dx, dy int) {
	s.cancelWrap()
	s.setCursorXY(s.cursor.x+dx, s.cursor.y+dy, true)
}

//----------

func (s *Screen) setCursor(c Cursor, raw bool) {
	s.setCursorX(c.x, raw)
	s.setCursorY(c.y, raw)
}
func (s *Screen) setCursorXY(x, y int, raw bool) {
	s.setCursorX(x, raw)
	s.setCursorY(y, raw)
}
func (s *Screen) setCursorY(y int, raw bool) {
	s.cursor.y = s.gbY.clamp(y, raw)
}
func (s *Screen) setCursorX(x int, raw bool) {
	s.cursor.x = s.gbX.clamp(x, raw)
}

func (s *Screen) IsCursor(x, y int) bool {
	return s.cursor.x == x && s.cursor.y == y
}

//----------

func (s *Screen) setScrollRegion(top1, bot1 int) {
	s.gbY.setInner(top1-1, bot1-1)
	//s.gbY.sub.on = true
}

// shifts up, blanks bottom
func (s *Screen) scrollUpRegion() {
	//if !s.gbY.inInnerAB(s.cursor.y) {
	//	return
	//}

	a, b := s.gbY.innerAB()
	s.copyLines(a, a+1, b+1) // move rows [top+1..bot] up by 1
	s.clearLines(b, 1)       // clear bottom row
}

// shift down, blanks top
func (s *Screen) scrollDownRegion() {
	//if !s.gbY.inInnerAB(s.cursor.y) {
	//	return
	//}

	a, b := s.gbY.innerAB()
	s.copyLines(a+1, a, b) // move rows [top..bot-1] down by 1
	s.clearLines(a, 1)     // clear top row
}

//----------

func (s *Screen) initTabStops() {
	s.tabStops = make([]bool, s.W)
	for x := 8; x < s.W; x += 8 { // every 8 cols
		s.tabStops[x] = true
	}
}

func (s *Screen) nextTabX(x int) int {
	maxX := s.gbX.B()
	for i := x + 1; i < maxX; i++ {
		if s.tabStops[i] {
			return i
		}
	}
	return maxX
}
func (s *Screen) prevTabX(x int) int {
	minX := s.gbX.A()
	for i := x - 1; i >= minX; i-- {
		if s.tabStops[i] {
			return i
		}
	}
	return minX
}

//----------
//----------

func (s *Screen) csiSlrm_setXMargins(left1, right1 int) {
	s.cancelWrap()

	//s.gbX.forceInner = true
	s.gbX.setInner(left1-1, right1-1)

	s.setCursorX(s.cursor.x, true) // clamp
}

//----------
//----------

func (s *Screen) csiCup_cursorPosition(row1, col1 int) {
	s.moveTo(row1, col1)
}

func (s *Screen) csiCuu_cursorUp(v int) {
	s.moveRel(0, -v)
}
func (s *Screen) csiCud_cursorDown(v int) {
	s.moveRel(0, v)
}
func (s *Screen) csiCuf_cursorForward(v int) {
	s.moveRel(v, 0)
}
func (s *Screen) csiCub_cursorBackward(v int) {
	s.moveRel(-v, 0)
}

func (s *Screen) csiEd_eraseInDisplay(mode int) {
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
		n := s.gbX.B() + 1 - s.cursor.x
		s.clearRange(y, s.cursor.x, n)
	case 1: // start to cursor
		x0 := s.gbX.A()
		n := s.cursor.x + 1 - x0
		s.clearRange(y, x0, n)
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
	y, x := s.cursor.y, s.cursor.x

	b := s.gbX.B()
	ins := b - x + 1
	if n > ins {
		n = ins
	}
	shift := ins - n

	s.copyRange(y, x+n, x, x+shift) // shift right

	s.clearRange(y, x, n)
}

func (s *Screen) csiDch_deleteChars(n int) {
	y, x := s.cursor.y, s.cursor.x

	b := s.gbX.B()
	rem := b - x + 1
	if n > rem {
		n = rem
	}
	shift := rem - n

	s.copyRange(y, x, x+n, x+n+shift) // shift left

	s.clearRange(y, b-n+1, n)
}

func (s *Screen) csiEch_eraseChars(n int) {
	x, y := s.cursor.x, s.cursor.y
	s.clearRange(y, x, n)
}

func (s *Screen) csiCpr_cursorPositionReport() (int, int) {
	row1 := s.gbY.report(s.cursor.y) + 1
	col1 := s.gbX.report(s.cursor.x) + 1
	return row1, col1
}

// csiIl_insertLines/DL operate only if cursor is inside scroll region.
// insert n blank lines at cursor row within [sTop..sBot].
func (s *Screen) csiIl_insertLines(n int) {
	y := s.cursor.y

	b := s.gbY.B()
	ins := b - y + 1
	if n > ins {
		n = ins
	}

	// shift down [y..sBot-n] → [y+n..sBot]
	s.copyLines(y+n, y, b-n+1)

	// clear inserted lines
	s.clearLines(y, n)
}

// delete n lines at cursor row within [sTop..sBot].
func (s *Screen) csiDl_deleteLines(n int) {
	y := s.cursor.y

	b := s.gbY.B()
	maxDel := b - y + 1
	if n > maxDel {
		n = maxDel
	}

	// shift up [y+n..sBot] → [y..sBot-n]
	s.copyLines(y, y+n, b+1)

	// clear vacated bottom lines
	s.clearLines(b-n+1, n)
}

func (s *Screen) csiSu_scrollUp(n int) {
	a, b := s.gbY.AB()
	h := b - a + 1
	if n > h {
		n = h
	}
	for i := 0; i < n; i++ {
		s.scrollUpRegion()
	}
}

func (s *Screen) csiSd_scrollDown(n int) {
	a, b := s.gbY.AB()
	h := b - a + 1
	if n > h {
		n = h
	}
	for i := 0; i < n; i++ {
		s.scrollDownRegion() // shifts down, blanks top
	}
}

func (s *Screen) csiCht_cursorHorizontalTabulation(n int) {
	s.escHt_tab(n)
}
func (s *Screen) csiCha_cursorHorizontalAbsolute(col1 int) {
	s.moveToCol(col1)
}
func (s *Screen) csiCbt_cursorBackwardTab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.x = s.prevTabX(s.cursor.x)
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
	needResize := len(s.grid1[0]) != cols
	if needResize {
		s.W = cols
		s.resize(s.W, s.H)
	}
	return needResize
}

func (s *Screen) csi_setResetMode(priv byte, a, b, c int, on bool, userCons ConsoleConn) {
	switch priv {
	case 0:
		switch a {
		case 20: // Automatic Newline (LNM)
			s.pmodes.set(a, on)
		}
	case '?':
		s.pmodes.set(a, on)
		switch a {
		case 3: // 32 Column Mode (DECCOLM)
			if needResize := s.csiColm_column132Mode(); needResize {
				userCons.SetSize(s.W, s.H)
			}
		case 6: // scroll origin mode
			s.gbY.forceInner = on
			if on {
				s.moveToOrigin()
			}
		case 69: // left/right margin mode
			s.gbX.forceInner = on
			if on {
				s.moveToOrigin()
			}
		case 47: // alternate screen buffer
			s.setGrid2(on)
		case 1047: // save cursor
			s.csiScp_saveCursorPos()
		case 1048: // save cursor, alternate screen buffer, clear
			s.csiScp_saveCursorPos()
			s.setGrid2(on)
			s.newGrid()
		}
	}
}

//----------
//----------

func (s *Screen) escInd_index() {
	s.cancelWrap()
	if s.cursor.y == s.gbY.iB {
		s.scrollUpRegion()
	} else {
		s.setCursorY(s.cursor.y+1, true)
	}
}

func (s *Screen) escRi_reverseIndex() {
	s.cancelWrap()
	if s.cursor.y == s.gbY.iA {
		s.scrollDownRegion()
	} else {
		//s.cursor.y--
		s.setCursorY(s.cursor.y-1, true)
	}
}

func (s *Screen) escHt_tab(n int) {
	s.cancelWrap()
	for ; n > 0; n-- {
		s.cursor.x = s.nextTabX(s.cursor.x)
	}
}

func (s *Screen) escHts_horizontalTabSet() {
	x := s.cursor.x
	if 0 <= x && x < s.W {
		s.tabStops[x] = true
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

func (s *Screen) escSc_saveCursor() {
	s.escSaveCursorAndAttr.save(s)
	s.escSaveCursorAndAttr.attr = s.curAttr
}
func (s *Screen) escRc_restoreCursor() {
	s.escSaveCursorAndAttr.restore(s)
	s.curAttr = s.escSaveCursorAndAttr.attr
}

func (s *Screen) escRis_reset(hard bool) {
	s.wrapNext = false
	s.cursor = Cursor{}
	s.gbX.resetInner()
	s.gbY.resetInner()
	s.pmodes = *newPrivModes()
	s.graphics = *newGraphics()

	s.initTabStops()

	if hard {
		s.newGrids()
	}
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

type Cursor struct {
	x, y int // 0-based
}

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
	wn bool
}

func (c *SaveCursor) save(s *Screen) {
	c.ok = true
	c.c = s.cursor
	c.wn = s.wrapNext
}
func (c *SaveCursor) restore(s *Screen) {
	if !c.ok {
		return
	}
	s.setCursor(c.c, true)
	s.wrapNext = c.wn
}

//----------
//----------
//----------

// PrivModes keeps DEC private modes (?n).
type PrivModes struct {
	m map[int]bool
}

func newPrivModes() *PrivModes {
	m := &PrivModes{m: make(map[int]bool)}
	// defaults
	m.set(7, true)  // autowrap
	m.set(20, true) // line feed newline mode
	m.set(25, true) // cursor
	return m
}

func (m *PrivModes) set(n int, on bool) { m.m[n] = on }
func (m *PrivModes) Is(n int) bool      { return m.m[n] }

func (m *PrivModes) clone() *PrivModes {
	m2 := *m // copy
	m2.m = maps.Clone(m.m)
	return &m2
}

//----------

func (m *PrivModes) AppCursorKeys() bool   { return m.Is(1) }
func (m *PrivModes) column132() bool       { return m.Is(3) }
func (m *PrivModes) reverseVideo() bool    { return m.Is(5) }
func (m *PrivModes) origin() bool          { return m.Is(6) }
func (m *PrivModes) autoWrap() bool        { return m.Is(7) }
func (m *PrivModes) autoRepeat() bool      { return m.Is(8) }
func (m *PrivModes) LineFeedNewline() bool { return m.Is(20) }
func (m *PrivModes) cursor() bool          { return m.Is(25) }
func (m *PrivModes) leftRightMargin() bool { return m.Is(69) }
func (m *PrivModes) BracketedPaste() bool  { return m.Is(2004) }

//----------
//----------
//----------

type Graphics struct {
	sel  string
	bank map[string]string
}

func newGraphics() *Graphics {
	gr := &Graphics{}
	gr.sel = "g0"
	gr.bank = map[string]string{}
	return gr
}

func (gr *Graphics) set(kind, typ string) {
	if typ != "" {
		gr.bank[kind] = typ // designate
	} else {
		gr.sel = kind // select
	}
}
func (gr *Graphics) clone() *Graphics {
	gr2 := *gr // copy
	gr2.bank = maps.Clone(gr.bank)
	return &gr2
}
func (gr *Graphics) isSpecial() bool {
	if u, ok := gr.bank[gr.sel]; ok {
		return u == "special"
	}
	return false
}

//----------
//----------
//----------

type GridBounds struct {
	oA, oB int // outer, inclusive
	iA, iB int // inner, inclusive

	// case 1: scroll // TODO

	forceInner bool // changes address
}

func newGridBounds(a, b int) *GridBounds {
	gb := &GridBounds{oA: a, oB: b}
	gb.resetInner()
	return gb
}

func (gb *GridBounds) resetInner() {
	gb.setInner(gb.oA, gb.oB)
}
func (gb *GridBounds) setInner(a, b int) {
	gb.iA = clamp(a, gb.oA, gb.oB)
	gb.iB = clamp(b, gb.oA, gb.oB)

	// invalid values
	if gb.iA > gb.iB {
		gb.iB = gb.iA
	}
}

func (gb *GridBounds) clamp(v int, raw bool) int { // output raw
	if gb.forceInner {
		if !raw {
			v += gb.iA
		}
		return clamp(v, gb.iA, gb.iB)
	}
	return clamp(v, gb.oA, gb.oB)
}

//----------

func (gb *GridBounds) report(v int) int { // input raw, output depends
	if gb.forceInner {
		return v - gb.iA
	}
	return v
}

//----------

func (gb *GridBounds) AB() (int, int) {
	return gb.A(), gb.B()
}
func (gb *GridBounds) A() int {
	if gb.forceInner {
		return gb.iA
	}
	return gb.oA
}
func (gb *GridBounds) B() int {
	if gb.forceInner {
		return gb.iB
	}
	return gb.oB
}

//----------

func (gb *GridBounds) innerAB() (int, int) {
	return gb.iA, gb.iB
}

func (gb *GridBounds) inInnerAB(v int) bool {
	a, b := gb.innerAB()
	return v >= a && v <= b
}

//----------
//----------
//----------

type Buffer struct {
	grid       [][]Cell
	scrollBack []Cell
}

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
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
