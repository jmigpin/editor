package widget

import (
	"fmt"
	"image/color"
	"time"

	"github.com/jmigpin/editor/util/drawutil"
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

func NewTextEditX(uiCtx UIContext) *TextEditX {
	te := &TextEditX{
		TextEdit: NewTextEdit(uiCtx),
	}

	opt := te.Text.Drawer.TextDrawerOptions()
	opt.Cursor.On = true

	// setup colorize order
	opt.Colorize.Groups = []*drawutil.ColorizeGroup{
		&opt.SyntaxHighlight.Group,
		&opt.ContentColorize.Group,
		{}, // 2=terminal
		&opt.WordHighlight.Group,
		&opt.ParenthesisHighlight.Group,
		{}, // 5=selection
		{}, // 6=flash
	}
	opt.Decorations.Groups = []*drawutil.DecorationGroup{
		{}, // 0=terminal
	}

	return te
}

const (
	cgIdxTerm      = 2
	cgIdxSelection = 5
	cgIdxFlash     = 6

	dgIdxTerm = 0
)

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
	opt := te.Drawer.TextDrawerOptions()
	g := opt.Colorize.Groups[cgIdxSelection]
	c := te.Cursor()
	if s, e, ok := c.SelectionIndexes(); ok {
		// colors
		pcol := te.TreeThemePaletteColor
		fg := pcol("text_selection_fg")
		bg := pcol("text_selection_bg")
		// colorize ops
		g.Ops = []*drawutil.ColorizeOp{
			{Offset: s, Fg: fg, Bg: bg},
			{Offset: e},
		}
		// don't draw other colorizations
		opt.WordHighlight.Group.Off = true
		opt.ParenthesisHighlight.Group.Off = true
	} else {
		g.Ops = nil
		// draw other colorizations
		opt.WordHighlight.Group.Off = false
		opt.ParenthesisHighlight.Group.Off = false
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
	te.uiCtx.RunOnUIGoRoutine(func() {
		te.flash.start = time.Now()
		te.flash.dur = 500 * time.Millisecond

		if line {
			// recalc index/len
			i0, i1 := te.flashLineIndexes(index)
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

func (te *TextEditX) flashLineIndexes(offset int) (int, int) {
	rd := te.EditCtx().LocalReader(offset)
	s, e, newline, err := iorw.LinesIndexes(rd, offset, offset)
	if err != nil {
		return 0, 0
	}
	if newline {
		e--
	}
	return s, e
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
		te.uiCtx.RunOnUIGoRoutine(func() {
			te.MarkNeedsPaint()
		})
	}
}

func (te *TextEditX) updateFlashOpt() {
	te.updateFlashOpt2(te.Drawer.TextDrawerOptions())
}

func (te *TextEditX) updateFlashOpt2(opt *drawutil.TextDrawerOptions) {
	g := opt.Colorize.Groups[cgIdxFlash]
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
	g.Ops = []*drawutil.ColorizeOp{
		{Offset: s, ProcColor: pc, Line: line},
		{Offset: e},
	}
}

//----------

func (te *TextEditX) EnableParenthesisMatch(v bool) {
	te.Drawer.TextDrawerOptions().ParenthesisHighlight.On = v
}

//----------

func (te *TextEditX) EnableSyntaxHighlight(v bool) {
	opt := te.Drawer.TextDrawerOptions()
	if opt.SyntaxHighlight.On == v {
		return
	}
	opt.SyntaxHighlight.On = v
	te.Drawer.TextDrawerOptionsChanged()
	te.MarkNeedsPaint()
}
func (te *TextEditX) SyntaxHighlight() bool {
	return te.Drawer.TextDrawerOptions().SyntaxHighlight.On
}

//----------

func (te *TextEditX) EnableCursorWordHighlight(v bool) {
	te.Drawer.TextDrawerOptions().WordHighlight.On = v
}

//----------

func (te *TextEditX) EnableGitColorize(v bool) {
	opt := te.Drawer.TextDrawerOptions()
	if opt.ContentColorize.Git.On == v {
		return
	}
	opt.ContentColorize.Git.On = v
	te.Drawer.TextDrawerOptionsChanged()
	te.MarkNeedsPaint()
}

//----------

func (te *TextEditX) SetCommentStrings(a ...any) {
	cs := []*drawutil.SyntaxComment{}
	for i, v := range a {
		// keep first definition for shortcut comment insertion
		if i == 0 {
			v2 := v // local closure
			te.ctx.Fns.CommentLineSym = func() any { return v2 }
		}

		switch t := v.(type) {
		case string:
			// line comment
			c := &drawutil.SyntaxComment{Start: t}
			cs = append(cs, c)
		case [2]string:
			// multiline comment
			c := &drawutil.SyntaxComment{Start: t[0], End: t[1]}
			cs = append(cs, c)
		default:
			panic(fmt.Sprintf("unexpected type: %v", t))
		}
	}

	opt := &te.Drawer.TextDrawerOptions().SyntaxHighlight
	opt.Comment.SCs = cs
	te.Drawer.TextDrawerOptionsChanged()
}

//----------

func (te *TextEditX) EnableTerminalColors(v bool) {
	opt := te.Drawer.TextDrawerOptions()
	opt.Colorize.Groups[cgIdxTerm].Off = !v
	opt.TextContrast.On = v
	te.MarkNeedsPaint()
}

func (te *TextEditX) SetTerminalColorOps(ops []*drawutil.ColorizeOp) {
	te.Text.Drawer.TextDrawerOptions().Colorize.Groups[cgIdxTerm].Ops = ops
	te.MarkNeedsPaint()
}

func (te *TextEditX) SetTerminalDecorations(entries []*drawutil.Decoration) {
	te.Text.Drawer.TextDrawerOptions().Decorations.Groups[dgIdxTerm].Entries = entries
	te.Text.Drawer.TextDrawerOptionsChanged()
	te.MarkNeedsPaint()
}

func (te *TextEditX) EnableTerminalDecorations(v bool) {
	te.Drawer.TextDrawerOptions().Decorations.Groups[dgIdxTerm].Off = !v
	te.MarkNeedsPaint()
}

//----------

func (te *TextEditX) OnThemeChange() {
	te.Text.OnThemeChange()

	pcol := te.TreeThemePaletteColor

	opt := te.Drawer.TextDrawerOptions()
	opt.TextContrast.Bg = pcol("text_bg")
	opt.Cursor.Fg = pcol("text_cursor_fg")
	opt.LineWrap.Fg = pcol("text_wrapline_fg")
	opt.LineWrap.Bg = pcol("text_wrapline_bg")

	// annotations
	opt.Annotations.Fg = pcol("text_annotations_fg")
	opt.Annotations.Bg = pcol("text_annotations_bg")
	opt.Annotations.Selected.Fg = pcol("text_annotations_select_fg")
	opt.Annotations.Selected.Bg = pcol("text_annotations_select_bg")

	// word highlight
	opt.WordHighlight.Fg = pcol("text_highlightword_fg")
	opt.WordHighlight.Bg = pcol("text_highlightword_bg")

	// parenthesis highlight
	opt.ParenthesisHighlight.Fg = pcol("text_parenthesis_fg")
	opt.ParenthesisHighlight.Bg = pcol("text_parenthesis_bg")

	// content colorize
	opt.ContentColorize.Git.AddFg = pcol("text_colorize_git_add_fg")
	opt.ContentColorize.Git.DeleteFg = pcol("text_colorize_git_delete_fg")

	// syntax highlight
	opt.SyntaxHighlight.Comment.Fg = pcol("text_colorize_comments_fg")
	opt.SyntaxHighlight.Comment.Bg = pcol("text_colorize_comments_bg")
	opt.SyntaxHighlight.String.Fg = pcol("text_colorize_string_fg")
	opt.SyntaxHighlight.String.Bg = pcol("text_colorize_string_bg")
	te.Drawer.TextDrawerOptionsChanged()
}
