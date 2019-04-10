package textutil

import (
	"bytes"
	"context"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Find(ctx context.Context, te *widget.TextEdit, str string) (bool, error) {
	if str == "" {
		return false, nil
	}

	// ignore case
	lowb := bytes.ToLower([]byte(str))

	tc := te.TextCursor
	i, err := find2(ctx, tc, lowb) // ignores case
	if err != nil || i < 0 {
		return false, err
	}
	tc.SetSelection(i, i+len(lowb)) // cursor at end to allow searching next
	return true, nil
}

func find2(ctx context.Context, tc *widget.TextCursor, b []byte) (int, error) {
	ci := tc.Index()
	l := tc.RW().Max()

	// index to end
	i, err := iorw.IndexCtx(ctx, tc.RW(), ci, b, true)
	if err != nil || i >= 0 {
		return i, err
	}

	// start to index
	e := ci + len(b) - 1
	if e > l {
		e = l
	}
	rd := iorw.NewLimitedReaderLen(tc.RW(), 0, e)
	k, err := iorw.IndexCtx(ctx, rd, 0, b, true)
	if err != nil {
		if err == iorw.ErrLimitReached {
			return -1, nil
		}
		return -1, err
	}
	return k, nil
}
