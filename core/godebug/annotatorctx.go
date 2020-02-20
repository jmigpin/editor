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

//func (ctx *Ctx) SetUpperValue(vname string, v interface{}) {
//	_, ctx2 := ctx.Value(vname)
//	ctx2.SetValue(vname, v)
//}

//----------

func (ctx *Ctx) WithBool(name string, v bool) *Ctx {
	return ctx.WithValue(name, v)
}
func (ctx *Ctx) ValueBool(name string) bool {
	v, _ := ctx.Value(name)
	if v == nil {
		return false
	}
	u := v.(bool)
	return u
}

//----------

// On avoiding visiting inserted debug stmts:
// - Create the debug stmts on a blockstmt (will keep debug index)
// - visit the stmt list (will not visit the blockstmt - not inserted)
// - insert the created blockstmt at the top

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

//----------

func (ctx *Ctx) replaceStmt(stmt ast.Stmt) { // TODO: rename replaceInStmtList
	if ctx.noAnnotations() {
		return
	}

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
	if ctx.noAnnotations() {
		return
	}

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
	if ctx.noAnnotations() {
		return
	}

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
	return ctx.WithBool("insert_in_stmt_list_after", after)
}
func (ctx *Ctx) insertAfterStmt() bool {
	return ctx.ValueBool("insert_in_stmt_list_after")
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
	if ctx.noAnnotations() {
		return
	}

	v, _ := ctx.Value("expr_ptr")
	if v == nil {
		return
	}
	u := v.(*ast.Expr)
	*u = expr
}

func (ctx *Ctx) replaceExprs(exprs []ast.Expr) {
	if ctx.noAnnotations() {
		return
	}

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
	return ctx.WithValue("n_results", n)
}
func (ctx *Ctx) nResults() int {
	v, _ := ctx.Value("n_results")
	if v == nil {
		return 0
	}
	u := v.(int)
	return u
}

//----------

func (ctx *Ctx) withStaticDebugIndex(v int) *Ctx {
	return ctx.WithValue("static_debug_index", v)
}

func (ctx *Ctx) withNoStaticDebugIndex() *Ctx {
	return ctx.WithValue("static_debug_index", nil)
}

func (ctx *Ctx) staticDebugIndex() (int, bool) {
	v, _ := ctx.Value("static_debug_index")
	if v == nil {
		return 0, false
	}
	u := v.(int)
	return u, true
}

func (ctx *Ctx) setUpperStaticDebugIndex(v int) {
	_, ctx2 := ctx.Value("static_debug_index")
	if ctx2 != nil {
		ctx2.SetValue("static_debug_index", v)
	}
}
func (ctx *Ctx) setUpperStaticDebugIndexToNil() {
	_, ctx2 := ctx.Value("static_debug_index")
	if ctx2 != nil {
		ctx2.SetValue("static_debug_index", nil)
	}
}

//----------

func (ctx *Ctx) withKeepDebugIndex() *Ctx {
	return ctx.WithBool("keep_debug_index", true)
}
func (ctx *Ctx) keepDebugIndex() bool {
	return ctx.ValueBool("keep_debug_index")
}

//----------

func (ctx *Ctx) withResultInVar(in bool) *Ctx {
	return ctx.WithBool("result_in_var", in)
}
func (ctx *Ctx) resultInVar() bool {
	return ctx.ValueBool("result_in_var")
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
	return ctx.WithBool("first_arg_is_type", true)
}
func (ctx *Ctx) firstArgIsType() bool {
	return ctx.ValueBool("first_arg_is_type")
}

//----------

func (ctx *Ctx) withNoAnnotations(v bool) *Ctx {
	return ctx.WithBool("no_annotations", v)
}
func (ctx *Ctx) noAnnotations() bool {
	return ctx.ValueBool("no_annotations")
}

//----------

func (ctx *Ctx) withLabeledStmt(stmt ast.Stmt) *Ctx {
	return ctx.WithValue("labeled_stmt", stmt)
}
func (ctx *Ctx) labeledStmt() (*ast.LabeledStmt, bool) {
	v, _ := ctx.Value("labeled_stmt")
	if v == nil {
		return nil, false
	}
	u := v.(*ast.LabeledStmt)
	return u, true
}

//----------

func (ctx *Ctx) valuesReset() *Ctx {
	ctx = ctx.WithValue("n_results", nil)
	ctx = ctx.WithValue("static_debug_index", nil)
	ctx = ctx.WithValue("result_in_var", nil)
	ctx = ctx.WithValue("assign_stmt_ignore_lhs", nil)
	ctx = ctx.WithValue("first_arg_is_type", nil)
	ctx = ctx.WithValue("labeled_stmt", nil)

	// Not reset:
	// 	no_annotations
	// 	exprs
	// 	expr_ptr
	// 	expr_iter
	// 	stmt_iter
	// 	insert_in_stmt_list_after
	// 	func_type

	return ctx
}
