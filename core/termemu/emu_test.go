package termemu

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCursorMoves(t *testing.T) {
	type mt struct {
		name           string
		startY, startX int    // 0-based
		seq            string // VT bytes after positioning at start
		wantY, wantX   int
		lnm            bool // linefeed newline mode
	}

	tests := []mt{
		{"home_H", 2, 2, "\x1b[H", 0, 0, true},
		{"cup_5_10", 0, 0, "\x1b[5;10H", 4, 9, true},
		{"cha_7G", 1, 3, "\x1b[7G", 1, 6, true},
		{"vpa_2d", 3, 5, "\x1b[2d", 1, 5, true},
		{"rel_0C", 1, 1, "\x1b[0C", 1, 2, true},
		{"rel_3C", 1, 1, "\x1b[3C", 1, 4, true},
		{"rel_2A", 2, 4, "\x1b[2A", 0, 4, true},
		{"cr", 1, 5, "\r", 1, 0, true},
		{"wrap_then_lf", 0, 8, "ABC", 1, 1, true},
		{"lf_no_scroll", 3, 9, "\n", 4, 9, false},
		{"lf_scroll", 4, 9, "\n", 4, 9, false},
		{"el_keep_pos", 2, 5, "\x1b[K", 2, 5, true},
		{"ed2_keep_pos", 1, 1, "\x1b[2J", 1, 1, true},
		{"cursor_showhide", 1, 1, "\x1b[?25l\x1b[?25h", 1, 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newUserMock()
			defer m.Close()

			te := newTestEmu(m, Opts{W: 10, H: 5})
			defer te.Close()

			te.scr.pmodes.set(20, tc.lnm)

			seq := cup(tc.startY, tc.startX) + tc.seq
			sendWithBarrier(t, te, seq)

			// Snapshot and verify.
			snap := te.Snapshot()
			if snap.cursor.y != tc.wantY || snap.cursor.x != tc.wantX {
				snap.PrintWithCursor()
				t.Fatalf("got cursor=(%d,%d), want=(%d,%d). seq=%q",
					snap.cursor.y, snap.cursor.x, tc.wantY, tc.wantX, printable(seq))
			}
		})
	}
}

func TestCPRRoundTrip(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 10, H: 5})
	defer te.Close()

	// Place cursor at (row=3,col=4) [0-based 2,3]
	sendWithBarrier(t, te, cup(2, 3))

	// Ask for CPR
	send(t, te, "\x1b[6n")

	// Read reply from the emu (it writes to readPw → Read())
	buf := make([]byte, 64)
	n, err := te.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	got := string(buf[:n])

	if want := "\x1b[3;4R"; got != want {
		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}
}

func TestScrollRegionAndOriginMode(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 5, H: 6})
	defer te.Close()

	// Region rows 2..5 (1-based); enable origin mode (?6h)
	sendWithBarrier(t, te, "\x1b[2;5r\x1b[?6h") // DECSTBM then set origin mode

	// Home within region should be (top, col1) in origin mode
	// Ask CPR to confirm relative coordinates
	send(t, te, "\x1b[6n")
	buf := make([]byte, 32)
	n, err := te.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(buf[:n]), "\x1b[1;1R"; got != want {
		snap := te.Snapshot()
		snap.Print()

		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}

	// Move down to bottom margin and LF to force region scroll
	sendWithBarrier(t, te, "\x1b[4B") // 4 down inside region
	snap := te.Snapshot()
	top, bot := snap.gbY.AB()
	if top != 1 || bot != 4 { // 0-based internally
		t.Fatalf("bad region top/bot: %d/%d", top, bot)
	}
	// Next LF should keep cursor at bottom margin
	sendWithBarrier(t, te, "\n")
	snap = te.Snapshot()
	if snap.cursor.y != bot {
		t.Fatalf("cursor not at bottom margin after LF: %d vs %d", snap.cursor.y, bot)
	}
}

func TestDchEch(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 6, H: 2})
	defer te.Close()

	sendWithBarrier(t, te, "ABCDEF")    // fills first row
	sendWithBarrier(t, te, "\r\x1b[3C") // to col 4 (0-based 3) over 'D'

	sendWithBarrier(t, te, "\x1b[2P") // DCH 2: delete D,E ⇒ row becomes ABCF__
	s := te.Snapshot()
	//s.Print()
	row := s.Grid[0]
	got := string([]rune{row[0].R, row[1].R, row[2].R, row[3].R, row[4].R, row[5].R})
	if got != "ABCF\x00\x00" {
		t.Fatalf("DCH got %q", got)
	}

	sendWithBarrier(t, te, "\r\x1b[2C")

	sendWithBarrier(t, te, "\r\x1b[1C\x1b[2X") // to col 2 then ECH 2: blank BC
	s = te.Snapshot()
	row = s.Grid[0]
	got = string([]rune{row[0].R, row[1].R, row[2].R, row[3].R, row[4].R, row[5].R})
	if got != "A\x00\x00F\x00\x00" {
		t.Fatalf("ECH got %q", got)
	}
}

func TestInsertDeleteLinesWithinRegion(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 4, H: 5})
	defer te.Close()

	// Fill with labels 1..5
	for i := 1; i <= 5; i++ {
		s := fmt.Sprintf("\r%-4d", i)
		if i < 5 {
			s += "\n" // avoid final scroll
		}
		sendWithBarrier(t, te, s)
	}

	//te.Snapshot().Print()

	// Region rows 2..4; put cursor on row 2 (1-based)
	sendWithBarrier(t, te, "\x1b[2;4r\x1b[2;1H\x1b[L") // IL 1
	snap := te.Snapshot()

	// Row texts after IL: 1, blank, 2, 3, 5  (2..4 moved down)
	if snap.Grid[1][0].R != '\x00' || snap.Grid[2][0].R != '2' {
		snap.Print()
		t.Fatalf("IL failed around region")
	}

	// Now DL 1 at row 3 (deletes the '3' line, pulls up within region)
	sendWithBarrier(t, te, "\x1b[3;1H\x1b[M")
	snap = te.Snapshot()
	//snap.Print()
	if snap.Grid[2][0].R != '3' {
		t.Fatalf("DL failed within region")
	}
}

func TestEnterIsCRNotLF(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 4, H: 2})
	defer te.Close()

	sendWithBarrier(t, te, "AB\r") // CR only
	snap := te.Snapshot()
	if snap.cursor.y != 0 || snap.cursor.x != 0 {
		t.Fatalf("CR should return to col 0 without moving row, got (%d,%d)", snap.cursor.y, snap.cursor.x)
	}
}

//----------

func TestDECALN(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 6, H: 3})
	defer te.Close()

	sendWithBarrier(t, te, "\x1b#8")

	s := te.Snapshot()
	if s.cursor.y != 0 || s.cursor.x != 0 {
		t.Fatalf("cursor at (%d,%d), want (0,0)", s.cursor.y, s.cursor.x)
	}
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if s.Grid[y][x].R != 'E' {
				t.Fatalf("cell(%d,%d)=%q, want 'E'", y, x, string(s.Grid[y][x].R))
			}
		}
	}
}

func TestINDandRI_RespectScrollRegion(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 4, H: 5})
	defer te.Close()

	// Region rows 2..4 (1-based)
	sendWithBarrier(t, te, "\x1b[2;4r")

	// Fill region with tags
	sendWithBarrier(t, te, cup(1, 0)+"AAAA") // row 2 (0-based 1)
	sendWithBarrier(t, te, cup(2, 0)+"BBBB") // row 3 (0-based 2)
	sendWithBarrier(t, te, cup(3, 0)+"CCCC") // row 4 (0-based 3)

	// IND at bottom margin scrolls up inside region
	sendWithBarrier(t, te, cup(3, 0)+"\x1bD")
	s := te.Snapshot()
	if got := string(runesOf(s.Grid[1][:4])); got != "BBBB" {
		s.Print()
		t.Fatalf("after IND, row2=%q, want BBBB", printable(got))
	}
	if got := string(runesOf(s.Grid[2][:4])); got != "CCCC" {
		t.Fatalf("after IND, row3=%q, want CCCC", printable(got))
	}
	if anyNonBlank(s.Grid[3][:4]) {
		t.Fatalf("after IND, row4 should be blank")
	}

	// RI at top margin scrolls down inside region
	sendWithBarrier(t, te, cup(1, 0)+"\x1bM")
	s = te.Snapshot()
	if anyNonBlank(s.Grid[1][:4]) {
		t.Fatalf("after RI, row2 should be blank")
	}
	if got := string(runesOf(s.Grid[2][:4])); got != "BBBB" {
		t.Fatalf("after RI, row3=%q, want BBBB", printable(got))
	}
	if got := string(runesOf(s.Grid[3][:4])); got != "CCCC" {
		t.Fatalf("after RI, row4=%q, want CCCC", printable(got))
	}
}

func TestNEL(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 5, H: 4})
	defer te.Close()

	sendWithBarrier(t, te, cup(1, 2)+"\x1bE") // from (1,2) → CR+LF to (2,0)
	s := te.Snapshot()
	if s.cursor.y != 2 || s.cursor.x != 0 {
		t.Fatalf("NEL cursor=(%d,%d), want (2,0)", s.cursor.y, s.cursor.x)
	}
}

func TestELandED(t *testing.T) {
	m := newUserMock()
	te := newTestEmu(m, Opts{W: 6, H: 3})
	defer te.Close()

	// Row0: "ABCDEF"
	sendWithBarrier(t, te, cup(0, 0)+"ABCDEF")
	// Row1: "ghijkl"
	sendWithBarrier(t, te, cup(1, 0)+"ghijkl")

	// EL0 at row0,col2 → "AB" + blanks
	sendWithBarrier(t, te, cup(0, 2)+"\x1b[0K")
	s := te.Snapshot()
	if got := string(runesOf(s.Grid[0][:])); got != "AB" {
		t.Fatalf("EL0 row0=%q, want 'AB' then blanks", printable(got))
	}

	// ED0 at row1,col3 → clears rest of screen from here
	sendWithBarrier(t, te, cup(1, 3)+"\x1b[0J")
	s = te.Snapshot()
	if got := string(runesOf(s.Grid[1][:3])); got != "ghi" {
		t.Fatalf("ED0 prefix row1=%q, want 'ghi'", printable(got))
	}
	if anyNonBlank(s.Grid[1][3:]) || anyNonBlank(s.Grid[2][:]) {
		t.Fatalf("ED0 should blank from cursor to end of screen")
	}

	// Refill and test ED2 (entire screen)
	sendWithBarrier(t, te, cup(0, 0)+"XXXXXX"+cup(1, 0)+"YYYYYY")
	sendWithBarrier(t, te, "\x1b[2J")
	s = te.Snapshot()
	for y := 0; y < s.H; y++ {
		if anyNonBlank(s.Grid[y][:]) {
			t.Fatalf("ED2 should blank entire screen")
		}
	}
}

func TestCSI0C_Equals1C(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 5, H: 2})
	defer te.Close()
	sendWithBarrier(t, te, "A\x1b[0CB") // 0C must move 1
	s := te.Snapshot()
	if s.Grid[0][0].R != 'A' || s.Grid[0][1].R != 0 || s.Grid[0][2].R != 'B' {
		t.Fatalf("got [%q %q %q], want ['A' NUL 'B']",
			s.Grid[0][0].R, s.Grid[0][1].R, s.Grid[0][2].R)
	}
}

func TestBackspaceMovesLeft(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 5, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "AB\bC") // C overwrites B
	s := te.Snapshot()
	if got := string([]rune{s.Grid[0][0].R, s.Grid[0][1].R}); got != "AC" {
		t.Fatalf("got %q, want AC", got)
	}
}

func TestHT_DefaultStopsEvery8(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 16, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "\tX") // start at col0; next stop at col8 → X at 8
	s := te.Snapshot()
	if s.cursor.y != 0 || s.cursor.x != 9 || s.Grid[0][8].R != 'X' {
		t.Fatalf("tab failed; cur=(%d,%d) cell8=%q", s.cursor.y, s.cursor.x, string(s.Grid[0][8].R))
	}
}

func TestCRandLF(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 5, H: 2})
	defer te.Close()
	sendWithBarrier(t, te, "ABC\rD\nE")
	s := te.Snapshot()
	if string([]rune{s.Grid[0][0].R, s.Grid[0][1].R}) != "DB" {
		t.Fatal("CR failed")
	}
	if s.Grid[1][0].R != 'E' {
		t.Fatal("LF failed")
	}
}

func TestED1_ClearsToCursor(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 6, H: 4})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b#8")           // fill E
	sendWithBarrier(t, te, "\x1b[3;4H\x1b[1J") // ED 1
	s := te.Snapshot()
	for y := 0; y < 2; y++ { // rows above
		if anyNonBlank(s.Grid[y][:]) {
			t.Fatal("ED1 failed above")
		}
	}
	for x := 0; x <= 3; x++ { // up to cursor inclusive
		if s.Grid[2][x].R != 0 {
			t.Fatal("ED1 failed at row 3 left side")
		}
	}
}

func TestEL1_ClearsLeftToCursor(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 6, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "ABCDEF\x1b[1G\x1b[3C") // go to col4
	sendWithBarrier(t, te, "\x1b[1K")              // EL 1
	s := te.Snapshot()
	for x := 0; x <= 3; x++ {
		if s.Grid[0][x].R != 0 {
			t.Fatal("EL1 failed")
		}
	}
	for x := 4; x < 6; x++ {
		if s.Grid[0][x].R == 0 {
			t.Fatal("EL1 overcleared")
		}
	}
}

func TestWrapPending_CancelledByCUB(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 4, H: 2})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[1;4H*") // put '*' at last col (wrap-pending)
	sendWithBarrier(t, te, "\x1b[1D")    // CUB 1 must cancel wrap
	sendWithBarrier(t, te, "X")          // writes SAME line, col 3
	s := te.Snapshot()
	if s.Grid[0][3].R != '*' || s.Grid[0][2].R != 'X' {
		t.Fatalf("wrap-pending not cancelled")
	}
	if anyNonBlank(s.Grid[1][:]) {
		t.Fatalf("unexpected scroll/wrap into next line")
	}
}

func TestDECSC_DECRC_PosRestored(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 10})
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[6;11H") // 1-based -> (5,10)
	sendWithBarrier(t, te, "\x1b7")      // DECSC
	sendWithBarrier(t, te, "\x1b[2;2H")  // move elsewhere
	sendWithBarrier(t, te, "\x1b8")      // DECRC

	s := te.Snapshot()
	if s.cursor.y != 5 || s.cursor.x != 10 {
		t.Fatalf("cursor=(%d,%d), want (5,10)", s.cursor.y, s.cursor.x)
	}
}

func TestLRMM_WrapAndCR(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 10, H: 3})
	defer te.Close()

	// Enable L/R margins 3..8 (1-based) and move to col=1 (→ left margin).
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s\x1b[1;1H")

	// Fill up to right margin; 'F' lands at x=7 and sets wrap-pending.
	sendWithBarrier(t, te, "ABCDEF")
	s := te.Snapshot()
	if s.Grid[0][7].R != 'F' {
		s.PrintWithCursor()
		t.Fatalf("want 'F' at right edge x=7, got %q", string(s.Grid[0][7].R))
	}

	// Next printable triggers the wrap into next line at the left margin (x=2).
	sendWithBarrier(t, te, "G")
	s = te.Snapshot()
	if s.Grid[1][2].R != 'G' {
		t.Fatalf("wrap failed: want 'G' at row=1,x=2 (left margin)")
	}

	// CR must move to left margin (not column 0) and overwrite at x=2.
	sendWithBarrier(t, te, "\rX")
	s = te.Snapshot()
	if s.Grid[1][2].R != 'X' {
		t.Fatalf("CR should move to left margin; got %q elsewhere", string(s.Grid[1][2].R))
	}
}

func TestCAN_SUB_Abort(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 5, H: 1})
	defer te.Close()
	send(t, te, "\x1b[9999")          // start a CSI
	send(t, te, string([]byte{0x18})) // CAN
	sendWithBarrier(t, te, "A")
	s := te.Snapshot()
	if s.Grid[0][0].R != 'A' {
		t.Fatal("CAN did not abort; parser stuck")
	}
}

func TestCUP_ColumnIsRelativeToLeftMargin_WhenLRMM(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 10, H: 4})
	defer te.Close()

	// Sanity: LRMM off → CUP 1;1 == absolute col 0
	sendWithBarrier(t, te, "\x1b[H")
	s := te.Snapshot()
	if s.cursor.y != 0 || s.cursor.x != 0 {
		t.Fatalf("LRMM off: want (0,0), got (%d,%d)", s.cursor.y, s.cursor.x)
	}

	// Enable LRMM and set left/right margins to 3..8 (1-based) → 0-based [2..7]
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s")

	// CUP 1;1 → should land at left margin (column 3 → x=2)
	sendWithBarrier(t, te, "\x1b[1;1H")
	s = te.Snapshot()
	if s.cursor.x != 2 {
		t.Fatalf("CUP 1;1 with LRMM: want x=2 (left margin), got %d", s.cursor.x)
	}

	// CUP 1;6 → left margin + 5 → x=7 (still within right margin)
	sendWithBarrier(t, te, "\x1b[1;6H")
	s = te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.x != 7 {
		t.Fatalf("CUP 1;6 with LRMM: want x=7, got %d", s.cursor.x)
	}

	// CUP 1;99 → clamp at right margin (x=7)
	sendWithBarrier(t, te, "\x1b[1;99H")
	s = te.Snapshot()
	if s.cursor.x != 7 {
		t.Fatalf("CUP 1;99 with LRMM: want x=7 (right margin), got %d", s.cursor.x)
	}

	// Turn LRMM off → CUP 1;1 back to absolute col 0
	sendWithBarrier(t, te, "\x1b[?69l\x1b[1;1H")
	s = te.Snapshot()
	if s.cursor.x != 0 {
		t.Fatalf("LRMM off again: CUP 1;1 should be x=0, got %d", s.cursor.x)
	}
}

func TestHVP_ColumnIsRelativeToLeftMargin_WhenLRMM(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 12, H: 4})
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[?69h\x1b[4;9s") // margins 4..9 → x in [3..8]
	sendWithBarrier(t, te, "\x1b[2;1f")          // HVP row2,col1 → x=3
	s := te.Snapshot()
	if s.cursor.x != 3 {
		t.Fatalf("HVP 2;1 with LRMM: want x=3 (left margin), got %d", s.cursor.x)
	}
}

func TestIND_PreservesColumn(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 6, H: 5})
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[2;3H+\x1b[1D\x1bD+")
	s := te.Snapshot()

	if s.Grid[1][2].R != '+' {
		t.Fatalf("want '+' at row2,col3")
	}
	if s.Grid[2][2].R != '+' {
		t.Fatalf("IND must keep X; want '+' at row3,col3")
	}
}

func TestCSI_I_CHT(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "A\x1b[I")  // default 1 tab -> col 8
	sendWithBarrier(t, te, "B\x1b[2I") // +2 tabs -> col 24 (clamped by W=20)
	s := te.Snapshot()
	if s.Grid[0][0].R != 'A' || s.Grid[0][8].R != 'B' {
		t.Fatalf("CHT failed")
	}
}

func TestCSI_ParamsIgnoreC0(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 10, H: 3})
	defer te.Close()
	seq := "\x1b[" + string([]byte{0x09}) + "2" + string([]byte{0x0D}) + ";" + string([]byte{0x08}) + "3H"
	sendWithBarrier(t, te, seq+"X") // CUP 2;3 with C0 mixed in
	s := te.Snapshot()
	if s.cursor.y != 1 || s.cursor.x != 3 || s.Grid[1][2].R != 'X' {
		t.Fatalf("C0 inside CSI not ignored; got (%d,%d)", s.cursor.y, s.cursor.x)
	}
}

func TestCSI_ParamsIgnore2(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 3})
	defer te.Close()
	seq := "A [2CB[4CC[6CD[8CE[10CF[12CG[14CH[16CI"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()

	//s.Print()
	u := stringOf(s.Grid[0][:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}
func TestCSI_ParamsIgnore3(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 3})
	defer te.Close()
	seq := "A[2CB[2CC[2CD[2CE[2CF[2CG[2CH[2CI[2C"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()
	//s.Print()

	u := stringOf(s.Grid[0][:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}
func TestCSI_ParamsIgnore4(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 3})
	defer te.Close()
	seq := "[3,1H[20lA [1AB [1AC [1AD [1AE [1AF [1AG [1AH [1AI [1A"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()

	//s.Print()
	u := stringOf(s.Grid[2][:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}

func TestTabs_DefaultsAndHTS_TBC(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 1})
	defer te.Close()

	// default every 8: '\tX' → X at col 8 (0-based)
	sendWithBarrier(t, te, "\tX")
	s := te.Snapshot()
	if s.Grid[0][8].R != 'X' {
		t.Fatalf("default tab stop failed")
	}

	// Clear all, set one at col 4 → '\tY' lands at x=3
	sendWithBarrier(t, te, "\x1b[3g\x1b[1;4H\x1bH\x1b[1;1H\tY")
	s = te.Snapshot()
	if s.Grid[0][3].R != 'Y' {
		t.Fatalf("HTS/TBC failed")
	}
}

func TestCHT_CBT(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 40, H: 1})
	defer te.Close()

	// Go to col 1, forward 2 tabs → x=16; back 1 tab → x=8
	sendWithBarrier(t, te, "\x1b[1;1H\x1b[2I"+"A\b"+"\x1b[1Z"+"B")
	s := te.Snapshot()
	if s.Grid[0][16].R != 'A' || s.Grid[0][8].R != 'B' {
		s.Print()
		t.Fatalf("CHT/CBT failed")
	}
}

func TestTab_RespectsLRMM(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[?69h\x1b[5;12s\x1b[1;5H") // margins 5..12 → x in [4..11]
	sendWithBarrier(t, te, "\tZ")                          // next stop but not beyond right margin
	s := te.Snapshot()
	if s.cursor.x < 4 || s.cursor.x > 11 {
		t.Fatalf("tab ignored LRMM")
	}
	if s.Grid[0][11].R == 0 { /* ok if it clamped to 11 */
	}
}

func TestCNL_CPL_LeftMarginAndScroll(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 10, H: 4})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s\x1b[1;1H") // LRMM 3..8; CUP row1,col1 → left margin
	sendWithBarrier(t, te, "A\x1b[2E")                    // CNL 2
	s := te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.y != 2 || s.cursor.x != 2 {
		t.Fatalf("CNL not at left margin")
	}
	sendWithBarrier(t, te, "B\x1b[1F") // CPL 1
	s = te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.y != 1 || s.cursor.x != 2 {
		t.Fatalf("CPL not at left margin")
	}
}

func TestVPR_HPR_Relative(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 8, H: 4})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[2;2H\x1b[3a\x1b[2eX") // → (row4,col5) write X
	s := te.Snapshot()
	if s.Grid[3][4].R != 'X' {
		t.Fatalf("HPR/VPR failed")
	}
}

func TestDECSpecial_OnOff(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 6, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b(0qqq") // G0=DEC Special, GL=G0
	s := te.Snapshot()
	//s.Print()
	if s.Grid[0][0].R != '─' || s.Grid[0][1].R != '─' || s.Grid[0][2].R != '─' {
		t.Fatalf("DEC special mapping failed")
	}
	sendWithBarrier(t, te, "\x1b(Bq") // back to ASCII
	s = te.Snapshot()
	//s.Print()
	if s.Grid[0][3].R != 'q' {
		t.Fatalf("ASCII after rmacs failed")
	}
}

func TestDECSpecial_SO_SI_G1(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 4, H: 1})
	defer te.Close()
	sendWithBarrier(t, te, "\x1b)0")                                          // G1 = DEC Special
	sendWithBarrier(t, te, string([]byte{0x0E})+"q"+string([]byte{0x0F})+"q") // SO q SI q
	s := te.Snapshot()
	if s.Grid[0][0].R != '─' || s.Grid[0][1].R != 'q' {
		t.Fatalf("SO/SI mapping failed")
	}
}

// vttest-like: place on far right using BS+TAB, then left edge via BS.
func Test_RightAndLeftEdges_WithBS_TAB(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 20, H: 6})
	defer te.Close()

	// Right edge: CUP 5;20, print 'C', BS, TAB (clamp to right edge), 'c'
	sendWithBarrier(t, te, "\x1b[5;20HC\b\tc")
	// Left edge: CUP 5;2, BS to col 1, print 'C'
	sendWithBarrier(t, te, "\x1b[5;2H\bC")

	s := te.Snapshot()
	if s.Grid[4][19].R != 'c' {
		t.Fatalf("want 'c' at (5,20)")
	}
	if s.Grid[4][0].R != 'C' {
		t.Fatalf("want 'C' at (5,1)")
	}
}

// The problematic bit: CR/BS inside CSI params must be ignored.
// This reproduces the “letters down the margins go missing” when CR is executed.
func Test_CSI_Params_Ignore_CR_BS_HT(t *testing.T) {
	te := newTestEmu(newUserMock(), Opts{W: 12, H: 3})
	defer te.Close()

	// CUP 2;10 but with CR and BS injected in params: should still land 2;10.
	seq := "\x1b[" + string([]byte{0x0d}) + "2;" + string([]byte{0x08}) + "10H"
	sendWithBarrier(t, te, seq+"X")

	s := te.Snapshot()
	//s.Print()
	if s.cursor.y != 1 || s.cursor.x != 10 || s.Grid[1][9].R != 'X' {
		t.Fatalf("C0 inside CSI not ignored; got cur=(%d,%d)", s.cursor.y, s.cursor.x)
	}
}

//// Column letters via CNL/CPL (LRMM off). If CR got executed inside a CSI elsewhere,
//// these left/right markers get lost; this keeps us honest.
//func Test_CNL_CPL_DrawMarginLetters(t *testing.T) {
//	te := newTestEmu(newUserMock(), Opts{W: 10, H: 6})
//	defer te.Close()

//	// Left column A,B,C downwards using CNL
//	sendWithBarrier(t, te, "\x1b[2;1HA\x1b[EB\x1b[EC")
//	// Right column x,y,z upwards using CHA to col 10 and CPL
//	sendWithBarrier(t, te, "\x1b[6;10Hx\x1b[Fy\x1b[Fz")

//	s := te.Snapshot()
//	s.Print()
//	if s.Grid[1][0].R != 'A' || s.Grid[2][0].R != 'B' || s.Grid[3][0].R != 'C' {
//		t.Fatalf("left margin letters missing")
//	}
//	if s.Grid[5][9].R != 'x' || s.Grid[4][9].R != 'y' || s.Grid[3][9].R != 'z' {
//		t.Fatalf("right margin letters missing")
//	}
//}

//----------
//----------
//----------

type userMock struct {
	ch chan struct{}
}

func newUserMock() *userMock {
	m := &userMock{}
	m.ch = make(chan struct{})
	return m
}

func (m *userMock) Read(p []byte) (int, error) {
	<-m.ch // simulate no keyboard input, just lock
	return len(p), nil
}
func (m *userMock) Write(p []byte) (int, error) {
	return len(p), nil // simulate output to a display
}
func (m *userMock) Close() error {
	if m.ch != nil {
		close(m.ch)
		m.ch = nil
	}
	return nil
}
func (m *userMock) SetSize(int, int) {}
func (m *userMock) Repaint()         {}
func (m *userMock) Error(error)      {}

//----------
//----------
//----------

func newTestEmu(cons ConsoleConn, opts Opts) *Emu {
	opts.Mode = ModeUI
	emu := NewEmu(cons, opts)
	return emu
}

func cup(row0, col0 int) string { // 0-based → VT 1-based
	return fmt.Sprintf("\x1b[%d;%dH", row0+1, col0+1)
}

func send(t *testing.T, te *Emu, s string) {
	t.Helper()
	_, _ = te.Write([]byte(s))
}
func sendWithBarrier(t *testing.T, te *Emu, seq string) {
	t.Helper()
	send(t, te, seq+"\x1b[5n") // seq+DSR 5
	expectedReply := "\x1b[0n"

	buf := make([]byte, len(expectedReply))
	n, err := te.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if u := string(buf[:n]); u != expectedReply {
		t.Fatalf("read: bad barrier: %q", u)
	}
}

//----------

// helpers for quick rune extraction

func stringOf(cells []Cell) string {
	buf := &bytes.Buffer{}
	for _, c := range cells {
		ru := c.R
		if ru == 0 {
			ru = ' '
		}
		buf.WriteRune(ru)
	}
	return buf.String()
}
func runesOf(cells []Cell) []rune {
	rs := make([]rune, len(cells))
	for i, c := range cells {
		rs[i] = c.R
	}
	// trim trailing NULs (blanks) for string compares
	i := len(rs)
	for i > 0 && rs[i-1] == 0 {
		i--
	}
	return rs[:i]
}

func anyNonBlank(cells []Cell) bool {
	for _, c := range cells {
		if c.R != 0 {
			return true
		}
	}
	return false
}

//----------

// printable helps debug control sequences in errors.
func printable(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			switch r {
			case '\x1b':
				b.WriteString("\\x1b")
			case '\n':
				b.WriteString("\\n")
			case '\r':
				b.WriteString("\\r")
			case '\t':
				b.WriteString("\\t")
			default:
				fmt.Fprintf(&b, "\\x%02x", r)
			}
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
