package termemu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/fontutil"
)

func TestScreenWraplines(t *testing.T) {
	s := newTestScreen(P{4, 2})

	u := ``
	for i := 0; i < 4*2+1; i++ {
		u += string('0' + rune(i%10))
	}

	writeScreenString(s, u)

	_, _ = s.setSize(P{2, 4})
	out := s.Sprint(true)

	exp := "01\ue00145\ue0018◙"
	if out != exp {
		t.Fatalf("got %q, want %q", out, exp)
	}
}

func TestScreenWraplines2(t *testing.T) {
	s := newTestScreen(P{3, 5})

	u := "AAAAA\nBBBBB\nCCCCC\nDDDDD\nEEEEE"
	writeScreenString(s, u)

	_, _ = s.setSize(P{3, 3})
	out := s.Sprint(true)

	exp := "AAA\ue001AA\nBBB\ue001BB\nCCC\ue001CC\nDDD\ue001∆∆∆\nDD\nEEE\ue001EE◙"
	if out != exp {
		t.Fatalf("got:\n%q\n", out)
	}
}

func TestScreenWraplines3(t *testing.T) {
	s := newTestScreen(P{3, 7})

	u := "AAAAA\nBBBBB\nCCCCC\nDDDDD\nEEEEE"
	writeScreenString(s, u)

	_, _ = s.setSize(P{3, 5})
	out := s.Sprint(true)

	exp := "AAA\ue001AA\nBBB\ue001BB\nCCC\ue001∆∆∆\nCC\nDDD\ue001DD\nEEE\ue001EE◙"
	if out != exp {
		t.Fatalf("got:\n%q", out)
	}
}

func TestScreenAutoWrapped(t *testing.T) {
	s := newTestScreen(P{4, 2})

	writeScreenString(s, "ABCD")
	if s.grid.line(0).AutoWrapped {
		t.Fatal("full line should not be marked before the pending wrap is consumed")
	}

	writeScreenString(s, "E")
	if !s.grid.line(0).AutoWrapped {
		t.Fatal("expected consumed autowrap to mark the previous line")
	}
}

func TestScreenAutoWrappedDoubleWidth(t *testing.T) {
	s := newTestScreen(P{4, 2})
	s.cursor = P{3, 0}

	s.putRune('界')

	if !s.grid.line(0).AutoWrapped {
		t.Fatal("expected double-width autowrap to mark the previous line")
	}
	if got := s.grid.line(1).cell(0).R; got != '界' {
		t.Fatalf("got %q at wrapped destination, want %q", got, '界')
	}
}

func TestScreenAutoWrappedClearedByLineMutation(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Screen)
	}{
		{"write", func(s *Screen) { s.putRune('X') }},
		{"erase", func(s *Screen) { s.grid.clearRangeX(P{}, 1) }},
		{"insert", func(s *Screen) { s.csiIch_insertChars(1) }},
		{"delete", func(s *Screen) { s.csiDch_deleteChars(1) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestScreen(P{4, 2})
			s.grid.line(0).AutoWrapped = true
			tc.fn(s)
			if s.grid.line(0).AutoWrapped {
				t.Fatal("line mutation should clear autowrap metadata")
			}
		})
	}
}

func TestScreenScrollBackPreservesAutoWrapped(t *testing.T) {
	s := newTestScreen(P{4, 2})
	writeLine := func(y int, text string, autoWrapped bool) {
		line := s.grid.line(y)
		for x, ru := range text {
			line.cells[x] = Cell{R: ru}
		}
		line.AutoWrapped = autoWrapped
	}
	writeLine(0, "ABCD", true)
	writeLine(1, "EF", false)

	s.grid.scrollUpR(s.grid.bounds(), 2)

	want := "ABCD" +
		string(rune(fontutil.TermWrapContinuousRune)) +
		"EF\n"
	if got := string(s.grid.scrollBack); got != want {
		t.Fatalf("scrollback=%q, want %q", got, want)
	}

	if got := s.grid.reinsertScrollBackLines(2); got != 2 {
		t.Fatalf("reinserted %d lines, want 2", got)
	}
	if len(s.grid.scrollBack) != 0 {
		t.Fatalf("scrollback not emptied: %q", s.grid.scrollBack)
	}
	if !s.grid.line(0).AutoWrapped || s.grid.line(1).AutoWrapped {
		t.Fatalf("unexpected autowrap metadata: %v, %v", s.grid.line(0).AutoWrapped, s.grid.line(1).AutoWrapped)
	}
	if got := string(runesOf(s.grid.line(0).cells)); got != "ABCD" {
		t.Fatalf("line0=%q, want %q", got, "ABCD")
	}
	if got := string(runesOf(s.grid.line(1).cells)); got != "EF" {
		t.Fatalf("line1=%q, want %q", got, "EF")
	}
}

func TestScreenResizeOverwritesLastRune(t *testing.T) {
	s := newTestScreen(P{10, 5})

	writeScreenString(s, "0123456789A")

	_, _ = s.setSize(P{15, 5})
	writeScreenString(s, "B")

	out := strings.TrimSpace(s.Sprint(false))
	exp := "0123456789\ue001AB"
	if out != exp {
		t.Errorf("got %q, want %q", out, exp)
	}
}

func TestScreenResizeWidthUsesStrictGrid(t *testing.T) {
	s := newTestScreen(P{20, 5})

	input := "0123456789ABCDE"
	writeScreenString(s, input)

	_, _ = s.setSize(P{5, 5})
	_, _ = s.setSize(P{20, 5})

	out := s.Sprint(false)
	if want := "01234"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestScreenEraseScrollback3J(t *testing.T) {
	s := newTestScreen(P{10, 2})

	writeScreenString(s, "L1\nL2\nL3\nL4")

	if len(s.grid.scrollBack) == 0 {
		t.Fatal("expected scrollback")
	}

	s.csiEd_eraseInDisplay(3)

	if len(s.grid.scrollBack) != 0 {
		t.Fatalf("expected empty scrollback, got %d bytes", len(s.grid.scrollBack))
	}
}

func TestScreenRisClearsScrollback(t *testing.T) {
	s := newTestScreen(P{10, 2})

	writeScreenString(s, "L1\nL2\nL3\nL4")

	if len(s.grid.scrollBack) == 0 {
		t.Fatal("expected scrollback")
	}

	s.escRis_reset(true)

	if len(s.grid.scrollBack) != 0 {
		t.Fatalf("expected empty scrollback, got %d bytes", len(s.grid.scrollBack))
	}
}

func TestScreenELandED(t *testing.T) {
	s := newTestScreen(P{6, 3})

	s.cursor = P{0, 0}
	s.cancelWrap()
	writeScreenString(s, "ABCDEF")
	s.cursor = P{0, 1}
	s.cancelWrap()
	writeScreenString(s, "ghijkl")

	s.cursor = P{2, 0}
	s.cancelWrap()
	s.csiEl_eraseInLine(0)
	if got := string(runesOf(s.grid1.lines[0].cells[:])); got != "AB" {
		t.Fatalf("EL0 row0=%q, want 'AB' then blanks", printable(got))
	}

	s.cursor = P{3, 1}
	s.cancelWrap()
	s.csiEd_eraseInDisplay(0)
	if got := string(runesOf(s.grid1.lines[1].cells[:3])); got != "ghi" {
		t.Fatalf("ED0 prefix row1=%q, want 'ghi'", printable(got))
	}
	if anyNonBlank(s.grid1.lines[1].cells[3:]) || anyNonBlank(s.grid1.lines[2].cells[:]) {
		t.Fatalf("ED0 should blank from cursor to end of screen")
	}

	s.cursor = P{0, 0}
	s.cancelWrap()
	writeScreenString(s, "XXXXXX")
	s.cursor = P{0, 1}
	s.cancelWrap()
	writeScreenString(s, "YYYYYY")
	s.cancelWrap()
	s.csiEd_eraseInDisplay(2)
	for y := 0; y < s.grid.size.Y; y++ {
		if anyNonBlank(s.grid1.lines[y].cells[:]) {
			t.Fatalf("ED2 should blank entire screen")
		}
	}
}

func TestScreenED1ClearsToCursor(t *testing.T) {
	s := newTestScreen(P{6, 4})

	for y := 0; y < s.grid.size.Y; y++ {
		for x := 0; x < s.grid.size.X; x++ {
			s.grid1.lines[y].cells[x].R = 'E'
		}
	}

	s.cursor = P{3, 2}
	s.csiEd_eraseInDisplay(1)

	for y := 0; y < 2; y++ {
		if anyNonBlank(s.grid1.lines[y].cells[:]) {
			t.Fatal("ED1 failed above")
		}
	}
	for x := 0; x <= 3; x++ {
		if s.grid1.lines[2].cells[x].R != 0 {
			t.Fatal("ED1 failed at row 3 left side")
		}
	}
}

func TestScreenEL1ClearsLeftToCursor(t *testing.T) {
	s := newTestScreen(P{6, 1})

	s.cursor = P{0, 0}
	s.cancelWrap()
	writeScreenString(s, "ABCDEF")

	s.cursor = P{3, 0}
	s.cancelWrap()
	s.csiEl_eraseInLine(1)

	for x := 0; x <= 3; x++ {
		if s.grid1.lines[0].cells[x].R != 0 {
			t.Fatal("EL1 failed")
		}
	}
	for x := 4; x < 6; x++ {
		if s.grid1.lines[0].cells[x].R == 0 {
			t.Fatal("EL1 overcleared")
		}
	}
}

func TestScreenInsertDeleteLinesWithinRegion(t *testing.T) {
	s := newTestScreen(P{4, 5})

	for i := 1; i <= 5; i++ {
		s.cursor = P{0, i - 1}
		s.cancelWrap()
		writeScreenString(s, fmt.Sprintf("%-4d", i))
	}

	s.setScrollRegion(2, 4)
	s.cursor = P{0, 1}
	s.csiIl_insertLines(1)

	if s.grid1.lines[1].cells[0].R != '\x00' || s.grid1.lines[2].cells[0].R != '2' {
		s.Print()
		t.Fatalf("IL failed around region")
	}

	s.cursor = P{0, 2}
	s.csiDl_deleteLines(1)
	if s.grid1.lines[2].cells[0].R != '3' {
		t.Fatalf("DL failed within region")
	}
}

func TestScreenEnterIsCRNotLF(t *testing.T) {
	s := newTestScreen(P{4, 2})

	writeScreenString(s, "AB\r")
	if s.cursor.Y != 0 || s.cursor.X != 0 {
		t.Fatalf("CR should return to col 0 without moving row, got (%d,%d)", s.cursor.Y, s.cursor.X)
	}
}

func TestScreenINDandRIRespectScrollRegion(t *testing.T) {
	s := newTestScreen(P{4, 5})

	s.setScrollRegion(2, 4)

	s.cursor = P{0, 1}
	s.cancelWrap()
	writeScreenString(s, "AAAA")
	s.cursor = P{0, 2}
	s.cancelWrap()
	writeScreenString(s, "BBBB")
	s.cursor = P{0, 3}
	s.cancelWrap()
	writeScreenString(s, "CCCC")

	s.cursor = P{0, 3}
	s.escInd_index()
	if got := string(runesOf(s.grid1.lines[1].cells[:4])); got != "BBBB" {
		s.PrintWithCursor()
		t.Fatalf("after IND, row2=%q, want BBBB", printable(got))
	}
	if got := string(runesOf(s.grid1.lines[2].cells[:4])); got != "CCCC" {
		t.Fatalf("after IND, row3=%q, want CCCC", printable(got))
	}
	if anyNonBlank(s.grid1.lines[3].cells[:4]) {
		t.Fatalf("after IND, row4 should be blank")
	}

	s.cursor = P{0, 1}
	s.escRi_reverseIndex()
	if anyNonBlank(s.grid1.lines[1].cells[:4]) {
		t.Fatalf("after RI, row2 should be blank")
	}
	if got := string(runesOf(s.grid1.lines[2].cells[:4])); got != "BBBB" {
		s.PrintWithCursor()
		t.Fatalf("after RI, row3=%q, want BBBB", printable(got))
	}
	if got := string(runesOf(s.grid1.lines[3].cells[:4])); got != "CCCC" {
		t.Fatalf("after RI, row4=%q, want CCCC", printable(got))
	}
}

func TestScreenNEL(t *testing.T) {
	s := newTestScreen(P{5, 4})

	s.cursor = P{2, 1}
	s.csiCnl_cursorNextLine(1)
	if s.cursor.Y != 2 || s.cursor.X != 0 {
		t.Fatalf("NEL cursor=(%d,%d), want (2,0)", s.cursor.Y, s.cursor.X)
	}
}

func TestScreenCRandLF(t *testing.T) {
	s := newTestScreen(P{5, 2})

	writeScreenString(s, "ABC\rD\nE")
	if string([]rune{s.grid1.lines[0].cells[0].R, s.grid1.lines[0].cells[1].R}) != "DB" {
		t.Fatal("CR failed")
	}
	if s.grid1.lines[1].cells[0].R != 'E' {
		t.Fatal("LF failed")
	}
}

func TestScreenResizeShrinkHeightKeepsVisibleContentStable(t *testing.T) {
	s := newTestScreen(P{4, 5})

	for i := 0; i < 5; i++ {
		s.cursor = P{0, i}
		s.cancelWrap()
		writeScreenString(s, fmt.Sprintf("L%d", i))
	}
	s.cursor = P{0, 4}
	s.cancelWrap()

	_, _ = s.setSize(P{4, 3})

	if got := string(runesOf(s.grid1.lines[0].cells[:])); got != "L2" {
		t.Fatalf("row0=%q, want L2", got)
	}
	if got := string(runesOf(s.grid1.lines[1].cells[:])); got != "L3" {
		t.Fatalf("row1=%q, want L3", got)
	}
	if got := string(runesOf(s.grid1.lines[2].cells[:])); got != "L4" {
		t.Fatalf("row2=%q, want L4", got)
	}
	if s.cursor.Y != 2 {
		t.Fatalf("cursor.Y=%d, want 2", s.cursor.Y)
	}
	if got := string(s.grid1.scrollBack); got != "L0\nL1\n" {
		t.Fatalf("scrollback=%q, want %q", got, "L0\nL1\n")
	}
}

func TestScreenResizeShrinkHeightThenRedrawBottomLine(t *testing.T) {
	s := newTestScreen(P{4, 5})

	for i := 0; i < 5; i++ {
		s.cursor = P{0, i}
		s.cancelWrap()
		writeScreenString(s, fmt.Sprintf("L%d", i))
	}
	s.cursor = P{0, 4}
	s.cancelWrap()

	_, _ = s.setSize(P{4, 3})

	// Simulate a program that, after being told the new size, redraws what it
	// believes is the bottom visible line.
	s.cursor = P{0, 2}
	s.cancelWrap()
	writeScreenString(s, "L4")

	//sp := NewScreenPrinter()
	//t.Log(string(sp.Bprint(s)))

	if got := string(runesOf(s.grid1.lines[0].cells[:])); got != "L2" {
		t.Fatalf("row0=%q, want L2", got)
	}
	if got := string(runesOf(s.grid1.lines[1].cells[:])); got != "L3" {
		t.Fatalf("row1=%q, want L3", got)
	}
	if got := string(runesOf(s.grid1.lines[2].cells[:])); got != "L4" {
		t.Fatalf("row2=%q, want L4", got)
	}
	if got := string(s.grid1.scrollBack); got != "L0\nL1\n" {
		t.Fatalf("scrollback=%q, want %q", got, "L0\nL1\n")
	}
}

func TestScreenResizeShrinkGrowThenRedrawVisibleLines(t *testing.T) {
	s := newTestScreen(P{4, 5})

	for i := 0; i < 5; i++ {
		s.cursor = P{0, i}
		s.cancelWrap()
		writeScreenString(s, fmt.Sprintf("L%d", i))
	}
	s.cursor = P{0, 4}
	s.cancelWrap()

	_, _ = s.setSize(P{4, 3})
	_, _ = s.setSize(P{4, 5})

	if got := string(s.grid1.scrollBack); got != "" {
		t.Fatalf("scrollback=%q, want empty after grow reinsertion", got)
	}
	if got := string(runesOf(s.grid1.lines[0].cells[:])); got != "L0" {
		t.Fatalf("row0=%q, want L0 after grow reinsertion", got)
	}
	if got := string(runesOf(s.grid1.lines[1].cells[:])); got != "L1" {
		t.Fatalf("row1=%q, want L1 after grow reinsertion", got)
	}

	for i, y := range []int{0, 1, 2, 3, 4} {
		s.cursor = P{0, i}
		s.cancelWrap()
		writeScreenString(s, fmt.Sprintf("L%d", y))
	}
	s.cancelWrap()

	//sp := NewScreenPrinter()
	//sp.CursorRune = '*'
	//t.Log(string(sp.Bprint(s)))

	if got := string(runesOf(s.grid1.lines[0].cells[:])); got != "L0" {
		t.Fatalf("row0=%q, want L0", got)
	}
	if got := string(runesOf(s.grid1.lines[1].cells[:])); got != "L1" {
		t.Fatalf("row1=%q, want L1", got)
	}
	if got := string(runesOf(s.grid1.lines[2].cells[:])); got != "L2" {
		t.Fatalf("row2=%q, want L2", got)
	}
	if got := string(runesOf(s.grid1.lines[3].cells[:])); got != "L3" {
		t.Fatalf("row3=%q, want L3", got)
	}
	if got := string(runesOf(s.grid1.lines[4].cells[:])); got != "L4" {
		t.Fatalf("row4=%q, want L4", got)
	}
	if got := string(s.grid1.scrollBack); got != "" {
		t.Fatalf("scrollback=%q, want empty", got)
	}
}

//----------

func newTestScreen(size P) *Screen {
	s := NewScreen()
	s.testing = true
	_, _ = s.setSize(size)
	return s
}

func writeScreenString(s *Screen, str string) {
	for _, ru := range str {
		switch ru {
		case '\n':
			s.carriageReturn()
			s.lineFeed()
		case '\r':
			s.carriageReturn()
		default:
			s.putRune(ru)
		}
	}
}
