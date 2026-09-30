package termemu

import (
	"reflect"
	"testing"
)

func TestRepeatPrecedingCharacter(t *testing.T) {
	tests := []struct {
		name, sequence, expanded string
		width, height            int
		errors                   int
	}{
		{"eleven", "A\x1b[11b", "AAAAAAAAAAAA", 16, 3, 0},
		{"six", "A\x1b[6b", "AAAAAAA", 16, 3, 0},
		{"eight", "A\x1b[8b", "AAAAAAAAA", 16, 3, 0},
		{"default", "A\x1b[b", "AA", 16, 3, 0},
		{"zero", "A\x1b[0b", "AA", 16, 3, 0},
		{"no preceding character", "\x1b[6b", "", 16, 3, 0},
		{"last character in chunk", "AB\x1b[3b", "ABBBB", 16, 3, 0},
		{"chained", "A\x1b[2b\x1b[3b", "AAAAAA", 16, 3, 0},
		{"spaces", "X \x1b[3bY", "X    Y", 16, 3, 0},
		{"unicode", "\u00e9\x1b[3b", "\u00e9\u00e9\u00e9\u00e9", 16, 3, 0},
		{"wide wrap", "\u754c\x1b[3b", "\u754c\u754c\u754c\u754c", 5, 3, 0},
		{"wrap", "A\x1b[6b", "AAAAAAA", 4, 2, 0},
		{"scroll", "A\x1b[9b", "AAAAAAAAAA", 4, 2, 0},
		{"no wrap", "\x1b[?7lA\x1b[6b", "\x1b[?7lAAAAAAA", 4, 2, 0},
		{"cursor movement", "A\x1b[2;2H\x1b[2b", "A\x1b[2;2HAA", 16, 3, 0},
		{"current attributes", "A\x1b[31m\x1b[2b", "A\x1b[31mAA", 16, 3, 0},
		{"mapped graphic", "\x1b(0q\x1b(B\x1b[2b", "\x1b(0q\x1b(B\u2500\u2500", 16, 3, 0},
		{"do not remap", "q\x1b(0\x1b[2b", "qqq\x1b(0", 16, 3, 0},
		{"hard reset", "A\x1bc\x1b[2b", "A\x1bc", 16, 3, 0},
		{"private form", "A\x1b[?2b", "A", 16, 3, 1},
		{"intermediate form", "A\x1b[2 b", "A", 16, 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := func(sequence string, errors int) *Screen {
				t.Helper()
				m := newTuiMock()
				defer m.Close()
				emu := newTestEmu(m, Opts{}, tt.width, tt.height)
				defer emu.Close()
				sendWithBarrier(t, emu, sequence)
				snapshot := emu.Snapshot()
				if got := m.GetErrors(); len(got) != errors {
					t.Fatalf("errors=%v, want %d errors", got, errors)
				}
				return snapshot
			}
			got, want := run(tt.sequence, tt.errors), run(tt.expanded, 0)
			if !reflect.DeepEqual(got.grid.lines, want.grid.lines) || string(got.grid.scrollBack) != string(want.grid.scrollBack) || got.cursor != want.cursor || got.wrapNext != want.wrapNext {
				t.Fatalf("REP differs from literal output: cursor=%v want=%v, scrollback=%q want=%q", got.cursor, want.cursor, got.grid.scrollBack, want.grid.scrollBack)
			}
		})
	}
}
