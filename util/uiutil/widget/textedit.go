package widget

import (
	"image"
	"log"

	"github.com/jmigpin/editor/util/iout/iorw"
)

type TextEdit struct {
	*Text
	ClipboardContext

	TextCursor  *TextCursor
	TextHistory *TextHistory
}

func NewTextEdit(ctx ImageContext, cctx ClipboardContext) *TextEdit {
	t := NewText(ctx)
	te := &TextEdit{Text: t, ClipboardContext: cctx}
	te.TextCursor = NewTextCursor(te)
	te.TextHistory = NewTextHistory(te)
	return te
}

//----------

func (te *TextEdit) SetBytes(b []byte) error {
	tc := te.TextCursor
	var err error
	tc.Edit(func() {
		err = iorw.DeleteInsertIfNotEqual(tc.RW(), 0, tc.RW().Len(), b)
	})
	return err
}

func (te *TextEdit) SetBytesClearPos(b []byte) error {
	tc := te.TextCursor
	var err error
	tc.Edit(func() {
		err = iorw.DeleteInsertIfNotEqual(tc.RW(), 0, tc.RW().Len(), b)
		// keep position in history record
		te.ClearPos()
	})
	return err
}

func (te *TextEdit) SetBytesClearHistory(b []byte) error {
	te.TextHistory.clear()
	return te.Text.SetBytes(b) // bypasses history
}

func (te *TextEdit) AppendBytesClearHistory(b []byte, maxSize int) error {
	te.TextHistory.clear()
	rw := te.brw // bypasses history

	l := rw.Len() + len(b)
	if l > maxSize {
		if err := rw.Delete(0, l-maxSize); err != nil {
			return err
		}
	}

	// run changes only once for delete+insert
	defer te.changes()

	return rw.Insert(rw.Len(), b)
}

//----------

func (te *TextEdit) SetStr(str string) error {
	return te.SetBytes([]byte(str))
}

func (te *TextEdit) SetStrClearPos(str string) error {
	return te.SetBytesClearPos([]byte(str))
}

func (te *TextEdit) SetStrClearHistory(str string) error {
	return te.SetBytesClearHistory([]byte(str))
}

func (te *TextEdit) AppendStrClearHistory(str string, maxSize int) error {
	return te.AppendBytesClearHistory([]byte(str), maxSize)
}

//----------

func (te *TextEdit) ClearPos() {
	te.TextCursor.SetSelectionOff()
	te.TextCursor.SetIndex(0)
	te.SetRuneOffset(0)
}

//----------

//func (te *TextEdit) OffsetIndex() int {
//	return te.Drawer.IndexOf(te.Offset())
//}

//func (te *TextEdit) SetOffsetIndex(index int) {
//	y := te.Drawer.PointOf(index).Y
//	te.SetOffsetY(y)
//}

//----------

func (te *TextEdit) GetPoint(i int) image.Point {
	return te.Drawer.BoundsPointOf(i)
}
func (te *TextEdit) GetIndex(p image.Point) int {
	return te.Drawer.BoundsIndexOf(p)
}

//----------

func (te *TextEdit) MakeIndexVisible(index int) {
	te.MakeRangeVisible(index, 0)
}

//func (te *TextEdit) MakeRangeVisible(index, le int) {
//	vr := te.VisibleRect()
//	ir := te.IndexLenRect(index, le)
//	o := te.indexRectOffset(ir, vr)
//	te.SetOffset(o)
//}

//func (te *TextEdit) MakeRangeCentered(index, le int) {
//	ir := te.IndexLenRect(index, le)
//	o := te.centeredIndexRectOffset(ir)
//	te.SetOffset(o)
//}

//----------

func (te *TextEdit) IsIndexVisible(index int) bool {
	return te.IsRangeVisible(index, 0)
}

//func (te *TextEdit) IsRangeVisible(index, le int) bool {
//	vr := te.VisibleRect()
//	ir := te.IndexLenRect(index, le)
//	return ir.Overlaps(vr)
//}

//----------

//func (te *TextEdit) indexRectOffset(ir, vr image.Rectangle) image.Point {
//	// all visible
//	if ir.In(vr) {
//		return vr.Min
//	}

//	// bigger then the view, set at top
//	if ir.Dy() > vr.Dy() {
//		return ir.Min
//	}

//	// align to closest top/bottom
//	if ir.Overlaps(vr) {
//		u := ir.Intersect(vr)
//		// align to the top
//		if u.Min.Y == vr.Min.Y {
//			return ir.Min
//		}
//		// align to the bottom
//		if u.Max.Y == vr.Max.Y {
//			w := ir.Max
//			w.Y -= te.Bounds.Dy()
//			return w
//		}
//	}

//	return te.centeredIndexRectOffset(ir)
//}

//func (te *TextEdit) centeredIndexRectOffset(ir image.Rectangle) image.Point {
//	bh := te.Bounds.Size().Div(2)
//	ih := ir.Size().Div(2)
//	w := ir.Min.Sub(bh).Add(ih)
//	return imageutil.MaxPoint(w, image.Point{})
//}

//----------

//func (te *TextEdit) IndexLenRect(index, le int) image.Rectangle {
//	a0 := te.Drawer.PointOf(index)

//	a1 := a0.Add(image.Point{1, 0}) // non-empty rectangle
//	if le != 0 {
//		a1 = te.Drawer.PointOf(index + le)
//		// could change line and max x is at the left
//		if a1.X <= a0.X {
//			a1.X = a0.X + 1 // non-empty rectangle
//		}
//	}
//	a1.Y += te.LineHeight()

//	return image.Rectangle{Min: a0, Max: a1}
//}

//func (te *TextEdit) VisibleRect() (vr image.Rectangle) {
//	vr = vr.Add(te.Offset())
//	vr.Max = vr.Max.Add(te.Bounds.Size())
//	return vr
//}

//----------

func (te *TextEdit) UpdateDuplicate(dup *TextEdit) {
	//// keep offset/cursor/selection position for restoration
	//oy := dup.Offset().Y
	//ip := dup.GetPoint(dup.TextCursor.Index())
	//var sip image.Point
	//if dup.TextCursor.SelectionOn() {
	//	sip = dup.GetPoint(dup.TextCursor.SelectionIndex())
	//}

	// update content and share history
	dup.TextHistory.New(0)
	b, err := te.Bytes()
	if err != nil {
		log.Print(err)
	}
	dup.SetBytes(b)
	dup.TextHistory.Use(te.TextHistory)

	//	// restore offset/cursor/selection position
	//	dup.SetOffsetY(oy)
	//	i := dup.GetIndex(ip)
	//	if dup.TextCursor.SelectionOn() {
	//		si := dup.GetIndex(sip)

	//		// commented: selection can change and result is incorrect
	//		//dup.TextCursor.SetSelection(si, i)

	//		// set selection off if the selection index changes
	//		if si != dup.TextCursor.SelectionIndex() {
	//			dup.TextCursor.SetSelectionOff()
	//		}
	//	} else {
	//		dup.TextCursor.SetIndex(i)
	//	}
}
