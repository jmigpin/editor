package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil"
)

type Layout struct {
	C       uiutil.Container
	UI      *UI
	Toolbar *Toolbar
	Cols    *Columns
	Sep     uiutil.Container
}

func NewLayout(ui *UI) *Layout {
	layout := &Layout{}
	layout.UI = ui

	layout.Toolbar = NewToolbar(ui)
	tb := layout.Toolbar
	tb.C.Style.DynamicMainSize = tb.TextArea.CalcUsedY
	tb.Data = layout
	tb.Colors = &ToolbarColors
	tb.OnSetText = func() {
		// dynamic toolbar bounds
		//layout.C.CalcChildsBounds()
		//layout.C.NeedPaint() // TODO: if bounds changed
	}

	sep := NewSeparator(ui, SeparatorWidth, SeparatorColor)

	layout.Cols = NewColumns(layout)

	layout.C.Style.Direction = uiutil.ColumnDirection
	layout.C.AppendChilds(&layout.Toolbar.C, &sep.C, &layout.Cols.C)

	return layout
}

func (layout *Layout) pointEvent(p *image.Point, ev interface{}) {
}

//func (layout *Layout) CalcArea(area *image.Rectangle) {
//a := *area
//layout.Area = a
//layout.Toolbar.CalcArea(&a)
//// separator
//a.Min.Y = layout.Toolbar.Area.Max.Y
//a.Min.Y += SeparatorWidth
//// cols
//layout.Cols.CalcArea(&a)
//}
//func (layout *Layout) Paint() {
//layout.Toolbar.Paint()
//// separator
//r1 := layout.Area
//r1.Min.Y = layout.Toolbar.Area.Max.Y
//r1.Max.Y = r1.Min.Y + SeparatorWidth
//layout.FillRectangle(&r1, &SeparatorColor)

//layout.Cols.Paint()
//}
