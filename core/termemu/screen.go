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

	Cursor     Cursor
	curAttr    Attr
	modes      Modes
	wrapNext   bool // autowrap support
	sTop, sBot int  // scrolling region [top..bottom], inclusive, 0-based // defaults to 0 and H-1

	csiSavedCursor struct {
		c  Cursor
		ok bool
	}
}

func NewScreen(w, h int) *Screen {
	s := &Screen{W: w, H: h}

	s.Grid = make([][]Cell, h)
	for i := range s.Grid {
		s.Grid[i] = make([]Cell, w)
	}

	s.sTop, s.sBot = 0, h-1

	s.modes = *NewModes()
	s.modes.set(25, true) // cursor
	return s
}

//----------

func (s *Screen) Clone() *Screen {
	cp := *s // copy
	cp.Grid = cloneGrid(cp.Grid)
	return &cp
}

func (s *Screen) PutRune(r rune) {
	if r == 0 {
		return
	}
	if r == '\t' {
		r = ' '
	}

	// apply pending wrap first
	if s.wrapNext {
		s.wrapNext = false
		s.Cursor.X = 0
		s.LF() // scrolls inside region
	}

	s.Grid[s.Cursor.Y][s.Cursor.X] = Cell{R: r, A: s.curAttr}

	if s.Cursor.X >= s.W-1 {
		if s.modes.AutoWrap() {
			// do not move now; set wrap for the *next* printable
			s.wrapNext = true
		} // else: stay at last column, overwrite subsequent prints
	} else {
		s.Cursor.X++
	}
}

//----------

func (s *Screen) CR() {
	s.wrapNext = false

	s.Cursor.X = 0
}

// move down one line, scrolling **inside the region**
func (s *Screen) LF() {
	s.wrapNext = false

	if s.Cursor.Y < s.sBot {
		s.Cursor.Y++
		return
	}
	// at bottom margin: scroll region up
	if s.sTop < s.sBot {
		s.scrollUpRegion(s.sTop, s.sBot)
	}
	// cursor stays on bottom margin
	s.Cursor.Y = s.sBot
}

// Reverse Index (move up), scrolling **inside the region**.
func (s *Screen) RI() {
	s.wrapNext = false

	if s.Cursor.Y > s.sTop {
		s.Cursor.Y--
		return
	}
	// at top margin: scroll region down
	if s.sTop < s.sBot {
		s.scrollDownRegion(s.sTop, s.sBot)
	}
	// cursor stays on top margin
	s.Cursor.Y = s.sTop
}

func (s *Screen) BS() {
	s.wrapNext = false

	if s.Cursor.X > 0 {
		s.Cursor.X--
	}
}

//----------

func (s *Screen) MoveTo(row1, col1 int) { // 1-based
	s.wrapNext = false
	s.MoveToRow(row1)
	s.MoveToCol(col1)
}
func (s *Screen) MoveToRow(row1 int) { // 1-based
	s.wrapNext = false
	s.setCursorY(row1 - 1)
}
func (s *Screen) MoveToCol(col1 int) { // 1-based
	s.wrapNext = false
	s.setCursorX(col1 - 1)
}

func (s *Screen) MoveRel(dy, dx int) {
	s.wrapNext = false
	s.setCursorYX(s.Cursor.Y+dy, s.Cursor.X+dx)
}

//----------

func (s *Screen) EraseDisplay(mode int) {
	switch mode {
	case 0: // cursor→end
		s.EraseLine(0)
		for y := s.Cursor.Y + 1; y < s.H; y++ {
			s.clearLine(y)
		}
	case 1: // home→cursor
		for y := 0; y < s.Cursor.Y; y++ {
			s.clearLine(y)
		}
		s.EraseLine(1)
	default: // 2 or others: entire screen
		for y := 0; y < s.H; y++ {
			s.clearLine(y)
		}
	}
}

func (s *Screen) EraseLine(mode int) {
	y := s.Cursor.Y
	switch mode {
	case 0: // cursor→end
		for x := s.Cursor.X; x < s.W; x++ {
			s.Grid[y][x] = Cell{}
		}
	case 1: // start→cursor
		for x := 0; x <= s.Cursor.X; x++ {
			s.Grid[y][x] = Cell{}
		}
	default: // 2: whole line
		s.clearLine(y)
	}
}

func (s *Screen) SetSGR(params []int) {
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

func (s *Screen) clearLine(y int) {
	row := s.Grid[y]
	for i := range row {
		row[i] = Cell{}
	}
}

//----------

// It clamps to the screen and ensures top<=bot. Caller may want to move the cursor to (top0,0) if emulating DECSTBM semantics.
func (s *Screen) SetScrollRegion(top1, bot1 int) {
	top := clamp(top1-1, 0, s.H-1)
	bot := clamp(bot1-1, 0, s.H-1)
	if top > bot {
		top, bot = 0, s.H-1
	}
	s.sTop, s.sBot = top, bot
}

func (s *Screen) ResetScrollRegion() { s.sTop, s.sBot = 0, s.H-1 }

// Region returns current [top..bottom], inclusive.
func (s *Screen) Region() (int, int) { return s.sTop, s.sBot }

func (s *Screen) scrollUpRegion(top, bot int) {
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

// In type Screen (0-based). DCH/ECH keep cursor, act on current line only.

func (s *Screen) DeleteChars(n int) {
	y, x := s.Cursor.Y, s.Cursor.X

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

func (s *Screen) EraseChars(n int) {
	x, y := s.Cursor.X, s.Cursor.Y

	row := s.Grid[y]

	end := x + n
	if end > s.W {
		end = s.W
	}
	blank := Cell{R: ' ', A: s.curAttr}
	for i := x; i < end; i++ {
		row[i] = blank
	}
}

//----------

// cursor position report
func (s *Screen) replyCPR() (int, int) {
	y, x := s.Cursor.Y, s.Cursor.X
	top := s.sTop // 0-based region top

	row1 := y + 1
	if s.modes.Origin() {
		row1 = (y - top) + 1
		if row1 < 1 {
			row1 = 1
		}
	}
	col1 := x + 1
	return row1, col1
}

//----------

func (s *Screen) moveToOrigin(on bool) {
	if on {
		top, _ := s.Region()
		s.MoveTo(top, 0) // (top, col 0)
	} else {
		s.MoveTo(0, 0)
	}
}

//----------

// SaveCursorPos implements CSI s (SCP).
func (s *Screen) SaveCursorPos() {
	s.csiSavedCursor.c = s.Cursor
	s.csiSavedCursor.ok = true
}

// RestoreCursorPos implements CSI u (RCP).
func (s *Screen) RestoreCursorPos() {
	if !s.csiSavedCursor.ok {
		return
	}
	c := s.csiSavedCursor.c
	s.setCursorYX(c.Y, c.X)
}

//----------

// insertLines/DL operate only if cursor is inside scroll region.
// insert n blank lines at cursor row within [sTop..sBot].
func (s *Screen) insertLines(n int) {
	y := s.Cursor.Y

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
func (s *Screen) deleteLines(n int) {
	y := s.Cursor.Y

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

//----------

func (s *Screen) scrollUp(n int) {
	if n <= 0 {
		return
	}
	h := s.sBot - s.sTop + 1
	if n > h {
		n = h
	}
	for i := 0; i < n; i++ {
		s.scrollUpRegion(s.sTop, s.sBot) // shifts up, blanks bottom
	}
}

func (s *Screen) scrollDown(n int) {
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

//----------

func (scr *Screen) clearCells(w []Cell) {
	for i := range w {
		// blank
		//w[i].R = 0
		w[i].R = ' '
		//w[i] = Cell{}
		//w[i] = Cell{Attr:scr.curAttr}
	}
}

//----------

func (s *Screen) setCursorYX(y, x int) {
	s.setCursorY(y)
	s.setCursorX(x)
}
func (s *Screen) setCursorY(y int) {
	s.Cursor.Y = clamp(y, 0, s.H-1)
}
func (s *Screen) setCursorX(x int) {
	s.Cursor.X = clamp(x, 0, s.W-1)
}

//----------

func (scr *Screen) Print() {
	fmt.Println(scr.String())
}
func (scr *Screen) String() string {
	return string(scr.Bytes(true, true))
	//return string(scr.Bytes(true, false))
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
				if scr.Cursor.X == x && scr.Cursor.Y == y {
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
	Y, X int // 0-based
}

//----------
//----------
//----------

// Modes keeps DEC private modes (?n).
type Modes struct {
	m map[int]bool
}

func NewModes() *Modes { return &Modes{m: make(map[int]bool)} }

func (md *Modes) set(n int, on bool) { md.m[n] = on }
func (md *Modes) Is(n int) bool      { return md.m[n] }

//----------

func (md *Modes) Origin() bool   { return md.Is(6) }
func (md *Modes) AutoWrap() bool { return md.Is(7) }
func (md *Modes) Cursor() bool   { return md.Is(25) }

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
