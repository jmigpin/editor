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
	CommentStr          string

	ui            *UI
	drawer        hsdrawer.HSDrawer
	scroller      widget.Scroller
	defaultCursor widget.Cursor

	history        *tahistory.History
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

	MeasureOpt struct {
		FirstLineOffsetX int
		lastHint         image.Point
		measurement      image.Point
	}

	flashLine struct {
		on    bool
		start time.Time
		p     image.Point
	}
	flashIndex struct {
		on         bool
		start      time.Time
		index, len int
	}
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui, CommentStr: "//"}

	// enable wrapline
	ta.drawer.WrapLineColorOpt = &loopers.WrapLineColorOpt{}

	ta.EvReg = evreg.NewRegister()
	ta.history = tahistory.NewHistory(128)

	ta.defaultCursor = widget.NoneCursor
	ta.Cursor = ta.defaultCursor

	return ta
}

func (ta *TextArea) Measure(hint image.Point) image.Point {

	// TODO: test if it has scroller in X

	// cache measurement
	face := ta.Theme.Font().Face(nil)
	if ta.str != ta.drawer.Str ||
		ta.MeasureOpt.FirstLineOffsetX != ta.drawer.FirstLineOffsetX ||
		face != ta.drawer.Face ||
		hint.X != ta.MeasureOpt.lastHint.X {

		// keep offset for restoration
		offsetIndex := 0
		changed := hint != ta.MeasureOpt.lastHint
		if changed {
			offsetIndex = ta.OffsetIndex()
		}

		ta.drawer.FirstLineOffsetX = ta.MeasureOpt.FirstLineOffsetX
		ta.drawer.Face = face
		ta.MeasureOpt.lastHint = hint
		ta.drawer.Str = ta.str

		// TODO: ensure the layout gives maximum space to not have to ignore Y in order for the textareas to work properly in dynamic sizes (toolbars)
		// ignore Y hint
		hint2 := image.Point{hint.X, 100000}

		ta.MeasureOpt.measurement = ta.drawer.Measure(hint2)

		// restore offset to keep the same first line while resizing
		if changed {
			ta.SetOffsetIndex(offsetIndex)
		}
	}
	return ta.MeasureOpt.measurement
}

func (ta *TextArea) CalcChildsBounds() {
	_ = ta.Measure(ta.Bounds.Size())
	ta.EmbedNode.CalcChildsBounds()
	ta.updateScroller()
}

func (ta *TextArea) StrHeight() int {
	h := ta.drawer.Measurement.Y
	min := ta.LineHeight()
	if h < min {
		h = min
	}
	return h
}

func (ta *TextArea) Paint() {
	bounds := ta.Bounds
	pal := ta.Theme.Palette()

	// fill background
	imageutil.FillRectangle(ta.ui.Image(), &bounds, pal.Normal.Bg)

	d := ta.drawer
	d.CursorIndex = &ta.cursorIndex
	d.Offset = ta.offset
	d.Fg = pal.Normal.Fg
	d.WrapLineColorOpt = ta.getDrawWrapLineColorOpt()
	d.SelectionOpt = ta.getDrawSelectionOpt()
	d.FlashSelectionOpt = ta.getDrawFlashSelectionOpt()
	d.HighlightWordOpt = ta.getDrawHighlightWordOpt()

	d.Draw(ta.ui.Image(), &bounds)

	ta.paintFlashLine()
}

func (ta *TextArea) getDrawWrapLineColorOpt() *loopers.WrapLineColorOpt {
	fgbg := NoSelectionColors(ta.Theme)
	return &loopers.WrapLineColorOpt{
		Fg: fgbg.Fg,
		Bg: fgbg.Bg,
	}
}
func (ta *TextArea) getDrawHighlightWordOpt() *loopers.HighlightWordOpt {
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
		Fg:    pal.Highlight.Fg,
		Bg:    pal.Highlight.Bg,
	}
}
func (ta *TextArea) getDrawSelectionOpt() *loopers.SelectionOpt {
	if ta.SelectionOn() {
		pal := ta.Theme.Palette()
		return &loopers.SelectionOpt{
			Fg:    pal.Selection.Fg,
			Bg:    pal.Selection.Bg,
			Start: ta.SelectionIndex(),
			End:   ta.CursorIndex(),
		}
	}
	return nil
}

func (ta *TextArea) paintFlashLine() {
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
	r := ta.Bounds
	r.Min.Y += ta.flashLine.p.Y - ta.OffsetY()
	r.Max.Y = r.Min.Y + ta.LineHeight()
	//r.Min.X += ta.flashLine.p.X // start flash from p.X
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
	ta.ui.RunOnUIThread(func() {
		ta.MarkNeedsPaint()
	})
}

func (ta *TextArea) getDrawFlashSelectionOpt() *loopers.FlashSelectionOpt {
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
		Bg:    ta.Theme.Palette().Normal.Bg,
	}

	// need to keep painting while flashing
	ta.ui.RunOnUIThread(func() {
		ta.MarkNeedsPaint()
	})

	return fsi
}

// Safe to use concurrently. Handles flashing the line independently of the number of runes that it contain, even if zero.
func (ta *TextArea) FlashIndexLine(index int) {
	ta.ui.RunOnUIThread(func() {
		ta.flashLine.on = true
		ta.flashLine.start = time.Now()
		ta.flashLine.p = ta.drawer.GetPoint(index)
		ta.MarkNeedsPaint()
	})
}

// Safe to use concurrently. Handles segments that span over more then one line.
func (ta *TextArea) FlashIndexLen(index, len int) {
	ta.ui.RunOnUIThread(func() {
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

	// ensure valid indexes
	ta.SetCursorIndex(ta.CursorIndex())
	ta.SetSelectionIndex(ta.SelectionIndex())

	ta.CalcChildsBounds()
	ta.MarkNeedsPaint()

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
		ta.history.Clear()
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
	ta.history.PushEdit(ta.edit)
	tahistory.TryToMergeLastTwoEdits(ta.history)

	u := ta.editStr
	cleanup()
	ta.setStr(u)
}

func (ta *TextArea) undo() {
	s, i, ok := ta.history.Undo(ta.Str())
	if !ok {
		return
	}
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOff()
}
func (ta *TextArea) redo() {
	s, i, ok := ta.history.Redo(ta.Str())
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
	//if v < 0 {
	//	v = 0
	//}
	//if v > ta.StrHeight() {
	//	v = ta.StrHeight()
	//}
	//if v != ta.offsetY {
	//	if ta.scroller != nil {
	//		// must have a scroller to change the offset
	//		ta.offsetY = v
	//		ta.MarkNeedsPaint()
	//	}
	//}

	o = widget.MaxPoint(o, image.Point{0, 0})
	o = widget.MinPoint(o, ta.drawer.Measurement)
	if o != ta.offset {
		// must have a scroller to change the offset
		if ta.scroller != nil {
			ta.offset = o
			ta.MarkNeedsPaint()
		}
	}
}

func (ta *TextArea) updateScroller() {
	if ta.scroller != nil {
		ta.scroller.SetScrollerOffset(ta.offset)
	}
}

func (ta *TextArea) OffsetIndex() int {
	return ta.drawer.GetIndex(&ta.offset)
}
func (ta *TextArea) SetOffsetIndex(i int) {
	p := ta.drawer.GetPoint(i)
	ta.SetOffsetY(p.Y)
}

func (ta *TextArea) MakeCursorVisible() {
	ta.MakeIndexVisible(ta.CursorIndex())
}
func (ta *TextArea) MakeIndexVisible(index int) {
	y0 := ta.OffsetY()
	y1 := y0 + ta.Bounds.Dy()

	// is all visible
	a0 := ta.drawer.GetPoint(index).Y
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
	a0 := ta.drawer.GetPoint(index).Y
	a1 := a0 + ta.LineHeight()
	if a0 >= y0 && a1 <= y1 {
		return true
	}
	return false
}

func (ta *TextArea) MakeIndexVisibleAtCenter(index int) {
	// set at half bounds
	p0 := ta.drawer.GetPoint(index).Y
	half := (ta.Bounds.Dy() - ta.LineHeight()) / 2
	offsetY := p0 - half
	ta.SetOffsetY(offsetY)
}

func (ta *TextArea) GetCPPaste(i event.CopyPasteIndex) (string, error) {
	return ta.ui.GetCPPaste(i)
}
func (ta *TextArea) SetCPCopy(i event.CopyPasteIndex, v string) error {
	return ta.ui.SetCPCopy(i, v)
}

func (ta *TextArea) LineHeight() int {
	return ta.drawer.LineHeight()
}

func (ta *TextArea) GetPoint(i int) image.Point {
	return ta.drawer.GetPoint(i)
}
func (ta *TextArea) GetIndex(p *image.Point) int {
	return ta.drawer.GetIndex(p)
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
	ta.ui.RunOnUIThread(func() {
		tautil.InsertString(ta, str)
	})
}

func (ta *TextArea) History() *tahistory.History {
	return ta.history
}
func (ta *TextArea) SetHistory(h *tahistory.History) {
	ta.history = h
}

func (ta *TextArea) GetBounds() image.Rectangle {
	return ta.Bounds
}

func (ta *TextArea) CommentString() string {
	return ta.CommentStr
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

	m := ta.drawer.Measurement
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
	TextAreaCmdEventId = iota
	TextAreaSetStrEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
	Index    int
}
type TextAreaSetStrEvent struct {
	TextArea *TextArea
}
