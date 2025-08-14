package termemu

type Screen struct {
	W, H    int
	Grid    [][]Cell
	Cursor  Cursor
	curAttr Attr

	// scrolling region [top..bottom], inclusive, 0-based
	sTop, sBot int // defaults to 0 and H-1

	modes *Modes

	// saved pos for CSI s/u
	saved struct {
		X, Y int
		OK   bool
	}
}

func NewScreen(w, h int) *Screen {
	s := &Screen{W: w, H: h}

	s.Grid = make([][]Cell, h)
	for i := range s.Grid {
		s.Grid[i] = make([]Cell, w)
	}

	s.sTop, s.sBot = 0, h-1

	s.modes = NewModes()
	s.modes.SetDEC(25, true) // cursor
	return s
}

//----------

func (s *Screen) Clone() *Screen {
	cp := Screen{W: s.W, H: s.H, Cursor: s.Cursor, curAttr: s.curAttr}
	cp.Grid = make([][]Cell, s.H)
	for i := range s.Grid {
		cp.Grid[i] = make([]Cell, s.W)
		copy(cp.Grid[i], s.Grid[i])
	}
	return &cp
}

func (s *Screen) PutRune(r rune) {
	if r == 0 {
		return
	}
	if r == '\t' {
		r = ' '
	}
	if s.Cursor.Y < 0 || s.Cursor.Y >= s.H {
		s.Cursor.Y = clamp(s.Cursor.Y, 0, s.H-1)
	}
	if s.Cursor.X < 0 || s.Cursor.X >= s.W {
		s.Cursor.X = clamp(s.Cursor.X, 0, s.W-1)
	}
	s.Grid[s.Cursor.Y][s.Cursor.X] = Cell{R: r, A: s.curAttr}
	s.Cursor.X++
	if s.Cursor.X >= s.W {
		s.Cursor.X = 0
		s.LF()
	}
}

func (s *Screen) CR() {
	s.Cursor.X = 0
}

//func (s *Screen) LF() {
//	if s.Cursor.Y == s.H-1 {
//		// scroll up
//		copy(s.Grid[0:], s.Grid[1:])
//		s.Grid[s.H-1] = make([]Cell, s.W)
//	} else {
//		s.Cursor.Y++
//	}
//}

// LF performs an index (move down one line), scrolling **inside the region**.
func (s *Screen) LF() {
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
	if s.Cursor.X > 0 {
		s.Cursor.X--
	}
}

func (s *Screen) MoveTo(row1, col1 int) { // 1-based
	s.MoveToRow(row1)
	s.MoveToCol(col1)
}
func (s *Screen) MoveToRow(row1 int) { // 1-based
	r := clamp(row1-1, 0, s.H-1)
	s.Cursor.Y = r
}
func (s *Screen) MoveToCol(col1 int) { // 1-based
	c := clamp(col1-1, 0, s.W-1)
	s.Cursor.X = c
}

func (s *Screen) MoveRel(dy, dx int) {
	s.Cursor.Y = clamp(s.Cursor.Y+dy, 0, s.H-1)
	s.Cursor.X = clamp(s.Cursor.X+dx, 0, s.W-1)
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
	copy(s.Grid[top:bot], s.Grid[top+1:bot+1])
	// clear bottom row
	s.Grid[bot] = make([]Cell, s.W)
}

func (s *Screen) scrollDownRegion(top, bot int) {
	// move rows [top..bot-1] down by 1
	copy(s.Grid[top+1:bot+1], s.Grid[top:bot])
	// clear top row
	s.Grid[top] = make([]Cell, s.W)
}

//----------

// In type Screen (0-based). DCH/ECH keep cursor, act on current line only.

func (s *Screen) DeleteChars(n int) {
	//y := clamp(s.Cursor.Y, 0, s.H-1)
	//x := clamp(s.Cursor.X, 0, s.W-1)
	x, y := s.Cursor.X, s.Cursor.Y

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
	blank := Cell{R: ' ', A: s.curAttr}
	for i := s.W - n; i < s.W; i++ {
		row[i] = blank
	}
}

func (s *Screen) EraseChars(n int) {
	//y := clamp(s.Cursor.Y, 0, s.H-1)
	//x := clamp(s.Cursor.X, 0, s.W-1)
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

func (s *Screen) homeOrigin(on bool) {
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
	s.saved.X, s.saved.Y = s.Cursor.X, s.Cursor.Y
	s.saved.OK = true
}

// RestoreCursorPos implements CSI u (RCP).
func (s *Screen) RestoreCursorPos() {
	if !s.saved.OK {
		return
	}
	s.Cursor.X = clamp(s.saved.X, 0, s.W-1)
	s.Cursor.Y = clamp(s.saved.Y, 0, s.H-1)
}

//----------

// insertLines/DL operate only if cursor is inside scroll region.
// insert n blank lines at cursor row within [sTop..sBot].
func (s *Screen) insertLines(n int) {
	//y := clamp(s.Cursor.Y, 0, s.H-1)
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
		copy(s.Grid[dst:s.sBot+1], s.Grid[y:s.sBot-n+1])
	}

	// clear inserted lines with spaces using current attr
	for r := y; r < y+n; r++ {
		row := s.Grid[r]
		blank := Cell{R: ' ', A: s.curAttr}
		for i := range row {
			row[i] = blank
		}
	}
}

// delete n lines at cursor row within [sTop..sBot].
func (s *Screen) deleteLines(n int) {
	//y := clamp(s.Cursor.Y, 0, s.H-1)
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
		copy(s.Grid[y:s.sBot-n+1], s.Grid[src:s.sBot+1])
	}

	// clear vacated bottom lines with spaces using current attr
	for r := s.sBot - n + 1; r <= s.sBot; r++ {
		row := s.Grid[r]
		blank := Cell{R: ' ', A: s.curAttr}
		for i := range row {
			row[i] = blank
		}
	}
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
	X, Y int // 0-based
}

//----------
//----------
//----------

// Modes keeps DEC private modes (?n).
type Modes struct {
	m map[int]bool
}

func NewModes() *Modes { return &Modes{m: make(map[int]bool)} }

func (md *Modes) SetDEC(n int, on bool) { md.m[n] = on }
func (md *Modes) Is(n int) bool         { return md.m[n] }

func (md *Modes) Origin() bool { return md.Is(6) }
func (md *Modes) Cursor() bool { return md.Is(25) }

//----------
//----------
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
