package xinput

import (
	"testing"

	"github.com/jezek/xgb/xproto"
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
