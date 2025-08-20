package termemu

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

type Screen struct {
	W, H int
	Grid [][]Cell

	cursor     Cursor
	curAttr    Attr
	modes      Modes
	wrapNext   bool   // autowrap support
	sTop, sBot int    // scrolling region, 0-based inclusive
	tabStops   []bool // len==W; true where a tab stop exists

	lrmm struct { // left right margin mode
		left, right int
	}

	csiSaveCursor        SaveCursor
	escSaveCursorAndAttr struct {
		SaveCursor
		attr Attr
	}
}

func NewScreen(w, h int) *Screen {
	s := &Screen{}
	s.resize(w, h)
	s.modes = *NewModes()
	return s
}

func (s *Screen) resize(w, h int) {
	s.W, s.H = w, h

	//// adjust size by keeping existing data
	//if len(s.Grid) != s.H {
	//	u := make([][]Cell, s.H)
	//	copy(u, s.Grid)
	//	s.Grid = u
	//}
	//if len(s.Grid[0]) != s.W {
	//	for y := range s.Grid {
	//		u := make([]Cell, s.W)
	//		copy(u, s.Grid[y])
	//		s.Grid[y] = u
	//	}
	//}

	// make new grid
	s.Grid = make([][]Cell, s.H)
	for y := range s.Grid {
		s.Grid[y] = make([]Cell, s.W)
	}

	s.sTop, s.sBot = 0, s.H-1
	s.setCursorYX(s.cursor.y, s.cursor.x) // clamp cursor
	s.initTabStops()
}

func (s *Screen) Clone() *Screen {
	s2 := *s // copy
	s2.Grid = cloneGrid(s2.Grid)
	return &s2
}

//----------

func (s *Screen) putRune(r rune) {
	// apply pending wrap first
	if s.wrapNext {
		s.cancelWrap()
		s.carriageReturn()
		s.lineFeed()
	}

	s.Grid[s.cursor.y][s.cursor.x] = Cell{R: r, A: s.curAttr}

	if s.cursor.x >= s.rightEdge() {
		if s.modes.autoWrap() {
			// do not move now; set wrap for the *next* printable
			s.wrapNext = true
		} // else: stay at last column, overwrite subsequent prints
	} else {
		s.cursor.x++
	}
}

//----------

func (s *Screen) carriageReturn() {
	s.cancelWrap()
	s.cursor.x = s.leftEdge()
}

// move down one line, scrolling **inside the region**
func (s *Screen) lineFeed() {
	s.cancelWrap()

	if s.modes.LineFeedNewlineMode() {
		s.carriageReturn()
	}

	if s.cursor.y < s.sBot {
		s.cursor.y++
		return
	}
	// at bottom margin: scroll region up
	s.scrollUpRegion()
	// cursor stays on bottom margin
	s.cursor.y = s.sBot
}

func (s *Screen) backspace() {
	s.moveRel(0, -1)
}

//----------

func (s *Screen) moveTo(row1, col1 int) { // 1-based
	s.cancelWrap()
	s.moveToRow(row1)
	s.moveToCol(col1)
}
func (s *Screen) moveToRow(row1 int) { // 1-based
	s.cancelWrap()
	s.setCursorY(row1 - 1)
}
func (s *Screen) moveToCol(col1 int) { // 1-based
	s.cancelWrap()
	s.setCursorX(col1 - 1)
}

func (s *Screen) moveRel(dy, dx int) {
	s.cancelWrap()
	s.setCursorYX(s.cursor.y+dy, s.cursor.x+dx)
}

func (s *Screen) moveToOrigin() {
	if s.modes.origin() { // TODO: only if currently inside the region?
		s.moveTo(s.sTop+1, 1)
	} else {
		s.moveTo(1, 1)
	}
}

//----------

func (s *Screen) cancelWrap() {
	s.wrapNext = false
}

func (s *Screen) clearLine(y int) {
	s.clearCells(s.Grid[y])
}

func (s *Screen) clearCells(w []Cell) {
	for i := range w {
		// blank
		//w[i].R = 0
		//w[i].R = ' '
		w[i] = Cell{}
		//w[i] = Cell{Attr:s.curAttr}
	}
}

//----------

func (s *Screen) setCursorYX(y, x int) {
	s.setCursorY(y)
	s.setCursorX(x)
}
func (s *Screen) setCursorY(y int) {
	s.cursor.y = clamp(y, 0, s.H-1)
}
func (s *Screen) setCursorX(x int) {
	if s.modes.leftRightMarginMode() {
		//s.cursor.x = clamp(x, s.lrmm.left, s.lrmm.right)
		s.cursor.x = clamp(s.lrmm.left+x, s.lrmm.left, s.lrmm.right)
	} else {
		s.cursor.x = clamp(x, 0, s.W-1)
	}
}

//----------

// It clamps to the screen and ensures top<=bot. Caller may want to move the cursor to (top0,0) if emulating DECSTBM semantics.
func (s *Screen) setScrollRegion(top1, bot1 int) {
	top := clamp(top1-1, 0, s.H-1)
	bot := clamp(bot1-1, 0, s.H-1)
	if top > bot {
		top, bot = 0, s.H-1
	}
	s.sTop, s.sBot = top, bot
}

func (s *Screen) resetScrollRegion() {
	s.sTop, s.sBot = 0, s.H-1
}

// shifts up, blanks bottom
func (s *Screen) scrollUpRegion() {
	top, bot := s.sTop, s.sBot

	// move rows [top+1..bot] up by 1
	copy(s.Grid[top:bot], cloneGrid(s.Grid[top+1:bot+1]))
	// clear bottom row
	s.clearCells(s.Grid[bot])
}

func (s *Screen) scrollDownRegion(top, bot int) {
	// move rows [top..bot-1] down by 1
	copy(s.Grid[top+1:bot+1], cloneGrid(s.Grid[top:bot]))
	// clear top row
	s.clearCells(s.Grid[top])
}

//----------

func (s *Screen) leftEdge() int {
	if s.modes.leftRightMarginMode() {
		return s.lrmm.left
	}
	return 0
}
func (s *Screen) rightEdge() int {
	if s.modes.leftRightMarginMode() {
		return s.lrmm.right
	}
	return s.W - 1
}

//----------

func (s *Screen) initTabStops() {
	s.tabStops = make([]bool, s.W)
	for x := 8; x < s.W; x += 8 { // every 8 cols
		s.tabStops[x] = true
	}
}

func (s *Screen) nextTabX(x int) int {
	maxX := s.rightEdge()
	for i := x + 1; i < maxX; i++ {
		if s.tabStops[i] {
			return i
		}
	}
	return maxX
}
func (s *Screen) prevTabX(x int) int {
	minX := s.leftEdge()
	for i := x - 1; i >= minX; i-- {
		if s.tabStops[i] {
			return i
		}
	}
	return minX
}

//----------
//----------

func (s *Screen) csiSlrm_lrmmSetMargins(left1, right1 int) {
	s.cancelWrap()

	// l1/r1 are 1-based per DECSLRM, inclusive
	l := clamp(left1-1, 0, s.W-1)
	r := clamp(right1-1, 0, s.W-1)
	if r < l { // at least 1 column
		r = l
	}
	s.lrmm.left, s.lrmm.right = l, r

	s.setCursorX(s.cursor.x - s.lrmm.left) // clamp
	//s.setCursorX(s.cursor.x) // clamp
}

//----------
//----------

func (s *Screen) csiCup_cursorPosition(row1, col1 int) {
	s.moveTo(row1, col1)
}

func (s *Screen) csiCuu_cursorUp(v int) {
	s.moveRel(-v, 0)
}
func (s *Screen) csiCud_cursorDown(v int) {
	s.moveRel(v, 0)
}
func (s *Screen) csiCuf_cursorForward(v int) {
	s.moveRel(0, v)
}
func (s *Screen) csiCub_cursorBackward(v int) {
	s.moveRel(0, -v)
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
		s.clearCells(s.Grid[y][s.cursor.x:s.W])
	case 1: // start to cursor
		s.clearCells(s.Grid[y][0 : s.cursor.x+1])
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
		case 30 <= p && p <= 37:
			s.curAttr.FG = p - 30
		case p == 39:
			s.curAttr.FG = 0
		case 40 <= p && p <= 47:
			s.curAttr.BG = p - 40
		case p == 49:
			s.curAttr.BG = 0
		}
	}
}

func (s *Screen) csiDch_deleteChars(n int) {
	y, x := s.cursor.y, s.cursor.x

	row := s.Grid[y]

	rem := s.W - x
	if rem <= 0 {
		return
	}

	if n > rem {
		n = rem
	}
	shift := rem - n
	if shift > 0 {
		copy(row[x:x+shift], row[x+n:x+n+shift]) // shift left
	}

	s.clearCells(row[s.W-n : s.W])
}

func (s *Screen) csiEch_eraseChars(n int) {
	x, y := s.cursor.x, s.cursor.y

	row := s.Grid[y]

	end := x + n
	if end > s.W {
		end = s.W
	}
	s.clearCells(row[x:end])
}

func (s *Screen) csiCpr_cursorPositionReport() (int, int) {
	y, x := s.cursor.y, s.cursor.x
	top := s.sTop // 0-based region top

	row1 := y + 1
	if s.modes.origin() {
		row1 = (y - top) + 1
		if row1 < 1 {
			row1 = 1
		}
	}
	col1 := x + 1
	return row1, col1
}

// csiIl_insertLines/DL operate only if cursor is inside scroll region.
// insert n blank lines at cursor row within [sTop..sBot].
func (s *Screen) csiIl_insertLines(n int) {
	y := s.cursor.y

	if y < s.sTop || y > s.sBot {
		return
	}
	maxIns := s.sBot - y + 1
	if n > maxIns {
		n = maxIns
	}

	// shift down [y..sBot-n] → [y+n..sBot]
	if dst := y + n; dst <= s.sBot {
		copy(s.Grid[dst:s.sBot+1], cloneGrid(s.Grid[y:s.sBot-n+1]))
	}

	// clear inserted lines with spaces using current attr
	for r := y; r < y+n; r++ {
		s.clearCells(s.Grid[r])
	}
}

// delete n lines at cursor row within [sTop..sBot].
func (s *Screen) csiDl_deleteLines(n int) {
	y := s.cursor.y

	if y < s.sTop || y > s.sBot {
		return
	}
	maxDel := s.sBot - y + 1
	if n > maxDel {
		n = maxDel
	}

	// shift up [y+n..sBot] → [y..sBot-n]
	if src := y + n; src <= s.sBot {
		copy(s.Grid[y:s.sBot-n+1], cloneGrid(s.Grid[src:s.sBot+1]))
	}

	// clear vacated bottom lines with spaces using current attr
	for r := s.sBot - n + 1; r <= s.sBot; r++ {
		s.clearCells(s.Grid[r])
	}
}

func (s *Screen) csiSu_scrollUp(n int) {
	if n <= 0 {
		return
	}
	h := s.sBot - s.sTop + 1
	if n > h {
		n = h
	}
	for i := 0; i < n; i++ {
		s.scrollUpRegion()
	}
}

func (s *Screen) csiSd_scrollDown(n int) {
	if n <= 0 {
		return
	}
	h := s.sBot - s.sTop + 1
	if n > h {
		n = h
	}
	for i := 0; i < n; i++ {
		s.scrollDownRegion(s.sTop, s.sBot) // shifts down, blanks top
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

//----------
//----------

// Reverse Index (move up), scrolling **inside the region**.
func (s *Screen) escRi_reverseIndex() {
	s.cancelWrap()

	if s.cursor.y > s.sTop {
		s.cursor.y--
		return
	}
	// at top margin: scroll region down
	if s.sTop < s.sBot {
		s.scrollDownRegion(s.sTop, s.sBot)
	}
	// cursor stays on top margin
	s.cursor.y = s.sTop
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
			s.Grid[y][x] = Cell{R: 'E', A: s.curAttr}
		}
	}
	s.cursor = Cursor{y: 0, x: 0}
}

func (s *Screen) escSc_saveCursor() {
	s.escSaveCursorAndAttr.save(s)
	s.escSaveCursorAndAttr.attr = s.curAttr
}
func (s *Screen) escRc_restoreCursor() {
	s.escSaveCursorAndAttr.restore(s)
	s.curAttr = s.escSaveCursorAndAttr.attr
}

func (s *Screen) escInd_index() {
	s.cancelWrap()
	if s.cursor.y == s.sBot {
		s.scrollUpRegion()
	} else {
		s.cursor.y++
	}
}

//----------

func (s *Screen) csiColm_column132Mode() bool {
	cols := 80
	if s.modes.column132Mode() {
		cols = 132
	}
	needResize := len(s.Grid[0]) != cols
	if needResize {
		s.W = cols
		s.resize(s.W, s.H)
	}
	return needResize
}

//----------
//----------

func (scr *Screen) Print() {
	fmt.Println(string(scr.Bytes(true, false)))
}
func (scr *Screen) PrintWithCursor() {
	fmt.Println(string(scr.Bytes(true, true)))
}

func (scr *Screen) Bytes(leftTopLines, cursor bool) []byte {
	buf := &bytes.Buffer{}

	width := len(scr.Grid[0])
	if leftTopLines {
		buf.WriteString("┌")
		buf.WriteString(strings.Repeat("─", width))
		buf.WriteString("┐\n")
	}

	for y, line := range scr.Grid {
		if leftTopLines {
			buf.WriteString("│")
		}
		for x, cell := range line {
			if cursor {
				if scr.cursor.x == x && scr.cursor.y == y {
					buf.WriteString("◙")
					continue
				}
			}

			if cell.R == 0 {
				buf.WriteString(" ")
				continue
			}
			buf.WriteString(string(cell.R))
		}
		buf.WriteString("│\n")
	}

	if leftTopLines {
		buf.WriteString("└")
	}
	buf.WriteString(strings.Repeat("─", width))
	buf.WriteString("┘\n")

	return buf.Bytes()
}

//----------
//----------
//----------

type Cell struct {
	R rune
	A Attr
}

type Attr struct {
	Bold bool
	FG   int
	BG   int
}

type Cursor struct {
	y, x int // 0-based
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
	s.setCursorYX(c.c.y, c.c.x)
	s.wrapNext = c.wn
}

//----------
//----------
//----------

// Modes keeps DEC private modes (?n).
type Modes struct {
	m map[int]bool
}

func NewModes() *Modes {
	m := &Modes{m: make(map[int]bool)}
	// defaults
	m.set(7, true)  // autowrap
	m.set(20, true) // line feed newline mode
	m.set(25, true) // cursor
	return m
}

func (m *Modes) set(n int, on bool) { m.m[n] = on }
func (m *Modes) Is(n int) bool      { return m.m[n] }

//----------

func (m *Modes) CursorKeysMode() bool      { return m.Is(1) }
func (m *Modes) column132Mode() bool       { return m.Is(3) }
func (m *Modes) origin() bool              { return m.Is(6) }
func (m *Modes) autoWrap() bool            { return m.Is(7) }
func (m *Modes) autoRepeat() bool          { return m.Is(8) }
func (m *Modes) LineFeedNewlineMode() bool { return m.Is(20) }
func (m *Modes) cursor() bool              { return m.Is(25) }
func (m *Modes) leftRightMarginMode() bool { return m.Is(69) }

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
