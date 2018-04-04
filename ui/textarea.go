package ui

import (
	"image"
	"time"
	"unicode"

	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/ui/tautil/tahistory"
	"github.com/jmigpin/editor/util/drawutil/hsdrawer"
	"github.com/jmigpin/editor/util/drawutil/loopers"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type TextArea struct {
	widget.EmbedNode
	EvReg               *evreg.Register
	HighlightCursorWord bool
	FlexibleParent      widget.Node // for dynamic text areas that change size

	Drawer  *hsdrawer.HSDrawer
	History *tahistory.History

	commentStr         string
	commentStrEnclosed [2]string // start, end

	ui            *UI
	scroller      widget.Scroller
	defaultCursor widget.Cursor

	edit           *tahistory.Edit
	editStr        string
	editOpenCursor int

	str         string
	cursorIndex int
	offset      image.Point
	selection   struct {
		on    bool
		index int // from index to cursorIndex
	}

	flashLine struct {
		on    bool
		start time.Time
		p1    image.Point
		p2    image.Point
	}
	flashIndex struct {
		on         bool
		start      time.Time
		index, len int
	}
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}

	ta.Drawer = &hsdrawer.HSDrawer{}
	ta.EvReg = evreg.NewRegister()
	ta.History = tahistory.NewHistory(128)

	ta.defaultCursor = widget.NoneCursor
	ta.Cursor = ta.defaultCursor

	return ta
}

func (ta *TextArea) Measure(hint image.Point) image.Point {
	d := ta.Drawer
	d.Args.Face = ta.Theme.Font().Face(nil)
	d.Args.WrapLineOpt = ta.wrapLineOpt()
	d.Args.ColorizeOpt = ta.colorizeOpt()
	d.Args.AnnotationsOpt = ta.annotationsOpt()

	//// TESTING
	//d.AnnotationsOpt = &loopers.AnnotationsOpt{
	//	Fg: color.White,
	//	Bg: color.RGBA{100, 0, 0, 255},
	//}
	//for i := 0; i < 1000; i++ {
	//	u := &d.AnnotationsOpt.OrderedEntries
	//	*u = append(*u, &loopers.AnnotationsEntry{i * 10, fmt.Sprintf("entry %v", i)})
	//}

	// keep offset for restoration of accurate offset index
	offsetIndex := 0
	indexOffsetY := 0
	neededMeasure := d.NeedMeasure(hint.X)
	if neededMeasure {
		offsetIndex = ta.OffsetIndex()
		p := ta.Drawer.GetPoint(offsetIndex)
		indexOffsetY = ta.offset.Y - p.Y
	}

	m := d.Measure(hint)

	// restore offset to keep the same first line while resizing
	if neededMeasure {
		p := ta.Drawer.GetPoint(offsetIndex)
		p.Y += indexOffsetY
		ta.SetOffsetY(p.Y)
	}

	return m
}

func (ta *TextArea) CalcChildsBounds() {
	_ = ta.Measure(ta.Bounds.Size())
	ta.EmbedNode.CalcChildsBounds()
	ta.updateScroller()
}

func (ta *TextArea) Paint() {
	// nothing to do
	if ta.Bounds.Dx() == 0 {
		// TODO: improve possible boxlayout requesting a paint with zero at init
		//debug.PrintStack()
		return
	}

	bounds := ta.Bounds

	pal := ta.Theme.Palette()

	// fill background
	imageutil.FillRectangle(ta.ui.Image(), &bounds, pal.Get("bg"))
	ta.paintFlashLineBg()

	d := ta.Drawer
	d.CursorIndex = &ta.cursorIndex
	d.Offset = ta.offset
	d.Fg = pal.Get("fg")
	d.SelectionOpt = ta.selectionOpt()
	d.FlashSelectionOpt = ta.flashSelectionOpt()
	d.HighlightWordOpt = ta.highlightWordOpt()

	d.Draw(ta.ui.Image(), &bounds)
}

func (ta *TextArea) annotationsOpt() *loopers.AnnotationsOpt {
	opt := ta.Drawer.Args.AnnotationsOpt // reuse drawer instance to avoid recalc
	if opt == nil {
		return nil
	}
	pal := ta.Theme.Palette()
	opt.Fg = pal.Get("annotations_fg")
	opt.Bg = pal.Get("annotations_bg")
	return opt
}
func (ta *TextArea) SetAnnotationsOrderedEntries(entries []*loopers.AnnotationsEntry) {
	// create new instance on drawer to use new entries
	ta.Drawer.Args.AnnotationsOpt = &loopers.AnnotationsOpt{OrderedEntries: entries}
}

func (ta *TextArea) colorizeOpt() *loopers.ColorizeOpt {
	// don't colorize if the comments are not set
	if ta.commentStr == "" && ta.commentStrEnclosed == [2]string{} {
		return nil
	}

	opt := ta.Drawer.Args.ColorizeOpt // reuse drawer instance to avoid recalc
	if opt == nil {
		opt = &loopers.ColorizeOpt{}
	}
	opt.Comment.Fg = UITheme.GetTextAreaCommentsFg()
	return opt
}

func (ta *TextArea) SetCommentStrings(cstr string, cstre [2]string) {
	ta.commentStr = cstr
	ta.commentStrEnclosed = cstre
	// create new instance to use the new settings
	ta.Drawer.Args.ColorizeOpt = &loopers.ColorizeOpt{}
	ta.Drawer.Args.ColorizeOpt.Comment.Line = ta.commentStr
	ta.Drawer.Args.ColorizeOpt.Comment.Enclosed = ta.commentStrEnclosed
}
func (ta *TextArea) CommentString() string {
	return ta.commentStr
}

func (ta *TextArea) wrapLineOpt() *loopers.WrapLineOpt {
	fg, bg := UITheme.NoSelectionColors(ta.Theme)
	opt := ta.Drawer.Args.WrapLineOpt // reuse drawer instance to avoid recalc
	if opt == nil {
		opt = &loopers.WrapLineOpt{}
	}
	opt.Fg = fg
	opt.Bg = bg
	return opt
}

func (ta *TextArea) highlightWordOpt() *loopers.HighlightWordOpt {
	if !ta.HighlightCursorWord {
		return nil
	}
	// don't highlight word if selection is on
	if ta.SelectionOn() {
		return nil
	}

	pal := ta.Theme.Palette()
	return &loopers.HighlightWordOpt{
		Index: ta.cursorIndex,
		Fg:    pal.Get("highlight_fg"),
		Bg:    pal.Get("highlight_bg"),
	}
}
func (ta *TextArea) selectionOpt() *loopers.SelectionOpt {
	if ta.SelectionOn() {
		pal := ta.Theme.Palette()
		return &loopers.SelectionOpt{
			Fg:    pal.Get("selection_fg"),
			Bg:    pal.Get("selection_bg"),
			Start: ta.SelectionIndex(),
			End:   ta.CursorIndex(),
		}
	}
	return nil
}

func (ta *TextArea) paintFlashLineBg() {
	if !ta.flashLine.on {
		return
	}

	now := time.Now()
	dur := FlashDuration
	end := ta.flashLine.start.Add(dur)

	// animation time ended
	if now.After(end) {
		ta.flashLine.on = false
		return
	}

	// rectangle to paint
	y1 := ta.flashLine.p1.Y - ta.OffsetY()
	y2 := ta.flashLine.p2.Y - ta.OffsetY()
	r := ta.Bounds
	r.Min.Y += y1
	r.Max.Y = r.Min.Y + (y2 - y1) + ta.LineHeight()
	r = r.Intersect(ta.Bounds)

	// tint percentage
	t := now.Sub(ta.flashLine.start)
	perc := 1.0 - (float64(t) / float64(dur))

	// paint
	img := ta.ui.Image()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := img.At(x, y)
			c2 := imageutil.TintOrShade(c, perc)
			img.Set(x, y, c2)
		}
	}

	// need to keep painting while flashing
	ta.ui.RunOnUIGoRoutine(func() {
		ta.MarkNeedsPaint()
	})
}

func (ta *TextArea) flashSelectionOpt() *loopers.FlashSelectionOpt {
	if !ta.flashIndex.on {
		return nil
	}

	now := time.Now()
	dur := FlashDuration
	end := ta.flashIndex.start.Add(dur)

	// animation time ended
	if now.After(end) {
		ta.flashIndex.on = false
		return nil
	}

	// tint percentage
	t := now.Sub(ta.flashIndex.start)
	perc := 1.0 - (float64(t) / float64(dur))

	fsi := &loopers.FlashSelectionOpt{
		Perc:  perc,
		Start: ta.flashIndex.index,
		End:   ta.flashIndex.index + ta.flashIndex.len,
		Bg:    ta.Theme.Palette().Get("bg"),
	}

	// need to keep painting while flashing
	ta.ui.RunOnUIGoRoutine(func() {
		ta.MarkNeedsPaint()
	})

	return fsi
}

// Safe to use concurrently. Handles flashing the line independently of the number of runes that it contain, even if zero.
func (ta *TextArea) FlashIndexLine(index int) {
	ta.ui.RunOnUIGoRoutine(func() {
		// start/end line indexes
		i0, i1 := 0, 0
		al := 0
		if index < len(ta.Str()) {
			i0 = tautil.LineStartIndex(ta.Str(), index)
			u, nl := tautil.LineEndIndexNextIndex(ta.Str(), index)
			if nl {
				u--
				// include newline index to flash annotations if present (they stay on newline index) but don't include the next line for flash (not added to "l").
				al = 1
			}
			i1 = u
		}

		// flash index (accurate runes)
		ta.flashIndex.on = true
		ta.flashIndex.start = time.Now()
		ta.flashIndex.index = i0
		ta.flashIndex.len = i1 - i0 + al

		// flash line bg
		ta.flashLine.on = true
		ta.flashLine.start = ta.flashIndex.start
		ta.flashLine.p1 = ta.Drawer.GetPoint(i0)
		ta.flashLine.p2 = ta.Drawer.GetPoint(i1)

		ta.MarkNeedsPaint()
	})
}

// Safe to use concurrently. Handles segments that span over more then one line.
func (ta *TextArea) FlashIndexLen(index, len int) {
	ta.ui.RunOnUIGoRoutine(func() {
		ta.flashIndex.on = true
		ta.flashIndex.start = time.Now()
		ta.flashIndex.index = index
		ta.flashIndex.len = len
		ta.MarkNeedsPaint()
	})
}

func (ta *TextArea) Str() string {
	if ta.edit != nil {
		// return edit str while editing
		return ta.editStr
	}
	return ta.str
}

func (ta *TextArea) setStr(s string) {
	if s == ta.str {
		return
	}
	ta.str = s
	ta.Drawer.Args.Str = ta.str

	// ensure valid indexes
	ta.SetCursorIndex(ta.CursorIndex())
	ta.SetSelectionIndex(ta.SelectionIndex())

	// calc bounds from flexibleparent
	if ta.FlexibleParent != nil {
		oldBounds := ta.Bounds

		// should trigger a call to ta.CalcChildsBounds
		ta.FlexibleParent.CalcChildsBounds()
		ta.FlexibleParent.Embed().MarkNeedsPaint()

		// Keep pointer inside if it was in before.
		// Need to test if it was in before to avoid warping on all changes.
		// Useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus.
		p, err := ta.ui.QueryPointer()
		if err == nil && p.In(oldBounds) && !p.In(ta.Bounds) {
			ta.ui.WarpPointerToRectanglePad(&ta.Bounds)
		}
	} else {
		ta.CalcChildsBounds()
		ta.MarkNeedsPaint()
	}

	ev := &TextAreaSetStrEvent{ta}
	ta.EvReg.RunCallbacks(TextAreaSetStrEventId, ev)
}

// TODO: have a set str, and a clear func
func (ta *TextArea) SetStrClear(str string, clearPosition, clearUndoQ bool) {
	ta.SetSelectionOff()
	if clearPosition {
		ta.SetCursorIndex(0)
		ta.SetOffsetY(0)
	}
	if clearUndoQ {
		ta.History.Clear()
		ta.setStr(str)
	} else {
		// replace string with edit to allow undo
		ta.EditOpen()
		ta.EditDelete(0, len(ta.Str()))
		ta.EditInsert(0, str)
		ta.EditCloseAfterSetCursor()
	}
}

func (ta *TextArea) EditOpen() {
	if ta.edit != nil {
		panic("edit already exists")
	}
	ta.edit = &tahistory.Edit{}
	ta.editStr = ta.str
	ta.editOpenCursor = ta.CursorIndex()
}
func (ta *TextArea) EditInsert(index int, str string) {
	ta.editStr = ta.edit.Insert(ta.editStr, index, str)
}
func (ta *TextArea) EditDelete(index, index2 int) {
	ta.editStr = ta.edit.Delete(ta.editStr, index, index2)
}
func (ta *TextArea) EditCloseAfterSetCursor() {
	cleanup := func() {
		ta.edit = nil
		ta.editStr = ""
		ta.editOpenCursor = 0
	}

	if ta.editStr == ta.str {
		cleanup()
		return
	}

	c1 := ta.editOpenCursor
	c2 := ta.CursorIndex()
	ta.edit.SetOpenCloseCursors(c1, c2)
	ta.History.PushEdit(ta.edit)
	tahistory.TryToMergeLastTwoEdits(ta.History)

	u := ta.editStr
	cleanup()
	ta.setStr(u)
}

func (ta *TextArea) undo() {
	s, i, ok := ta.History.Undo(ta.Str())
	if !ok {
		return
	}
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOff()
}
func (ta *TextArea) redo() {
	s, i, ok := ta.History.Redo(ta.Str())
	if !ok {
		return
	}
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOff()
}

func (ta *TextArea) CursorIndex() int {
	return ta.cursorIndex
}
func (ta *TextArea) SetCursorIndex(v int) {
	v = ta.validIndex(v)
	if v != ta.cursorIndex {
		ta.cursorIndex = v
		ta.validateSelection()
		ta.MarkNeedsPaint()
	}
}
func (ta *TextArea) SelectionIndex() int {
	return ta.selection.index
}
func (ta *TextArea) SetSelectionIndex(v int) {
	v = ta.validIndex(v)
	if v != ta.selection.index {
		ta.selection.index = v
		ta.validateSelection()
		ta.MarkNeedsPaint()
	}
}
func (ta *TextArea) SetSelection(si, ci int) {
	ta.SetSelectionIndex(si)
	ta.SetCursorIndex(ci)
	ta.setSelectionOn(ta.somethingSelected())
}

func (ta *TextArea) SelectionOn() bool {
	return ta.selection.on && ta.somethingSelected()
}
func (ta *TextArea) SetSelectionOff() {
	ta.setSelectionOn(false)
}
func (ta *TextArea) setSelectionOn(v bool) {
	if v != ta.selection.on {
		ta.selection.on = v
		ta.MarkNeedsPaint()
	}
}

func (ta *TextArea) validIndex(v int) int {
	if v < 0 {
		v = 0
	} else if v > len(ta.Str()) {
		v = len(ta.Str())
	}
	return v
}
func (ta *TextArea) validateSelection() {
	if !ta.somethingSelected() {
		ta.SetSelectionOff()
	}
}
func (ta *TextArea) somethingSelected() bool {
	si := ta.SelectionIndex()
	ci := ta.CursorIndex()
	return si != ci
}

func (ta *TextArea) OffsetY() int {
	return ta.offset.Y
}
func (ta *TextArea) SetOffsetY(v int) {
	ta._setOffset(image.Point{ta.offset.X, v})
	ta.updateScroller()
}
func (ta *TextArea) _setOffset(o image.Point) {
	// must have a scroller to change the offset
	if ta.scroller == nil {
		return
	}
	o = imageutil.MaxPoint(o, image.Point{0, 0})
	o = imageutil.MinPoint(o, ta.Drawer.MeasurementFullY())
	if o != ta.offset {
		ta.offset = o
		ta.MarkNeedsPaint()
	}
}

func (ta *TextArea) updateScroller() {
	if ta.scroller != nil {
		ta.scroller.SetScrollerOffset(ta.offset)
	}
}

func (ta *TextArea) OffsetIndex() int {
	return ta.Drawer.GetIndex(&ta.offset)
}
func (ta *TextArea) SetOffsetIndex(i int) {
	p := ta.Drawer.GetPoint(i)
	ta.SetOffsetY(p.Y)
}

func (ta *TextArea) MakeCursorVisible() {
	ta.MakeIndexVisible(ta.CursorIndex())
}
func (ta *TextArea) MakeIndexVisible(index int) {
	y0 := ta.OffsetY()
	y1 := y0 + ta.Bounds.Dy()

	// is all visible
	a0 := ta.Drawer.GetPoint(index).Y
	a1 := a0 + ta.LineHeight()
	if a0 >= y0 && a1 <= y1 {
		return
	}

	// is partially visible
	if y0 >= a0 && y0 <= a1 {
		// partially visible at top
		ta.SetOffsetY(a0)
		return
	}
	if y1 >= a0 && y1 <= a1 {
		// partially visible at bottom
		sy := ta.Bounds.Dy()
		ta.SetOffsetY(a0 - sy + ta.LineHeight())
		return
	}

	// set at half bounds
	half := ta.Bounds.Dy() / 2
	ta.SetOffsetY(a0 - half)
}

func (ta *TextArea) IndexIsVisible(index int) bool {
	y0 := ta.OffsetY()
	y1 := y0 + ta.Bounds.Dy()

	// is all visible
	a0 := ta.Drawer.GetPoint(index).Y
	a1 := a0 + ta.LineHeight()
	if a0 >= y0 && a1 <= y1 {
		return true
	}
	return false
}

func (ta *TextArea) MakeIndexVisibleAtCenter(index int) {
	//// set at half bounds
	//p0 := ta.Drawer.GetPoint(index).Y
	//half := (ta.Bounds.Dy() - ta.LineHeight()) / 2
	//offsetY := p0 - half
	//ta.SetOffsetY(offsetY)

	ta.MakeIndexVisible(index)
}

func (ta *TextArea) GetCPPaste(i event.CopyPasteIndex) (string, error) {
	return ta.ui.GetCPPaste(i)
}
func (ta *TextArea) SetCPCopy(i event.CopyPasteIndex, v string) error {
	return ta.ui.SetCPCopy(i, v)
}

func (ta *TextArea) LineHeight() int {
	return ta.Drawer.LineHeight()
}

func (ta *TextArea) GetPoint(i int) image.Point {
	return ta.Drawer.GetPoint(i)
}
func (ta *TextArea) GetIndex(p *image.Point) int {
	return ta.Drawer.GetIndex(p)
}

func (ta *TextArea) IndexPoint(i int) image.Point {
	p := ta.GetPoint(i)
	p.Y -= ta.OffsetY()
	return p.Add(ta.Bounds.Min)
}
func (ta *TextArea) PointIndex(p *image.Point) int {
	p2 := p.Sub(ta.Bounds.Min)
	p2.Y += ta.OffsetY()
	return ta.GetIndex(&p2)
}

func (ta *TextArea) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch ev := ev0.(type) {
	case *event.KeyDown:
		ta.onKeyDown(ev)
	case *event.MouseDown:

		// annotations
		switch ev.Button {
		case event.ButtonWheelUp, event.ButtonWheelDown:
			if ev.Modifiers.HasAny(event.ModControl) {
				if ta.annotationMouseDown(ev, true) {
					return true
				}
			}
		default:
			if ta.annotationMouseDown(ev, false) {
				return true
			}
		}

		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = widget.PointerCursor
		case event.ButtonLeft:
			if ev.Modifiers.Is(event.ModShift) {
				tautil.MoveCursorToPoint(ta, &ev.Point, true)
			} else {
				tautil.MoveCursorToPoint(ta, &ev.Point, false)
			}
		}
	case *event.MouseUp:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = ta.defaultCursor
		}
	case *event.MouseDragStart:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = ta.defaultCursor
		}
	case *event.MouseDragMove:
		if ev.Buttons.Has(event.ButtonLeft) {
			tautil.MoveCursorToPoint(ta, &ev.Point, true)
			ta.MakeCursorVisible()
		}
	case *event.MouseDragEnd:
		switch ev.Button {
		case event.ButtonLeft:
			tautil.MoveCursorToPoint(ta, &ev.Point, true)
		}
	case *event.MouseClick:
		return ta.onMouseClick(ev)
	case *event.MouseDoubleClick:
		return ta.onMouseDoubleClick(ev)
	case *event.MouseTripleClick:
		return ta.onMouseTripleClick(ev)
	}

	return false
}

func (ta *TextArea) onMouseClick(ev *event.MouseClick) bool {
	switch ev.Button {
	case event.ButtonRight:
		if !ta.PointIndexInsideSelection(&ev.Point) {
			tautil.MoveCursorToPoint(ta, &ev.Point, false)
		}
		i := ta.PointIndex(&ev.Point)
		ev2 := &TextAreaCmdEvent{ta, i}
		ta.EvReg.RunCallbacks(TextAreaCmdEventId, ev2)
		return true
	case event.ButtonMiddle:
		tautil.MoveCursorToPoint(ta, &ev.Point, false)
		tautil.Paste(ta, event.PrimaryCPI)
		return true
	}
	return false
}
func (ta *TextArea) onMouseDoubleClick(ev *event.MouseDoubleClick) bool {
	switch ev.Button {
	case event.ButtonLeft:
		tautil.MoveCursorToPoint(ta, &ev.Point, false)
		tautil.SelectWord(ta)
		return true
	}
	return false
}
func (ta *TextArea) onMouseTripleClick(ev *event.MouseTripleClick) bool {
	switch ev.Button {
	case event.ButtonLeft:
		tautil.MoveCursorToPoint(ta, &ev.Point, false)
		tautil.SelectLine(ta)
		return true
	}
	return false
}

func (ta *TextArea) annotationMouseDown(ev *event.MouseDown, force bool) bool {
	// annotations index
	p2 := ev.Point.Sub(ta.Bounds.Min)
	p2.Y += ta.OffsetY()
	ai, aio, ok := ta.Drawer.GetAnnotationsIndex(&p2)

	// still send event with index -1
	if !ok && force {
		ai, aio, ok = -1, -1, true
	}

	if ok {
		ev3 := &TextAreaAnnotationClickEvent{ta, ai, aio, ev.Button}
		ta.EvReg.RunCallbacks(TextAreaAnnotationClickEventId, ev3)
		return true
	}
	return false
}

func (ta *TextArea) PointIndexInsideSelection(p *image.Point) bool {
	if ta.SelectionOn() {
		i := ta.PointIndex(p)
		s, e := tautil.SelectionStringIndexes(ta)
		return i >= s && i < e
	}
	return false
}

func (ta *TextArea) onKeyDown(ev *event.KeyDown) {
	switch ev.Code {
	case event.KCodeAltL,
		event.KCodeAltGr,
		event.KCodeShiftL,
		event.KCodeShiftR,
		event.KCodeControlL,
		event.KCodeControlR,
		event.KCodeCapsLock,
		event.KCodeNumLock,
		event.KCodeInsert,
		event.KCodePageUp,
		event.KCodePageDown,
		event.KCodeSuperL: // windows key
		// ignore these
	default:
		ta.onKeyDown2(ev)
	}
}
func (ta *TextArea) onKeyDown2(ev *event.KeyDown) {
	//defer ta.MakeCursorVisible()
	//log.Printf("%+v", ev)

	switch ev.Code {
	case event.KCodeRight:
		ta.MakeCursorVisible() // before and after
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift):
			tautil.MoveCursorJumpRight(ta, true)
		case ev.Modifiers.Is(event.ModControl):
			tautil.MoveCursorJumpRight(ta, false)
		case ev.Modifiers.Is(event.ModShift):
			tautil.MoveCursorRight(ta, true)
		default:
			tautil.MoveCursorRight(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeLeft:
		ta.MakeCursorVisible() // before and after
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift):
			tautil.MoveCursorJumpLeft(ta, true)
		case ev.Modifiers.Is(event.ModControl):
			tautil.MoveCursorJumpLeft(ta, false)
		case ev.Modifiers.Is(event.ModShift):
			tautil.MoveCursorLeft(ta, true)
		default:
			tautil.MoveCursorLeft(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeUp:
		ta.MakeCursorVisible() // before and after
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModAlt):
			tautil.MoveLineUp(ta)
		case ev.Modifiers.HasAny(event.ModShift):
			tautil.MoveCursorUp(ta, true)
		default:
			tautil.MoveCursorUp(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeDown:
		ta.MakeCursorVisible() // before and after
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift | event.ModAlt):
			tautil.DuplicateLines(ta)
		case ev.Modifiers.Is(event.ModControl | event.ModAlt):
			tautil.MoveLineDown(ta)
		case ev.Modifiers.HasAny(event.ModShift):
			tautil.MoveCursorDown(ta, true)
		default:
			tautil.MoveCursorDown(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeHome:
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift):
			tautil.StartOfString(ta, true)
		case ev.Modifiers.Is(event.ModControl):
			tautil.StartOfString(ta, false)
		case ev.Modifiers.Is(event.ModShift):
			tautil.StartOfLine(ta, true)
		default:
			tautil.StartOfLine(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeEnd:
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift):
			tautil.EndOfString(ta, true)
		case ev.Modifiers.Is(event.ModControl):
			tautil.EndOfString(ta, false)
		case ev.Modifiers.Is(event.ModShift):
			tautil.EndOfLine(ta, true)
		default:
			tautil.EndOfLine(ta, false)
		}
		ta.MakeCursorVisible()
	case event.KCodeBackspace:
		tautil.Backspace(ta)
		ta.MakeCursorVisible()
	case event.KCodeDelete:
		tautil.Delete(ta)
	case event.KCodeReturn:
		tautil.AutoIndent(ta)
		ta.MakeCursorVisible()
	case event.KCodeTab:
		switch {
		case ev.Modifiers.Is(event.ModShift):
			tautil.TabLeft(ta)
		default:
			tautil.TabRight(ta)
		}
		ta.MakeCursorVisible()
	case ' ':
		// ensure space even if modifiers are present
		tautil.InsertString(ta, " ")
		ta.MakeCursorVisible()
	default:
		// shortcuts with printable runes - also avoids non-defined shortcuts to get a rune printed
		switch {
		case ev.Modifiers.Is(event.ModControl | event.ModShift):
			switch ev.Code {
			case 'd':
				tautil.Uncomment(ta)
			case 'z':
				ta.redo()
				ta.MakeCursorVisible()
			}
		case ev.Modifiers.Is(event.ModControl):
			switch ev.Code {
			case 'd':
				tautil.Comment(ta)
			case 'c':
				tautil.Copy(ta)
			case 'x':
				tautil.Cut(ta)
			case 'v':
				tautil.Paste(ta, event.ClipboardCPI)
			case 'k':
				tautil.RemoveLines(ta)
			case 'a':
				tautil.SelectAll(ta)
			case 'z':
				ta.undo()
				ta.MakeCursorVisible()
			}
		case ev.Code >= event.KCodeF1 && ev.Code <= event.KCodeF12,
			ev.Code == event.KCodeEscape:
			// do nothing
		case !unicode.IsPrint(ev.Rune):
			// do nothing
		default:
			tautil.InsertString(ta, string(ev.Rune))
			ta.MakeCursorVisible()
		}
	}
}

func (ta *TextArea) InsertStringAsync(str string) {
	ta.ui.RunOnUIGoRoutine(func() {
		tautil.InsertString(ta, str)
	})
}

func (ta *TextArea) GetBounds() image.Rectangle {
	return ta.Bounds
}

func (ta *TextArea) Error(err error) {
	ta.ui.OnError(err)
}

// Implement widget.Scrollable
func (ta *TextArea) SetScroller(scroller widget.Scroller) {
	ta.scroller = scroller
}

// Implement widget.Scrollable
func (ta *TextArea) SetScrollableOffset(p image.Point) {
	ta._setOffset(p)
}

// Implement widget.Scrollable
func (ta *TextArea) ScrollableSize() image.Point {
	// extra height allows to scroll past the str height
	visible := 2 * ta.LineHeight() // keep n lines visible at the end
	extra := ta.Bounds.Dy() - visible

	//y := ta.StrHeight() + extra
	//return image.Point{ta.Bounds.Dx(), y}

	m := ta.Drawer.MeasurementFullY()
	m.Y += extra
	return m
}

// Implement widget.Scrollable
func (ta *TextArea) ScrollablePagingMargin() int {
	return ta.LineHeight() * 1
}

// Implement widget.Scrollable
func (ta *TextArea) ScrollableScrollJump() int {
	return ta.LineHeight() * 4
}

const (
	TextAreaSetStrEventId = iota
	TextAreaCmdEventId
	TextAreaAnnotationClickEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
	Index    int
}
type TextAreaSetStrEvent struct {
	TextArea *TextArea
}
type TextAreaAnnotationClickEvent struct {
	TextArea    *TextArea
	Index       int
	IndexOffset int
	Button      event.MouseButton
}
