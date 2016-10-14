package edit

import (
	"context"
	"sync"

	"github.com/jmigpin/editor/ui"
)

var rowCtx = &RowCtx{m: make(map[*ui.Row]*RowCtxData)}

// processes running in rows, the context allows canceling
type RowCtx struct {
	sync.Mutex
	m map[*ui.Row]*RowCtxData
}
type RowCtxData struct {
	ctx    context.Context
	cancel context.CancelFunc
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
func (rctx *RowCtx) ClearIfCtx(row *ui.Row, ctx context.Context) {
	rctx.Lock()
	defer rctx.Unlock()
	e, ok := rctx.m[row]
	if !ok {
		return
	}
	if e.ctx == ctx {
		delete(rctx.m, row)
	}
}
