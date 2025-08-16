package termemu

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestCursorMoves(t *testing.T) {
	type mt struct {
		name           string
		startY, startX int    // 0-based
		seq            string // VT bytes after positioning at start
		wantY, wantX   int
	}

	tests := []mt{
		{"home_H", 2, 2, "\x1b[H", 0, 0},
		{"cup_5_10", 0, 0, "\x1b[5;10H", 4, 9},
		{"cha_7G", 1, 3, "\x1b[7G", 1, 6},
		{"vpa_2d", 3, 5, "\x1b[2d", 1, 5},
		{"rel_0C", 1, 1, "\x1b[0C", 1, 2},
		{"rel_3C", 1, 1, "\x1b[3C", 1, 4},
		{"rel_2A", 2, 4, "\x1b[2A", 0, 4},
		{"cr", 1, 5, "\r", 1, 0},
		{"lf_no_scroll", 3, 9, "\n", 4, 9},
		//{"wrap_then_lf", 0, 8, "ABC", 1, 1}, // needs autowrap
		{"lf_scroll", 4, 9, "\n", 4, 9},
		{"el_keep_pos", 2, 5, "\x1b[K", 2, 5},
		{"ed2_keep_pos", 1, 1, "\x1b[2J", 1, 1},
		{"cursor_showhide", 1, 1, "\x1b[?25l\x1b[?25h", 1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newPtyMock()
			defer m.Close()

			te := newTestEmu(m, Opts{W: 10, H: 5})
			//te.scr.modes.set(7, true) // autowrap
			defer te.Close()

			seq := cup(tc.startY, tc.startX) + tc.seq
			sendWithBarrier(t, te, seq)

			// Snapshot and verify.
			snap := te.Snapshot()
			if snap.Cursor.Y != tc.wantY || snap.Cursor.X != tc.wantX {
				t.Fatalf("got cursor=(%d,%d), want=(%d,%d). seq=%q",
					snap.Cursor.Y, snap.Cursor.X, tc.wantY, tc.wantX, printable(seq))
			}
		})
	}
}

func TestCPRRoundTrip(t *testing.T) {
	m := newPtyMock()
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
	m := newPtyMock()
	te := newTestEmu(m, Opts{W: 5, H: 6})
	defer te.Close()

	// Region rows 2..5 (1-based); enable origin mode (?6h)
	sendWithBarrier(t, te, "\x1b[2;5r\x1b[?6h") // DECSTBM then set origin mode

	// Home within region should be (top, col1) in origin mode
	// Ask CPR to confirm relative coordinates
	send(t, te, "\x1b[6n")
	buf := make([]byte, 32)
	n, _ := te.Read(buf)
	if got, want := string(buf[:n]), "\x1b[1;1R"; got != want {
		t.Fatalf("got %q, want %q", printable(got), printable(want))
	}

	// Move down to bottom margin and LF to force region scroll
	sendWithBarrier(t, te, "\x1b[4B") // 4 down inside region
	snap := te.Snapshot()
	top, bot := snap.Region()
	if top != 1 || bot != 4 { // 0-based internally
		t.Fatalf("bad region top/bot: %d/%d", top, bot)
	}
	// Next LF should keep cursor at bottom margin
	sendWithBarrier(t, te, "\n")
	snap = te.Snapshot()
	if snap.Cursor.Y != bot {
		t.Fatalf("cursor not at bottom margin after LF: %d vs %d", snap.Cursor.Y, bot)
	}
}

func TestDCH_ECH(t *testing.T) {
	m := newPtyMock()
	te := newTestEmu(m, Opts{W: 6, H: 2})
	defer te.Close()

	sendWithBarrier(t, te, "ABCDEF")    // fills first row
	sendWithBarrier(t, te, "\r\x1b[3C") // to col 4 (0-based 3) over 'D'

	sendWithBarrier(t, te, "\x1b[2P") // DCH 2: delete D,E ⇒ row becomes ABCF__
	s := te.Snapshot()
	row := s.Grid[0]
	got := string([]rune{row[0].R, row[1].R, row[2].R, row[3].R, row[4].R, row[5].R})
	if got != "ABCF  " {
		t.Fatalf("DCH got %q", got)
	}

	sendWithBarrier(t, te, "\r\x1b[2C")

	sendWithBarrier(t, te, "\r\x1b[1C\x1b[2X") // to col 2 then ECH 2: blank BC
	s = te.Snapshot()
	row = s.Grid[0]
	got = string([]rune{row[0].R, row[1].R, row[2].R, row[3].R, row[4].R, row[5].R})
	if got != "A  F  " {
		t.Fatalf("ECH got %q", got)
	}
}

func TestInsertDeleteLinesWithinRegion(t *testing.T) {
	m := newPtyMock()
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

	// Region rows 2..4; put cursor on row 3 (1-based)
	//sendWithBarrier(t, te, "\x1b[2;4r\x1b[3;1H\x1b[L") // IL 1

	// Region rows 2..4; put cursor on row 2 (1-based)
	sendWithBarrier(t, te, "\x1b[2;4r\x1b[2;1H\x1b[L") // IL 1
	snap := te.Snapshot()

	// Row texts after IL: 1, blank, 3, 4, 5  (2..4 moved down)
	//if snap.Grid[1][0].R != ' ' || snap.Grid[2][0].R != '3' {

	// Row texts after IL: 1, blank, 2, 3, 5  (2..4 moved down)
	//te.Snapshot().Print()
	if snap.Grid[1][0].R != ' ' || snap.Grid[2][0].R != '2' {
		t.Fatalf("IL failed around region")
	}

	// Now DL 1 at row 3 (deletes the '3' line, pulls up within region)
	sendWithBarrier(t, te, "\x1b[3;1H\x1b[M")
	//te.Snapshot().Print()
	snap = te.Snapshot()
	if snap.Grid[2][0].R != '3' {
		t.Fatalf("DL failed within region")
	}
}

func TestEnterIsCRNotLF(t *testing.T) {
	m := newPtyMock()
	te := newTestEmu(m, Opts{W: 4, H: 2})
	defer te.Close()

	sendWithBarrier(t, te, "AB\r") // CR only
	snap := te.Snapshot()
	if snap.Cursor.Y != 0 || snap.Cursor.X != 0 {
		t.Fatalf("CR should return to col 0 without moving row, got (%d,%d)", snap.Cursor.Y, snap.Cursor.X)
	}
}

//----------
//----------
//----------

type ptyMock struct {
	pr *io.PipeReader
	pw *io.PipeWriter
}

func newPtyMock() *ptyMock {
	pr, pw := io.Pipe()
	return &ptyMock{pr: pr, pw: pw}
}

func (m *ptyMock) Read(p []byte) (int, error) {
	//return m.pr.Read(p)
	return len(p), nil
}
func (m *ptyMock) Write(p []byte) (int, error) {
	//return m.pw.Write(p)
	return len(p), nil
}
func (m *ptyMock) Close() error {
	_ = m.pr.Close()
	_ = m.pw.Close()
	return nil
}

//----------
//----------
//----------

func newTestEmu(rwc io.ReadWriteCloser, opts Opts) *Emu {
	opts.Mode = ModeUI
	return NewEmu(rwc, opts)
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
	deadline := time.After(10000 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting DSR(5) reply")
		default:
			//buf := make([]byte, 256)
			buf := make([]byte, 4)
			n, err := te.Read(buf)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			// expecting reply: ESC[0n
			if n > 0 && buf[n-1] == 'n' {
				return
			}
		}
	}
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
