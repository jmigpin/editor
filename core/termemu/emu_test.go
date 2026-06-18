package termemu

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
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
			m := newTuiMock()
			defer m.Close()

			te := newTestEmu(m, Opts{}, 10, 5)
			defer te.Close()

			te.scr.privModes.set("20", tc.lnm)

			seq := cup(tc.startY, tc.startX) + tc.seq
			sendWithBarrier(t, te, seq)

			// Snapshot and verify.
			snap := te.Snapshot()
			if snap.cursor.Y != tc.wantY || snap.cursor.X != tc.wantX {
				snap.PrintWithCursor()
				t.Fatalf("got cursor=(%d,%d), want=(%d,%d). seq=%q",
					snap.cursor.Y, snap.cursor.X, tc.wantY, tc.wantX, printable(seq))
			}
		})
	}
}

func TestCPRRoundTrip(t *testing.T) {
	m := newTuiMock()
	te := newTestEmu(m, Opts{}, 10, 5)
	defer te.Close()

	// Place cursor at (row=3,col=4) [0-based 2,3]
	sendWithBarrier(t, te, cup(2, 3))

	// Ask for CPR
	send(t, te, "\x1b[6n")

	// Read reply from the emu
	got := receive(t, te, 64)

	if want := "\x1b[3;4R"; got != want {
		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}
}

func TestScrollRegionAndOriginMode(t *testing.T) {
	m := newTuiMock()
	te := newTestEmu(m, Opts{}, 5, 6)
	defer te.Close()

	// Region rows 2..5 (1-based); enable origin mode (?6h)
	sendWithBarrier(t, te, "\x1b[2;5r\x1b[?6h") // DECSTBM then set origin mode

	// Home within region should be (top, col1) in origin mode
	// Ask CPR to confirm relative coordinates
	send(t, te, "\x1b[6n")
	got := receive(t, te, 32)
	if want := "\x1b[1;1R"; got != want {
		//s := te.Snapshot()
		//s.PrintWithCursor()
		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}

	// Move down to bottom margin and LF to force region scroll
	sendWithBarrier(t, te, "\x1b[4B") // 4 down inside region
	s := te.Snapshot()
	r := s.dynBounds(s.cursor)
	top, bot := r.Min.Y, r.Max.Y-1
	if top != 1 || bot != 4 { // 0-based internally
		s.PrintWithCursor()
		t.Fatalf("bad region top/bot: %d/%d", top, bot)
	}
	// Next LF should keep cursor at bottom margin
	sendWithBarrier(t, te, "\n")
	s = te.Snapshot()
	if s.cursor.Y != bot {
		t.Fatalf("cursor not at bottom margin after LF: %d vs %d", s.cursor.Y, bot)
	}
}

func TestDchEch(t *testing.T) {
	m := newTuiMock()
	te := newTestEmu(m, Opts{}, 6, 2)
	defer te.Close()

	sendWithBarrier(t, te, "ABCDEF")    // fills first row
	sendWithBarrier(t, te, "\r\x1b[3C") // to col 4 (0-based 3) over 'D'

	sendWithBarrier(t, te, "\x1b[2P") // DCH 2: delete D,E ⇒ row becomes ABCF__
	s := te.Snapshot()
	//s.Print()
	row := s.grid1.lines[0]
	got := string([]rune{row.cells[0].R, row.cells[1].R, row.cells[2].R, row.cells[3].R, row.cells[4].R, row.cells[5].R})
	if got != "ABCF\x00\x00" {
		t.Fatalf("DCH got %q", got)
	}

	sendWithBarrier(t, te, "\r\x1b[2C")

	sendWithBarrier(t, te, "\r\x1b[1C\x1b[2X") // to col 2 then ECH 2: blank BC
	s = te.Snapshot()
	row = s.grid1.lines[0]
	got = string([]rune{row.cells[0].R, row.cells[1].R, row.cells[2].R, row.cells[3].R, row.cells[4].R, row.cells[5].R})
	if got != "A\x00\x00F\x00\x00" {
		t.Fatalf("ECH got %q", got)
	}
}

//----------

func TestDECALN(t *testing.T) {
	m := newTuiMock()
	te := newTestEmu(m, Opts{}, 6, 3)
	defer te.Close()

	sendWithBarrier(t, te, "\x1b#8")

	s := te.Snapshot()
	if s.cursor.Y != 0 || s.cursor.X != 0 {
		t.Fatalf("cursor at (%d,%d), want (0,0)", s.cursor.Y, s.cursor.X)
	}
	for y := 0; y < s.grid.size.Y; y++ {
		for x := 0; x < s.grid.size.X; x++ {
			if s.grid1.lines[y].cells[x].R != 'E' {
				t.Fatalf("cell(%d,%d)=%q, want 'E'", y, x, string(s.grid1.lines[y].cells[x].R))
			}
		}
	}
}

func TestCSI0C_Equals1C(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 5, 2)
	defer te.Close()
	sendWithBarrier(t, te, "A\x1b[0CB") // 0C must move 1
	s := te.Snapshot()
	if s.grid1.lines[0].cells[0].R != 'A' || s.grid1.lines[0].cells[1].R != 0 || s.grid1.lines[0].cells[2].R != 'B' {
		t.Fatalf("got [%q %q %q], want ['A' NUL 'B']",
			s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R, s.grid1.lines[0].cells[2].R)
	}
}

func TestBackspaceMovesLeft(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 5, 1)
	defer te.Close()
	sendWithBarrier(t, te, "AB\bC") // C overwrites B
	s := te.Snapshot()
	if got := string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R}); got != "AC" {
		t.Fatalf("got %q, want AC", got)
	}
}

func TestHT_DefaultStopsEvery8(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 16, 1)
	defer te.Close()
	sendWithBarrier(t, te, "\tX") // start at col0; next stop at col8 → X at 8
	s := te.Snapshot()
	if s.cursor.Y != 0 || s.cursor.X != 9 || s.grid1.lines[0].cells[8].R != 'X' {
		t.Fatalf("tab failed; cur=(%d,%d) cell8=%q", s.cursor.Y, s.cursor.X, string(s.grid1.lines[0].cells[8].R))
	}
}

func TestWrapPending_CancelledByCUB(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 4, 2)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[1;4H*") // put '*' at last col (wrap-pending)
	sendWithBarrier(t, te, "\x1b[1D")    // CUB 1 must cancel wrap
	sendWithBarrier(t, te, "X")          // writes SAME line, col 3
	s := te.Snapshot()
	if s.grid1.lines[0].cells[3].R != '*' || s.grid1.lines[0].cells[2].R != 'X' {
		t.Fatalf("wrap-pending not cancelled")
	}
	if anyNonBlank(s.grid1.lines[1].cells[:]) {
		t.Fatalf("unexpected scroll/wrap into next line")
	}
}

func TestWrapPending_CancelledByAutoWrapOff(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 4, 2)
	defer te.Close()

	sendWithBarrier(t, te, "ABCD\x1b[?7lE")

	s := te.Snapshot()
	if s.grid.line(0).AutoWrapped {
		t.Fatal("autowrap off should cancel the pending wrap")
	}
	if got := string(runesOf(s.grid.line(0).cells)); got != "ABCE" {
		t.Fatalf("line0=%q, want %q", got, "ABCE")
	}
	if anyNonBlank(s.grid.line(1).cells) {
		t.Fatal("unexpected wrap into the next line")
	}
}

func TestDECSC_DECRC_PosRestored(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 10)
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[6;11H") // 1-based -> (5,10)
	sendWithBarrier(t, te, "\x1b7")      // DECSC
	sendWithBarrier(t, te, "\x1b[2;2H")  // move elsewhere
	sendWithBarrier(t, te, "\x1b8")      // DECRC

	s := te.Snapshot()
	if s.cursor.Y != 5 || s.cursor.X != 10 {
		t.Fatalf("cursor=(%d,%d), want (5,10)", s.cursor.Y, s.cursor.X)
	}
}

func TestLRMM_WrapAndCR(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 10, 3)
	defer te.Close()

	// Enable L/R margins 3..8 (1-based) and move to col=1 (→ left margin).
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s\x1b[1;1H")
	//s2 := te.Snapshot()
	//s2.PrintWithCursor()

	// Fill up to right margin; 'F' lands at x=7 and sets wrap-pending.
	sendWithBarrier(t, te, "ABCDEF")
	s := te.Snapshot()
	if s.grid1.lines[0].cells[7].R != 'F' {
		s.PrintWithCursor()
		t.Fatalf("want 'F' at right edge x=7, got %q", string(s.grid1.lines[0].cells[7].R))
	}

	// Next printable triggers the wrap into next line at the left margin (x=2).
	sendWithBarrier(t, te, "G")
	s = te.Snapshot()
	if s.grid1.lines[1].cells[2].R != 'G' {
		t.Fatalf("wrap failed: want 'G' at row=1,x=2 (left margin)")
	}

	// CR must move to left margin (not column 0) and overwrite at x=2.
	sendWithBarrier(t, te, "\rX")
	s = te.Snapshot()
	if s.grid1.lines[1].cells[2].R != 'X' {
		t.Fatalf("CR should move to left margin; got %q elsewhere", string(s.grid1.lines[1].cells[2].R))
	}
}

func TestCAN_SUB_Abort(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 5, 1)
	defer te.Close()
	send(t, te, "\x1b[9999")          // start a CSI
	send(t, te, string([]byte{0x18})) // CAN
	sendWithBarrier(t, te, "A")
	s := te.Snapshot()
	if s.grid1.lines[0].cells[0].R != 'A' {
		t.Fatal("CAN did not abort; parser stuck")
	}
}

func TestCUP_ColumnIsRelativeToLeftMargin_WhenLRMM(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 10, 4)
	defer te.Close()

	// Sanity: LRMM off → CUP 1;1 == absolute col 0
	sendWithBarrier(t, te, "\x1b[H")
	s := te.Snapshot()
	if s.cursor.Y != 0 || s.cursor.X != 0 {
		t.Fatalf("LRMM off: want (0,0), got (%d,%d)", s.cursor.Y, s.cursor.X)
	}

	// Enable LRMM and set left/right margins to 3..8 (1-based) → 0-based [2..7]
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s")

	// CUP 1;1 → should land at left margin (column 3 → x=2)
	sendWithBarrier(t, te, "\x1b[1;1H")
	s = te.Snapshot()
	if s.cursor.X != 2 {
		t.Fatalf("CUP 1;1 with LRMM: want x=2 (left margin), got %d", s.cursor.X)
	}

	// CUP 1;6 → left margin + 5 → x=7 (still within right margin)
	sendWithBarrier(t, te, "\x1b[1;6H")
	s = te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.X != 7 {
		t.Fatalf("CUP 1;6 with LRMM: want x=7, got %d", s.cursor.X)
	}

	// CUP 1;99 → clamp at right margin (x=7)
	sendWithBarrier(t, te, "\x1b[1;99H")
	s = te.Snapshot()
	if s.cursor.X != 7 {
		t.Fatalf("CUP 1;99 with LRMM: want x=7 (right margin), got %d", s.cursor.X)
	}

	// Turn LRMM off → CUP 1;1 back to absolute col 0
	sendWithBarrier(t, te, "\x1b[?69l\x1b[1;1H")
	s = te.Snapshot()
	if s.cursor.X != 0 {
		t.Fatalf("LRMM off again: CUP 1;1 should be x=0, got %d", s.cursor.X)
	}
}

func TestHVP_ColumnIsRelativeToLeftMargin_WhenLRMM(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 12, 4)
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[?69h\x1b[4;9s") // margins 4..9 → x in [3..8]
	sendWithBarrier(t, te, "\x1b[2;1f")          // HVP row2,col1 → x=3
	s := te.Snapshot()
	if s.cursor.X != 3 {
		t.Fatalf("HVP 2;1 with LRMM: want x=3 (left margin), got %d", s.cursor.X)
	}
}

func TestIND_PreservesColumn(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 6, 5)
	defer te.Close()

	sendWithBarrier(t, te, "\x1b[2;3H+\x1b[1D\x1bD+")
	s := te.Snapshot()

	if s.grid1.lines[1].cells[2].R != '+' {
		t.Fatalf("want '+' at row2,col3")
	}
	if s.grid1.lines[2].cells[2].R != '+' {
		t.Fatalf("IND must keep X; want '+' at row3,col3")
	}
}

func TestCSI_I_CHT(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 1)
	defer te.Close()
	sendWithBarrier(t, te, "A\x1b[I")  // default 1 tab -> col 8
	sendWithBarrier(t, te, "B\x1b[2I") // +2 tabs -> col 24 (clamped by W=20)
	s := te.Snapshot()
	if s.grid1.lines[0].cells[0].R != 'A' || s.grid1.lines[0].cells[8].R != 'B' {
		t.Fatalf("CHT failed")
	}
}

func TestCSI_ParamsIgnoreC0(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 10, 3)
	defer te.Close()
	seq := "\x1b[" + string([]byte{0x09}) + "2" + string([]byte{0x0D}) + ";" + string([]byte{0x08}) + "3H"
	sendWithBarrier(t, te, seq+"X") // CUP 2;3 with C0 mixed in
	s := te.Snapshot()
	if s.cursor.Y != 1 || s.cursor.X != 3 || s.grid1.lines[1].cells[2].R != 'X' {
		t.Fatalf("C0 inside CSI not ignored; got (%d,%d)", s.cursor.Y, s.cursor.X)
	}
}

func TestCSI_ParamsIgnore2(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 3)
	defer te.Close()
	seq := "A \x1b[\r2CB\x1b[\r4CC\x1b[\r6CD\x1b[\r8CE\x1b[\r10CF\x1b[\r12CG\x1b[\r14CH\x1b[\r16CI"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()

	//s.Print()
	u := stringOf(s.grid1.lines[0].cells[:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}
func TestCSI_ParamsIgnore3(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 3)
	defer te.Close()
	seq := "A\x1b[2\bCB\x1b[2\bCC\x1b[2\bCD\x1b[2\bCE\x1b[2\bCF\x1b[2\bCG\x1b[2\bCH\x1b[2\bCI\x1b[2\bC"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()
	//s.Print()

	u := stringOf(s.grid1.lines[0].cells[:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}
func TestCSI_ParamsIgnore4(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 3)
	defer te.Close()
	seq := "\x1b[3,1H\x1b[20lA \x1b[1\vAB \x1b[1\vAC \x1b[1\vAD \x1b[1\vAE \x1b[1\vAF \x1b[1\vAG \x1b[1\vAH \x1b[1\vAI \x1b[1\vA\r\r"
	sendWithBarrier(t, te, seq)
	s := te.Snapshot()

	//s.Print()
	u := stringOf(s.grid1.lines[2].cells[:17])
	exp := "A B C D E F G H I"
	if u != exp {
		t.Fatalf("expected %q, got %q", exp, u)
	}
}

func TestTabs_DefaultsAndHTS_TBC(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 1)
	defer te.Close()

	// default every 8: '\tX' → X at col 8 (0-based)
	sendWithBarrier(t, te, "\tX")
	s := te.Snapshot()
	if s.grid1.lines[0].cells[8].R != 'X' {
		t.Fatalf("default tab stop failed")
	}

	// Clear all, set one at col 4 → '\tY' lands at x=3
	sendWithBarrier(t, te, "\x1b[3g\x1b[1;4H\x1bH\x1b[1;1H\tY")
	s = te.Snapshot()
	if s.grid1.lines[0].cells[3].R != 'Y' {
		t.Fatalf("HTS/TBC failed")
	}

	// Reset to default every 8 columns using CSI ? 5 W
	sendWithBarrier(t, te, "\x1b[?5W\x1b[1;1H\tZ")
	s = te.Snapshot()
	if s.grid1.lines[0].cells[8].R != 'Z' {
		t.Fatalf("DECST8C failed to restore tab stops")
	}
}

func TestCHT_CBT(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 40, 1)
	defer te.Close()

	// Go to col 1, forward 2 tabs → x=16; back 1 tab → x=8
	sendWithBarrier(t, te, "\x1b[1;1H\x1b[2I"+"A\b"+"\x1b[1Z"+"B")
	s := te.Snapshot()
	if s.grid1.lines[0].cells[16].R != 'A' || s.grid1.lines[0].cells[8].R != 'B' {
		s.Print()
		t.Fatalf("CHT/CBT failed")
	}
}

func TestTab_RespectsLRMM(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 1)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[?69h\x1b[5;12s\x1b[1;5H") // margins 5..12 → x in [4..11]
	sendWithBarrier(t, te, "\tZ")                          // next stop but not beyond right margin
	s := te.Snapshot()
	if s.cursor.X < 4 || s.cursor.X > 11 {
		t.Fatalf("tab ignored LRMM")
	}
	if s.grid1.lines[0].cells[11].R == 0 { /* ok if it clamped to 11 */
	}
}

func TestCNL_CPL_LeftMarginAndScroll(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 10, 4)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[?69h\x1b[3;8s\x1b[1;1H") // LRMM 3..8; CUP row1,col1 → left margin
	sendWithBarrier(t, te, "A\x1b[2E")                    // CNL 2
	s := te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.Y != 2 || s.cursor.X != 2 {
		t.Fatalf("CNL not at left margin")
	}
	sendWithBarrier(t, te, "B\x1b[1F") // CPL 1
	s = te.Snapshot()
	//s.PrintWithCursor()
	if s.cursor.Y != 1 || s.cursor.X != 2 {
		t.Fatalf("CPL not at left margin")
	}
}

func TestVPR_HPR_Relative(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 8, 4)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b[2;2H\x1b[3a\x1b[2eX") // → (row4,col5) write X
	s := te.Snapshot()
	if s.grid1.lines[3].cells[4].R != 'X' {
		t.Fatalf("HPR/VPR failed")
	}
}

func TestDECSpecial_OnOff(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 6, 1)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b(0qqq") // G0=DEC Special, GL=G0
	s := te.Snapshot()
	//s.Print()
	if s.grid1.lines[0].cells[0].R != '─' || s.grid1.lines[0].cells[1].R != '─' || s.grid1.lines[0].cells[2].R != '─' {
		t.Fatalf("DEC special mapping failed")
	}
	sendWithBarrier(t, te, "\x1b(Bq") // back to ASCII
	s = te.Snapshot()
	//s.Print()
	if s.grid1.lines[0].cells[3].R != 'q' {
		t.Fatalf("ASCII after rmacs failed")
	}
}

func TestDECSpecial_SO_SI_G1(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 4, 1)
	defer te.Close()
	sendWithBarrier(t, te, "\x1b)0")                                          // G1 = DEC Special
	sendWithBarrier(t, te, string([]byte{0x0E})+"q"+string([]byte{0x0F})+"q") // SO q SI q
	s := te.Snapshot()
	if s.grid1.lines[0].cells[0].R != '─' || s.grid1.lines[0].cells[1].R != 'q' {
		t.Fatalf("SO/SI mapping failed")
	}
}

// vttest-like: place on far right using BS+TAB, then left edge via BS.
func Test_RightAndLeftEdges_WithBS_TAB(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 20, 6)
	defer te.Close()

	// Right edge: CUP 5;20, print 'C', BS, TAB (clamp to right edge), 'c'
	sendWithBarrier(t, te, "\x1b[5;20HC\b\tc")
	// Left edge: CUP 5;2, BS to col 1, print 'C'
	sendWithBarrier(t, te, "\x1b[5;2H\bC")

	s := te.Snapshot()
	if s.grid1.lines[4].cells[19].R != 'c' {
		t.Fatalf("want 'c' at (5,20)")
	}
	if s.grid1.lines[4].cells[0].R != 'C' {
		t.Fatalf("want 'C' at (5,1)")
	}
}

// The problematic bit: CR/BS inside CSI params must be ignored.
// This reproduces the “letters down the margins go missing” when CR is executed.
func Test_CSI_Params_Ignore_CR_BS_HT(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 12, 3)
	defer te.Close()

	// CUP 2;10 but with CR and BS injected in params: should still land 2;10.
	seq := "\x1b[" + string([]byte{0x0d}) + "2;" + string([]byte{0x08}) + "10H"
	sendWithBarrier(t, te, seq+"X")

	s := te.Snapshot()
	//s.Print()
	if s.cursor.Y != 1 || s.cursor.X != 10 || s.grid1.lines[1].cells[9].R != 'X' {
		t.Fatalf("C0 inside CSI not ignored; got cur=(%d,%d)", s.cursor.Y, s.cursor.X)
	}
}

func TestVT52_DCA(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 10, 6)
	defer te.Close()
	// Enter VT52 mode
	sendWithBarrierVT52(t, te, "\x1b[?2l")
	// ESC Y (row=3,col=5) → bytes 0x20+3, 0x20+5
	seq := "\x1bY" + string([]byte{0x20 - 1 + 3, 0x20 - 1 + 5})
	sendWithBarrierVT52(t, te, seq+"X")
	s := te.Snapshot()
	if s.cursor.Y != 2 || s.cursor.X != 5 || s.grid1.lines[2].cells[4].R != 'X' {
		s.PrintWithCursor()
		t.Fatalf("VT52 DCA failed: cur=(%d,%d) r='%c'", s.cursor.Y, s.cursor.X, s.grid1.lines[2].cells[4].R)
	}
	// Back to ANSI
	sendWithBarrier(t, te, "\x1b<")
}

func TestVT52_F_G(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 6, 1)
	defer te.Close()
	sendWithBarrierVT52(t, te, "\x1b[?2l") // VT52 mode
	sendWithBarrierVT52(t, te, "\x1bFqqx") // ESC F: graphics on
	sendWithBarrierVT52(t, te, "\x1bGq")   // ESC G: graphics off
	s := te.Snapshot()
	if string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R, s.grid1.lines[0].cells[2].R}) != "──│" {
		t.Fatal("graphics map failed")
	}
	if s.grid1.lines[0].cells[3].R != 'q' {
		t.Fatal("exit graphics failed")
	}
}

func TestIRM_InsertMode(t *testing.T) {
	te := newTestEmu(newTuiMock(), Opts{}, 6, 2)
	defer te.Close()

	// Row 1: "ABCDEF"
	sendWithBarrier(t, te, "\x1b[1;1HABCDEF")

	// IRM on; CUP 1;3; print 'X' -> "ABXCDE" (F drops)
	sendWithBarrier(t, te, "\x1b[4h\x1b[1;3HX")
	s := te.Snapshot()
	got := string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R, s.grid1.lines[0].cells[2].R, s.grid1.lines[0].cells[3].R, s.grid1.lines[0].cells[4].R, s.grid1.lines[0].cells[5].R})
	if got != "ABXCDE" {
		t.Fatalf("IRM insert failed: got %q, want %q", got, "ABXCDE")
	}

	// IRM off; overwrite at next col with 'Y' -> "ABXYDE"
	sendWithBarrier(t, te, "\x1b[4lY")
	s = te.Snapshot()
	got = string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R, s.grid1.lines[0].cells[2].R, s.grid1.lines[0].cells[3].R, s.grid1.lines[0].cells[4].R, s.grid1.lines[0].cells[5].R})
	if got != "ABXYDE" {
		t.Fatalf("Replace after IRM off failed: got %q, want %q", got, "ABXYDE")
	}

	// IRM on at last col; insert 'Z' -> last char replaced, no wrap: "ABXYDZ"
	sendWithBarrier(t, te, "\x1b[4h\x1b[1;6HZ")
	s = te.Snapshot()
	got = string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R, s.grid1.lines[0].cells[2].R, s.grid1.lines[0].cells[3].R, s.grid1.lines[0].cells[4].R, s.grid1.lines[0].cells[5].R})
	if got != "ABXYDZ" {
		t.Fatalf("Insert at right edge failed: got %q, want %q", got, "ABXYDZ")
	}
}

//----------

func TestDecrqm(t *testing.T) {
	m := newTuiMock()
	te := newTestEmu(m, Opts{}, 10, 5)
	defer te.Close()

	send(t, te, "\x1b[?2026$p")
	got := receive(t, te, 64)
	if want := "\x1b[?2026;2$y"; got != want {
		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}
}

func TestXTVERSION(t *testing.T) {
	tests := []string{
		"\x1b[>q",
		"\x1b[>0q",
	}

	for _, seq := range tests {
		t.Run(printable(seq), func(t *testing.T) {
			m := newTuiMock()
			te := newTestEmu(m, Opts{}, 10, 5)
			defer te.Close()

			send(t, te, seq)
			got := receive(t, te, 64)
			if want := "\x1bP>|editor-termemu\x1b\\"; got != want {
				t.Fatalf("got %q, want %q", printable(got), printable(want))
			}
		})
	}
}

//----------

func TestUTF8FragmentedEmit(t *testing.T) {
	pr, pw := io.Pipe()

	detectedError := false
	p := NewVTParser(pr, func(op *TermOp) {
		if op.kind == "print" {
			for _, ru := range op.s {
				if ru == utf8.RuneError {
					detectedError = true
				}
			}
		}
	})

	go func() {
		_ = p.Run()
	}()

	// 1. Send 'A' and the first 2 bytes of '€' (E2 82 AC)
	_, _ = pw.Write([]byte{'A', 0xE2, 0x82})

	// Small pause to let the parser process what it has
	time.Sleep(50 * time.Millisecond)

	// 2. Send the last byte of '€'
	_, _ = pw.Write([]byte{0xAC})

	time.Sleep(50 * time.Millisecond)

	if detectedError {
		t.Errorf("UTF-8 was corrupted by partial emit!")
	}
	_ = pw.Close()
}

//----------
//----------
//----------

// NOTE: long tests
// paste content into a constant using go's backquotes ``
// remove cmds that expect reply to avoid stalling (ex: [0c)
// use "[9n" for custom print/pause for debuging the screen state

func _TestSnapshot0(t *testing.T) {

	opts := Opts{}
	//opts.Mode = ModeRaw
	//opts.Debug = true
	te := newTestEmu(newTuiMock(), opts, 4, 4)
	defer te.Close()

	u := ``
	for i := 0; i < 30; i++ {
		u += "W"
	}

	//2 1[4h [4l
	//[4h [4lb

	sendWithBarrier(t, te, u)
	s := te.Snapshot()
	_ = s
	s.PrintWithCursor()
	t.Fatalf("todo")
}

//----------
//----------
//----------

type TuiMock struct {
	mu     sync.Mutex
	ch     chan struct{}
	Errors []error
}

func newTuiMock() *TuiMock {
	m := &TuiMock{}
	m.ch = make(chan struct{})
	return m
}

func (m *TuiMock) Read(p []byte) (int, error) {
	<-m.ch // simulate no keyboard input, just lock
	return len(p), nil
}
func (m *TuiMock) Write(p []byte) (int, error) {
	// DEBUG
	//fmt.Printf("%s\n", string(p))

	return len(p), nil // simulate output to a display
}
func (m *TuiMock) Close() error {
	if m.ch != nil {
		close(m.ch)
		m.ch = nil
	}
	return nil
}
func (m *TuiMock) OnColumnModeChange() {}
func (m *TuiMock) Paint()              {}
func (m *TuiMock) Error(err error) {
	m.mu.Lock()
	m.Errors = append(m.Errors, err)
	m.mu.Unlock()
	fmt.Println(err)
}
func (m *TuiMock) GetErrors() []error {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]error, len(m.Errors))
	copy(res, m.Errors)
	return res
}
func (m *TuiMock) Print(v any) { fmt.Println(v) }

//----------
//----------
//----------

func newTestEmu(tui *TuiMock, opts Opts, w, h int) *Emu {
	if opts.Mode == ModeOff {
		opts.Mode = ModeGrid
	}
	emu := NewEmu(tui, tui, opts)
	emu.scr.testing = true
	emu.SetSize(P{w, h})

	//go func() {
	// read all cmds sent to exec
	// but then it won't be able to read for testing
	//io.Copy(io.Discard, emu)
	//}()

	return emu
}

//----------

func send(t *testing.T, te *Emu, s string) {
	t.Helper()
	_, _ = te.Write([]byte(s))
}

func sendWithBarrier(t *testing.T, te *Emu, seq string) {
	t.Helper()
	ping, pong := "\x1b[5n", "\x1b[0n" // DSR 5
	sendWithBarrier2(t, te, seq, ping, pong)
}
func sendWithBarrierVT52(t *testing.T, te *Emu, seq string) {
	t.Helper()
	ping, pong := "\x1bZ", "\x1b/Z"
	sendWithBarrier2(t, te, seq, ping, pong)
}

func sendWithBarrier2(t *testing.T, te *Emu, seq string, ping, pong string) {
	t.Helper()
	send(t, te, seq+ping)
	expectedReply := pong

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

func cup(row0, col0 int) string { // 0-based → VT 1-based
	return fmt.Sprintf("\x1b[%d;%dH", row0+1, col0+1)
}

//----------

func receive(t *testing.T, te *Emu, size int) string {
	t.Helper()
	buf := make([]byte, size)
	n, err := te.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	return string(buf[:n])
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

func TestCsiQ(t *testing.T) {
	mock := newTuiMock()
	defer mock.Close()

	emu := NewEmu(mock, mock, Opts{})
	emu.scr.testing = true
	emu.SetSize(P{10, 5})
	defer emu.Close()

	// 1. Send CSI 1 SP q (DECSCUSR)
	sendWithBarrier(t, emu, "\x1b[1 q")

	// 2. Send CSI 1 q (DECLL)
	sendWithBarrier(t, emu, "\x1b[1q")

	if errs := mock.GetErrors(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// 3. Send CSI = 0 c (tertiary DA query)
	send(t, emu, "\x1b[=0c")
	resp := receive(t, emu, 32)
	if want := "\x1b[>0;1;1c"; resp != want {
		t.Fatalf("got tertiary DA response %q, want %q", printable(resp), printable(want))
	}

	// 4. Send CSI ! p (DECSTR / Soft Reset)
	sendWithBarrier(t, emu, "\x1b[!p")

	if errs := mock.GetErrors(); len(errs) > 0 {
		t.Fatalf("unexpected errors after DECSTR: %v", errs)
	}

	// 5. Send CSI = 1;1 u (kitty kb protocol set flags)
	sendWithBarrier(t, emu, "\x1b[=1;1u")

	if errs := mock.GetErrors(); len(errs) > 0 {
		t.Fatalf("unexpected errors after CSI = u: %v", errs)
	}
}

//----------
