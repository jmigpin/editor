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
	OnWriteOp   func(*iorw.RWCallbackWriteOp)

	rwcb iorw.ReadWriter // rwcallback (write ops)
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
	te.rwcb = &iorw.RWCallback{rw, te.writeOpCallback}
	//te.TextCursor.SetRW(te.rwcb)
	te.TextCursor.hrw = &writeOpHistoryRW{te.rwcb, te.TextCursor}
}

//----------

func (te *TextEdit) writeOpCallback(u *iorw.RWCallbackWriteOp) {
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
		err = rw.Overwrite(rw.Min(), iorw.MMLen(rw), b)
	})
	return err
}

func (te *TextEdit) SetBytesClearPos(b []byte) error {
	tc := te.TextCursor
	var err error
	tc.Edit(func() {
		rw := tc.RW()
		err = rw.Overwrite(rw.Min(), iorw.MMLen(rw), b)
		te.ClearPos() // position will be kept in history record
	})
	return err
}

// Keeps position (useful for file save)
func (te *TextEdit) SetBytesClearHistory(b []byte) error {
	rw := te.rwcb // bypass history
	if err := rw.Overwrite(rw.Min(), iorw.MMLen(rw), b); err != nil {
		return err
	}
	te.TextHistory.clear()
	te.contentChanged()
	return nil
}

func (te *TextEdit) AppendBytesClearHistory(b []byte) error {
	rw := te.rwcb // bypass history
	if err := rw.Overwrite(rw.Max(), 0, b); err != nil {
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

//----------

func (te *TextEdit) ClearPos() {
	te.TextCursor.SetSelectionOff()
	te.TextCursor.SetIndex(0)
	te.MakeIndexVisible(0)
}

//----------

var limitedReaderPadding = 2500

func (te *TextEdit) LimitedReaderPad(offset int) iorw.Reader {
	return te.LimitedReaderPad2(offset, offset)
}

func (te *TextEdit) LimitedReaderPad2(min, max int) iorw.Reader {
	return iorw.NewLimitedReaderPad(te.rw, min, max, limitedReaderPadding)
}

//----------

func (te *TextEdit) LinesIndexes(min, max int) (int, int, bool, error) {
	rd := te.LimitedReaderPad2(min, max)
	return iorw.LinesIndexes(rd, min, max)
}

func (te *TextEdit) LineStartIndex(offset int) (int, error) {
	rd := te.LimitedReaderPad(offset)
	return iorw.LineStartIndex(rd, offset)
}
func (te *TextEdit) LineEndIndex(offset int) (int, bool, error) {
	rd := te.LimitedReaderPad(offset)
	return iorw.LineEndIndex(rd, offset)
}

func (te *TextEdit) IndexFunc(offset int, truth bool, f func(rune) bool) (index, size int, err error) {
	rd := te.LimitedReaderPad(offset)
	return iorw.IndexFunc(rd, offset, truth, f)
}
func (te *TextEdit) LastIndexFunc(offset int, truth bool, f func(rune) bool) (index, size int, err error) {
	rd := te.LimitedReaderPad(offset)
	return iorw.LastIndexFunc(rd, offset, truth, f)
}

//----------

func (te *TextEdit) UpdateDuplicate(dup *TextEdit) {
	dup.SetRW(te.Text.rw)               // share readwriter
	dup.TextHistory.Use(te.TextHistory) // share history
	dup.contentChanged()
}

//----------

func (te *TextEdit) UpdatePositionOnWriteOp(u *iorw.RWCallbackWriteOp) {
	s := u.Index
	e := s + u.Dn
	e2 := s + u.In

	// update cursor/selection position
	tc := te.TextCursor
	tci := tc.Index()
	v1 := te.editValue(s, e, e2, tci)
	if !tc.SelectionOn() {
		tc.SetIndex(tci + v1)
	} else {
		si := tc.SelectionIndex()
		v3 := te.editValue(s, e, e2, si)
		tc.SetSelection(si+v3, tci+v1)
	}

	// update offset position
	ro := te.RuneOffset()
	v2 := te.editValue(s, e, e2, ro)
	te.SetRuneOffset(ro + v2)
}

func (te *TextEdit) editValue(s, e, e2, o int) int {
	v := 0
	if s < o {
		k := mathutil.Smallest(e, o)
		v = k - s
		//if typ == iorw.WopDelete {
		//	v = -v
		//}
		//if typ == iorw.WopOverwrite {
		v = -v
		k = mathutil.Smallest(e2, o)
		v += k - s
		//}
	}
	return v
}
