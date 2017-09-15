package cmdutil

import (
	"context"
	"sync"

	"github.com/jmigpin/editor/ui"
)

// Cancels processes running in rows.
type RowCtx struct {
	sync.Mutex
	m map[*ui.Row]*RowCtxData
}

func NewRowCtx() *RowCtx {
	return &RowCtx{m: make(map[*ui.Row]*RowCtxData)}
}
func (rctx *RowCtx) Add(row *ui.Row, ctx context.Context) context.Context {
	rctx.Lock()
	defer rctx.Unlock()
	_, ok := rctx.m[row]
	if ok {
		panic("entry already exists")
	}
	ctx2, cancel := context.WithCancel(ctx)
	rctx.m[row] = &RowCtxData{ctx2, cancel}
	return ctx2
}
func (rctx *RowCtx) Cancel(row *ui.Row) {
	rctx.Lock()
	defer rctx.Unlock()
	e, ok := rctx.m[row]
	if !ok {
		return
	}
	e.cancel()
	delete(rctx.m, row)
}
func (rctx *RowCtx) ClearIfNotNewCtx(row *ui.Row, ctx context.Context, fn func()) {
	rctx.Lock()
	defer rctx.Unlock()
	e, ok := rctx.m[row]
	if !ok {
		fn()
		return
	}
	if e.ctx == ctx {
		delete(rctx.m, row)
		fn()
	}
}

type RowCtxData struct {
	ctx    context.Context
	cancel context.CancelFunc
}

var gRowCtx = NewRowCtx()

func RowCtxCancel(row *ui.Row) {
	gRowCtx.Cancel(row)
}
