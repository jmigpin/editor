package godebug

import (
	"go/ast"
)

type Ctx struct {
	pctx  *Ctx
	id    ctxId
	value interface{}
}

func (ctx *Ctx) WithValue(id ctxId, value interface{}) *Ctx {
	return &Ctx{ctx, id, value}
}
func (ctx *Ctx) Value(id ctxId) (interface{}, *Ctx) {
	for c := ctx; c != nil; c = c.pctx {
		if c.id == id {
			return c.value, c
		}
	}
	return nil, nil
}
func (ctx *Ctx) SetValue(value interface{}) {
	ctx.value = value
}

//----------

func (ctx *Ctx) withBoolean(id ctxId, v bool) *Ctx {
	return ctx.WithValue(id, v)
}
func (ctx *Ctx) boolean(id ctxId) bool {
	v, _ := ctx.Value(id)
	if v == nil {
		return false
	}
	return v.(bool)
}

//----------

func (ctx *Ctx) withExpr(e *ast.Expr) *Ctx {
	return ctx.WithValue(ctxIdExpr, e)
}
func (ctx *Ctx) replaceExpr(e ast.Expr) {
	v, _ := ctx.Value(ctxIdExpr)
	if v == nil {
		panic("ctx: missing expr")
	}
	u := v.(*ast.Expr)
	*u = e
}

//----------

func (ctx *Ctx) withExprs(es *[]ast.Expr) *Ctx {
	return ctx.WithValue(ctxIdExprs, es)
}
func (ctx *Ctx) replaceExprs(es []ast.Expr) {
	v, _ := ctx.Value(ctxIdExprs)
	if v == nil {
		panic("ctx: missing exprs")
	}
	u := v.(*[]ast.Expr)
	*u = es
}

//----------

func (ctx *Ctx) withCallExpr(cep **ast.CallExpr) *Ctx {
	return ctx.WithValue(ctxIdCallExpr, cep)
}
func (ctx *Ctx) replaceCallExpr(ce *ast.CallExpr) {
	v, _ := ctx.Value(ctxIdCallExpr)
	if v == nil {
		panic("ctx: missing call expr")
	}
	u := v.(**ast.CallExpr)
	*u = ce
}

//----------

func (ctx *Ctx) withStmtsIter(si *StmtsIter) *Ctx {
	return ctx.WithValue(ctxIdStmtsIter, si)
}
func (ctx *Ctx) stmtsIter() *StmtsIter {
	v, _ := ctx.Value(ctxIdStmtsIter)
	if v == nil {
		panic("ctx: stmtsiter not set")
	}
	return v.(*StmtsIter)
}

//----------

func (ctx *Ctx) withStmts(stmts *[]ast.Stmt) *Ctx {
	si := &StmtsIter{stmts: stmts}
	return ctx.withStmtsIter(si)
}
func (ctx *Ctx) insertStmt(stmt ast.Stmt) {
	si := ctx.stmtsIter()
	if ctx.insertStmtAfter() {
		si.after++
		k := si.index + si.after
		*si.stmts = insertStmtAt(*si.stmts, k, stmt)
		return
	}
	*si.stmts = insertStmtAt(*si.stmts, si.index, stmt)
	si.index++
}
func (ctx *Ctx) replaceStmt(stmt ast.Stmt) {
	si := ctx.stmtsIter()
	(*si.stmts)[si.index] = stmt
}
func (ctx *Ctx) nilifyStmt(stmt *ast.Stmt) {
	*stmt = nil
}
func (ctx *Ctx) curStmt() ast.Stmt { // can be nil
	si := ctx.stmtsIter()
	if si.index >= len(*si.stmts) {
		return nil
	}
	return (*si.stmts)[si.index]
}
func (ctx *Ctx) nextStmt() ast.Stmt { // can be nil
	si := ctx.stmtsIter()

	// advance
	si.index += si.after + 1
	si.after = 0

	return ctx.curStmt()
}

//----------

func (ctx *Ctx) withInsertStmtAfter(v bool) *Ctx {
	return ctx.withBoolean(ctxIdStmtsIterInsertAfter, v)
}
func (ctx *Ctx) insertStmtAfter() bool {
	return ctx.boolean(ctxIdStmtsIterInsertAfter)
}

//----------

func (ctx *Ctx) withNResults(n int) *Ctx {
	return ctx.WithValue(ctxIdNResults, n)
}
func (ctx *Ctx) nResults() int {
	v, _ := ctx.Value(ctxIdNResults)
	if v == nil {
		return 0
	}
	u := v.(int)
	return u
}

//----------

func (ctx *Ctx) withDebugIndex(v int) *Ctx {
	return ctx.WithValue(ctxIdDebugIndex, &v)
}
func (ctx *Ctx) debugIndex() *int {
	v, _ := ctx.Value(ctxIdDebugIndex)
	if v == nil {
		panic("ctx: debugindex not set")
	}
	return v.(*int)
}
func (ctx *Ctx) nextDebugIndex() int {
	u := ctx.debugIndex()
	r := *u

	fdi, ok := ctx.fixedDebugIndex()
	if ok {
		if fdi.added {
			return fdi.index
		}
		fdi.added = true
		fdi.index = r
	}

	*u++
	return r
}

//----------

func (ctx *Ctx) withFixedDebugIndex() *Ctx {
	if _, ok := ctx.fixedDebugIndex(); ok {
		return ctx
	}
	v := &FixedDebugIndex{}
	return ctx.WithValue(ctxIdFixedDebugIndex, v)
}
func (ctx *Ctx) fixedDebugIndex() (*FixedDebugIndex, bool) {
	v, _ := ctx.Value(ctxIdFixedDebugIndex)
	if v == nil {
		return nil, false
	}
	return v.(*FixedDebugIndex), true
}
func (ctx *Ctx) withNilFixedDebugIndex() *Ctx {
	return ctx.WithValue(ctxIdFixedDebugIndex, nil)
}

//----------

func (ctx *Ctx) withFuncType(ft *ast.FuncType) *Ctx {
	return ctx.WithValue(ctxIdFuncType, ft)
}
func (ctx *Ctx) funcType() (*ast.FuncType, bool) {
	v, _ := ctx.Value(ctxIdFuncType)
	if v == nil {
		return nil, false
	}
	u := v.(*ast.FuncType)
	return u, true
}

//----------

func (ctx *Ctx) withTakingVarAddress(e ast.Expr) *Ctx {
	return ctx.WithValue(ctxIdTakingVarAddress, e)
}
func (ctx *Ctx) takingVarAddress() (ast.Expr, bool) {
	v, _ := ctx.Value(ctxIdTakingVarAddress)
	if v == nil {
		return nil, false
	}
	return v.(ast.Expr), true
}

//----------

func (ctx *Ctx) withTypeInsteadOfValue(e *ast.Expr) *Ctx {
	return ctx.WithValue(ctxIdTypeInsteadOfValue, e)
}
func (ctx *Ctx) typeInsteadOfValue() (*ast.Expr, bool) {
	v, _ := ctx.Value(ctxIdTypeInsteadOfValue)
	if v == nil {
		return nil, false
	}
	return v.(*ast.Expr), true
}

//----------

func (ctx *Ctx) withLabeledStmt(ls *ast.LabeledStmt) *Ctx {
	return ctx.WithValue(ctxIdLabeledStmt, ls)
}
func (ctx *Ctx) withoutLabeledStmt() *Ctx {
	return ctx.WithValue(ctxIdLabeledStmt, nil)
}
func (ctx *Ctx) labeledStmt() (*ast.LabeledStmt, bool) {
	v, _ := ctx.Value(ctxIdLabeledStmt)
	if v == nil {
		return nil, false
	}
	return v.(*ast.LabeledStmt), true
}

//----------

func (ctx *Ctx) withResetForFuncLit() *Ctx {
	ctx2 := &Ctx{} // new ctx (full reset)

	v, _ := ctx.Value(ctxIdDebugIndex)
	ctx2 = ctx2.WithValue(ctxIdDebugIndex, v)

	v2 := ctx.boolean(ctxIdNoAnnotations)
	if v2 {
		ctx2 = ctx2.withBoolean(ctxIdNoAnnotations, v2)
	}

	return ctx2
}

//----------
//----------
//----------

type ctxId int

const (
	ctxIdNone ctxId = iota
	ctxIdFuncType
	ctxIdTakingVarAddress
	ctxIdTypeInsteadOfValue
	ctxIdLabeledStmt
	ctxIdNResults // int
	ctxIdStmtsIter
	ctxIdStmtsIterInsertAfter // bool
	ctxIdExpr
	ctxIdExprs
	ctxIdCallExpr        // pointer
	ctxIdDebugIndex      // int
	ctxIdFixedDebugIndex // struct

	ctxIdExprInLhs          // bool
	ctxIdNoAnnotations      // bool
	ctxIdNameInsteadOfValue // bool
	ctxIdFirstArgIsType     // bool
	ctxIdInTypeArg          // bool
	ctxIdInDeclStmt         // bool
)

//----------

type StmtsIter struct {
	stmts *[]ast.Stmt
	index int // current
	after int // n inserted after
}

//----------

type DebugIndex struct {
	index int
}
type FixedDebugIndex struct {
	index int
	added bool
}

//----------

func insertStmtAt(ss []ast.Stmt, index int, stmt ast.Stmt) []ast.Stmt {
	if len(ss) <= index { // nil or empty slice or after last element
		return append(ss, stmt)
	}
	ss = append(ss[:index+1], ss[index:]...) // get space, index < len(a)
	ss[index] = stmt
	return ss
}
