package rwedit

import (
	"context"
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestAll1(t *testing.T) {
	type state struct {
		s              string
		ci             int  // cursor index
		si             int  // selected index
		son            bool // selection on
		commentLineSym any
	}
	type test struct {
		st  state
		est state // expected state
		f   func(*Ctx) error
	}

	var testEntry func(*test)

	var testFns = []func(){
		func() {
			testEntry(&test{
				st:  state{s: "123\nabc\n", ci: 0},
				est: state{s: "123\n123\nabc\n", si: 4, ci: 7, son: true},
				f:   DuplicateLines,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "123", ci: 0},
				est: state{s: "123\n123", si: 4, ci: 7, son: true},
				f:   DuplicateLines,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "   123", ci: 4},
				est: state{s: "   1\n   23", ci: 8},
				f:   func(ctx *Ctx) error { return AutoIndent(ctx) },
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "   123", ci: 4, si: 5, son: true},
				est: state{s: "   1\n   3", ci: 8, son: false},
				f:   AutoIndent,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n   abc", ci: 10},
				est: state{s: "0123\n   ab\n   c", ci: 14, son: false},
				f:   AutoIndent,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n   abc", ci: 6},
				est: state{s: "0123\n \n   abc", ci: 8, son: false},
				f:   AutoIndent,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "123", ci: 2},
				est: state{s: "13", ci: 1},
				f:   Backspace,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   Backspace,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234"},
				est: state{s: "//01234", ci: 2},
				f:   Comment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n  abc\n efg\n", si: 9, ci: 14, son: true},
				est: state{s: "0123\n // abc\n //efg\n", si: 5, ci: 19, son: true},
				f:   Comment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc", ci: 3, commentLineSym: [2]string{"/*", "*/"}},
				est: state{s: "/*0123*/\nabc", ci: 5},
				f:   Comment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc", si: 1, ci: 5, son: true, commentLineSym: [2]string{"/*", "*/"}},
				est: state{s: "/*0123*/\n/*abc*/", si: 0, ci: 16, son: true},
				f:   Comment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n // abc\n //efg\n", si: 5, ci: 19, son: true},
				est: state{s: "0123\n  abc\n efg\n", si: 5, ci: 15, son: true},
				f:   Uncomment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\n <!-- abc --> \n \n<!--efg-->\n", si: 2, ci: 23, son: true, commentLineSym: [2]string{"<!--", "-->"}},
				est: state{s: "0123\n  abc  \n \nefg\n", si: 0, ci: 18, son: true},
				f:   Uncomment,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   Cut,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2, si: 4, son: true},
				est: state{s: "014", ci: 2},
				f:   Delete,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234", ci: 2},
				est: state{s: "0134", ci: 2},
				f:   Delete,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc", ci: 7},
				est: state{s: "0123\nabc", ci: 5},
				f: func(ctx *Ctx) error {
					return StartOfLine(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc"},
				est: state{s: "0123\nabc", ci: 4},
				f: func(ctx *Ctx) error {
					return EndOfLine(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc", ci: 6},
				est: state{s: "0123\nabc", si: 6, ci: 8, son: true},
				f: func(ctx *Ctx) error {
					return EndOfLine(ctx, true)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012"},
				est: state{s: "012", ci: 3},
				f: func(ctx *Ctx) error {
					return EndOfLine(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012\nabc", ci: 7},
				est: state{s: "012\nabc", ci: 0},
				f: func(ctx *Ctx) error {
					StartOfString(ctx, false)
					return nil
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012\nabc"},
				est: state{s: "012\nabc", ci: 7},
				f: func(ctx *Ctx) error {
					EndOfString(ctx, false)
					return nil
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234\nabc"},
				est: state{s: "01234\nabc", si: 6, ci: 8, son: true},
				f: func(ctx *Ctx) error {
					cctx := context.Background()
					opt := &iorw.IndexOpt{}
					_, err := Find(cctx, ctx, "ab", false, opt)
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01234\nabc", ci: 7},
				est: state{s: "01234\nabc", si: 6, ci: 8, son: true},
				f: func(ctx *Ctx) error {
					cctx := context.Background()
					opt := &iorw.IndexOpt{}
					_, err := Find(cctx, ctx, "ab", false, opt)
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "01ab23", ci: 4},
				f: func(ctx *Ctx) error {
					return InsertString(ctx, "ab")
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "0123", ci: 1},
				f: func(ctx *Ctx) error {
					return MoveCursorLeft(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 2},
				est: state{s: "0123", ci: 3},
				f: func(ctx *Ctx) error {
					return MoveCursorRight(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: " 0123 ", ci: 3},
				est: state{s: " 0123 ", ci: 5},
				f: func(ctx *Ctx) error {
					return MoveCursorJumpRight(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabcd", ci: 5},
				est: state{s: "abcd\n0123", ci: 0},
				f:   MoveLineUp,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabcd\n", ci: 5},
				est: state{s: "abcd\n0123\n", ci: 0},
				f:   MoveLineUp,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy", si: 4, ci: 7, son: true},
				est: state{s: "ab\nzy\n01", si: 0, ci: 5, son: true},
				f:   MoveLineUp,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy"},
				est: state{s: "ab\n01\nzy", ci: 3},
				f:   MoveLineDown,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy\n", si: 0, ci: 4, son: true},
				est: state{s: "zy\n01\nab\n", si: 3, ci: 8, son: true},
				f:   MoveLineDown,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\nzy", ci: 4},
				est: state{s: "01\nzy\nab", ci: 7},
				f:   MoveLineDown,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "01\nab\n", ci: 4},
				est: state{s: "01\n\nab", ci: 5},
				f:   MoveLineDown,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "aaa\nbbb\n", ci: 4, si: 5, son: true},
				est: state{s: "aaa\n\nbbb", ci: 8, si: 5, son: true},
				f:   MoveLineDown,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "aaa\nbbb\n\n", ci: 4, si: 6, son: true},
				est: state{s: "aaa\n\n", ci: 4},
				f:   RemoveLines,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123401234", ci: 4},
				est: state{s: "0404", ci: 1},
				f: func(ctx *Ctx) error {
					_, err := Replace(ctx, "123", "")
					return err
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "012 -- abc", ci: 4},
				est: state{s: "012 -- abc", si: 4, ci: 5, son: true},
				f:   SelectWord,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "--abc--", ci: 3},
				est: state{s: "--abc--", si: 2, ci: 5, son: true},
				f:   SelectWord,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "--abc--", ci: 5},
				est: state{s: "--abc--", si: 5, ci: 6, son: true},
				f:   SelectWord,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "abc\n   0123", ci: 10},
				est: state{s: "abc\n   0123", ci: 7},
				f: func(ctx *Ctx) error {
					return StartOfLine(ctx, false)
				},
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123", ci: 0},
				est: state{s: "\t0123", ci: 1},
				f:   TabRight,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "0123\nabc\n", si: 2, ci: 6, son: true},
				est: state{s: "\t0123\n\tabc\n", si: 0, ci: 10, son: true},
				f:   TabRight,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "\t0123\n\tabc\n", si: 2, ci: 8, son: true},
				est: state{s: "0123\nabc\n", si: 0, ci: 8, son: true},
				f:   TabLeft,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "\t0123\n\tabc\n", ci: 4},
				est: state{s: "\t0123\n\tabc\n", si: 0, ci: 11, son: true},
				f:   SelectAll,
			})
		},
		func() {
			testEntry(&test{
				st:  state{s: "\t0123\n\tabc\n", ci: 8},
				est: state{s: "\t0123\n\tabc\n", ci: 11, si: 6, son: true},
				f:   SelectLine,
			})
		},
	}

	// TODO: movecursorup/movecursordown
	// TODO: copy
	// TODO: paste

	testEntry = func(w *test) {
		t.Helper()

		// init
		ctx := NewCtx()
		ctx.Fns = EmptyCtxFns()
		cls := w.st.commentLineSym
		if cls == nil {
			cls = "//"
		}
		ctx.Fns.CommentLineSym = func() any { return cls }
		ctx.RW = iorw.NewBytesReadWriterAt([]byte(w.st.s))

		if w.st.son {
			ctx.C.SetSelection(w.st.si, w.st.ci)
		} else {
			ctx.C.SetIndex(w.st.ci)
		}

		// func error
		if err := w.f(ctx); err != nil {
			t.Fatal(err)
		}
		// content
		b, err := iorw.ReadFastFull(ctx.RW)
		if err != nil {
			t.Fatal(err)
		}
		// state
		est := state{
			s:   string(b),
			ci:  ctx.C.Index(),
			si:  ctx.C.SelectionIndex(),
			son: ctx.C.HaveSelection(),
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
