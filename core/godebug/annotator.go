package godebug

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"math/big"
	"strings"
)

type Annotator struct {
	fset               *token.FileSet
	file               *SrcFile
	debugPkgName       string
	debugVarPrefix     string
	debugVarNameIndex  int
	fileIndex          int
	debugIndex         int
	builtDebugLineStmt bool
}

func NewAnnotator(fset *token.FileSet, f *SrcFile) *Annotator {
	ann := &Annotator{fset: fset, file: f}
	ann.debugPkgName = string('Σ')
	ann.debugVarPrefix = string('Σ')
	return ann
}

//----------

func (ann *Annotator) AnnotateAstFile(astFile *ast.File) {
	ctx := &Ctx{}

	// if annotating only blocks, start with annotations off
	if ann.file.annType == AnnotationTypeBlock {
		ctx = ctx.withNoAnnotations(true)
	}

	ann.visitFile(ctx, astFile)
	ann.removeInnerFuncComments(astFile)
}

//----------

func (ann *Annotator) removeInnerFuncComments(astFile *ast.File) {
	// ensure comments are not in between stmts in the middle of declarations inside functions (solves test100)
	// Other comments stay in place since they might be needed (build comments, "c" package comments, ...)
	u := []*ast.CommentGroup{}
	for _, cg := range astFile.Comments { // all comments
		for _, d := range astFile.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				if fd.Pos() > cg.End() { // passed comment
					break
				}
				inside := cg.Pos() >= fd.Pos() && cg.Pos() < fd.End()
				if !inside {
					u = append(u, cg)
				}
			}
		}
	}
	astFile.Comments = u
}

//----------

func (ann *Annotator) sprintNode(n ast.Node) (string, error) {
	buf := &bytes.Buffer{}
	cfg := &printer.Config{Mode: printer.RawFormat}
	if err := cfg.Fprint(buf, ann.fset, n); err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

func (ann *Annotator) printNode(n ast.Node) {
	s, err := ann.sprintNode(n)
	if err != nil {
		s = err.Error()
	}
	fmt.Println(s)
}

//----------

func (ann *Annotator) visitFile(ctx *Ctx, file *ast.File) {
	for _, d := range file.Decls {
		ann.visitDeclFromFile(ctx, d)
	}
}

func (ann *Annotator) visitDeclFromFile(ctx *Ctx, decl ast.Decl) {
	switch t := decl.(type) {
	case *ast.FuncDecl:
		ann.visitFuncDecl(ctx, t)
	case *ast.GenDecl:
		// do nothing
	default:
		fmt.Printf("todo: decl: %T\n", t)
	}
}

func (ann *Annotator) visitFuncDecl(ctx *Ctx, fd *ast.FuncDecl) {
	// catch directive at the top of the function decl
	on, ok := ann.annotationsOn(fd)
	if ok && !on {
		ctx = ctx.withNoAnnotations(!on)
	}

	// don't annotate these functions to avoid endless loop recursion
	noAnnSpecialFunc := false
	if fd.Recv != nil && fd.Body != nil && fd.Name.Name == "String" && len(fd.Type.Params.List) == 0 {
		noAnnSpecialFunc = true
	}
	if fd.Recv != nil && fd.Body != nil && fd.Name.Name == "Error" && len(fd.Type.Params.List) == 0 {
		noAnnSpecialFunc = true
	}
	// insert debugging step to show it ran
	if noAnnSpecialFunc {
		//s := fmt.Sprintf("special func %v()", fd.Name.Name)
		//re := basicLitStringQ(s)
		//ce := ann.newDebugCallExpr("INAnn", re)
		//stmt := ann.newDebugLineStmt(ctx, fd.Type.End(), ce)
		//ctx2, _ := ctx.withStmtIter(&fd.Body.List)
		//ctx2.insertInStmtListBefore(stmt)
		return
	}

	// create new blockstmt to contain args debug stmts
	pos := fd.Type.End()
	bs := &ast.BlockStmt{List: []ast.Stmt{}}

	// body can be nil (external func decls, only the header declared)
	hasParamsBody := len(fd.Type.Params.List) > 0 && fd.Body != nil
	if hasParamsBody {
		// visit parameters
		ctx3, _ := ctx.withStmtIter(&bs.List)
		exprs := ann.visitFieldList(ctx3, fd.Type.Params)

		// debug line
		ce := ann.newDebugCallExpr("IL", exprs...)
		stmt2 := ann.newDebugLineStmt(ctx3, pos, ce)
		ctx3.insertInStmtList(stmt2)
	}

	// visit body
	ctx2 := ctx.withFuncType(fd.Type)
	if fd.Body != nil {
		ann.visitBlockStmt(ctx2, fd.Body)
	}

	if hasParamsBody {
		// insert blockstmt at the top of the body
		ctx4, _ := ctx.withStmtIter(&fd.Body.List)
		ctx4.insertInStmtListBefore(bs) // index 0
	}
}

//----------

func (ann *Annotator) visitBlockStmt(ctx *Ctx, bs *ast.BlockStmt) {
	ann.visitStmtList(ctx, &bs.List)
}

func (ann *Annotator) visitExprStmt(ctx *Ctx, es *ast.ExprStmt) {
	pos := es.End() // replacements could make position 0
	e := ann.visitExpr(ctx, &es.X)
	stmt := ann.newDebugLineStmt(ctx, pos, e)
	ctx.insertInStmtList(stmt)
}

func (ann *Annotator) visitAssignStmt(ctx *Ctx, as *ast.AssignStmt) {
	pos := as.End() // replacements could make position 0

	ctx2 := ctx.withInsertStmtAfter(false)

	ctx3 := ctx2
	if len(as.Rhs) >= 2 {
		// ex: a[i], a[j] = a[j], a[i] // a[j] returns 1 result
		ctx3 = ctx3.withNResults(1)
	} else {
		ctx3 = ctx3.withNResults(len(as.Lhs))
	}
	ctx3 = ctx3.withResultInVar(true)
	rhs := ann.visitExprList(ctx3, &as.Rhs)

	if ctx.assignStmtIgnoreLhs() {
		ce1 := ann.newDebugCallExpr("IL", rhs...)
		stmt2 := ann.newDebugLineStmt(ctx, pos, ce1)
		ctx.insertInStmtList(stmt2)
		return
	}

	rhsId := ann.newDebugCallExpr("IL", rhs...)

	ctx4 := ctx.withInsertStmtAfter(true)
	ctx4 = ctx4.withNResults(1) // a[i] // a returns 1 result, not zero
	lhs := ann.visitExprList(ctx4, &as.Lhs)

	lhsId := ann.newDebugCallExpr("IL", lhs...)

	ce3 := ann.newDebugCallExpr("IA", lhsId, rhsId)
	stmt2 := ann.newDebugLineStmt(ctx4, pos, ce3)
	ctx4.insertInStmtList(stmt2)
}

func (ann *Annotator) visitTypeSwitchStmt(ctx *Ctx, tss *ast.TypeSwitchStmt) {
	if tss.Init != nil && !ctx.noAnnotations() {
		// wrap in block stmt to have init variables valid only in block
		bs := ann.wrapInitInBlockStmt(ctx, tss, nil)
		ann.visitBlockStmt(ctx, bs)
		return
	}

	ctx2 := ctx.withAssignStmtIgnoreLhs() // don't debug lhs
	ann.visitStmt(ctx2, tss.Assign)

	ann.visitBlockStmt(ctx, tss.Body)
}

func (ann *Annotator) visitSwitchStmt(ctx *Ctx, ss *ast.SwitchStmt) {
	if ss.Init != nil && !ctx.noAnnotations() {
		bs := ann.wrapInitInBlockStmt(ctx, ss, nil)
		ann.visitBlockStmt(ctx, bs)
		return
	}

	if ss.Tag != nil {
		pos := ss.Tag.End() // replacements could make position 0

		ctx2 := ctx
		if _, ok := ss.Tag.(*ast.CallExpr); ok {
			// special callexpr case: switch handles only 1 result
			ctx2 = ctx2.withNResults(1)
		}
		ctx2 = ctx2.withResultInVar(true)

		e := ann.visitExpr(ctx2, &ss.Tag)
		stmt2 := ann.newDebugLineStmt(ctx, pos, e)
		ctx.insertInStmtListBefore(stmt2)
	}

	ann.visitBlockStmt(ctx, ss.Body)
}

func (ann *Annotator) visitIfStmt(ctx *Ctx, is *ast.IfStmt) {
	if is.Init != nil && !ctx.noAnnotations() {
		// wrap in block stmt to have init variables valid only in block
		bs := ann.wrapInitInBlockStmt(ctx, is, nil)
		ann.visitBlockStmt(ctx, bs)
		return
	}

	// condition
	pos := is.Cond.End() // replacements could make position 0
	ctx2 := ctx.withNResults(1).withResultInVar(true)
	e := ann.visitExpr(ctx2, &is.Cond)
	stmt2 := ann.newDebugLineStmt(ctx, pos, e)
	ctx.insertInStmtListBefore(stmt2)

	ann.visitBlockStmt(ctx, is.Body)

	switch t := is.Else.(type) {
	case nil: // nothing to do
	case *ast.IfStmt: // "else if ..."
		if !ctx.noAnnotations() {
			bs := ann.wrapInitInBlockStmt(ctx, t, &is.Else)
			ann.visitBlockStmt(ctx, bs)
		} else {
			ann.visitStmt(ctx, t) // init won't be annotated, just visit
		}
	case *ast.BlockStmt: // "else ..."
		ann.visitBlockStmt(ctx, t)
	default:
		fmt.Printf("todo: visitIfStmt: else: %T\n", t)
	}
}

func (ann *Annotator) visitForStmt(ctx *Ctx, fs *ast.ForStmt) {
	if fs.Init != nil && !ctx.noAnnotations() {
		// wrap in block stmt to have init variables valid only in block
		bs := ann.wrapInitInBlockStmt(ctx, fs, nil)
		ann.visitBlockStmt(ctx, bs)
		return
	}

	if fs.Cond != nil && !ctx.noAnnotations() {
		pos := fs.Cond.End()

		// create ifstmt to break the loop
		ue := &ast.UnaryExpr{Op: token.NOT, X: fs.Cond} // negate
		is := &ast.IfStmt{If: fs.Pos(), Cond: ue, Body: &ast.BlockStmt{}}
		fs.Cond = nil // clear forstmt condition

		// insert break inside ifstmt
		brk := &ast.BranchStmt{Tok: token.BREAK}
		is.Body.List = append(is.Body.List, brk)

		// blockstmt to contain the code to be inserted
		bs := &ast.BlockStmt{List: []ast.Stmt{is}}

		// visit condition
		ctx3, _ := ctx.withStmtIter(&bs.List) // index at 0
		e := ann.visitExpr(ctx3, &ue.X)

		// ifstmt condition debug line (create debug line before visiting body)
		stmt2 := ann.newDebugLineStmt(ctx3, pos, e)
		ctx3.insertInStmtListBefore(stmt2)

		// visit body (creates bigger debug line indexes)
		ann.visitBlockStmt(ctx, fs.Body)

		// insert created blockstmt at the top (after visiting body).
		ctx4, _ := ctx.withStmtIter(&fs.Body.List) // index at 0
		ctx4.insertInStmtListBefore(bs)

		return
	}

	ann.visitBlockStmt(ctx, fs.Body)
}

func (ann *Annotator) visitRangeStmt(ctx *Ctx, rs *ast.RangeStmt) {
	pos := rs.X.End()

	ctx2 := ctx.withNResults(1)
	x := ann.visitExpr(ctx2, &rs.X)

	// TODO: context to discard X when visiting rs.X above?
	// assign x to anon (not using X value, showing just length value instead as it will be less verbose and more useful)
	as2 := ann.newAssignStmt11(anonIdent(), x)
	as2.Tok = token.ASSIGN
	ctx.insertInStmtList(as2)

	// length of x
	ce5 := &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{rs.X}}
	//// call everytime (go only calls len once in range())
	//ce8 := ann.newDebugCallExpr("IVr", ce5)
	//lenId := ce8
	//rangeLenId := ce8
	// reuse len value
	ce6 := ann.newDebugCallExpr("IVr", ce5)
	ce7 := ann.assignToNewIdent(ctx, ce6)
	lenId := ce7
	rangeLenId := ce7

	// show step with length result (in case it is zero, it would not show)
	ctx5 := ctx.withKeepDebugIndex()
	stmt5 := ann.newDebugLineStmt(ctx5, pos, lenId)
	ctx.insertInStmtListBefore(stmt5)

	// key and value
	lhs := []ast.Expr{}
	if rs.Key != nil {
		ce := ann.newDebugCallExpr("IV", rs.Key)
		if isAnonIdent(rs.Key) {
			ce = ann.newDebugCallExpr("IAn")
		}
		lhs = append(lhs, ce)
	}
	if rs.Value != nil {
		ce := ann.newDebugCallExpr("IV", rs.Value)
		if isAnonIdent(rs.Value) {
			ce = ann.newDebugCallExpr("IAn")
		}
		lhs = append(lhs, ce)
	}

	// blockstmt to contain the code to be inserted
	bs := &ast.BlockStmt{}
	ctx3, _ := ctx.withStmtIter(&bs.List) // index at 0

	rhs := []ast.Expr{rangeLenId}
	ce1 := ann.newDebugCallExpr("IL", rhs...)
	rhsId := ann.assignToNewIdent(ctx3, ce1)

	ce2 := ann.newDebugCallExpr("IL", lhs...)
	lhsId := ann.assignToNewIdent(ctx3, ce2)

	as1 := ann.newDebugCallExpr("IA", lhsId, rhsId)

	// create debug line before visiting range body
	stmt2 := ann.newDebugLineStmt(ctx3, pos, as1)
	ctx3.insertInStmtListBefore(stmt2)

	// visit range body
	ann.visitBlockStmt(ctx, rs.Body)

	// insert created blockstmt at the top (after visiting body).
	ctx4, _ := ctx.withStmtIter(&rs.Body.List) // index at 0
	ctx4.insertInStmtListBefore(bs)
}

func (ann *Annotator) visitLabeledStmt(ctx *Ctx, ls *ast.LabeledStmt) {
	// Problem:
	// -	label1: ; // inserting empty stmt breaks compilation
	// 	for { break label1 } // compile error: invalide break label
	// -	using block stmts won't work
	// 	label1:
	// 	{ for { break label1} } // compile error
	// No way to insert debug stmts between the label and the stmt.
	// Just make a debug step with "label" where a warning can be shown.
	//
	// Ex: in the case of the ast.forstmt, the init variables need to be enclosed in a blockstmt such that they are only valid in that block. Since the label is linked to the forstmt, it is not possible to insert debug lines for the init.
	// unless the label itself is inside a new block!!

	ce := ann.newDebugCallExpr("ILa")
	stmt := ann.newDebugLineStmt(ctx, ls.Pos(), ce)
	ctx.insertInStmtListBefore(stmt)

	if ls.Stmt != nil {
		ctx = ctx.withLabeledStmt(ls)
		ann.visitStmt(ctx, ls.Stmt)
	}
}

func (ann *Annotator) visitReturnStmt(ctx *Ctx, rs *ast.ReturnStmt) {
	ft, ok := ctx.funcType()
	if !ok {
		return
	}

	// functype number of results to return
	ftNResults := ft.Results.NumFields()
	if ftNResults == 0 {
		// show debug step
		ce := ann.newDebugCallExpr("ISt")
		stmt := ann.newDebugLineStmt(ctx, rs.End(), ce)
		ctx.insertInStmtListBefore(stmt)
		return
	}

	pos := rs.End()

	// naked return, use results ids
	if len(rs.Results) == 0 {
		var w []ast.Expr
		for _, f := range ft.Results.List {
			for _, id := range f.Names {
				w = append(w, id)
			}
		}
		rs.Results = w
	}

	// visit results
	n := ftNResults
	if len(rs.Results) > 1 { // ex: return 1, f(1), 1
		n = 1
	}
	ctx2 := ctx.withNResults(n).withResultInVar(true)
	exprs := ann.visitExprList(ctx2, &rs.Results)

	ce := ann.newDebugCallExpr("IL", exprs...)
	stmt2 := ann.newDebugLineStmt(ctx, pos, ce)
	ctx.insertInStmtListBefore(stmt2)
}

func (ann *Annotator) visitDeferStmt(ctx *Ctx, ds *ast.DeferStmt) {
	ann.visitDeferCallStmt(ctx, &ds.Call)
}
func (ann *Annotator) visitDeferCallStmt(ctx *Ctx, cep **ast.CallExpr) {
	// assign arguments to tmp variables
	ce := *cep
	if len(ce.Args) > 0 {
		args2 := make([]ast.Expr, len(ce.Args))
		copy(args2, ce.Args)
		ids := ann.assignToNewIdents(ctx, len(ce.Args), args2...)
		for i := range ce.Args {
			ce.Args[i] = ids[i]
		}
	}

	// replace func call with wrapped function
	bs := &ast.BlockStmt{List: []ast.Stmt{
		&ast.ExprStmt{X: ce},
	}}
	*cep = &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{Params: &ast.FieldList{}},
			Body: bs,
		},
	}

	ann.visitBlockStmt(ctx, bs)
}

func (ann *Annotator) visitDeclStmt(ctx *Ctx, ds *ast.DeclStmt) {
	if gd, ok := ds.Decl.(*ast.GenDecl); ok {
		// transform this decl into several individual decls to allow inserting stmts between the declarations
		if len(gd.Specs) >= 2 {
			ann.splitGenDeclStmt(ctx, gd)
		}
		for _, s := range gd.Specs {
			ann.visitSpec(ctx, s)
		}
	}
}

func (ann *Annotator) splitGenDeclStmt(ctx *Ctx, gd *ast.GenDecl) {
	// make this decl single spec (will be copied below)
	gd.Lparen = 0
	gd.Rparen = 0

	switch gd.Tok {
	case token.CONST, token.VAR:
		// make new statements from the 2nd beyond
		for i := 1; i < len(gd.Specs); i++ {
			s := gd.Specs[i]
			// build stmt
			gd2 := *gd
			gd2.Specs = []ast.Spec{s}
			stmt := &ast.DeclStmt{&gd2}

			ctx.insertInStmtListAfter(stmt)
		}
		// reset counter to have the other specs be visited
		if iter, ok := ctx.stmtIter(); ok {
			iter.step -= len(gd.Specs) - 1
		}
		// make this decl single spec
		gd.Specs = gd.Specs[:1]
	}
}

func (ann *Annotator) visitBranchStmt(ctx *Ctx, bs *ast.BranchStmt) {
	pos := bs.Pos()
	ce := ann.newDebugCallExpr("IBr")
	stmt := ann.newDebugLineStmt(ctx, pos, ce)
	ctx.insertInStmtList(stmt)
}

func (ann *Annotator) visitIncDecStmt(ctx *Ctx, ids *ast.IncDecStmt) {
	pos := ids.End()

	e1 := ann.visitExpr(ctx, &ids.X)

	ctx2 := ctx.withInsertStmtAfter(true)
	e2 := ann.visitExpr(ctx2, &ids.X)

	l1 := ann.newDebugCallExpr("IL", e1)
	l2 := ann.newDebugCallExpr("IL", e2)

	ce3 := ann.newDebugCallExpr("IA", l2, l1)
	stmt := ann.newDebugLineStmt(ctx, pos, ce3)
	ctx2.insertInStmtList(stmt)
}

func (ann *Annotator) visitSendStmt(ctx *Ctx, ss *ast.SendStmt) {
	pos := ss.End()

	ctx2 := ctx.withNResults(1).withResultInVar(true)
	val := ann.visitExpr(ctx2, &ss.Value)

	ctx3 := ctx.withInsertStmtAfter(true)

	ch := ann.visitExpr(ctx3, &ss.Chan)

	ce := ann.newDebugCallExpr("IS", ch, val)
	stmt := ann.newDebugLineStmt(ctx3, pos, ce)
	ctx3.insertInStmtList(stmt)
}

func (ann *Annotator) visitGoStmt(ctx *Ctx, gs *ast.GoStmt) {
	ann.visitDeferCallStmt(ctx, &gs.Call)
}

func (ann *Annotator) visitSelectStmt(ctx *Ctx, ss *ast.SelectStmt) {
	// debug step to show it is waiting on the select statement
	ce := ann.newDebugCallExpr("ISt")
	stmt := ann.newDebugLineStmt(ctx, ss.Pos(), ce)
	ctx.insertInStmtListBefore(stmt)

	ann.visitBlockStmt(ctx, ss.Body)
}

//----------

func (ann *Annotator) visitCaseClause(ctx *Ctx, cc *ast.CaseClause) {
	// debug step showing the case was entered
	// create first to keep debug index
	ce := ann.newDebugCallExpr("ISt")
	stmt := ann.newDebugLineStmt(ctx, cc.Colon, ce)

	// visit body
	ann.visitStmtList(ctx, &cc.Body)

	// insert debug step into body
	ctx2, _ := ctx.withStmtIter(&cc.Body)
	ctx2.insertInStmtList(stmt)
}

func (ann *Annotator) visitCommClause(ctx *Ctx, cc *ast.CommClause) {
	// debug step showing the case was entered
	// create first to keep debug index
	ce := ann.newDebugCallExpr("ISt")
	stmt := ann.newDebugLineStmt(ctx, cc.Colon, ce)

	// visit body
	ann.visitStmtList(ctx, &cc.Body)

	// insert debug step into body
	ctx2, _ := ctx.withStmtIter(&cc.Body)
	ctx2.insertInStmtList(stmt)
}

//----------

func (ann *Annotator) visitSpec(ctx *Ctx, spec ast.Spec) {
	// specs: import, value, type

	switch t := spec.(type) {
	case *ast.ValueSpec:
		if len(t.Values) > 0 {
			// Ex: var a,b int = 1, 2; var a, b = f()
			// use an assignstmt

			lhs := []ast.Expr{}
			for _, id := range t.Names {
				lhs = append(lhs, id)
			}

			as := &ast.AssignStmt{
				Lhs:    lhs,
				TokPos: t.Pos(),
				Tok:    token.ASSIGN,
				Rhs:    t.Values,
			}
			ann.visitAssignStmt(ctx, as)
		}
	}
}

//----------

func (ann *Annotator) wrapInitInBlockStmt(ctx *Ctx, stmt ast.Stmt, directStmt *ast.Stmt) *ast.BlockStmt {
	bs := &ast.BlockStmt{List: []ast.Stmt{stmt}}
	if directStmt != nil {
		*directStmt = bs
	} else {
		// keep the labeled stmt attached to its stmt
		if ls, ok := ctx.labeledStmt(); ok && ls.Stmt == stmt {
			bs = &ast.BlockStmt{List: []ast.Stmt{ls}}
		}
		ctx.replaceStmt(bs)
	}
	// add init stmts to top of the block stmt list
	switch t := stmt.(type) {
	case *ast.IfStmt:
		if t.Init != nil {
			bs.List = append([]ast.Stmt{t.Init}, bs.List...)
			t.Init = nil
		}
	case *ast.ForStmt:
		if t.Init != nil {
			bs.List = append([]ast.Stmt{t.Init}, bs.List...)
			t.Init = nil
		}
	case *ast.SwitchStmt:
		if t.Init != nil {
			bs.List = append([]ast.Stmt{t.Init}, bs.List...)
			t.Init = nil
		}
	case *ast.TypeSwitchStmt:
		if t.Init != nil {
			bs.List = append([]ast.Stmt{t.Init}, bs.List...)
			t.Init = nil
		}
	default:
		panic("todo")
	}
	return bs
}

//----------

func (ann *Annotator) visitCallExpr(ctx *Ctx, ce *ast.CallExpr) {
	ctx = ctx.withInsertStmtAfter(false)

	// stepping in function name
	// also: first arg is type in case of new/make functions
	ctx2 := ctx
	fname := "f"
	isPanic := false
	switch t := ce.Fun.(type) {
	case *ast.Ident:
		fname = t.Name
		// handle builtin funcs
		switch t.Name {
		case "panic":
			if ann.isBuiltin(t) {
				isPanic = true
			}
		case "new", "make":
			if ann.isBuiltin(t) {
				ctx2 = ctx2.withFirstArgIsType()
			}
		}
	case *ast.SelectorExpr:
		fname = t.Sel.Name
		if !isSelectorIdents(t) { // check if there will be a debug step
			// visit X on its own debug step
			ctx3 := ctx2.withNResults(1)
			pos := ce.Fun.End()
			x := ann.visitExpr(ctx3, &t.X)
			stmt := ann.newDebugLineStmt(ctx, pos, x)
			ctx.insertInStmtList(stmt)
		}
	case *ast.FuncLit:
		pos := ce.Fun.End()
		e := ann.visitExpr(ctx, &ce.Fun)
		stmt := ann.newDebugLineStmt(ctx, pos, e)
		ctx.insertInStmtList(stmt)
	case *ast.InterfaceType:
		fname = "interface{}"
	}
	fnamee := basicLitStringQ(fname)

	// n results
	nResults := 1
	if len(ce.Args) == 1 {
		arg := ce.Args[0]
		typ, ok := ann.file.astExprType(arg)
		if ok {
			switch t := typ.Type.(type) {
			case *types.Tuple:
				nResults = t.Len()
			}
		}
	}

	// visit args
	ctx2 = ctx2.withNResults(nResults)
	ctx2 = ctx2.withResultInVar(true)
	args := ann.visitExprList(ctx2, &ce.Args)

	// insert before calling the function (shows stepping in)
	args2 := append([]ast.Expr{fnamee}, args...)
	ce4 := ann.newDebugCallExpr("ICe", args2...)
	ctx3 := ctx.withKeepDebugIndex()
	stmt := ann.newDebugLineStmt(ctx3, ce.Rparen, ce4)
	ctx.insertInStmtList(stmt)

	// avoid "line unreachable" compiler errors
	if isPanic {
		// nil arg: newDebugLineStmt will generate an emptyStmt
		ctx.pushExprs(emptyExpr())
		return
	}

	ctx4 := ctx.withResultInVar(true)
	result := ann.getResultExpr(ctx4, ce)

	// args after exiting func
	args3 := append([]ast.Expr{fnamee, result}, args...)
	ce3 := ann.newDebugCallExpr("IC", args3...)
	id := ann.assignToNewIdent(ctx, ce3)
	ctx.pushExprs(id)
}

func (ann *Annotator) visitBinaryExpr(ctx *Ctx, be *ast.BinaryExpr) {
	ctx = ctx.withNResults(1)
	switch be.Op {
	case token.LAND, token.LOR:
		ann.visitBinaryExpr3(ctx, be)
	default:
		ann.visitBinaryExpr2(ctx, be)
	}
}
func (ann *Annotator) visitBinaryExpr2(ctx *Ctx, be *ast.BinaryExpr) {
	// keep isdirect before visiting expr
	// ex: "a:=1*f()" is not direct, but "d0:=f();a:=1*d0" is because d0 is an ident (that could refer to a const)
	direct := isDirectExpr(be)

	x := ann.visitExpr(ctx, &be.X)
	y := ann.visitExpr(ctx, &be.Y)

	ctx2 := ctx
	ctx2 = ctx2.withResultInVar(!direct)
	result := ann.getResultExpr(ctx2, be)

	opbl := basicLitInt(int(be.Op))
	ce3 := ann.newDebugCallExpr("IB", result, opbl, x, y)
	id1 := ann.assignToNewIdent(ctx, ce3)
	ctx.pushExprs(id1)
}

func (ann *Annotator) visitBinaryExpr3(ctx *Ctx, be *ast.BinaryExpr) {
	// ex: f1() || f2() // f2 should not be called if f1 is true
	// ex: f1() && f2() // f2 should not be called if f1 is false

	x := ann.visitExpr(ctx, &be.X)

	// y if be.Y doesn't run
	q := ann.newDebugCallExpr("IVs", basicLitStringQ("?"))
	y := ann.assignToNewIdent(ctx, q)

	// create final result variable, initially with be.X
	ctx2 := ctx.withInsertStmtAfter(false)
	finalResult := ann.assignToNewIdent(ctx2, be.X)
	ctx.replaceExpr(finalResult)

	// create ifstmt to run be.Y if be.X is true
	var xcond ast.Expr = finalResult // token.LAND
	if be.Op == token.LOR {
		xcond = &ast.UnaryExpr{Op: token.NOT, X: xcond}
	}
	is := &ast.IfStmt{If: be.Pos(), Cond: xcond, Body: &ast.BlockStmt{}}
	ctx.insertInStmtListBefore(is)

	// (inside ifstmt) assign be.Y to result variable
	as2 := ann.newAssignStmt11(finalResult, be.Y)
	as2.Tok = token.ASSIGN
	is.Body.List = append(is.Body.List, as2)

	// (inside ifstmt) run be.Y
	ctx3, _ := ctx.withStmtIter(&is.Body.List) // index at 0
	y2 := ann.visitExpr(ctx3, &as2.Rhs[0])

	// (inside ifstmt) assign debug result to y
	as3 := ann.newAssignStmt11(y, y2)
	as3.Tok = token.ASSIGN
	is.Body.List = append(is.Body.List, as3)

	result := ann.newDebugCallExpr("IV", finalResult)

	opbl := basicLitInt(int(be.Op))
	ce3 := ann.newDebugCallExpr("IB", result, opbl, x, y)
	id1 := ann.assignToNewIdent(ctx, ce3)
	ctx.pushExprs(id1)
}

func (ann *Annotator) visitUnaryExpr(ctx *Ctx, ue *ast.UnaryExpr) {
	pos := ue.End()

	// X expression
	ctx2 := ctx
	ctx2 = ctx2.withInsertStmtAfter(false)
	ctx2 = ctx2.withNResults(1)
	if ue.Op == token.AND {
		// Ex: f1(&c[i]) -> d0:=c[i]; f1(&d0) // d0 wrong address
		ctx2 = ctx2.withResultInVar(false)
	}
	x := ann.visitExpr(ctx2, &ue.X)

	// show entering
	if ue.Op == token.ARROW {
		opbl := basicLitInt(int(ue.Op))
		ce4 := ann.newDebugCallExpr("IUe", opbl, x)
		ctx3 := ctx.withKeepDebugIndex()
		stmt := ann.newDebugLineStmt(ctx3, pos, ce4)
		ctx2.insertInStmtList(stmt)
	}

	// result
	ctx3 := ctx
	direct := isDirectExpr(ue)
	ctx3 = ctx3.withResultInVar(!direct)
	result := ann.getResultExpr(ctx3, ue)

	opbl := basicLitInt(int(ue.Op))
	ce3 := ann.newDebugCallExpr("IU", result, opbl, x)
	id := ann.assignToNewIdent(ctx, ce3)
	ctx.pushExprs(id)
}

func (ann *Annotator) visitSelectorExpr(ctx *Ctx, se *ast.SelectorExpr) {

	// simplify if the tree is just made of idents
	if isSelectorIdents(se) {
		ce := ann.newDebugCallExpr("IV", se)
		id := ann.assignToNewIdent(ctx, ce)
		ctx.pushExprs(id)
		return
	}

	ctx2 := ctx.withInsertStmtAfter(false)
	x := ann.visitExpr(ctx2, &se.X)
	ce2 := ann.newDebugCallExpr("IV", se) // selector value
	ce := ann.newDebugCallExpr("ISel", x, ce2)
	id := ann.assignToNewIdent(ctx, ce)
	ctx.pushExprs(id)
}

func (ann *Annotator) visitIndexExpr(ctx *Ctx, ie *ast.IndexExpr) {
	// ex: a, ok := c[f1()] // map access, more then 1 result
	// ex: a, b = c[i], d[j]

	// X expr
	var x ast.Expr
	switch ie.X.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		x = nilIdent()
	default:
		x = ann.visitExpr(ctx, &ie.X)
	}

	// Index expr
	ctx2 := ctx.withResultInVar(true)
	ctx2 = ctx2.withNResults(1) // a = b[f()] // f() returns 1 result
	ix := ann.visitExpr(ctx2, &ie.Index)

	result := ann.getResultExpr(ctx, ie)

	ce3 := ann.newDebugCallExpr("II", result, x, ix)
	ctx.pushExprs(ce3)
}

func (ann *Annotator) visitSliceExpr(ctx *Ctx, se *ast.SliceExpr) {
	var x ast.Expr
	switch se.X.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		x = nilIdent()
	default:
		x = ann.visitExpr(ctx, &se.X)
	}

	var ix []ast.Expr
	for _, e := range []*ast.Expr{&se.Low, &se.High, &se.Max} {
		var r ast.Expr
		if *e == nil {
			r = nilIdent()
		} else {
			r = ann.visitExpr(ctx, e)
		}
		ix = append(ix, r)
	}

	result := ann.getResultExpr(ctx, se)

	// slice3: 2 colons present
	s := "false"
	if se.Slice3 {
		s = "true"
	}
	bl := basicLitString(s)

	ce := ann.newDebugCallExpr("II2", result, x, ix[0], ix[1], ix[2], bl)
	ctx.pushExprs(ce)
}

func (ann *Annotator) visitKeyValueExpr(ctx *Ctx, kv *ast.KeyValueExpr) {
	var k ast.Expr
	if id, ok := kv.Key.(*ast.Ident); ok {
		k = ann.newDebugCallExpr("IVs", basicLitStringQ(id.Name))
	} else {
		k = ann.visitExpr(ctx, &kv.Key)
	}

	v := ann.visitExpr(ctx, &kv.Value)

	ce := ann.newDebugCallExpr("IKV", k, v)
	ctx.pushExprs(ce)
}

func (ann *Annotator) visitTypeAssertExpr(ctx *Ctx, tae *ast.TypeAssertExpr) {
	// don't show type assertion for "X.(sometype)", just visit the expr
	if tae.Type != nil {
		ctx2 := ctx.withNResults(1)
		x := ann.visitExpr(ctx2, &tae.X)
		ctx.pushExprs(x)
		return
	}
	// from here it is of type "switch X.(type)"

	// simplify if it is just an ident
	if _, ok := tae.X.(*ast.Ident); ok {
		ce := ann.newDebugCallExpr("IVt", tae.X)
		ctx.pushExprs(ce)
		return
	}

	ctx2 := ctx
	ctx2 = ctx2.withResultInVar(true)
	ctx2 = ctx2.withNResults(1)
	x := ann.visitExpr(ctx2, &tae.X)
	ce1 := ann.newDebugCallExpr("IVt", tae.X)

	ce2 := ann.newDebugCallExpr("ITA", x, ce1)
	ctx.pushExprs(ce2)
}

func (ann *Annotator) visitParenExpr(ctx *Ctx, pe *ast.ParenExpr) {
	x := ann.visitExpr(ctx, &pe.X)
	ce := ann.newDebugCallExpr("IP", x)
	ctx.pushExprs(ce)
}

func (ann *Annotator) visitStarExpr(ctx *Ctx, se *ast.StarExpr) {
	// Ex: *a=1
	ctx = ctx.withNResults(1)

	x := ann.visitExpr(ctx, &se.X)

	ctx3 := ctx.withResultInVar(false)
	result := ann.getResultExpr(ctx3, se)

	opbl := basicLitInt(int(token.MUL))
	ce3 := ann.newDebugCallExpr("IU", result, opbl, x)
	id := ann.assignToNewIdent(ctx, ce3)
	ctx.pushExprs(id)
}

//----------

func (ann *Annotator) visitBasicLit(ctx *Ctx, bl *ast.BasicLit) {
	ce := ann.newDebugCallExpr("IV", bl)
	id := ann.assignToNewIdent(ctx, ce)
	ctx.pushExprs(id)
}

func (ann *Annotator) visitFuncLit(ctx *Ctx, fl *ast.FuncLit) {
	ctx = ctx.valuesReset()

	id := ann.assignToNewIdent(ctx, fl)
	ctx.replaceExpr(id)

	// create new blockstmt to contain args debug stmts
	pos := fl.Type.End()
	bs := &ast.BlockStmt{List: []ast.Stmt{}}

	hasParams := len(fl.Type.Params.List) > 0
	if hasParams {
		// visit parameters
		ctx3, _ := ctx.withStmtIter(&bs.List)
		exprs := ann.visitFieldList(ctx3, fl.Type.Params)

		// debug line
		ce := ann.newDebugCallExpr("IL", exprs...)
		stmt2 := ann.newDebugLineStmt(ctx3, pos, ce)
		ctx3.insertInStmtList(stmt2)
	}

	// visit body
	ctx2 := ctx.withFuncType(fl.Type)
	ann.visitBlockStmt(ctx2, fl.Body)

	if hasParams {
		// insert blockstmt at the top of the body
		ctx4, _ := ctx.withStmtIter(&fl.Body.List)
		ctx4.insertInStmtListBefore(bs) // index 0
	}

	ce := ann.newDebugCallExpr("IV", id)
	id2 := ann.assignToNewIdent(ctx, ce)
	ctx.pushExprs(id2)
}

func (ann *Annotator) visitCompositeLit(ctx *Ctx, cl *ast.CompositeLit) {
	u := ann.visitExprList(ctx, &cl.Elts)
	ce := ann.newDebugCallExpr("ILit", u...)
	ctx.pushExprs(ce)
}

//----------

func (ann *Annotator) visitIdent(ctx *Ctx, id *ast.Ident) {
	if isAnonIdent(id) {
		ce := ann.newDebugCallExpr("IAn")
		ctx.pushExprs(ce)
		return
	}
	ce := ann.newDebugCallExpr("IV", id)
	id2 := ann.assignToNewIdent(ctx, ce)
	ctx.pushExprs(id2)
}

//----------

//func (ann *Annotator) visitArrayType(ctx *Ctx, at *ast.ArrayType) {
//	e := ann.visitType(ctx)
//	ctx.pushExprs(e)
//}

func (ann *Annotator) visitType(ctx *Ctx) ast.Expr {
	bl := basicLitStringQ("type")
	ce := ann.newDebugCallExpr("IVs", bl)
	id := ann.assignToNewIdent(ctx, ce)
	return id
}

//----------

func (ann *Annotator) visitFieldList(ctx *Ctx, fl *ast.FieldList) []ast.Expr {
	exprs := []ast.Expr{}
	for _, f := range fl.List {
		w := ann.visitField(ctx, f)
		exprs = append(exprs, w...)
	}
	return exprs
}

func (ann *Annotator) visitField(ctx *Ctx, field *ast.Field) []ast.Expr {
	// set field name if it has no names (otherwise it won't output)
	if len(field.Names) == 0 {
		field.Names = append(field.Names, ann.newIdent(ctx))
	}

	exprs := []ast.Expr{}
	for _, id := range field.Names {
		ctx2 := ctx.withNewExprs()
		ann.visitIdent(ctx2, id)
		w := ctx2.popExprs()
		exprs = append(exprs, w...)
	}
	return exprs
}

//----------

func (ann *Annotator) visitStmtList(ctx *Ctx, list *[]ast.Stmt) {
	ctx2, iter := ctx.withStmtIter(list)

	for iter.index < len(*list) {
		stmt := (*list)[iter.index]

		// on/off annotation stmts
		on, ok := ann.annotationsOn(stmt)
		if ok {
			ctx2 = ctx2.withNoAnnotations(!on)
		}

		// stmts defaults
		ctx3 := ctx2
		switch stmt.(type) {
		case *ast.ExprStmt, *ast.AssignStmt:
			// ast.SwitchStmt needs false
			ctx3 = ctx3.withInsertStmtAfter(true)
		}

		ann.visitStmt(ctx3, stmt)

		iter.index += 1 + iter.step
		iter.step = 0
	}
}

func (ann *Annotator) visitStmt(ctx *Ctx, stmt ast.Stmt) {
	ctx = ctx.withNewExprs()
	ctx = ctx.withNResults(0)
	ctx = ctx.withNoStaticDebugIndex() // setup to be able to set upper
	switch t := stmt.(type) {
	case *ast.ExprStmt:
		ann.visitExprStmt(ctx, t)
	case *ast.AssignStmt:
		ann.visitAssignStmt(ctx, t)
	case *ast.TypeSwitchStmt:
		ann.visitTypeSwitchStmt(ctx, t)
	case *ast.SwitchStmt:
		ann.visitSwitchStmt(ctx, t)
	case *ast.IfStmt:
		ann.visitIfStmt(ctx, t)
	case *ast.ForStmt:
		ann.visitForStmt(ctx, t)
	case *ast.RangeStmt:
		ann.visitRangeStmt(ctx, t)
	case *ast.LabeledStmt:
		ann.visitLabeledStmt(ctx, t)
	case *ast.ReturnStmt:
		ann.visitReturnStmt(ctx, t)
	case *ast.DeferStmt:
		ann.visitDeferStmt(ctx, t)
	case *ast.DeclStmt:
		ann.visitDeclStmt(ctx, t)
	case *ast.BranchStmt:
		ann.visitBranchStmt(ctx, t)
	case *ast.IncDecStmt:
		ann.visitIncDecStmt(ctx, t)
	case *ast.SendStmt:
		ann.visitSendStmt(ctx, t)
	case *ast.GoStmt:
		ann.visitGoStmt(ctx, t)
	case *ast.SelectStmt:
		ann.visitSelectStmt(ctx, t)
	case *ast.BlockStmt:
		ann.visitBlockStmt(ctx, t)
	case *ast.CaseClause:
		ann.visitCaseClause(ctx, t)
	case *ast.CommClause:
		ann.visitCommClause(ctx, t)
	case nil: // do nothing
	case *ast.EmptyStmt: // do nothing
	default:
		fmt.Printf("visitstmt: %#v\n", t)
	}
}

func (ann *Annotator) visitExprList(ctx *Ctx, list *[]ast.Expr) []ast.Expr {
	var exprs []ast.Expr
	ctx2, iter := ctx.withExprIter(list)
	for iter.index < len(*list) {
		exprPtr := &(*list)[iter.index]

		if iter.index == 0 && ctx.firstArgIsType() {
			e := ann.visitType(ctx)
			exprs = append(exprs, e)
		} else {
			e := ann.visitExpr(ctx2, exprPtr)
			exprs = append(exprs, e)
		}

		iter.index += 1 + iter.step
		iter.step = 0
	}
	return exprs
}

func (ann *Annotator) visitExpr(ctx *Ctx, exprPtr *ast.Expr) ast.Expr {
	ctx = ctx.withNewExprs()
	ctx = ctx.withExprPtr(exprPtr)

	e1 := *exprPtr

	switch t := e1.(type) {
	case *ast.CallExpr:
		ann.visitCallExpr(ctx, t)
	case *ast.BinaryExpr:
		ann.visitBinaryExpr(ctx, t)
	case *ast.UnaryExpr:
		ann.visitUnaryExpr(ctx, t)
	case *ast.SelectorExpr:
		ann.visitSelectorExpr(ctx, t)
	case *ast.IndexExpr:
		ann.visitIndexExpr(ctx, t)
	case *ast.SliceExpr:
		ann.visitSliceExpr(ctx, t)
	case *ast.KeyValueExpr:
		ann.visitKeyValueExpr(ctx, t)
	case *ast.TypeAssertExpr:
		ann.visitTypeAssertExpr(ctx, t)
	case *ast.ParenExpr:
		ann.visitParenExpr(ctx, t)
	case *ast.StarExpr:
		ann.visitStarExpr(ctx, t)
	case *ast.BasicLit:
		ann.visitBasicLit(ctx, t)
	case *ast.FuncLit:
		ann.visitFuncLit(ctx, t)
	case *ast.CompositeLit:
		ann.visitCompositeLit(ctx, t)
	case *ast.Ident:
		ann.visitIdent(ctx, t)

	// TODO: _=new([]int,3), called if builtin not being detected
	//case *ast.ArrayType:
	//	ann.visitArrayType(ctx, t)

	default:
		err := fmt.Errorf("todo: visitExpr: %T", e1)
		err2 := errorPos(err, ann.fset, e1.Pos())
		fmt.Println(err2)
	}

	exprs := ctx.popExprs()
	if len(exprs) == 1 {
		return exprs[0]
	}

	// debug
	err := fmt.Errorf("todo: visitExpr: %T, len(exprs)=%v\n", e1, len(exprs))
	err2 := errorPos(err, ann.fset, e1.Pos())
	fmt.Println(err2)

	return nilIdent()
}

//----------

func (ann *Annotator) getResultExpr(ctx *Ctx, e ast.Expr) ast.Expr {
	nres := ctx.nResults()
	if nres == 0 {
		return nilIdent()
	}
	if nres >= 2 {
		u := ann.assignToNewIdents(ctx, nres, e)
		ctx.replaceExprs(u)

		var u2 []ast.Expr
		for _, e := range u {
			ce := ann.newDebugCallExpr("IV", e)
			u2 = append(u2, ce)
		}

		ce := ann.newDebugCallExpr("IL", u2...)
		return ann.assignToNewIdent(ctx, ce)
	}

	if ctx.resultInVar() {
		// putting the result in a variable is never inserted after
		ctx = ctx.withInsertStmtAfter(false)
		e = ann.assignToNewIdent(ctx, e)
		ctx.replaceExpr(e)
	}

	ce := ann.newDebugCallExpr("IV", e)
	return ann.assignToNewIdent(ctx, ce)
}

//----------

func (ann *Annotator) assignToNewIdent(ctx *Ctx, e ast.Expr) ast.Expr {
	u := ann.assignToNewIdents(ctx, 1, e)
	return u[0]
}

func (ann *Annotator) assignToNewIdents(ctx *Ctx, nids int, exprs ...ast.Expr) []ast.Expr {
	ids := []ast.Expr{}
	for i := 0; i < nids; i++ {
		ids = append(ids, ann.newIdent(ctx))
	}
	stmt := ann.newAssignStmt(ids, exprs)
	ctx.insertInStmtList(stmt)
	return ids
}

//----------

func (ann *Annotator) newDebugCallExpr(fname string, u ...ast.Expr) *ast.CallExpr {
	// wrap big constants that cannot be used as args (compile error)
	if fname == "IV" && len(u) == 1 {
		e1 := &u[0]
		if e2, ok := ann.basicLitStringQIfBigConstant(*e1); ok {
			fname = "IVs"
			*e1 = e2
		}
	}

	se := &ast.SelectorExpr{
		X:   ast.NewIdent(ann.debugPkgName),
		Sel: ast.NewIdent(fname),
	}
	return &ast.CallExpr{Fun: se, Args: u}
}

func (ann *Annotator) newDebugLineStmt(ctx *Ctx, pos token.Pos, e ast.Expr) ast.Stmt {
	if ctx.noAnnotations() {
		return &ast.EmptyStmt{}
	}

	if e == emptyExpr() {
		return &ast.EmptyStmt{}
	}

	var di int
	if i, ok := ctx.staticDebugIndex(); ok {
		di = i
	} else {
		di = ann.debugIndex
		ann.debugIndex++
	}
	if ctx.keepDebugIndex() {
		ctx.setUpperStaticDebugIndex(di)
	}

	position := ann.fset.Position(pos)
	lineOffset := position.Offset

	args := []ast.Expr{
		basicLitInt(ann.fileIndex),
		basicLitInt(di),
		basicLitInt(lineOffset),
		e,
	}

	se := &ast.SelectorExpr{
		X:   ast.NewIdent(ann.debugPkgName),
		Sel: ast.NewIdent("Line"),
	}
	es := &ast.ExprStmt{X: &ast.CallExpr{Fun: se, Args: args}}

	ann.builtDebugLineStmt = true
	return es
}

//----------

func (ann *Annotator) newIdent(ctx *Ctx) *ast.Ident {
	return &ast.Ident{Name: ann.newVarName(ctx)}
}
func (ann *Annotator) newVarName(ctx *Ctx) string {
	if ctx.noAnnotations() {
		return ""
	}
	s := fmt.Sprintf(ann.debugVarPrefix+"%d", ann.debugVarNameIndex)
	ann.debugVarNameIndex++
	return s
}

func (ann *Annotator) newAssignStmt11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return ann.newAssignStmt([]ast.Expr{lhs}, []ast.Expr{rhs})
}
func (ann *Annotator) newAssignStmt(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.DEFINE, Lhs: lhs, Rhs: rhs}
}

//----------

func (ann *Annotator) basicLitStringQIfBigConstant(e ast.Expr) (ast.Expr, bool) {
	// handles big constants:
	// _=uint64(1<<64 - 1)
	// _=uint64(math.MaxUint64)
	//
	// the annotator would generate IV(1<<64), which will give a compile error since "1<<64" overflows an int (consts are assigned to int by default)

	t, ok := ann.file.astExprType(e)
	if ok && t.Value != nil {
		switch t.Value.Kind() { // performance (not necessary)
		case constant.Int, constant.Float, constant.Complex:

			u := constant.Val(t.Value)
			switch t2 := u.(type) { // necessary
			case *big.Int:
				return basicLitStringQ(t2.String()), true
			case *big.Float:
				return basicLitStringQ(t2.String()), true
			case *big.Rat:
				return basicLitStringQ(t2.String()), true
			}
		}
	}
	return e, false
}

//----------

// Returns on/off, ok.
func (ann *Annotator) annotationsOn(n ast.Node) (bool, bool) {
	at := ann.file.files.NodeAnnType(n)
	switch at {
	case AnnotationTypeOff:
		return false, true
	case AnnotationTypeBlock:
		return true, true
	}
	return false, false
}

//----------

func (ann *Annotator) isBuiltin(id *ast.Ident) bool {
	typ, ok := ann.file.astIdentObj(id)
	if ok {
		_, ok2 := typ.(*types.Builtin)
		return ok2
	}
	// always returning true if the object is not found (helpful for tests)
	return true
}

//----------

func nilIdent() *ast.Ident {
	return &ast.Ident{Name: "nil"}
}
func anonIdent() *ast.Ident {
	return &ast.Ident{Name: "_"}
}
func isAnonIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

//----------

var _emptyExpr = &ast.Ident{Name: "*emptyExpr*"}

func emptyExpr() ast.Expr { return _emptyExpr }

//----------

func basicLitString(v string) *ast.BasicLit {
	s := strings.ReplaceAll(v, "%", "%%")
	return &ast.BasicLit{Kind: token.STRING, Value: s}
}
func basicLitStringQ(v string) *ast.BasicLit {
	s := strings.ReplaceAll(v, "%", "%%")
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", s)}
}
func basicLitInt(v int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}

//----------

// Can't create vars from the expr or it could create a var of different type.
func isDirectExpr(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.ParenExpr:
		return isDirectExpr(t.X)
	case *ast.UnaryExpr:
		switch t.Op {
		case token.ADD, // +
			token.SUB, // -
			token.XOR: // ^
			return isDirectExpr(t.X)
		}
	case *ast.BasicLit:
		// This fails for all basic literals
		// type A int
		// a:=A(1)
		// b:=1
		// if a==b {} // type mismatch A!=int

		// Could pass context "not-direct-from-here" for the values of "==" but then there could be other cases missing
		return true
	case *ast.Ident:
		// if "a" is const, it would be type int if assigned to a tmp var
		// ex: var a int32 = 0 | a
		// ex: f(1+a) with f=func(int32)
		return true
	case *ast.SelectorExpr:
		return true // TODO: document why
	case *ast.BinaryExpr:
		// ex: var a float =1*2 // (1*2) gives type int if assigned to a tmp var
		switch t.Op {
		case token.ADD, // +
			token.SUB,     // -
			token.MUL,     // *
			token.QUO,     // /
			token.REM,     // %
			token.AND,     // &
			token.OR,      // |
			token.XOR,     // ^
			token.SHL,     // <<
			token.SHR,     // >>
			token.AND_NOT: // &^
			return isDirectExpr(t.X) && isDirectExpr(t.Y)
		}
	}
	return false
}

//----------

func isSelectorIdents(e ast.Expr) bool {
	se, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch t := se.X.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isSelectorIdents(t)
	default:
		return false
	}
}

//----------

// TODO
//func avoidCallExpr(name string) bool {
//	// fixes
//	switch name {
//	case "bool",
//		"int", "int8", "int16", "int32", "int64",
//		"uint", "uint8", "uint16", "uint32", "uint64",
//		"uintptr",
//		"float32", "float64",
//		"complex64", "complex128",
//		"string",
//		"rune", "byte":
//		return true
//	}
//	return false
//}
