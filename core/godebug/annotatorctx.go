package godebug

import "go/ast"

type Ctx struct {
	pctx  *Ctx
	vname string
	value interface{}
}

func (ctx *Ctx) WithValue(vname string, value interface{}) *Ctx {
	return &Ctx{pctx: ctx, vname: vname, value: value}
}

func (ctx *Ctx) Value(vname string) (interface{}, *Ctx) {
	for c := ctx; c != nil; c = c.pctx {
		if c.vname == vname {
			return c.value, c
		}
	}
	return nil, nil
}

func (ctx *Ctx) SetValue(vname string, value interface{}) {
	ctx.vname = vname
	ctx.value = value
}

//----------

type StmtIter struct {
	list        *[]ast.Stmt
	index, step int
}

func (ctx *Ctx) withStmtIter(list *[]ast.Stmt) (*Ctx, *StmtIter) {
	iter := &StmtIter{list: list}
	ctx2 := ctx.WithValue("stmt_iter", iter)
	return ctx2, iter
}

func (ctx *Ctx) stmtIter() (*StmtIter, bool) {
	v, _ := ctx.Value("stmt_iter")
	if v == nil {
		return nil, false
	}
	si := v.(*StmtIter)
	return si, true
}

func (ctx *Ctx) replaceStmt(stmt ast.Stmt) {
	iter, ok := ctx.stmtIter()
	if !ok {
		return
	}
	i := iter.index
	(*iter.list)[i] = stmt
}

func (ctx *Ctx) insertInStmtList(stmt ast.Stmt) {
	after := ctx.insertAfterStmt()
	if after {
		ctx.insertInStmtListAfter(stmt)
	} else {
		ctx.insertInStmtListBefore(stmt)
	}
}

func (ctx *Ctx) insertInStmtListBefore(stmt ast.Stmt) {
	iter, ok := ctx.stmtIter()
	if !ok {
		return
	}
	*iter.list = append(*iter.list, nil)
	i := iter.index
	copy((*iter.list)[i+1:], (*iter.list)[i:])
	(*iter.list)[i] = stmt
	iter.index++
}
func (ctx *Ctx) insertInStmtListAfter(stmt ast.Stmt) {
	iter, ok := ctx.stmtIter()
	if !ok {
		return
	}
	*iter.list = append(*iter.list, nil)
	i := iter.index + 1 + iter.step
	if i < len(*iter.list) {
		copy((*iter.list)[i+1:], (*iter.list)[i:])
	}
	(*iter.list)[i] = stmt
	iter.step++
}

//----------

func (ctx *Ctx) withInsertStmtAfter(after bool) *Ctx {
	return ctx.WithValue("insert_in_stmt_list_after", after)
}
func (ctx *Ctx) insertAfterStmt() bool {
	v, _ := ctx.Value("insert_in_stmt_list_after")
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}

//----------

type ExprIter struct {
	list        *[]ast.Expr
	index, step int
}

func (ctx *Ctx) withExprIter(list *[]ast.Expr) (*Ctx, *ExprIter) {
	iter := &ExprIter{list: list}
	ctx2 := ctx.WithValue("expr_iter", iter)
	return ctx2, iter
}

func (ctx *Ctx) exprIter() (*ExprIter, bool) {
	v, _ := ctx.Value("expr_iter")
	if v == nil {
		return nil, false
	}
	u := v.(*ExprIter)
	return u, true
}

//----------

func (ctx *Ctx) withExprPtr(exprPtr *ast.Expr) *Ctx {
	return ctx.WithValue("expr_ptr", exprPtr)
}

//----------

func (ctx *Ctx) replaceExpr(expr ast.Expr) {
	v, _ := ctx.Value("expr_ptr")
	if v == nil {
		return
	}
	u := v.(*ast.Expr)
	*u = expr
}

func (ctx *Ctx) replaceExprs(exprs []ast.Expr) {
	iter, ok := ctx.exprIter()
	if !ok {
		return
	}
	*iter.list = exprs
	// advance to end
	iter.index = len(exprs)
}

//----------

func (ctx *Ctx) withNResults(n int) *Ctx {
	return ctx.WithValue("nresults", n)
}
func (ctx *Ctx) nResults() int {
	v, _ := ctx.Value("nresults")
	if v == nil {
		return 0
	}
	u := v.(int)
	return u
}

//----------

func (ctx *Ctx) withCallExprDebugIndex() *Ctx {
	return ctx.WithValue("call_expr_debug_index", -1)
}

func (ctx *Ctx) setupCallExprDebugIndex(ann *Annotator) *Ctx {
	v, ctx2 := ctx.Value("call_expr_debug_index")
	if v == nil {
		return ctx
	}
	u := v.(int)
	if u == -2 {
		return ctx
	}
	if u == -1 {
		i := ann.debugIndex
		ann.debugIndex++
		ctx2.SetValue("call_expr_debug_index", i)
		return ctx.withStaticDebugIndex(i)
	}
	return ctx.withStaticDebugIndex(u)
}

func (ctx *Ctx) callExprDebugIndex() *Ctx {
	v, _ := ctx.Value("call_expr_debug_index")
	if v == nil {
		return ctx
	}
	u := v.(int)
	if u < 0 {
		return ctx
	}
	return ctx.withStaticDebugIndex(u)
}

//----------

func (ctx *Ctx) withStaticDebugIndex(v int) *Ctx {
	return ctx.WithValue("static_debug_index", v)
}

func (ctx *Ctx) withNoStaticDebugIndex() *Ctx {
	return ctx.WithValue("static_debug_index", -1)
}

func (ctx *Ctx) staticDebugIndex() (int, bool) {
	v, _ := ctx.Value("static_debug_index")
	if v == nil {
		return 0, false
	}
	u := v.(int)
	if u < 0 {
		return 0, false
	}
	return u, true
}

//----------

func (ctx *Ctx) withResultInVar() *Ctx {
	return ctx.WithValue("result_in_var", true)
}
func (ctx *Ctx) withNoResultInVar() *Ctx {
	return ctx.WithValue("result_in_var", nil)
}
func (ctx *Ctx) resultInVar() bool {
	v, _ := ctx.Value("result_in_var")
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}

//----------

func (ctx *Ctx) withFuncType(ft *ast.FuncType) *Ctx {
	return ctx.WithValue("func_type", ft)
}

func (ctx *Ctx) funcType() (*ast.FuncType, bool) {
	v, _ := ctx.Value("func_type")
	if v == nil {
		return nil, false
	}
	u := v.(*ast.FuncType)
	return u, true
}

//----------

func (ctx *Ctx) withAssignStmtIgnoreLhs() *Ctx {
	return ctx.WithValue("assign_stmt_ignore_lhs", true)
}
func (ctx *Ctx) withNoAssignStmtIgnoreLhs() *Ctx {
	return ctx.WithValue("assign_stmt_ignore_lhs", nil)
}
func (ctx *Ctx) assignStmtIgnoreLhs() bool {
	v, _ := ctx.Value("assign_stmt_ignore_lhs")
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}

//----------

func (ctx *Ctx) withNewExprs() *Ctx {
	u := []ast.Expr{}
	return ctx.WithValue("exprs", &u)
}
func (ctx *Ctx) pushExprs(e ...ast.Expr) {
	v, _ := ctx.Value("exprs")
	if v == nil {
		return
	}
	u := v.(*[]ast.Expr)
	*u = append(*u, e...)
}
func (ctx *Ctx) popExprs() []ast.Expr {
	v, _ := ctx.Value("exprs")
	if v == nil {
		return nil
	}
	u := v.(*[]ast.Expr)
	r := *u
	*u = []ast.Expr{}
	return r
}

//----------

func (ctx *Ctx) withFirstArgIsType() *Ctx {
	return ctx.WithValue("first_arg_is_type", true)
}

func (ctx *Ctx) firstArgIsType() bool {
	v, _ := ctx.Value("first_arg_is_type")
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}

//----------

func (ctx *Ctx) valuesReset() *Ctx {
	ctx = ctx.WithValue("nresults", nil)
	ctx = ctx.WithValue("call_expr_debug_index", nil)
	ctx = ctx.WithValue("static_debug_index", nil)
	ctx = ctx.WithValue("result_in_var", nil)
	ctx = ctx.WithValue("assign_stmt_ignore_lhs", nil)
	ctx = ctx.WithValue("first_arg_is_type", nil)
	return ctx
}

//----------

func (ctx *Ctx) withNoAnnotationsFalse() *Ctx {
	return ctx.WithValue("no_annotations", false)
}

func (ctx *Ctx) setUpperNoAnnotationsTrue() {
	_, ctx2 := ctx.Value("no_annotations")
	ctx2.SetValue("no_annotations", true)
}

func (ctx *Ctx) noAnnotations() bool {
	v, _ := ctx.Value("no_annotations")
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}
