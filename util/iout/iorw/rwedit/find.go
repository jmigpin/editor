package rwedit

import (
	"context"
	"errors"
	"io"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func Find(cctx context.Context, ectx *Ctx, str string, reverse bool, opt *iorw.IndexOpt) (bool, error) {
	if str == "" {
		return false, nil
	}
	if reverse {
		i, n, err := find2Rev(cctx, ectx, []byte(str), opt)
		if err != nil || i < 0 {
			return false, err
		}
		ectx.C.SetSelection(i+n, i) // cursor at start to allow searching next
	} else {
		i, n, err := find2(cctx, ectx, []byte(str), opt)
		if err != nil || i < 0 {
			return false, err
		}
		ectx.C.SetSelection(i, i+n) // cursor at end to allow searching next
	}

	return true, nil
}
func find2(cctx context.Context, ectx *Ctx, b []byte, opt *iorw.IndexOpt) (int, int, error) {
	ci := ectx.C.Index()
	// index to end
	i, n, err := iorw.IndexCtx(cctx, ectx.RW, ci, b, opt)
	if err != nil || i >= 0 {
		return i, n, err
	}
	// start to index
	e := ci + len(b) - 1
	rd := iorw.NewLimitedReaderAt(ectx.RW, 0, e)
	k, n, err := iorw.IndexCtx(cctx, rd, 0, b, opt)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, 0, nil
		}
		return -1, 0, err
	}
	return k, n, nil
}
func find2Rev(cctx context.Context, ectx *Ctx, b []byte, opt *iorw.IndexOpt) (int, int, error) {
	ci := ectx.C.Index()
	// start to index (in reverse)
	i, n, err := iorw.LastIndexCtx(cctx, ectx.RW, ci, b, opt)
	if err != nil || i >= 0 {
		return i, n, err
	}
	// index to end (in reverse)
	s := ci - len(b) + 1
	e := ectx.RW.Max()
	rd2 := iorw.NewLimitedReaderAt(ectx.RW, s, e)
	k, n, err := iorw.LastIndexCtx(cctx, rd2, e, b, opt)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, 0, nil
		}
		return -1, 0, err
	}
	return k, n, nil
}
