package termemu

import (
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestPixelSizeQueries(t *testing.T) {
	m := newTuiMock()
	defer m.Close()
	emu := newTestEmu(m, Opts{}, 80, 24)
	defer emu.Close()
	sendWithBarrier(t, emu, "ABC\x1b[2;3H")

	// No response is fabricated before valid cell metrics are available.
	sendWithBarrier(t, emu, "\x1b[14t\x1b[15t\x1b[16t")
	for _, cell := range []P{{8, 16}, {10, 20}} {
		if _, changed := emu.SetSize(P{80, 24}, cell); changed {
			t.Fatal("updating cell metrics changed the grid size")
		}
		before := emu.Snapshot()
		for _, query := range []int{14, 15, 16} {
			px := cell
			if query != 16 {
				px = P{80 * cell.X, 24 * cell.Y}
			}
			want := fmt.Sprintf("\x1b[%d;%d;%dt", query-10, px.Y, px.X)
			send(t, emu, fmt.Sprintf("\x1b[%dt", query))
			buf := make([]byte, len(want))
			if _, err := io.ReadFull(emu, buf); err != nil {
				t.Fatal(err)
			}
			if string(buf) != want {
				t.Fatalf("query=%d: reply=%q, want %q", query, buf, want)
			}
		}
		after := emu.Snapshot()
		if before.cursor != after.cursor || !reflect.DeepEqual(before.grid.lines, after.grid.lines) {
			t.Fatal("size queries modified the screen")
		}
	}

	// Queries use the effective clamped grid, including terminal column mode changes.
	sendWithBarrier(t, emu, "\x1b[?3h")
	cr, _ := emu.SetSize(P{80, 24}, P{8, 16})
	if cr.X != 132 {
		t.Fatalf("columns=%d, want 132", cr.X)
	}
	send(t, emu, "\x1b[14t")
	want := "\x1b[4;384;1056t"
	buf := make([]byte, len(want))
	if _, err := io.ReadFull(emu, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != want {
		t.Fatalf("clamped grid reply=%q, want %q", buf, want)
	}
	emu.Snapshot()
	if errors := m.GetErrors(); len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
}
