package termemu

import (
	"image/color"
	"testing"
)

func TestScreenPrinterCursorUsesCellColors(t *testing.T) {
	scr := NewScreen()
	_, _ = scr.setSize(P{3, 1})

	scr.grid1.lines[0].cells[0] = Cell{
		R: 'A',
		A: Attr{
			Fg: NewTermColorRGB(0x00, 0x00, 0x00),
			Bg: NewTermColorRGB(0xd9, 0xd9, 0xd9),
		},
	}
	scr.cursor = P{0, 0}

	sp := NewScreenPrinter()

	got := struct {
		fg      TermColor
		bg      TermColor
		inverse bool
		ok      bool
	}{}
	sp.ColorFn = func(_ int, fg, bg TermColor, inverse bool) {
		if got.ok {
			return
		}
		got.fg = fg
		got.bg = bg
		got.inverse = inverse
		got.ok = true
	}

	_ = sp.Bprint(scr)

	if !got.ok {
		t.Fatal("missing color op")
	}
	if got.fg.Kind() != TermColorRGB || got.bg.Kind() != TermColorRGB {
		t.Fatalf("want rgb colors, got fg=%+v bg=%+v", got.fg, got.bg)
	}
	if gotFg := got.fg.RGBA(); gotFg != (colorRGBA(0x00, 0x00, 0x00)) {
		t.Fatalf("fg=%v", gotFg)
	}
	if gotBg := got.bg.RGBA(); gotBg != (colorRGBA(0xd9, 0xd9, 0xd9)) {
		t.Fatalf("bg=%v", gotBg)
	}
	if !got.inverse {
		t.Fatal("expected inverse")
	}
}

func TestScreenPrinterCursorVisibleAtLineEnd(t *testing.T) {
	scr := NewScreen()
	_, _ = scr.setSize(P{2, 1})

	scr.grid1.lines[0].cells[0] = Cell{
		R: 'A',
		A: Attr{
			Fg: NewTermColorRGB(0x00, 0x00, 0x00),
			Bg: NewTermColorRGB(0xd9, 0xd9, 0xd9),
		},
	}
	scr.grid1.lines[0].cells[1] = Cell{
		R: ' ',
		A: Attr{
			Fg: NewTermColorRGB(0x00, 0x00, 0x00),
			Bg: NewTermColorRGB(0xd9, 0xd9, 0xd9),
		},
	}
	scr.cursor = P{1, 0}

	sp := NewScreenPrinter()

	var gotFg, gotBg TermColor
	var gotInverse bool
	var gotAtSpace bool
	callN := 0
	sp.ColorFn = func(offset int, fg, bg TermColor, inverse bool) {
		callN++
		if callN == 2 {
			gotFg = fg
			gotBg = bg
			gotInverse = inverse
			gotAtSpace = true
		}
	}

	bs := sp.Bprint(scr)
	if got := string(bs); got != "A " {
		t.Fatalf("print=%q want %q", got, "A ")
	}
	if !gotAtSpace {
		t.Fatal("missing cursor color op at trailing space")
	}
	if gotFg.Kind() != TermColorRGB || gotBg.Kind() != TermColorRGB {
		t.Fatalf("want rgb colors, got fg=%+v bg=%+v", gotFg, gotBg)
	}
	if !gotInverse {
		t.Fatal("expected inverse on cursor cell")
	}
}

//----------

func colorRGBA(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 0xff}
}
