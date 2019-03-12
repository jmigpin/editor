package widget

import (
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
	defer te.contentChanged()

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
	te.MakeIndexVisible(0)
}

//----------

func (te *TextEdit) UpdateDuplicate(dup *TextEdit) {
	// share readwriter
	dup.brw = te.brw
	dup.TextCursor.tcrw.ReadWriter = dup.brw
	dup.Drawer.SetReader(dup.brw)

	// share history
	dup.TextHistory.Use(te.TextHistory)

	dup.contentChanged()
}

//func (te *TextEdit) UpdateDuplicate_(dup *TextEdit) {
//	// keep offset/cursor/selection position for restoration
//	//ip := dup.GetPoint(dup.TextCursor.Index())
//	//ip = ip.Add(image.Point{2, 2})
//	//op := dup.GetPoint(dup.RuneOffset())
//	//op = op.Add(image.Point{2, 2})

//	// keep offset/cursor/selection position for restoration
//	//oy := dup.Offset().Y
//	//ip := dup.GetPoint(dup.TextCursor.Index())
//	//var sip image.Point
//	//if dup.TextCursor.SelectionOn() {
//	//	sip = dup.GetPoint(dup.TextCursor.SelectionIndex())
//	//}

//	// update content and share history
//	dup.TextHistory.New(0)
//	b, err := te.Bytes()
//	if err != nil {
//		log.Print(err)
//		return
//	}
//	dup.SetBytes(b)
//	dup.TextHistory.Use(te.TextHistory)

//	// restore offset/cursor/selection position
//	//i := dup.GetIndex(ip)
//	//dup.TextCursor.SetIndex(i)
//	//dup.TextCursor.SetSelectionOff()
//	//i2 := dup.GetIndex(op)
//	//dup.SetRuneOffset(i2)

//	// restore offset/cursor/selection position
//	//	dup.SetOffsetY(oy)
//	//i := dup.GetIndex(ip)
//	//	if dup.TextCursor.SelectionOn() {
//	//		si := dup.GetIndex(sip)

//	//		// commented: selection can change and result is incorrect
//	//		//dup.TextCursor.SetSelection(si, i)

//	//		// set selection off if the selection index changes
//	//		if si != dup.TextCursor.SelectionIndex() {
//	//			dup.TextCursor.SetSelectionOff()
//	//		}
//	//	} else {
//	//		dup.TextCursor.SetIndex(i)
//	//	}
//}
