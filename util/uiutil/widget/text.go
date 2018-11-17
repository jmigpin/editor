package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/iout"
)

type Text struct {
	ENode
	TextScroll

	Drawer   drawer3.Drawer
	OnSetStr func() // TODO: rename

	scrollable struct{ x, y bool }
	ctx        ImageContext
	fg, bg     color.Color

	brw iout.ReadWriter // base rw
	//trw iout.ReadWriter // text rw (calls changes)
}

func NewText(ctx ImageContext) *Text {
	t := &Text{ctx: ctx}

	t.brw = iout.NewRW(nil)
	//t.trw = &tRW{ReadWriter: t.brw, t: t}

	t.TextScroll.Text = t

	t.Drawer = drawer3.NewPosDrawer()
	t.Drawer.SetReader(t.brw)
	return t
}

//----------

//func (t *Text) BaseRW() iout.ReadWriter {
//	return t.brw
//}

//func (t *Text) RW() iout.ReadWriter {
//	return t.trw
//}

//----------

func (t *Text) Len() int {
	return t.brw.Len()
}

// Result might not be a copy, so changes to the slice might affect the text data.
func (t *Text) Bytes() ([]byte, error) {
	return t.brw.ReadNSliceAt(0, t.brw.Len())
}

func (t *Text) SetBytes(b []byte) error {
	if err := t.brw.Delete(0, t.brw.Len()); err != nil {
		return err
	}

	// run changes only once for delete+insert
	defer t.changes()

	return t.brw.Insert(0, b)
}

//----------

func (t *Text) Str() string {
	p, err := t.Bytes()
	if err != nil {
		return ""
	}
	return string(p)
}

func (t *Text) SetStr(str string) error {
	return t.SetBytes([]byte(str))
}

//----------

func (t *Text) changes() {
	t.Drawer.SetNeedMeasure(true)
	t.MarkNeedsLayoutAndPaint()

	// TODO: move this to somewhere else.
	// Because it will layout now, it needs to set the exts options
	if d, ok := t.Drawer.(*drawer3.PosDrawer); ok {
		max := 75 * 1024
		v := t.Len() < max
		if !v {
			d.WrapLine.SetOn(v)
			d.ColorizeSyntax.SetOn(v)
			//d.Segments.SetOn(v)
		}
	}

	// make possible measurements available immediately
	t.Layout()

	if t.OnSetStr != nil {
		t.OnSetStr()
	}
}

//----------

func (t *Text) Offset() image.Point {
	return t.Drawer.Offset()
}

func (t *Text) SetOffset(o image.Point) {
	// set only if scrollable
	u := image.Point{}
	if t.scrollable.x {
		u.X = o.X
	}
	if t.scrollable.y {
		u.Y = o.Y
	}

	if u != t.Drawer.Offset() {
		t.Drawer.SetOffset(u)
		t.MarkNeedsLayoutAndPaint()
	}
}

func (t *Text) SetOffsetY(y int) {
	o := t.Offset()
	o.Y = y
	t.SetOffset(o)
}

// implements Scrollable interface.
func (t *Text) SetScrollable(x, y bool) {
	t.scrollable.x = x
	t.scrollable.y = y
}

//----------

func (t *Text) FullMeasurement() image.Point {
	t.Drawer.SetBounds(t.Bounds)
	return t.Drawer.Measure()
}

func (t *Text) LineHeight() int {
	return t.Drawer.LineHeight()
}

//----------

func (t *Text) Measure(hint image.Point) image.Point {
	t.Drawer.SetBoundsSize(hint)
	m := t.Drawer.Measure()
	return imageutil.MinPoint(m, hint)
}

//----------

func (t *Text) Layout() {
	t.Drawer.SetBounds(t.Bounds)
	if t.Drawer.NeedMeasure() {
		_ = t.Drawer.Measure()
		t.MarkNeedsPaint()
	}
}

//----------

func (t *Text) PaintBase() {
	imageutil.FillRectangle(t.ctx.Image(), &t.Bounds, t.bg)
}
func (t *Text) Paint() {
	t.Drawer.SetBounds(t.Bounds)
	t.Drawer.Draw(t.ctx.Image(), t.fg)
}

//----------

func (t *Text) OnThemeChange() {
	t.fg = t.TreeThemePaletteColor("text_fg")
	t.bg = t.TreeThemePaletteColor("text_bg")

	f := t.TreeThemeFont().Face(nil)
	t.Drawer.SetFace(f)

	if t.Drawer.NeedMeasure() {
		t.MarkNeedsLayoutAndPaint()
	} else {
		t.MarkNeedsPaint()
	}
}

//----------

//// Auto calls t.changes() on write operations.
//type tRW struct {
//	iout.ReadWriter
//	t *Text
//}

//func (rw *tRW) Insert(i int, p []byte) error {
//	err := rw.ReadWriter.Insert(i, p)
//	if err != nil {
//		return err
//	}
//	rw.t.changes()
//	return nil
//}

//func (rw *tRW) Delete(i, len int) error {
//	err := rw.ReadWriter.Delete(i, len)
//	if err != nil {
//		return err
//	}
//	rw.t.changes()
//	return nil
//}
