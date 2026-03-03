package termemu

import (
	"bytes"
	"image/color"
)

type ScreenPrinter struct {
	//Border    bool
	//Seperator bool
	ColorFn func(offset int, fg, bg color.Color, inverse bool)

	CursorRune rune // mostly for testing where there are no colors, so a rune is printed for guidance

	// double buffer to avoid writing over the currently displayed bytes
	bufK int
	bufs [2]bytes.Buffer
}

func NewScreenPrinter() *ScreenPrinter {
	sp := &ScreenPrinter{}
	sp.ColorFn = func(_ int, _, _ color.Color, _ bool) {}

	//sp.Border = true // TESTING
	//sp.Seperator = true

	return sp
}

//func (sp *ScreenPrinter) Bprint(scr *Screen) []byte {
//	// choose buffer
//	sp.bufK = (sp.bufK + 1) % len(sp.bufs)
//	buf := sp.bufs[sp.bufK]

//	buf.Reset()

//	//----------

//	if scr.ScrollBackBuf1 != nil {
//		sb := scr.ScrollBackBuf1
//		buf.Write(sb)
//		if len(sb) > 0 && sb[len(sb)-1] != '\n' {
//			buf.WriteString("\n")
//		}
//	}

//	//----------

//	border := func(s string) {
//		if sp.Border {
//			buf.WriteString(s)
//		}
//	}

//	isCursor := func(x, y int) bool {
//		return scr.IsCursor(x, y) && scr.privModes.showCursor()
//	}

//	width := len(scr.Grid.lines[0].cells)

//	if sp.Seperator {
//		//s:="↕"+strings.Repeat("─", width) + "┐\n")
//		buf.WriteString("" + strings.Repeat("─", width-1) + "┼\n")
//	}

//	border("┌")
//	border(strings.Repeat("─", width))
//	border("┐\n")

//	maxOffset := 0
//	for y, line := range scr.Grid.lines {

//		// when there is no border, backtrack runes from end to find first non empty - needs to be done here to have correct color positions
//		max2 := len(line.cells) // exclusive
//		if !sp.Border {
//			for ; max2 > 0; max2-- {
//				x := max2 - 1
//				c := line.cells[x]
//				empty := (c.R == 0 || c.R == ' ') &&
//					c.A.Bg == nil && !c.A.Inverse &&
//					!isCursor(x, y)
//				if !empty {
//					break
//				}
//			}
//		}

//		border("│")
//		for x, cell := range line.cells {
//			offset := buf.Len()
//			maxOffset = offset

//			if !sp.Border && x >= max2 {
//				break
//			}

//			ru := cell.R
//			if ru == 0 {
//				ru = ' '
//			}

//			if isCursor(x, y) {
//				sp.ColorFn(offset, nil, nil, true)
//				if sp.CursorRune != 0 {
//					ru = sp.CursorRune
//				}
//			} else {
//				sp.ColorFn(
//					offset,
//					cell.A.Fg.Color(),
//					cell.A.Bg.Color(),
//					cell.A.Inverse,
//				)
//			}

//			buf.WriteRune(ru)
//		}
//		border("│")
//		buf.WriteString("\n")
//	}

//	border("└")
//	border(strings.Repeat("─", width))
//	border("┘")

//	if !sp.Border {
//		b2 := buf.Bytes()[:maxOffset]
//		return b2
//	}

//	return buf.Bytes()
//}

func (sp *ScreenPrinter) Bprint(scr *Screen) []byte {
	// use old buffer
	//if scr.privModes.SynchronizedOutput() {
	//if scr.grid.cleared {
	//	scr.grid.cleared = false
	//	return sp.bufs[sp.bufK].Bytes()
	//}

	// choose buffer
	sp.bufK = (sp.bufK + 1) % len(sp.bufs)
	buf := sp.bufs[sp.bufK]

	buf.Reset()

	//----------

	if scr.ScrollBackBuf1 != nil {
		sb := scr.ScrollBackBuf1
		buf.Write(sb)
		if len(sb) > 0 && sb[len(sb)-1] != '\n' {
			buf.WriteString("\n")
		}
	}

	//----------

	isCursor := func(x, y int) bool {
		return scr.IsCursor(x, y) && scr.privModes.showCursor()
	}

	for y := range scr.grid.size.Y {
		line := scr.grid.line(y)

		//  backtrack runes from end to find first non empty - needs to be done here to have correct color positions
		max2 := len(line.cells) // exclusive
		for ; max2 > 0; max2-- {
			x := max2 - 1
			c := line.cells[x]
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
			//maxOffset = offset

			if x >= max2 {
				break
			}

			ru := cell.R
			if ru == 0 {
				ru = ' '
			}

			if isCursor(x, y) {
				sp.ColorFn(offset, nil, nil, true)
				if sp.CursorRune != 0 {
					ru = sp.CursorRune
				}
			} else {
				sp.ColorFn(
					offset,
					cell.A.Fg.Color(),
					cell.A.Bg.Color(),
					cell.A.Inverse,
				)
			}

			buf.WriteRune(ru)
		}
		buf.WriteString("\n")
	}

	//if !sp.Border {
	//	b2 := buf.Bytes()[:maxOffset]
	//	return b2
	//}

	return buf.Bytes()
}
