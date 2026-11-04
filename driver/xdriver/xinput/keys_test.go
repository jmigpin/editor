package xinput

import (
	"testing"

	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKeysymUnicode(t *testing.T) {
	for _, tc := range []struct {
		keysym xproto.Keysym
		want   rune
	}{
		{0x20, ' '},
		{0xe7, '\u00e7'},
		{0x1ec, '\u011b'},
		{0x1e8, '\u010d'},
		{0x7e1, '\u03b1'},
		{0x20ac, '\u20ac'},
		{0xfe08, 0},
		{0xff0d, 0},
		{0xffbe, 0},
		{0x1008ff13, 0},
		{0x0100fe08, '\ufe08'},
		{0x0101f600, '\U0001f600'},
		{0x0100d800, 0},
		{0x01110000, 0},
	} {
		if got := keysymRune(tc.keysym); got != tc.want {
			t.Errorf("keysym %#x: got %U, want %U", tc.keysym, got, tc.want)
		}
	}
	if got := finalKeysymRune(keysymToEventKeysym(0xfe51), 0xfe51); got != '\u00b4' {
		t.Errorf("dead acute: got %U", got)
	}
}

func TestMouseButtonsIgnoreXKBGroupBits(t *testing.T) {
	for _, tt := range []struct {
		state uint16
		want  event.MouseButtons
	}{
		{0x0, 0},
		{0x2000, 0},
		{0x4000, 0},
		{0x100, event.MouseButtons(event.ButtonLeft)},
		{0x2100, event.MouseButtons(event.ButtonLeft)},
		{0x4100, event.MouseButtons(event.ButtonLeft)},
		{0x1f00, event.MouseButtons(event.ButtonLeft | event.ButtonMiddle | event.ButtonRight | event.ButtonWheelUp | event.ButtonWheelDown)},
	} {
		if got := translateModifiersToEventMouseButtons(tt.state); got != tt.want {
			t.Errorf("state %#x: got %#x, want %#x", tt.state, got, tt.want)
		}
	}

	for _, tt := range []struct {
		button xproto.Button
		want   event.MouseButton
	}{
		{6, event.ButtonWheelLeft},
		{7, event.ButtonWheelRight},
		{8, event.ButtonBackward},
		{9, event.ButtonForward},
	} {
		if got := translateButtonToEventButton(tt.button); got != tt.want {
			t.Errorf("button %d: got %v, want %v", tt.button, got, tt.want)
		}
	}
}
