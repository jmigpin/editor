package rwedit

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/jmigpin/editor/v2/util/iout/iorw"
)

func Find(cctx context.Context, ctx *Ctx, str string) (bool, error) {
	if str == "" {
		return false, nil
	}

	// ignore case
	lowb := bytes.ToLower([]byte(str))

	i, err := find2(cctx, ctx, lowb) // ignores case
	if err != nil || i < 0 {
		return false, err
	}
	ctx.C.SetSelection(i, i+len(lowb)) // cursor at end to allow searching next
	return true, nil
}

func find2(cctx context.Context, ctx *Ctx, b []byte) (int, error) {
	ci := ctx.C.Index()

	// index to end
	i, err := iorw.IndexCtx(cctx, ctx.RW, ci, b, true)
	if err != nil || i >= 0 {
		return i, err
	}

	// start to index
	e := ci + len(b) - 1
	rd := iorw.NewLimitedReaderAt(ctx.RW, 0, e)
	k, err := iorw.IndexCtx(cctx, rd, 0, b, true)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, nil
		}
		return -1, err
	}
	return k, nil
}
