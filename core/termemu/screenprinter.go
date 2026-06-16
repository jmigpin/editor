package termemu

import (
	"bytes"

	"github.com/jmigpin/editor/util/fontutil"
)

type ScreenPrinter struct {
	ColorFn func(offset int, fg, bg TermColor, inverse bool)
	SepFn   func(offset int)

	CursorRune rune // mostly for testing where there are no colors, so a rune is printed for guidance

	// double buffer to avoid writing over the currently displayed bytes
	bufK int
	bufs [2]bytes.Buffer

	scrollbackSep string

	testing bool
}

func NewScreenPrinter() *ScreenPrinter {
	sp := &ScreenPrinter{}
	sp.ColorFn = func(_ int, _, _ TermColor, _ bool) {}
	sp.SepFn = func(_ int) {}

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
			sp.SepFn(buf.Len())
			if sp.testing {
				buf.WriteString(sbs)
			}
		}
	}

	//----------

	// We intentionally keep cursor-only cells as visible content when  computing the printable line end. While an app is running, many terminals show the cursor via color/inverse (without a dedicated cursor rune), and trimming that cell would make the cursor disappear at line end. Side effect: after the app exits, a trailing " " can remain in text dumps (for example "\n "). This is expected and preferred over losing the live cursor.
	isCursor := func(x, y int) bool {
		return scr.IsCursor(x, y) && scr.privModes.showCursor()
	}

	//----------

	revv := scr.privModes.reverseVideo()
	effectiveCell := func(c *Cell, cursor bool) Cell {
		c2 := *c
		if revv {
			c2.A.Inverse = !c2.A.Inverse
		}
		if cursor {
			c2.A.Inverse = !c2.A.Inverse
		}
		return c2
	}
	coloredBg := func(c *Cell, cursor bool) bool {
		c2 := effectiveCell(c, cursor)
		empty := c2.A.Bg.IsDefault() && !c2.A.Inverse
		return !empty
	}

	//----------

	for y := range scr.grid.size.Y {
		line := scr.grid.line(y)

		// find last non empty to avoid end spaces when copying - done here to have correct color positions after the cut
		max2 := len(line.cells) // exclusive
		for ; max2 > 0; max2-- {
			x := max2 - 1
			c := line.cell(x)
			cursor := isCursor(x, y)
			empty := (c.R == 0 || c.R == ' ') && !coloredBg(c, cursor)
			if !empty {
				break
			}
		}

		for x := range len(line.cells) {
			cell := line.cell(x)

			if x >= max2 {
				break
			}

			offset := buf.Len()
			ru, ok := cell.printableRune()
			if !ok {
				continue
			}
			cursor := isCursor(x, y)

			cell2 := effectiveCell(cell, cursor)

			if cursor {
				sp.ColorFn(offset, cell2.A.Fg, cell2.A.Bg, cell2.A.Inverse)
				if sp.testing && sp.CursorRune != 0 {
					ru = sp.CursorRune
				}
			} else {
				sp.ColorFn(offset, cell2.A.Fg, cell2.A.Bg, cell2.A.Inverse)
			}

			buf.WriteRune(ru)
		}

		// newline
		if line.AutoWrapped {
			buf.WriteRune(fontutil.TermWrapContinuousRune)
		} else {
			buf.WriteByte('\n')
		}
	}

	bs := buf.Bytes()

	// clear ending newlines to prevent the last added newline to push the screen up and make the autoscroll move
	bs = bytes.TrimRight(bs, "\n")

	return bs
}
