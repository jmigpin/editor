package widget

import (
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/mathutil"
)

type TextEdit struct {
	*Text
	ClipboardContext

	TextCursor  *TextCursor
	TextHistory *TextHistory
	OnWriteOp   func(*RWWriteOpCb)

	crw iorw.ReadWriter // write op callback rw: OnWriteOp(...)
}

func NewTextEdit(ctx ImageContext, cctx ClipboardContext) *TextEdit {
	t := NewText(ctx)
	te := &TextEdit{Text: t, ClipboardContext: cctx}
	te.TextCursor = NewTextCursor(te)
	te.TextHistory = NewTextHistory(te)
	te.SetRW(te.Text.rw)
	return te
}

//----------

func (te *TextEdit) SetRW(rw iorw.ReadWriter) {
	te.Text.SetRW(rw)
	te.crw = &writeOpCbRW{rw, te}
	te.TextCursor.hrw = &writeOpHistoryRW{te.crw, te.TextCursor}
}

//----------

func (te *TextEdit) writeOpCallback(u *RWWriteOpCb) {
	if te.OnWriteOp != nil {
		te.OnWriteOp(u)
	}
}

//----------

func (te *TextEdit) SetBytes(b []byte) error {
	tc := te.TextCursor
	var err error
	tc.Edit(func() {
		rw := tc.RW()
		err = rw.Overwrite(0, rw.Len(), b)
	})
	return err
}

func (te *TextEdit) SetBytesClearPos(b []byte) error {
	tc := te.TextCursor
	var err error
	tc.Edit(func() {
		rw := tc.RW()
		err = rw.Overwrite(0, rw.Len(), b)
		te.ClearPos() // position will be kept in history record
	})
	return err
}

// Keeps position (useful for file save)
func (te *TextEdit) SetBytesClearHistory(b []byte) error {
	rw := te.crw // bypass history
	if err := rw.Overwrite(0, rw.Len(), b); err != nil {
		return err
	}
	te.TextHistory.clear()
	te.contentChanged()
	return nil
}

func (te *TextEdit) AppendBytesClearHistory(b []byte) error {
	rw := te.crw // bypass history
	if err := rw.Insert(rw.Len(), b); err != nil {
		return err
	}
	te.TextHistory.clear()
	te.contentChanged()
	return nil
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

func (te *TextEdit) AppendStrClearHistory(str string) error {
	return te.AppendBytesClearHistory([]byte(str))
}

//----------

func (te *TextEdit) ClearPos() {
	te.TextCursor.SetSelectionOff()
	te.TextCursor.SetIndex(0)
	te.MakeIndexVisible(0)
}

//----------

func (te *TextEdit) UpdateDuplicate(dup *TextEdit) {
	dup.SetRW(te.Text.rw)               // share readwriter
	dup.TextHistory.Use(te.TextHistory) // share history
	dup.contentChanged()
}

//----------

func (te *TextEdit) UpdateWriteOp(u *RWWriteOpCb) {
	s := u.Index
	e := s + u.Length1
	e2 := s + u.Length2

	// update cursor/selection position
	tc := te.TextCursor
	tci := tc.Index()
	v1 := te.editValue(u.Type, s, e, e2, tci)
	if !tc.SelectionOn() {
		tc.SetIndex(tci + v1)
	} else {
		si := tc.SelectionIndex()
		v3 := te.editValue(u.Type, s, e, e2, si)
		tc.SetSelection(si+v3, tci+v1)
	}

	// update offset position
	ro := te.RuneOffset()
	v2 := te.editValue(u.Type, s, e, e2, ro)
	te.SetRuneOffset(ro + v2)
}

func (te *TextEdit) editValue(typ iorw.WriterOp, s, e, e2, o int) int {
	v := 0
	if s < o {
		k := mathutil.Smallest(e, o)
		v = k - s
		if typ == iorw.DeleteWOp {
			v = -v
		}
		if typ == iorw.OverwriteWOp {
			v = -v
			k := mathutil.Smallest(e2, o)
			v += k - s
		}
	}
	return v
}

//----------

// Runs callback on write operations.
type writeOpCbRW struct {
	iorw.ReadWriter
	te *TextEdit
}

func (rw *writeOpCbRW) Insert(i int, p []byte) error {
	if err := rw.ReadWriter.Insert(i, p); err != nil {
		return err
	}
	u := &RWWriteOpCb{iorw.InsertWOp, i, len(p), 0}
	rw.te.writeOpCallback(u)
	return nil
}

func (rw *writeOpCbRW) Delete(i, length int) error {
	if err := rw.ReadWriter.Delete(i, length); err != nil {
		return err
	}
	u := &RWWriteOpCb{iorw.DeleteWOp, i, length, 0}
	rw.te.writeOpCallback(u)
	return nil
}

func (rw *writeOpCbRW) Overwrite(i, length int, p []byte) error {
	if err := rw.ReadWriter.Overwrite(i, length, p); err != nil {
		return err
	}
	u := &RWWriteOpCb{iorw.OverwriteWOp, i, length, len(p)}
	rw.te.writeOpCallback(u)
	return nil
}

//----------

type RWWriteOpCb struct {
	Type    iorw.WriterOp
	Index   int
	Length1 int
	Length2 int
}
