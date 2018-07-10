package textutil

import (
	"testing"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

//----------

type cctx struct{}

func (*cctx) GetCPPaste(i event.CopyPasteIndex, fn func(string, bool)) {}
func (*cctx) SetCPCopy(i event.CopyPasteIndex, v string)               {}
func (*cctx) RunOnUIGoRoutine(f func())                                { f() }

//----------

func TestAll1(t *testing.T) {
	type test struct {
		s   string
		ci  int  // cursor index
		si  int  // selected index
		son bool // selection on
		f   func(*widget.TextEditX) error
		// expected
		es   string
		eci  int
		esi  int
		eson bool
	}

	var tests = []*test{
		{
			s: "123\nabc\n", ci: 0,
			es: "123\n123\nabc\n", esi: 4, eci: 7, eson: true,
			f: func(tex *widget.TextEditX) error { return DuplicateLines(tex.TextEdit) },
		},
		{
			s: "123", ci: 0,
			es: "123\n123", esi: 4, eci: 7, eson: true,
			f: func(tex *widget.TextEditX) error { return DuplicateLines(tex.TextEdit) },
		},

		{
			s: "   123", ci: 4,
			es: "   1\n   23", eci: 8,
			f: func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
		},
		{
			s: "   123", ci: 4, si: 5, son: true,
			es: "   1\n   3", eci: 8, eson: false,
			f: func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
		},
		{
			s: "0123\n   abc", ci: 10,
			es: "0123\n   ab\n   c", eci: 14, eson: false,
			f: func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
		},
		{
			s: "0123\n   abc", ci: 6,
			es: "0123\n \n   abc", eci: 8, eson: false,
			f: func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
		},

		{
			s: "123", ci: 2,
			es: "13", eci: 1,
			f: func(tex *widget.TextEditX) error { return Backspace(tex.TextEdit) },
		},
		{
			s: "01234", ci: 2, si: 4, son: true,
			es: "014", eci: 2,
			f: func(tex *widget.TextEditX) error { return Backspace(tex.TextEdit) },
		},
		{
			s:  "01234",
			es: "//01234", eci: 2,
			f: func(tex *widget.TextEditX) error {
				tex.SetCommentStrings("//", [2]string{})
				return Comment(tex)
			},
		},
		{
			s: "0123\n  abc\n efg\n", si: 9, ci: 14, son: true,
			es: "0123\n // abc\n //efg\n", esi: 5, eci: 19, eson: true,
			f: func(tex *widget.TextEditX) error {
				tex.SetCommentStrings("//", [2]string{})
				return Comment(tex)
			},
		},
		{
			s: "0123\n // abc\n //efg\n", si: 5, ci: 19, son: true,
			es: "0123\n  abc\n efg\n", esi: 5, eci: 15, eson: true,
			f: func(tex *widget.TextEditX) error {
				tex.SetCommentStrings("//", [2]string{})
				return Uncomment(tex)
			},
		},
		{
			s: "01234", ci: 2, si: 4, son: true,
			es: "014", eci: 2,
			f: func(tex *widget.TextEditX) error { return Cut(tex.TextEdit) },
		},
		{
			s: "01234", ci: 2, si: 4, son: true,
			es: "014", eci: 2,
			f: func(tex *widget.TextEditX) error { return Delete(tex.TextEdit) },
		},
		{
			s: "01234", ci: 2,
			es: "0134", eci: 2,
			f: func(tex *widget.TextEditX) error { return Delete(tex.TextEdit) },
		},
		{
			s:  "0123\nabc",
			es: "0123\nabc", eci: 4,
			f: func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, false) },
		},
		{
			s: "0123\nabc", ci: 6,
			es: "0123\nabc", esi: 6, eci: 8, eson: true,
			f: func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, true) },
		},
		{
			s:  "012",
			es: "012", eci: 3,
			f: func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, false) },
		},
		{
			s:  "012\nabc",
			es: "012\nabc", eci: 7,
			f: func(tex *widget.TextEditX) error {
				EndOfString(tex.TextEdit, false)
				return nil
			},
		},
		{
			s:  "01234\nabc",
			es: "01234\nabc", esi: 6, eci: 8, eson: true,
			f: func(tex *widget.TextEditX) error {
				_, err := Find(tex.TextEdit, "ab")
				return err
			},
		},
		{
			s: "01234\nabc", ci: 7,
			es: "01234\nabc", esi: 6, eci: 8, eson: true,
			f: func(tex *widget.TextEditX) error {
				_, err := Find(tex.TextEdit, "ab")
				return err
			},
		},
		{
			s: "0123", ci: 2,
			es: "01ab23", eci: 4,
			f: func(tex *widget.TextEditX) error {
				return InsertString(tex.TextEdit, "ab")
			},
		},
		{
			s: "0123", ci: 2,
			es: "0123", eci: 1,
			f: func(tex *widget.TextEditX) error {
				return MoveCursorLeft(tex.TextEdit, false)
			},
		},
		{
			s: "0123", ci: 2,
			es: "0123", eci: 3,
			f: func(tex *widget.TextEditX) error {
				return MoveCursorRight(tex.TextEdit, false)
			},
		},
		{
			s: " 0123 ", ci: 3,
			es: " 0123 ", eci: 5,
			f: func(tex *widget.TextEditX) error {
				return MoveCursorJumpRight(tex.TextEdit, false)
			},
		},

		{
			s: "0123\nabcd", ci: 5,
			es: "abcd\n0123", eci: 0,
			f: func(tex *widget.TextEditX) error {
				return MoveLineUp(tex.TextEdit)
			},
		},
		{
			s: "0123\nabcd\n", ci: 5,
			es: "abcd\n0123\n", eci: 0,
			f: func(tex *widget.TextEditX) error {
				return MoveLineUp(tex.TextEdit)
			},
		},
		{
			s: "01\nab\nzy", si: 4, ci: 7, son: true,
			es: "ab\nzy\n01", esi: 0, eci: 5, eson: true,
			f: func(tex *widget.TextEditX) error {
				return MoveLineUp(tex.TextEdit)
			},
		},
		{
			s:  "01\nab\nzy",
			es: "ab\n01\nzy", eci: 3,
			f: func(tex *widget.TextEditX) error {
				return MoveLineDown(tex.TextEdit)
			},
		},
		{
			s: "01\nab\nzy\n", si: 0, ci: 4, son: true,
			es: "zy\n01\nab\n", esi: 3, eci: 8, eson: true,
			f: func(tex *widget.TextEditX) error {
				return MoveLineDown(tex.TextEdit)
			},
		},
		{
			s: "01\nab\nzy", ci: 4,
			es: "01\nzy\nab", eci: 7,
			f: func(tex *widget.TextEditX) error {
				return MoveLineDown(tex.TextEdit)
			},
		},
		{
			s: "01\nab\nzy", ci: 4,
			es: "01\nzy", eci: 3,
			f: func(tex *widget.TextEditX) error {
				return RemoveLines(tex.TextEdit)
			},
		},

		{
			s: "0123401234", ci: 4,
			es: "0404", eci: 1,
			f: func(tex *widget.TextEditX) error {
				_, err := Replace(tex.TextEdit, "123", "")
				return err
			},
		},

		{
			s: "012 -- abc", ci: 4,
			es: "012 -- abc", esi: 4, eci: 5, eson: true,
			f: func(tex *widget.TextEditX) error {
				return SelectWord(tex.TextEdit)
			},
		},
		{
			s: "012 -- abc", ci: 1,
			es: "012 -- abc", esi: 0, eci: 3, eson: true,
			f: func(tex *widget.TextEditX) error {
				return SelectWord(tex.TextEdit)
			},
		},
		{
			s: "--abc--", ci: 3,
			es: "--abc--", esi: 2, eci: 5, eson: true,
			f: func(tex *widget.TextEditX) error {
				return SelectWord(tex.TextEdit)
			},
		},
		{
			s: "--abc--", ci: 5,
			es: "--abc--", esi: 5, eci: 6, eson: true,
			f: func(tex *widget.TextEditX) error {
				return SelectWord(tex.TextEdit)
			},
		},
		{
			s: "abc\n   0123", ci: 10,
			es: "abc\n   0123", eci: 7,
			f: func(tex *widget.TextEditX) error {
				return StartOfLine(tex.TextEdit, false)
			},
		},
		{
			s: "0123", ci: 0,
			es: "\t0123", eci: 1,
			f: func(tex *widget.TextEditX) error {
				return TabRight(tex.TextEdit)
			},
		},
		{
			s: "0123\nabc\n", si: 2, ci: 6, son: true,
			es: "\t0123\n\tabc\n", esi: 0, eci: 10, eson: true,
			f: func(tex *widget.TextEditX) error {
				return TabRight(tex.TextEdit)
			},
		},
		{
			s: "\t0123\n\tabc\n", si: 2, ci: 8, son: true,
			es: "0123\nabc\n", esi: 0, eci: 8, eson: true,
			f: func(tex *widget.TextEditX) error {
				return TabLeft(tex.TextEdit)
			},
		},

		// secondary
		// TODO: movecursorup/movecursordown
		// TODO: copy
		// TODO: paste
		// TODO: startofstring
		// TODO: selectall
		// TODO: selectline
	}

	for x, w := range tests {
		// init
		tex := widget.NewTextEditX(nil, &cctx{})
		tex.Text.SetStr(w.s)
		tc := tex.TextCursor
		tc.SetIndex(w.ci)
		tc.SetSelection(w.si, w.ci)
		if !w.son {
			tc.SetSelectionOff()
		}

		// func error
		if err := w.f(tex); err != nil {
			t.Fatalf("x=%v err=%v", x, err)
		}
		// content
		s, err := tc.RW().ReadNAt(0, tc.RW().Len())
		if err != nil {
			t.Fatal(err)
		}
		s1 := string(s)
		if s1 != w.es {
			t.Fatalf("x=%v %q != %q", x, s1, w.es)
		}
		// cursor index
		if tc.Index() != w.eci {
			t.Fatalf("x=%v index %v != %v", x, tc.Index(), w.eci)
		}
		// selection index
		if tc.SelectionOn() != w.eson {
			t.Fatalf("x=%v selectionon %v != %v", x, tc.SelectionOn(), w.eson)
		}
		if tc.SelectionOn() && tc.SelectionIndex() != w.esi {
			t.Fatalf("x=%v selectionindex %v != %v", x, tc.SelectionIndex(), w.esi)
		}
	}
}
