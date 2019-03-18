package widget

import (
	"image/color"
	"time"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

// textedit with extensions
type TextEditX struct {
	*TextEdit

	commentLineStr string // Used in comment/uncomment lines

	flash struct {
		start time.Time
		now   time.Time
		dur   time.Duration
		line  struct {
			on bool
		}
		index struct {
			on    bool
			index int
			len   int
		}
	}
}

func NewTextEditX(ctx ImageContext, cctx ClipboardContext) *TextEditX {
	te := &TextEditX{
		TextEdit: NewTextEdit(ctx, cctx),
	}

	if d, ok := te.Text.Drawer.(*drawer4.Drawer); ok {
		d.Opt.Cursor.On = true
		d.Opt.Colorize.Groups = []*drawer4.ColorizeGroup{
			&d.Opt.SyntaxHighlight.Group,
			&d.Opt.WordHighlight.Group,
			&d.Opt.ParenthesisHighlight.Group,
			{}, // 3=selection
			{}, // 4=flash
		}
	}

	return te
}

//----------

func (te *TextEditX) PaintBase() {
	te.TextEdit.PaintBase()
	te.iterateFlash()
}

func (te *TextEditX) Paint() {
	te.updateSelectionOpt()
	te.updateFlashOpt()
	te.TextEdit.Paint()
}

//----------

func (te *TextEditX) updateSelectionOpt() {
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		g := d.Opt.Colorize.Groups[3]
		if te.TextCursor.SelectionOn() {
			// colors
			pcol := te.TreeThemePaletteColor
			fg := pcol("text_selection_fg")
			bg := pcol("text_selection_bg")
			// colorize ops
			s, e := te.TextCursor.SelectionIndexes()
			g.Ops = []*drawer4.ColorizeOp{
				{Offset: s, Fg: fg, Bg: bg},
				{Offset: e},
			}
			// don't draw other colorizations
			d.Opt.WordHighlight.Group.Off = true
			d.Opt.ParenthesisHighlight.Group.Off = true
		} else {
			g.Ops = nil
			// draw other colorizations
			d.Opt.WordHighlight.Group.Off = false
			d.Opt.ParenthesisHighlight.Group.Off = false
		}
	}
}

//----------

func (te *TextEditX) FlashLine(index int) {
	te.startFlash(index, 0, true)
}

func (te *TextEditX) FlashIndexLen(index int, len int) {
	te.startFlash(index, len, len == 0)
}

// Safe to use concurrently. If line is true then len is calculated.
func (te *TextEditX) startFlash(index, len int, line bool) {
	te.RunOnUIGoRoutine(func() {
		te.flash.start = time.Now()
		te.flash.dur = 500 * time.Millisecond

		if line {
			// recalc index/len
			i0, i1 := te.lineIndexes(index)
			index = i0
			len = i1 - index

			te.flash.line.on = true
			// need at least len 1 or the colorize op will be canceled
			if len == 0 {
				len = 1
			}
		}

		// flash index (accurate runes)
		te.flash.index.on = true
		te.flash.index.index = index
		te.flash.index.len = len

		te.MarkNeedsPaint()
	})
}

func (te *TextEditX) lineIndexes(offset int) (int, int) {
	lsi, err := iorw.LineStartIndex(te.Drawer.Reader(), offset)
	if err != nil {
		return 0, 0
	}
	lei, nl, err := iorw.LineEndIndex(te.Drawer.Reader(), offset)
	if err != nil {
		return 0, 0
	}
	if nl {
		lei--
	}
	return lsi, lei
}

//----------

func (te *TextEditX) iterateFlash() {
	if !te.flash.line.on && !te.flash.index.on {
		return
	}

	te.flash.now = time.Now()
	end := te.flash.start.Add(te.flash.dur)

	// animation time ended
	if te.flash.now.After(end) {
		te.flash.index.on = false
		te.flash.line.on = false
	} else {
		te.RunOnUIGoRoutine(func() {
			te.MarkNeedsPaint()
		})
	}
}

func (te *TextEditX) updateFlashOpt() {
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		te.updateFlashOpt4(d)
	}
}

func (te *TextEditX) updateFlashOpt4(d *drawer4.Drawer) {
	g := d.Opt.Colorize.Groups[4]
	if !te.flash.index.on {
		g.Ops = nil
		return
	}

	// tint percentage
	t := te.flash.now.Sub(te.flash.start)
	perc := 1.0 - (float64(t) / float64(te.flash.dur))

	// process color function
	bg3 := te.TreeThemePaletteColor("text_bg")
	pc := func(fg, bg color.Color) (_, _ color.Color) {
		fg2 := imageutil.TintOrShade(fg, perc)
		if bg == nil {
			bg = bg3
		}
		bg2 := imageutil.TintOrShade(bg, perc)
		return fg2, bg2
	}

	s := te.flash.index.index
	e := s + te.flash.index.len
	line := te.flash.line.on
	g.Ops = []*drawer4.ColorizeOp{
		{Offset: s, ProcColor: pc, Line: line},
		{Offset: e},
	}
}

//----------

func (te *TextEditX) EnableParenthesisMatch(v bool) {
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		d.Opt.ParenthesisHighlight.On = v
	}
}

//----------

func (te *TextEditX) EnableColorizeSyntax(v bool) {
	// TODO
	//if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
	//	d.ColorizeSyntax.SetOn(v)
	//}
}

//----------

func (te *TextEditX) EnableHighlightCursorWord(v bool) {
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		d.Opt.WordHighlight.On = v
		if !v {
			g := d.Opt.Colorize.Groups[1]
			g.Ops = nil
		}
	}
}

//----------

func (te *TextEditX) SetCommentStrings(line string, enclosed [2]string) {
	te.commentLineStr = line // keep for comment/uncomment lines shortcut

	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		opt := &d.Opt.SyntaxHighlight
		opt.Comment.Line.S = line
		opt.Comment.Enclosed.S = enclosed[0]
		opt.Comment.Enclosed.E = enclosed[1]
	}
}

func (te *TextEditX) CommentLineSymbol() string {
	return te.commentLineStr
}

//----------

func (te *TextEditX) OnThemeChange() {
	te.Text.OnThemeChange()

	pcol := te.TreeThemePaletteColor

	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		d.Opt.Cursor.Fg = pcol("text_cursor_fg")
		d.Opt.LineWrap.Fg = pcol("text_wrapline_fg")
		d.Opt.LineWrap.Bg = pcol("text_wrapline_bg")

		// annotations
		d.Opt.Annotations.Fg = pcol("text_annotations_fg")
		d.Opt.Annotations.Bg = pcol("text_annotations_bg")
		d.Opt.Annotations.Selected.Fg = pcol("text_annotations_select_fg")
		d.Opt.Annotations.Selected.Bg = pcol("text_annotations_select_bg")

		// word highlight
		d.Opt.WordHighlight.Fg = pcol("text_highlightword_fg")
		d.Opt.WordHighlight.Bg = pcol("text_highlightword_bg")

		// parenthesis highlight
		d.Opt.ParenthesisHighlight.Fg = pcol("text_parenthesis_fg")
		d.Opt.ParenthesisHighlight.Bg = pcol("text_parenthesis_bg")

		// syntax highlight
		opt := &d.Opt.SyntaxHighlight
		opt.Comment.Line.Fg = pcol("text_colorize_comments_fg")
		opt.Comment.Line.Bg = pcol("text_colorize_comments_bg")
		opt.Comment.Enclosed.Fg = pcol("text_colorize_comments_fg")
		opt.Comment.Enclosed.Bg = pcol("text_colorize_comments_bg")
		opt.String.Fg = pcol("text_colorize_string_fg")
		opt.String.Bg = pcol("text_colorize_string_bg")
	}
}
