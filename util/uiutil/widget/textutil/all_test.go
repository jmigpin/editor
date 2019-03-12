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
	type state struct {
		s   string
		ci  int  // cursor index
		si  int  // selected index
		son bool // selection on
	}
	type test struct {
		st  state
		est state // expected state
		f   func(*widget.TextEditX) error
	}

	var testEntry func(*test)

	var testFns = []func(){
		func() {
			testEntry(&test{
				st:  state{s: "123\nabc\n", ci: 0},
				est: state{s: "123\n123\nabc\n", si: 4, ci: 7, son: true},
				f:   func(tex *widget.TextEditX) error { return DuplicateLines(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "123", ci: 0},
				est: state{s: "123\n123", si: 4, ci: 7, son: true},
				f:   func(tex *widget.TextEditX) error { return DuplicateLines(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "   123", ci: 4},
				est: state{s: "   1\n   23", ci: 8},
				f:   func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "   123", ci: 4, si: 5, son: true},
				est: state{s: "   1\n   3", ci: 8, son: false},
				f:   func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n   abc", ci: 10},
				est: state{s: "0123\n   ab\n   c", ci: 14, son: false},
				f:   func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n   abc", ci: 6},
				est: state{s: "0123\n \n   abc", ci: 8, son: false},
				f:   func(tex *widget.TextEditX) error { return AutoIndent(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "123", ci: 2},
				est: state{s: "13", ci: 1},
				f:   func(tex *widget.TextEditX) error { return Backspace(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   func(tex *widget.TextEditX) error { return Backspace(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234"},
				est: state{s: "//01234", ci: 2},
				f: func(tex *widget.TextEditX) error {
					tex.SetCommentStrings("//", [2]string{})
					return Comment(tex)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n  abc\n efg\n", si: 9, ci: 14, son: true},
				est: state{s: "0123\n // abc\n //efg\n", si: 5, ci: 19, son: true},
				f: func(tex *widget.TextEditX) error {
					tex.SetCommentStrings("//", [2]string{})
					return Comment(tex)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n // abc\n //efg\n", si: 5, ci: 19, son: true},
				est: state{s: "0123\n  abc\n efg\n", si: 5, ci: 15, son: true},
				f: func(tex *widget.TextEditX) error {
					tex.SetCommentStrings("//", [2]string{})
					return Uncomment(tex)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   func(tex *widget.TextEditX) error { return Cut(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   func(tex *widget.TextEditX) error { return Delete(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2},
				est: state{s: "0134", ci: 2},
				f:   func(tex *widget.TextEditX) error { return Delete(tex.TextEdit) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc"},
				est: state{s: "0123\nabc", ci: 4},
				f:   func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, false) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc", ci: 6},
				est: state{s: "0123\nabc", si: 6, ci: 8, son: true},
				f:   func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, true) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012"},
				est: state{s: "012", ci: 3},
				f:   func(tex *widget.TextEditX) error { return EndOfLine(tex.TextEdit, false) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012\nabc"},
				est: state{s: "012\nabc", ci: 7},
				f: func(tex *widget.TextEditX) error {
					EndOfString(tex.TextEdit, false)
					return nil
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234\nabc"},
				est: state{s: "01234\nabc", si: 6, ci: 8, son: true},
				f: func(tex *widget.TextEditX) error {
					_, err := Find(tex.TextEdit, "ab")
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234\nabc", ci: 7},
				est: state{s: "01234\nabc", si: 6, ci: 8, son: true},
				f: func(tex *widget.TextEditX) error {
					_, err := Find(tex.TextEdit, "ab")
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "01ab23", ci: 4},
				f: func(tex *widget.TextEditX) error {
					return InsertString(tex.TextEdit, "ab")
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "0123", ci: 1},
				f: func(tex *widget.TextEditX) error {
					return MoveCursorLeft(tex.TextEdit, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "0123", ci: 3},
				f: func(tex *widget.TextEditX) error {
					return MoveCursorRight(tex.TextEdit, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: " 0123 ", ci: 3},
				est: state{s: " 0123 ", ci: 5},
				f: func(tex *widget.TextEditX) error {
					return MoveCursorJumpRight(tex.TextEdit, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabcd", ci: 5},
				est: state{s: "abcd\n0123", ci: 0},
				f: func(tex *widget.TextEditX) error {
					return MoveLineUp(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabcd\n", ci: 5},
				est: state{s: "abcd\n0123\n", ci: 0},
				f: func(tex *widget.TextEditX) error {
					return MoveLineUp(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy", si: 4, ci: 7, son: true},
				est: state{s: "ab\nzy\n01", si: 0, ci: 5, son: true},
				f: func(tex *widget.TextEditX) error {
					return MoveLineUp(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy"},
				est: state{s: "ab\n01\nzy", ci: 3},
				f: func(tex *widget.TextEditX) error {
					return MoveLineDown(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy\n", si: 0, ci: 4, son: true},
				est: state{s: "zy\n01\nab\n", si: 3, ci: 8, son: true},
				f: func(tex *widget.TextEditX) error {
					return MoveLineDown(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy", ci: 4},
				est: state{s: "01\nzy\nab", ci: 7},
				f: func(tex *widget.TextEditX) error {
					return MoveLineDown(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\n", ci: 4},
				est: state{s: "01\n\nab", ci: 5},
				f: func(tex *widget.TextEditX) error {
					return MoveLineDown(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "aaa\nbbb\n", ci: 4, si: 5, son: true},
				est: state{s: "aaa\n\nbbb", ci: 8, si: 5, son: true},
				f: func(tex *widget.TextEditX) error {
					return MoveLineDown(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "aaa\nbbb\n\n", ci: 4, si: 6, son: true},
				est: state{s: "aaa\n\n", ci: 4},
				f: func(tex *widget.TextEditX) error {
					return RemoveLines(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123401234", ci: 4},
				est: state{s: "0404", ci: 1},
				f: func(tex *widget.TextEditX) error {
					_, err := Replace(tex.TextEdit, "123", "")
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012 -- abc", ci: 4},
				est: state{s: "012 -- abc", si: 4, ci: 5, son: true},
				f: func(tex *widget.TextEditX) error {
					return SelectWord(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "--abc--", ci: 3},
				est: state{s: "--abc--", si: 2, ci: 5, son: true},
				f: func(tex *widget.TextEditX) error {
					return SelectWord(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "--abc--", ci: 5},
				est: state{s: "--abc--", si: 5, ci: 6, son: true},
				f: func(tex *widget.TextEditX) error {
					return SelectWord(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "abc\n   0123", ci: 10},
				est: state{s: "abc\n   0123", ci: 7},
				f: func(tex *widget.TextEditX) error {
					return StartOfLine(tex.TextEdit, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 0},
				est: state{s: "\t0123", ci: 1},
				f: func(tex *widget.TextEditX) error {
					return TabRight(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc\n", si: 2, ci: 6, son: true},
				est: state{s: "\t0123\n\tabc\n", si: 0, ci: 10, son: true},
				f: func(tex *widget.TextEditX) error {
					return TabRight(tex.TextEdit)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "\t0123\n\tabc\n", si: 2, ci: 8, son: true},
				est: state{s: "0123\nabc\n", si: 0, ci: 8, son: true},
				f: func(tex *widget.TextEditX) error {
					return TabLeft(tex.TextEdit)
				},
			})
		},
	}

	// secondary
	// TODO: movecursorup/movecursordown
	// TODO: copy
	// TODO: paste
	// TODO: startofstring
	// TODO: selectall
	// TODO: selectline

	testEntry = func(w *test) {
		t.Helper()

		// init
		tex := widget.NewTextEditX(nil, &cctx{})
		tex.Text.SetStr(w.st.s)
		tc := tex.TextCursor
		if w.st.son {
			tc.SetSelection(w.st.si, w.st.ci)
		} else {
			tc.SetIndex(w.st.ci)
		}

		// func error
		if err := w.f(tex); err != nil {
			t.Fatal(err)
		}
		// content
		s, err := tc.RW().ReadNCopyAt(0, tc.RW().Len())
		if err != nil {
			t.Fatal(err)
		}
		// state
		est := state{
			s:   string(s),
			ci:  tc.Index(),
			si:  tc.SelectionIndex(),
			son: tc.SelectionOn(),
		}
		if est != w.est {
			t.Fatalf("expected:\n%v\ngot:\n%v\n", w.est, est)
		}
	}

	// run tests
	for _, fn := range testFns {
		fn()
	}
}
