package godebug

import (
	"fmt"
	"go/ast"
	"slices"
)

type Ctx struct {
	pctx *Ctx
	id   ctxId
	v    any

	ann *Annotator
}

func newCtx(ann *Annotator) *Ctx {
	return &Ctx{ann: ann}
}

//----------

func (ctx *Ctx) withValue(id ctxId, value any) *Ctx {
	return &Ctx{ctx, id, value, ctx.ann}
}
func (ctx *Ctx) withValue2(ids []ctxId, value any) *Ctx {
	ctx2 := ctx
	for _, id := range ids {
		ctx2 = ctx2.withValue(id, value)
	}
	return ctx2
}
func (ctx *Ctx) withInherit(match, v any, cids ...ctxId) *Ctx {
	cids2 := ctx.valueMatch3(cids, match)
	return ctx.withValue2(cids2, v)
}

//----------

func (ctx *Ctx) value(id ctxId) (any, *Ctx, bool) {
	for c := ctx; c != nil; c = c.pctx {
		if c.id == id {
			return c.v, c, true
		}
	}
	return nil, nil, false
}
func (ctx *Ctx) value2(id ctxId) (any, *Ctx) {
	v, ctx2, ok := ctx.value(id)
	if !ok {
		return nil, nil
	}
	return v, ctx2
}
func (ctx *Ctx) setValue(v any) {
	ctx.v = v
}
func (ctx *Ctx) mustValue(id ctxId) interface{} {
	v, _, ok := ctx.value(id)
	if !ok {
		err := fmt.Errorf("must value: %v", id)
		ctx.panic(err)
	}
	return v
}

//----------

func (ctx *Ctx) valueMatch(id ctxId, v any) (*Ctx, bool) {
	ctx2 := ctx
	for ctx2 != nil {
		v2, ctx3, ok := ctx2.value(id)
		if !ok {
			break
		}
		if v2 == v {
			return ctx3, true
		}
		ctx2 = ctx3.pctx
	}
	return nil, false
}
func (ctx *Ctx) valueMatch2(id ctxId, v any) bool {
	_, ok := ctx.valueMatch(id, v)
	return ok
}
func (ctx *Ctx) valueMatch3(ids []ctxId, v any) []ctxId {
	w := []ctxId{}
	for _, id := range ids {
		if ctx.valueMatch2(id, v) {
			w = append(w, id)
		}
	}
	return w
}

//----------

func (ctx *Ctx) boolean(id ctxId) bool {
	v, _, ok := ctx.boolean2(id)
	return ok && v
}
func (ctx *Ctx) boolean2(id ctxId) (bool, *Ctx, bool) {
	v, ctx2, ok := ctx.value(id)
	if !ok {
		return false, nil, false
	}
	return v.(bool), ctx2, true
}

//----------

func (ctx *Ctx) integer(id ctxId) (int, bool) {
	v, _, ok := ctx.value(id)
	if !ok {
		return 0, false
	}
	return v.(int), true
}

//----------

func (ctx *Ctx) replaceExprs(exprs ...ast.Expr) {
	if len(exprs) == 1 {
		expr := exprs[0]
		v, _, ok := ctx.value(cidnExpr)
		if !ok {
			ctx.panic("ctx: missing expr")
		}
		u := v.(*ast.Expr)
		*u = expr
		return
	}

	v, _, ok := ctx.value(cidnExprs)
	if !ok {
		ctx.panic("ctx: missing exprs")
	}
	u := v.(*[]ast.Expr)
	*u = exprs
}

//----------

func (ctx *Ctx) stmtsIter() *StmtsIter {
	v, _, ok := ctx.value(cidStmtsIter)
	if !ok {
		err := fmt.Errorf("stmtsiter not set")
		ctx.panic(err)
	}
	return v.(*StmtsIter)
}

//----------

func (ctx *Ctx) withStmts(stmts *[]ast.Stmt) *Ctx {
	si := newStmtsIter(ctx, stmts)
	return ctx.withValue(cidStmtsIter, si)
}
func (ctx *Ctx) withStmt(stmt *ast.Stmt) *Ctx {
	si := newStmtsIter2(ctx, stmt)
	return ctx.withValue(cidStmtsIter, si)
}
func (ctx *Ctx) insertStmt(stmt ast.Stmt) {
	si := ctx.stmtsIter()
	after := ctx.boolean(cidbInsertStmtAfter)
	si.insert(stmt, after)
}
func (ctx *Ctx) replaceStmt(stmt ast.Stmt) {
	si := ctx.stmtsIter()
	si.replace(stmt)
}

//----------

func (ctx *Ctx) stmtVisited(stmt ast.Stmt) bool {
	_, ok := ctx.ann.ctxData.visited[stmt]
	return ok
}
func (ctx *Ctx) setStmtVisited(stmt ast.Stmt, v bool) {
	if v {
		ctx.ann.ctxData.visited[stmt] = struct{}{}
	} else {
		delete(ctx.ann.ctxData.visited, stmt)
	}
}

//----------

func (ctx *Ctx) getDebugIndex() int {
	v, _, ok := ctx.value(cidiFixedDebugIndex)
	if ok {
		u := v.(int)
		if u >= 0 {
			return u
		}
	}
	return ctx.nextDebugIndex()
}
func (ctx *Ctx) nextDebugIndex() int {
	u := ctx.ann.ctxData.debugIndex
	ctx.ann.ctxData.debugIndex++
	return u
}
func (ctx *Ctx) withFixedDebugIndex(fixed bool) *Ctx {
	index := -1 // allows reset if already existed
	if fixed {
		index = ctx.nextDebugIndex() // new unique index
	}
	return ctx.withValue(cidiFixedDebugIndex, index)
}

//----------

func (ctx *Ctx) withResetForFuncLit() *Ctx {
	ctx2 := newCtx(ctx.ann) // full reset
	ctx2 = ctx2.withNoAnnotationsInstance2(ctx)
	return ctx2
}

//----------

func (ctx *Ctx) funcNode() (ast.Node, *ast.FuncType, *ast.BlockStmt) {
	v2 := ctx.mustValue(cidnFuncNode)
	switch t := v2.(type) {
	case *ast.FuncLit:
		return t, t.Type, t.Body
	case *ast.FuncDecl:
		return t, t.Type, t.Body
	default:
		panic("expecting func node")
	}
}

//----------

func (ctx *Ctx) withNoAnnotationsInstance() *Ctx {
	return ctx.withNoAnnotationsInstance2(ctx)
}
func (ctx *Ctx) withNoAnnotationsInstance2(ctx0 *Ctx) *Ctx {
	// ensure noannotations instance; useful to inherit value and allow to be changed only for this block
	v := ctx0.boolean(cidbNoBlockAnnotations)
	return ctx.withValue(cidbNoBlockAnnotations, v)
}
func (ctx *Ctx) withNoAnnotationsUpdated(node ast.Node) *Ctx {
	if on, ok := ctx.ann.nodeAnnotationBlockOn(node); ok {
		_, ctx2, ok2 := ctx.boolean2(cidbNoBlockAnnotations)
		if ok2 {
			ctx2.setValue(!on) // change value, keep ctx
		} else {
			ctx = ctx.withValue(cidbNoBlockAnnotations, !on)
		}
	}
	return ctx
}

//----------

func (ctx *Ctx) panic(v any) error {
	s := fmt.Sprint(v)
	if u, ok := ctx.curStmtSrc(); ok {
		s += "\n" + u
	}
	panic(s)
}
func (ctx *Ctx) curStmtSrc() (string, bool) {
	if v, _, ok := ctx.value(cidStmtsIter); ok {
		si := v.(*StmtsIter)
		stmt := (ast.Stmt)(nil)
		if si.stmts != nil {
			i := si.index - 1
			if i < len(*si.stmts) {
				stmt = (*si.stmts)[i]
			}
		} else if si.stmt != nil {
			stmt = *si.stmt
		}
		if stmt != nil {
			s := ctx.ann.debug2(stmt)
			//err = fmt.Errorf("%w:\n%v", err, s)
			return s, true
		}
	}
	return "", false
}

//----------
//----------
//----------

type ctxId int

const (
	cidNone ctxId = iota

	cidStmtsIter // struct

	cidbNoBlockAnnotations
	cidiSliceExprsLimit
	cidiFixedDebugIndex
	cidbInsertStmtAfter

	cidnExpr
	cidnExprs
	cidnFuncNode
	cidnNameInsteadOfValue

	cidnResNil
	cidnResAssignDebugToVar
	cidnResNotReplaceable
	cidnResReplaceWithVar

	cidnIsConstSpec
	cidnIsExprStmtExpr
	cidnIsTypeSwitchStmtAssign
	cidnIsCaseClauseListItem
	cidnIsLabeledStmtStmt
	cidnIsCallExprFun
)

//----------
//----------
//----------

type StmtsIter struct {
	index int // current stmt index
	stmts *[]ast.Stmt
	stmt  *ast.Stmt // meaningful if stmts is nil
	ctx   *Ctx
}

func newStmtsIter(ctx *Ctx, stmts *[]ast.Stmt) *StmtsIter {
	return &StmtsIter{ctx: ctx, stmts: stmts}
}
func newStmtsIter2(ctx *Ctx, stmt *ast.Stmt) *StmtsIter {
	return &StmtsIter{ctx: ctx, stmt: stmt}
}
func (si *StmtsIter) iterate(fn func(ast.Stmt) error) error {
	for ; si.index < len(*si.stmts); si.index++ {
		if err := fn((*si.stmts)[si.index]); err != nil {
			return err
		}
	}
	return nil
}
func (si *StmtsIter) replace(stmt ast.Stmt) {
	if si.stmts == nil || len(*si.stmts) == 0 {
		if si.stmt == nil {
			err := fmt.Errorf("replace: len(stmts)=0 && stmt=nil")
			panic(err)
		}
		*si.stmt = stmt
	} else {
		(*si.stmts)[si.index] = stmt
	}
	si.ctx.setStmtVisited(stmt, true)
}
func (si *StmtsIter) insert(stmt ast.Stmt, after bool) {
	if si.stmts == nil {
		err := fmt.Errorf("insert: stmts=nil")
		panic(err)
	}

	k := si.index
	if after {
		k++
	} else {
		si.index++
	}
	*si.stmts = slices.Insert(*si.stmts, k, stmt)
	si.ctx.setStmtVisited(stmt, true)
}
