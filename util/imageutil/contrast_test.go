package imageutil

import (
	"image/color"
	"testing"
)

func TestEnsureContrastColor(t *testing.T) {
	tests := []struct {
		name string
		fg   color.Color
		bg   color.Color
		want color.Color
	}{
		{
			name: "keeps readable pair",
			fg:   color.RGBA{0x00, 0x00, 0x00, 0xff},
			bg:   color.RGBA{0xc6, 0xee, 0x9e, 0xff},
			want: color.RGBA{0x00, 0x00, 0x00, 0xff},
		},
		{
			name: "darkens light fg on light bg",
			fg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
			bg:   color.RGBA{0xc6, 0xee, 0x9e, 0xff},
			want: color.RGBA{0x66, 0x66, 0x66, 0xff},
		},
		{
			name: "tints dark fg on dark bg",
			fg:   color.RGBA{0x10, 0x10, 0x10, 0xff},
			bg:   color.RGBA{0x00, 0x20, 0x00, 0xff},
			want: color.RGBA{0x93, 0x93, 0x93, 0xff},
		},
		{
			name: "keeps nil fg",
			fg:   nil,
			bg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EnsureContrastColor(tc.fg, tc.bg)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
