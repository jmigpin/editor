package widget

import (
	"image"
	"image/color"
	"time"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/iout"
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
			on     bool
			p1, p2 image.Point
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

	if d, ok := te.Text.Drawer.(*drawer3.PosDrawer); ok {
		d.Cursor.SetOn(true)
		// segment groups:
		//	0=selection (always on)
		//	1=word
		//	2=parenthesis
		//	3=flash
		d.Segments.SetOn(true)
		d.Segments.Opt.SetupNGroups(4)
	}

	if d, ok := te.Text.Drawer.(*drawer4.Drawer); ok {
		d.Opt.Cursor.On = true
		d.Opt.Colorize.Groups = []*drawer4.ColorizeGroup{
			&d.Opt.SyntaxHighlight.Group,
			&d.Opt.WordHighlight.Group,
			&d.Opt.ParenthesisHighlight.Group,
			&drawer4.ColorizeGroup{}, // 3=selection
			&drawer4.ColorizeGroup{}, // 4=flash
		}
	}

	return te
}

//----------

func (te *TextEditX) PaintBase() {
	te.TextEdit.PaintBase()
	te.iterateFlash()
	te.paintFlashLineBg()
}

func (te *TextEditX) Paint() {
	te.updateSelectionOpt()
	te.updateHighlightWordOpt()
	te.updateFlashOpt()
	te.updateParenthesisOpt()
	te.TextEdit.Paint()
}

//----------

func (te *TextEditX) updateSelectionOpt() {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		if !d.Segments.On() {
			return
		}
		sg := d.Segments.Opt.Groups[0]
		if te.TextCursor.SelectionOn() {
			sg.On = true
			s, e := te.TextCursor.SelectionIndexes()
			seg := &drawer3.Segment{s, e}
			sg.Segs = []*drawer3.Segment{seg}
		} else {
			sg.On = false
			sg.Segs = nil
		}
	}
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
				&drawer4.ColorizeOp{Offset: s, Fg: fg, Bg: bg},
				&drawer4.ColorizeOp{Offset: e},
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
			i0, i1 := te.lineIndexes(index)
			index = i0
			len = i1 - index
		}

		// flash index (accurate runes)
		te.flash.index.on = true
		te.flash.index.index = index
		te.flash.index.len = len

		// flash line bg
		if line {
			te.flash.line.on = true
			te.flash.line.p1 = te.Drawer.PointOf(index)
			te.flash.line.p2 = te.Drawer.PointOf(index + len)
		}

		te.MarkNeedsPaint()
	})
}

func (te *TextEditX) lineIndexes(index int) (int, int) {
	i0, i1 := 0, 0
	if index <= len(te.Str()) {
		i0 = parseutil.LineStartIndex(te.Str(), index)
		u, nl := parseutil.LineEndIndexNextIndex(te.Str(), index)
		if nl {
			u--
		}
		i1 = u
	}
	return i0, i1
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

func (te *TextEditX) paintFlashLineBg() {
	if !te.flash.line.on {
		return
	}

	// rectangle to paint
	y1 := te.flash.line.p1.Y - te.Offset().Y
	y2 := te.flash.line.p2.Y - te.Offset().Y
	r := te.Bounds
	r.Min.Y += y1
	r.Max.Y = r.Min.Y + (y2 - y1) + te.LineHeight()
	r = r.Intersect(te.Bounds)

	// tint percentage
	t := te.flash.now.Sub(te.flash.start)
	perc := 1.0 - (float64(t) / float64(te.flash.dur))

	// paint
	img := te.ctx.Image()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := img.At(x, y)
			c2 := imageutil.TintOrShade(c, perc)
			img.Set(x, y, c2)
		}
	}
}

func (te *TextEditX) updateFlashOpt() {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		te.updateFlashOpt3(d)
	}
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		te.updateFlashOpt4(d)
	}
}

func (te *TextEditX) updateFlashOpt3(d *drawer3.PosDrawer) {
	if !d.Segments.On() {
		return
	}

	sg := d.Segments.Opt.Groups[3]
	if !te.flash.index.on {
		sg.On = false
		sg.Segs = nil
		return
	}

	sg.On = true
	sg.Segs = []*drawer3.Segment{
		{
			Pos: te.flash.index.index,
			End: te.flash.index.index + te.flash.index.len,
		},
	}

	// tint percentage
	t := te.flash.now.Sub(te.flash.start)
	perc := 1.0 - (float64(t) / float64(te.flash.dur))

	bg3 := te.TreeThemePaletteColor("text_bg")

	// set process color function
	sg.ProcColor = func(fg, bg color.Color) (_, _ color.Color) {
		fg2 := imageutil.TintOrShade(fg, perc)
		if bg == nil {
			bg = bg3
		}
		bg2 := imageutil.TintOrShade(bg, perc)
		return fg2, bg2
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
	g.Ops = []*drawer4.ColorizeOp{
		&drawer4.ColorizeOp{Offset: s, ProcColor: pc},
		&drawer4.ColorizeOp{Offset: e},
	}
}

//----------

func (te *TextEditX) EnableParenthesisMatch(v bool) {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		sg := d.Segments.Opt.Groups[2]
		sg.On = v
	}
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		d.Opt.ParenthesisHighlight.On = v
	}

}

func (te *TextEditX) updateParenthesisOpt() {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		te.updateParenthesisOpt3(d)
	}
}

func (te *TextEditX) updateParenthesisOpt3(d *drawer3.PosDrawer) {
	if !d.Segments.On() {
		return
	}

	sg := d.Segments.Opt.Groups[2]
	sg.Segs = nil // might find segments or not, always start with nil
	if !sg.On {
		return
	}

	tc := te.TextCursor

	// read current rune
	ci := tc.Index()
	cru, _, err := tc.RW().ReadRuneAt(ci)
	if err != nil {
		return
	}

	// find parenthesis type
	pars := []rune{
		'{', '}',
		'(', ')',
		'[', ']',
	}
	var pi int
	for ; pi < len(pars); pi++ {
		if pars[pi] == cru {
			break
		}
	}
	if pi >= len(pars) {
		return
	}

	// assign open/close parenthesis
	var open, close rune
	isOpen := pi%2 == 0
	if isOpen {
		open, close = pars[pi], pars[pi+1]
	} else {
		open, close = pars[pi-1], pars[pi]
	}

	if isOpen {
		te.findParenthesisClose(ci, cru, open, close, sg)
	} else {
		te.findParenthesisOpen(ci, cru, open, close, sg)
	}
}

func (te *TextEditX) findParenthesisClose(ci int, cru, open, close rune, sg *drawer3.SegGroup) {
	tc := te.TextCursor
	earlyExitIndex := te.visibleBottomIndex()

	seg1 := &drawer3.Segment{ci, ci + len(string(open))}
	sg.Segs = append(sg.Segs, seg1)

	c := 0                     // match count
	i := ci + len(string(cru)) // start searching on next rune
	for {
		if i >= earlyExitIndex {
			return
		}

		ru, size, err := tc.RW().ReadRuneAt(i)
		if err != nil {
			return
		}

		if ru == open {
			c++
		}
		if ru == close {
			if c > 0 {
				c--
			} else {
				seg2 := &drawer3.Segment{i, i + len(string(close))}
				sg.Segs = append(sg.Segs, seg2)
				return
			}
		}

		i += size
	}
}

func (te *TextEditX) findParenthesisOpen(ci int, cru, open, close rune, sg *drawer3.SegGroup) {
	tc := te.TextCursor
	earlyExitIndex := te.visibleTopIndex()

	seg2 := &drawer3.Segment{ci, ci + len(string(close))}
	sg.Segs = append(sg.Segs, seg2)

	c := 0 // match count
	for i := ci; ; {
		if i < earlyExitIndex {
			return
		}

		ru, size, err := tc.RW().ReadLastRuneAt(i)
		if err != nil {
			return
		}
		i -= size

		if ru == close {
			c++
		}
		if ru == open {
			if c > 0 {
				c--
			} else {
				seg1 := &drawer3.Segment{i, i + len(string(open))}
				// prepend
				sg.Segs = append([]*drawer3.Segment{seg1}, sg.Segs...)
				return
			}
		}

	}
}

//----------

func (te *TextEditX) EnableWrapLines(v bool) {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		d.WrapLine.SetOn(v)
	}
}

//----------

func (te *TextEditX) EnableColorizeSyntax(v bool) {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		d.ColorizeSyntax.SetOn(v)
	}
}

//----------

func (te *TextEditX) EnableHighlightCursorWord(v bool) {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		sg := d.Segments.Opt.Groups[1]
		sg.On = v
	}
	if d, ok := te.Drawer.(*drawer4.Drawer); ok {
		d.Opt.WordHighlight.On = v
		if !v {
			g := d.Opt.Colorize.Groups[1]
			g.Ops = nil
		}
	}
}

func (te *TextEditX) updateHighlightWordOpt() {
	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		te.updateHighlightWordOpt3(d)
	}
}

func (te *TextEditX) updateHighlightWordOpt3(d *drawer3.PosDrawer) {
	sg := d.Segments.Opt.Groups[1]
	sg.Segs = nil
	if !sg.On {
		return
	}

	tc := te.TextCursor

	if tc.SelectionOn() {
		return
	}

	word, _, err := iout.WordAtIndex(tc.RW(), tc.Index(), 100)
	if err != nil {
		return
	}

	// indexes of visible text
	a, b := te.visibleTopIndex(), te.visibleBottomIndex()
	a -= len(word)
	b += len(word)
	if a < 0 {
		a = 0
	}
	l := tc.RW().Len()
	if b > l {
		b = l
	}

	// search segments
	for i := a; i < b; {
		// find word
		j, err := iout.Index(tc.RW(), i, b-i, word, false)
		if err != nil {
			return
		}
		if j < 0 {
			break
		}

		// isolated word
		if iout.WordIsolated(tc.RW(), j, len(word)) {
			seg := &drawer3.Segment{j, j + len(word)}
			sg.Segs = append(sg.Segs, seg)
		}

		i = j + len(word)
	}
}

//----------

func (te *TextEditX) SetCommentStrings(line string, enclosed [2]string) {
	te.commentLineStr = line // keep for comment/uncomment lines shortcut

	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		d.ColorizeSyntax.Opt.Comment.Line = line
		d.ColorizeSyntax.Opt.Comment.Enclosed = enclosed
	}
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

	if d, ok := te.Drawer.(*drawer3.PosDrawer); ok {
		d.Cursor.Opt.Fg = pcol("text_cursor_fg")

		// selection
		sg := d.Segments.Opt.Groups[0]
		sg.Fg = pcol("text_selection_fg")
		sg.Bg = pcol("text_selection_bg")

		// word
		sg = d.Segments.Opt.Groups[1]
		sg.Fg = pcol("text_highlightword_fg")
		sg.Bg = pcol("text_highlightword_bg")

		// parenthesis
		sg = d.Segments.Opt.Groups[2]
		sg.Fg = pcol("text_parenthesis_fg")
		sg.Bg = pcol("text_parenthesis_bg")

		d.WrapLine.Opt.Fg = pcol("text_wrapline_fg")
		d.WrapLine.Opt.Bg = pcol("text_wrapline_bg")

		d.ColorizeSyntax.Opt.String.Fg = pcol("text_colorize_string_fg")
		d.ColorizeSyntax.Opt.Comment.Fg = pcol("text_colorize_comments_fg")

		d.Annotations.Opt.Fg = pcol("text_annotations_fg")
		d.Annotations.Opt.Bg = pcol("text_annotations_bg")
		d.Annotations.Opt.Select.Fg = pcol("text_annotations_select_fg")
		d.Annotations.Opt.Select.Bg = pcol("text_annotations_select_bg")
	}

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

//----------

func (te *TextEditX) visibleTopIndex() int {
	return te.Drawer.IndexOf(te.Offset())
}
func (te *TextEditX) visibleBottomIndex() int {
	// TODO: needs improvement for lines with big X

	// first rune of line after last visible line
	y := te.Offset().Y + te.Bounds.Size().Y + te.LineHeight()

	return te.Drawer.IndexOf(image.Point{0, y})
}
