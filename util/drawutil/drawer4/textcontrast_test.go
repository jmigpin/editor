package drawer4

import (
	"image/color"
	"testing"

	"github.com/jmigpin/editor/util/imageutil"
)

func TestTextContrastAdjust(t *testing.T) {
	tests := []struct {
		name   string
		on     bool
		fg     color.Color
		bg     color.Color
		lineBg color.Color
		optBg  color.Color
		want   color.Color
	}{
		{
			name: "off keeps low contrast pair",
			fg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
			bg:   color.RGBA{0xc6, 0xee, 0x9e, 0xff},
			want: color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			name: "on adjusts low contrast bg",
			on:   true,
			fg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
			bg:   color.RGBA{0xc6, 0xee, 0x9e, 0xff},
			want: imageutil.EnsureContrastColor(color.RGBA{0xff, 0xff, 0xff, 0xff}, color.RGBA{0xc6, 0xee, 0x9e, 0xff}),
		},
		{
			name:   "on uses line bg fallback",
			on:     true,
			fg:     color.RGBA{0xff, 0xff, 0xff, 0xff},
			lineBg: color.RGBA{0xc6, 0xee, 0x9e, 0xff},
			want:   imageutil.EnsureContrastColor(color.RGBA{0xff, 0xff, 0xff, 0xff}, color.RGBA{0xc6, 0xee, 0x9e, 0xff}),
		},
		{
			name:  "on uses option bg fallback",
			on:    true,
			fg:    color.RGBA{0xff, 0xff, 0xff, 0xff},
			optBg: color.RGBA{0xff, 0xff, 0xff, 0xff},
			want:  imageutil.EnsureContrastColor(color.RGBA{0xff, 0xff, 0xff, 0xff}, color.RGBA{0xff, 0xff, 0xff, 0xff}),
		},
		{
			name: "on keeps nil effective bg",
			on:   true,
			fg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
			want: color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New()
			d.Opt.TextContrast.On = tc.on
			d.st.curColors.fg = tc.fg
			d.st.curColors.bg = tc.bg
			d.st.curColors.lineBg = tc.lineBg
			d.Opt.TextContrast.Bg = tc.optBg
			tci := &TextContrast{d: d}
			tci.adjust()
			if d.st.curColors.fg != tc.want {
				t.Fatalf("got %v, want %v", d.st.curColors.fg, tc.want)
			}
		})
	}
}
