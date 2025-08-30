package termemu

import (
	"bytes"
	"image/color"
	"strings"
)

type ScreenPrinter struct {
	Border     bool
	Cursor     bool
	CursorRune rune
	ColorFn    func(offset int, fg, bg color.Color, inverse bool)

	buf bytes.Buffer
}

func NewScreenPrinter() *ScreenPrinter {
	sp := &ScreenPrinter{}
	sp.ColorFn = func(_ int, _, _ color.Color, _ bool) {}
	return sp
}

func (sp *ScreenPrinter) Bprint(scr *Screen) []byte {
	buf := &sp.buf
	buf.Reset()

	//----------

	// TODO: needs clone - or just on the user side
	sb := *scr.ScrollBack
	buf.Write(sb)
	if len(sb) > 0 && sb[len(sb)-1] != '\n' {
		buf.WriteString("\n")
	}

	//----------

	border := func(s string) {
		if sp.Border {
			buf.WriteString(s)
		}
	}

	width := len((*scr.Grid)[0])
	border("┌")
	border(strings.Repeat("─", width))
	border("┐\n")

	for y, line := range *scr.Grid {

		// backtrack runes from end to find first non empty
		max2 := len(line)
		if !sp.Border {
			for ; max2 > 0; max2-- {
				x := max2 - 1
				c := line[x]
				empty := (c.R == 0 || c.R == ' ') &&
					c.A.Bg == nil && !c.A.Inverse &&
					!scr.IsCursor(x, y)
				if !empty {
					break
				}
			}
		}

		border("│")
		for x, cell := range line {

			if !sp.Border && x >= max2 {
				break
			}

			offset := buf.Len()

			sp.ColorFn(
				offset,
				cell.A.Fg.Color(),
				cell.A.Bg.Color(),
				cell.A.Inverse,
			)

			ru := cell.R
			if ru == 0 {
				ru = ' '
			}

			if sp.Cursor && scr.IsCursor(x, y) {
				sp.ColorFn(offset, nil, nil, true)
				if sp.CursorRune != 0 {
					ru = sp.CursorRune
				}
			}

			buf.WriteRune(ru)
		}
		border("│")
		buf.WriteString("\n")
	}

	border("└")
	border(strings.Repeat("─", width))
	border("┘")

	return buf.Bytes()
}
