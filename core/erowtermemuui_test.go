package core

import (
	"image/color"
	"io"
	"testing"
	"time"

	"github.com/jmigpin/editor/core/termemu"
)

func TestTermCellColors(t *testing.T) {
	rgb := func(r, g, b uint8) color.RGBA {
		return color.RGBA{r, g, b, 0xff}
	}
	termRGB := func(r, g, b uint8) termemu.TermColor {
		return termemu.NewTermColorRGB(r, g, b)
	}

	tests := []struct {
		name         string
		fg           termemu.TermColor
		bg           termemu.TermColor
		inverse      bool
		defaultFg    color.Color
		defaultBg    color.Color
		useGrayscale bool
		isLightTheme bool
		wantOk       bool
		wantFg       color.RGBA
		wantBg       color.RGBA
		wantSetBg    bool
	}{
		{
			name:         "noop/default/default",
			defaultFg:    rgb(0x11, 0x22, 0x33),
			defaultBg:    rgb(0xee, 0xdd, 0xcc),
			isLightTheme: false,
			wantOk:       false,
		},
		{
			name:         "inverse/default/default",
			inverse:      true,
			defaultFg:    rgb(0x10, 0x20, 0x30),
			defaultBg:    rgb(0xe0, 0xd0, 0xc0),
			isLightTheme: true,
			wantOk:       true,
			wantFg:       rgb(0xe0, 0xd0, 0xc0),
			wantBg:       rgb(0x10, 0x20, 0x30),
			wantSetBg:    true,
		},
		{
			name:         "inverse/explicit-pair",
			fg:           termRGB(0xaa, 0xbb, 0xcc),
			bg:           termRGB(0x01, 0x02, 0x03),
			inverse:      true,
			defaultFg:    rgb(0x11, 0x22, 0x33),
			defaultBg:    rgb(0xee, 0xdd, 0xcc),
			isLightTheme: false,
			wantOk:       true,
			wantFg:       rgb(0x01, 0x02, 0x03),
			wantBg:       rgb(0xaa, 0xbb, 0xcc),
			wantSetBg:    true,
		},
		{
			name:         "light/implicit-bg-adjust-fg",
			fg:           termRGB(0xf0, 0xf0, 0xf0),
			defaultFg:    rgb(0x40, 0x40, 0x40),
			defaultBg:    rgb(0xff, 0xff, 0xff),
			isLightTheme: true,
			wantOk:       true,
			wantFg:       rgba8Of(ensureContrastColor(rgb(0xf0, 0xf0, 0xf0), rgb(0xff, 0xff, 0xff))),
			wantBg:       rgb(0xff, 0xff, 0xff),
			wantSetBg:    false,
		},
		{
			name:         "light/explicit-bg-keep-pair",
			fg:           termRGB(0xf0, 0xf0, 0xf0),
			bg:           termRGB(0x00, 0x80, 0x00),
			defaultFg:    rgb(0x40, 0x40, 0x40),
			defaultBg:    rgb(0xff, 0xff, 0xff),
			isLightTheme: true,
			wantOk:       true,
			wantFg:       rgb(0xf0, 0xf0, 0xf0),
			wantBg:       rgb(0x00, 0x80, 0x00),
			wantSetBg:    true,
		},
		{
			name:         "grayscale/implicit-bg-only-fg",
			fg:           termRGB(0x10, 0x50, 0x90),
			defaultFg:    rgb(0x20, 0x30, 0x40),
			defaultBg:    rgb(0xfa, 0xfa, 0xfa),
			useGrayscale: true,
			isLightTheme: false,
			wantOk:       true,
			wantFg:       rgba8Of(grayscaleColor(rgb(0x10, 0x50, 0x90))),
			wantBg:       rgb(0xfa, 0xfa, 0xfa),
			wantSetBg:    false,
		},
		{
			name:         "grayscale/explicit-bg-too",
			fg:           termRGB(0x10, 0x50, 0x90),
			bg:           termRGB(0x90, 0x50, 0x10),
			defaultFg:    rgb(0x20, 0x30, 0x40),
			defaultBg:    rgb(0xfa, 0xfa, 0xfa),
			useGrayscale: true,
			isLightTheme: false,
			wantOk:       true,
			wantFg:       rgba8Of(grayscaleColor(rgb(0x10, 0x50, 0x90))),
			wantBg:       rgba8Of(grayscaleColor(rgb(0x90, 0x50, 0x10))),
			wantSetBg:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fg2, bg2, setBg, ok := termCellColors(tc.fg, tc.bg, tc.inverse, tc.defaultFg, tc.defaultBg, tc.useGrayscale, tc.isLightTheme)

			if ok != tc.wantOk {
				t.Fatalf("ok=%v want %v", ok, tc.wantOk)
			}
			if !ok {
				return
			}
			if got, want := rgba8Of(fg2), tc.wantFg; got != want {
				t.Fatalf("fg=%v want %v", got, want)
			}
			if got, want := rgba8Of(bg2), tc.wantBg; got != want {
				t.Fatalf("bg=%v want %v", got, want)
			}
			if setBg != tc.wantSetBg {
				t.Fatalf("setBg=%v want %v", setBg, tc.wantSetBg)
			}
		})
	}
}

func TestTermCellColorsCursorUsesUnderlyingCellColors(t *testing.T) {
	defaultFg := color.RGBA{0x00, 0x00, 0x00, 0xff}
	defaultBg := color.RGBA{0xff, 0xff, 0xea, 0xff}
	cellBg := color.RGBA{0xd9, 0xd9, 0xd9, 0xff}

	emu := termemu.NewEmu(nilReadWriter{}, testTui{}, termemu.Opts{})
	defer emu.Close()
	emu.SetSize(termemu.P{3, 1})

	seq := "\x1b[38;2;0;0;0m\x1b[48;2;217;217;217mA\b"
	if _, err := emu.Write([]byte(seq)); err != nil {
		t.Fatal(err)
	}
	waitForTermCell(t, emu, 'A')

	scr := emu.Snapshot()

	sp := termemu.NewScreenPrinter()
	type result struct {
		fg      termemu.TermColor
		bg      termemu.TermColor
		inverse bool
		ok      bool
	}
	got0 := result{}
	sp.ColorFn = func(_ int, fg, bg termemu.TermColor, inverse bool) {
		if got0.ok {
			return
		}
		got0 = result{fg: fg, bg: bg, inverse: inverse, ok: true}
	}
	_ = sp.Bprint(scr)

	if !got0.ok {
		t.Fatal("missing color op")
	}

	fg2, bg2, setBg, ok := termCellColors(got0.fg, got0.bg, got0.inverse, defaultFg, defaultBg, false, true)
	if !ok {
		t.Fatal("expected colors")
	}
	if got, want := rgba8Of(fg2), cellBg; got != want {
		t.Fatalf("fg=%v want %v", got, want)
	}
	wantBg := color.RGBA{0x00, 0x00, 0x00, 0xff}
	if got, want := rgba8Of(bg2), wantBg; got != want {
		t.Fatalf("bg=%v want %v", got, want)
	}
	if !setBg {
		t.Fatal("expected bg to be set")
	}
}

//----------

type testTui struct{}

func (testTui) OnColumnModeChange() {}
func (testTui) SyncScreen()              {}
func (testTui) Print(any)           {}
func (testTui) Error(error)         {}

type nilReadWriter struct{}

func (nilReadWriter) Read([]byte) (int, error)    { return 0, io.EOF }
func (nilReadWriter) Write(p []byte) (int, error) { return len(p), nil }

func waitForTermCell(t *testing.T, emu *termemu.Emu, ru rune) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		scr := emu.Snapshot()
		if scr.Bprint(false) != nil && len(scr.Bprint(false)) > 0 {
			if scr.Bprint(false)[0] == byte(ru) {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout waiting for terminal cell")
}

func rgba8Of(c color.Color) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}
