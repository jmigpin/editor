package ui

import (
	"sync"
	"time"

	"github.com/jmigpin/editor/drawutil2/hsdrawer"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/widget"
)

type Toolbar struct {
	*TextArea
	parent widget.Node

	flash struct {
		sync.Mutex
		on     bool
		start  time.Time
		colors *hsdrawer.Colors
	}
}

func NewToolbar(ui *UI, parent widget.Node) *Toolbar {
	tb := &Toolbar{parent: parent}
	tb.TextArea = NewTextArea(ui)

	tb.DisableHighlightCursorWord = true
	tb.Colors = &ToolbarColors

	tb.TextArea.EvReg.Add(TextAreaSetStrEventId, tb.onTextAreaSetStr)

	return tb
}
func (tb *Toolbar) onTextAreaSetStr(ev0 interface{}) {
	ev := ev0.(*TextAreaSetStrEvent)

	// dynamic toolbar bounds that change when edited
	// if toolbar bounds changed due to text change (dynamic height) then the parent container needs paint
	b := tb.Bounds()
	tb.parent.CalcChildsBounds()
	if !tb.Bounds().Eq(b) {
		tb.parent.Embed().MarkNeedsPaint()
	}

	// keep pointer inside if it was in before -- need to test if it was in before otherwise and auto-change that edits the toolbar will warp the pointer
	// useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus
	b2 := tb.Bounds() // new recalc'ed bounds
	p, ok := tb.ui.QueryPointer()
	if ok && p.In(ev.OldBounds) && !p.In(b2) {
		tb.ui.WarpPointerToRectanglePad(&b2)
	}
}

// Safe to use concurrently
func (tb *Toolbar) Flash() {
	tb.flash.Lock()
	// keep but don't override the original colors
	if !tb.flash.on {
		tb.flash.colors = tb.Colors
	}

	tb.flash.on = true
	tb.flash.start = time.Now()
	tb.flash.Unlock()

	tb.MarkNeedsPaint()
}
func (tb *Toolbar) Paint() {
	if tb.flash.on {
		// setup flash colors
		tb.flash.Lock()
		now := time.Now()
		end := tb.flash.start.Add(250 * time.Millisecond)
		if now.After(end) {
			tb.flash.on = false
			tb.Colors = tb.flash.colors
		} else {
			t := now.Sub(tb.flash.start)
			d := end.Sub(tb.flash.start)
			perc := 1.0 - (float64(t) / float64(d))
			c1 := *tb.flash.colors
			nc := &tb.flash.colors.Normal
			c1.Normal.Fg = imageutil.TintOrShade(nc.Fg, perc)
			c1.Normal.Bg = imageutil.TintOrShade(nc.Bg, perc)
			tb.Colors = &c1
		}
		tb.flash.Unlock()

		tb.MarkNeedsPaint()
	}
	tb.TextArea.Paint()
}
