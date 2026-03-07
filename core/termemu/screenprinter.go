package termemu

import (
	"bytes"
	"image/color"
)

type ScreenPrinter struct {
	ColorFn func(offset int, fg, bg color.Color, inverse bool)

	CursorRune rune // mostly for testing where there are no colors, so a rune is printed for guidance

	// double buffer to avoid writing over the currently displayed bytes
	bufK int
	bufs [2]bytes.Buffer

	scrollbackSep string

	testing bool
}

func NewScreenPrinter() *ScreenPrinter {
	sp := &ScreenPrinter{}
	sp.ColorFn = func(_ int, _, _ color.Color, _ bool) {}

	sp.scrollbackSep = "▲▲▲\n"

	return sp
}

func (sp *ScreenPrinter) Bprint(scr *Screen) []byte {
	// choose buffer
	sp.bufK = (sp.bufK + 1) % len(sp.bufs)
	buf := sp.bufs[sp.bufK]

	buf.Reset()

	//----------

	sbs := sp.scrollbackSep
	if sp.testing {
		sbs = "∆∆∆\n"
	}

	if scr.grid.hasScrollBack {
		sb := scr.grid.scrollBack
		if len(sb) > 0 {
			buf.Write(sb)
			if len(sb) > 0 && sb[len(sb)-1] != '\n' {
				buf.WriteString("\n")
			}
			buf.WriteString(sbs)
		}
	}

	//----------

	// We intentionally keep cursor-only cells as visible content when  computing the printable line end. While an app is running, many terminals show the cursor via color/inverse (without a dedicated cursor rune), and trimming that cell would make the cursor disappear at line end. Side effect: after the app exits, a trailing " " can remain in text dumps (for example "\n "). This is expected and preferred over losing the live cursor.
	isCursor := func(x, y int) bool {
		return scr.IsCursor(x, y) && scr.privModes.showCursor()
	}

	for y := range scr.grid.size.Y {
		line := scr.grid.line(y)

		//  backtrack runes from end to find first non empty - needs to be done here to have correct color positions
		max2 := len(line.cells) // exclusive
		for ; max2 > 0; max2-- {
			x := max2 - 1
			c := line.cell(x)
			empty := (c.R == 0 || c.R == ' ') &&
				c.A.Bg == nil && !c.A.Inverse &&
				!isCursor(x, y)
			if !empty {
				break
			}
		}

		for x := range len(line.cells) {
			cell := line.cell(x)

			offset := buf.Len()

			if x >= max2 {
				break
			}

			ru := cell.printableRune()

			if isCursor(x, y) {
				sp.ColorFn(offset, nil, nil, true)

				if sp.testing && sp.CursorRune != 0 {
					ru = sp.CursorRune
				}

			} else {
				sp.ColorFn(
					offset,
					cell.A.Fg,
					cell.A.Bg,
					cell.A.Inverse,
				)
			}

			buf.WriteRune(ru)
		}
		buf.WriteString("\n")
	}

	bs := buf.Bytes()

	// clear ending newlines to prevent the last added newline to push the screen up and make the autoscroll move
	bs = bytes.TrimRight(bs, "\n")

	return bs
}
