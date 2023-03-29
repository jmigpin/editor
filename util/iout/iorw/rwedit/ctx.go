package rwedit

import (
	"fmt"
	"image"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//godebug:annotatefile

type Ctx struct {
	RW  iorw.ReadWriterAt
	C   Cursor
	Fns CtxFns
}

func NewCtx() *Ctx {
	ctx := &Ctx{C: &SimpleCursor{}, Fns: EmptyCtxFns()}
	return ctx
}

func (ctx *Ctx) CursorSelectionLinesIndexes() (int, int, bool, error) {
	a, b, ok := ctx.C.SelectionIndexes()
	if !ok {
		a = ctx.C.Index()
		b = a
	}
	rd := ctx.LocalReader2(a, b)
	return iorw.LinesIndexes(rd, a, b)
}

func (ctx *Ctx) Selection() ([]byte, bool) {
	a, b, ok := ctx.C.SelectionIndexes()
	if !ok {
		return nil, false
	}
	w, err := ctx.RW.ReadFastAt(a, b-a)
	if err != nil {
		return nil, false
	}
	return iout.CopyBytes(w), true
}

func (ctx *Ctx) LocalReader(i int) iorw.ReaderAt {
	return ctx.LocalReader2(i, i)
}
func (ctx *Ctx) LocalReader2(min, max int) iorw.ReaderAt {
	pad := 2500
	return iorw.NewLimitedReaderAtPad(ctx.RW, min, max, pad)
}

//----------

type CtxFns struct {
	Error func(error)

	GetPoint         func(int) image.Point
	GetIndex         func(image.Point) int
	LineHeight       func() int
	CommentLineSym   func() interface{}
	MakeIndexVisible func(int)
	PageUp           func(up bool)
	ScrollUp         func(up bool)

	SetClipboardData func(event.ClipboardIndex, string)
	GetClipboardData func(event.ClipboardIndex, func(string, error)) // setter should wrap fn to run on ui goroutine

	Undo func() error
	Redo func() error
}

func EmptyCtxFns() CtxFns {
	u := CtxFns{}

	u.Error = func(err error) { fmt.Println(err) }

	u.GetPoint = func(int) image.Point { return image.ZP }
	u.GetIndex = func(image.Point) int { return 0 }
	u.LineHeight = func() int { return 0 }
	u.CommentLineSym = func() interface{} { return nil }
	u.MakeIndexVisible = func(int) {}
	u.PageUp = func(bool) {}
	u.ScrollUp = func(bool) {}

	u.SetClipboardData = func(event.ClipboardIndex, string) {}
	u.GetClipboardData = func(event.ClipboardIndex, func(string, error)) {}

	u.Undo = func() error { return nil }
	u.Redo = func() error { return nil }

	return u
}
