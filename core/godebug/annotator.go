package godebug

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/astut"
	"github.com/jmigpin/editor/util/reflectutil"
)

type Annotator struct {
	fset *token.FileSet

	typesInfo    *types.Info
	nodeAnnTypes map[ast.Node]AnnotationType

	fileIndex          int
	debugPkgName       string
	debugVarPrefix     string // will have integer appended
	debugVarNameIndex  int
	debugLastIndex     int  // n indexes were used
	builtDebugLineStmt bool // at least one debug stmt inserted

	simpleTestMode bool // just testing annotator (no types)
}

func NewAnnotator(fset *token.FileSet) *Annotator {
	ann := &Annotator{fset: fset}
	// defaults for tests; used values are in annotatorset
	ann.debugPkgName = "Σ"
	ann.debugVarPrefix = "Σ"
	return ann
}

//----------

func (ann *Annotator) AnnotateAstFile(astFile *ast.File) {
	ctx := &Ctx{}
	ctx = ctx.withDebugIndex(0)

	ann.vis(ctx, astFile)

	// commented: setting astfile.comments to nil seems to fix things for the ast printer.
	//ann.removeInnerFuncComments(astFile)
	astFile.Comments = nil

	ann.debugLastIndex = ann.correctDebugIndexes(astFile)

	//fmt.Println("result----")
	//ann.printNode(astFile)
}

//----------

func (ann *Annotator) log(args ...interface{}) {
	//log.Print(args...)
	s := fmt.Sprint(args...)
	s = strings.TrimRight(s, "\n") + "\n"
	fmt.Print(s)
}
func (ann *Annotator) logf(f string, args ...interface{}) {
	ann.log(fmt.Sprintf(f, args...))
}

//----------

func (ann *Annotator) vis(ctx *Ctx, v interface{}) {
	switch t := v.(type) {
	case nil: // nothing todo
	case ast.Node:
		ann.vis2(ctx, t)
	case *[]ast.Stmt:
		ann.visStmts(ctx, t)
	default:
		panic(fmt.Sprintf("todo: %T", t))
	}
}

// Should only be called throught vis()
func (ann *Annotator) vis2(ctx *Ctx, node ast.Node) {
	//ann.logf("vis2 %T\n", node)
	//ann.logf("vis2 %T (%v)\n", node, ann.sprintNode(node))

	// handle here for top level directives (ex: func decl)
	ctx = ann.annotationBlockCtx(ctx, node)
	if ctx.boolean(ctxIdNoAnnotations) {
		ann.visNoAnnotations(ctx, node)
		return
	}

	// build name: visit node if there is a function defined for it
	name, err := reflectutil.TypeNameBase(node)
	if err != nil {
		return
	}
	res, err := reflectutil.InvokeByName(ann, "Vis"+name, ctx, node)
	if err != nil {
		ann.logf("todo: vis2: %v", err)
		return
	}
	if len(res) != 0 {
		panic(fmt.Sprintf("len of res not zero: %v", len(res)))
	}
}

//----------

func (ann *Annotator) visNoAnnotations(ctx *Ctx, n1 ast.Node) {
	ast.Inspect(n1, func(n2 ast.Node) bool {
		if n2 != n1 {
			ctx2 := ann.annotationBlockCtx(ctx, n2)
			if !ctx2.boolean(ctxIdNoAnnotations) {
				ann.vis(ctx2, n2)
				return false // done with noannotations
			}
		}

		// special handling of a few nodes to setup the ctx in case annotations become enabled
		switch t2 := n2.(type) {
		case *ast.BlockStmt:
			ann.vis(ctx, &t2.List)
			return false
		case *ast.CaseClause:
			ann.vis(ctx, &t2.Body) // not a blockstmt
			return false
		case *ast.CommClause:
			ann.vis(ctx, &t2.Body) // not a blockstmt
			return false

		case *ast.FuncDecl:
			ctx2 := ctx.withFuncType(t2.Type)
			ann.vis(ctx2, t2.Body)
			return false
		case *ast.FuncLit:
			ctx2 := ctx.withFuncType(t2.Type)
			ann.vis(ctx2, t2.Body)
			return false

		case *ast.DeclStmt:
			ann.VisDeclStmt(ctx, t2) // sets ctx boolean
			return false

		// TODO: godebug directive in "else if" between "else" and "if"?

		default:
			return true // visit childs
		}
	})
}

//----------

// the resulting (res) "debug" expr for the given expr
func (ann *Annotator) resExpr(ctx *Ctx, e *ast.Expr) ast.Expr {
	ctx = ctx.withExpr(e)
	node := *e

	//ann.logf("resExpr %T\n", node)
	//ann.logf("resExpr %T (%v)\n", *e, ann.sprintNode2(node))

	// buildname: visit node if there is a function defined for it
	name, err := reflectutil.TypeNameBase(node)
	if err != nil {
		return nilIdent()
	}
	res, err := reflectutil.InvokeByName(ann, "Res"+name, ctx, node)
	if err != nil {
		ann.logf("todo: resExpr: %v", err)
		return nilIdent()
	}
	result := res[0].Interface().(ast.Expr) // must succeed
	if result == nil {
		return nilIdent()
	}
	return result
}

func (ann *Annotator) resExprs(ctx *Ctx, es *[]ast.Expr) ast.Expr {
	ctx = ctx.withExprs(es)
	w := []ast.Expr{}
	for i := range *es {
		ctx2 := ctx
		e1 := &(*es)[i]
		res := (ast.Expr)(nil)
		if i == 0 && ctx2.boolean(ctxIdFirstArgIsType) {
			ctx2 = ctx2.withBoolean(ctxIdInTypeArg, true)
		}
		res = ann.resExpr(ctx2, e1)
		if res == nil {
			res = nilIdent()
		}
		w = append(w, res)
	}
	if len(w) == 0 {
		return nilIdent()
	}
	return ann.newDebugI(ctx, "IL", w...)
}

func (ann *Annotator) resFieldList(ctx *Ctx, fl *ast.FieldList) ast.Expr {
	w := []ast.Expr{}
	for _, f := range fl.List {
		// set field name if it has no names (otherwise it won't output)
		if len(f.Names) == 0 {
			f.Names = append(f.Names, ann.newIdent(ctx))
		}

		for i := range f.Names {
			e1 := ast.Expr(f.Names[i])
			e := ann.resExpr(ctx, &e1) // e1 won't be replaceable (local var)
			w = append(w, e)
		}
	}
	if len(w) == 0 {
		return nilIdent()
	}
	return ann.newDebugI(ctx, "IL", w...)
}

//----------
//----------
//----------

func (ann *Annotator) VisFile(ctx *Ctx, file *ast.File) {
	for _, d := range file.Decls {
		ann.vis(ctx, d)
	}
}

func (ann *Annotator) VisGenDecl(ctx *Ctx, gd *ast.GenDecl) {
	switch gd.Tok {
	case token.CONST:
		// TODO: handle iota
	case token.VAR:
		// annotate only if inside decl stmt
		// TODO: top level var decls that have a func lit assigned?
		if !ctx.boolean(ctxIdInDeclStmt) {
			break
		}

		if len(gd.Specs) >= 2 {
			ann.splitGenDecl(ctx, gd)
		}
		for _, spec := range gd.Specs {
			ann.vis(ctx, spec)
		}
	}
}

func (ann *Annotator) VisImportSpec(ctx *Ctx, is *ast.ImportSpec) {
}

func (ann *Annotator) VisDeclStmt(ctx *Ctx, ds *ast.DeclStmt) {
	// DeclStmt is a decl in a stmt list => gendecl is only const/type/var
	ctx = ctx.withBoolean(ctxIdInDeclStmt, true)
	ann.vis(ctx, ds.Decl)
}

func (ann *Annotator) VisFuncDecl(ctx *Ctx, fd *ast.FuncDecl) {
	// body can be nil (external func decls, only the header declared)
	if fd.Body == nil {
		return
	}

	//// don't annotate these functions to avoid endless loop recursion
	//if fd.Recv != nil && fd.Body != nil && fd.Name.Name == "String" && len(fd.Type.Params.List) == 0 {
	//	return
	//}
	//if fd.Recv != nil && fd.Body != nil && fd.Name.Name == "Error" && len(fd.Type.Params.List) == 0 {
	//	return
	//}

	// visit body first to avoid annotating params insertions
	ctx2 := ctx.withFuncType(fd.Type)
	ann.vis(ctx2, fd.Body)

	// visit params
	ann.visFieldList(ctx, fd.Type.Params, fd.Body)
}

func (ann *Annotator) VisBlockStmt(ctx *Ctx, bs *ast.BlockStmt) {
	ann.vis(ctx, &bs.List)
}

func (ann *Annotator) VisExprStmt(ctx *Ctx, es *ast.ExprStmt) {
	ctx = ctx.withFixedDebugIndex()
	xPos := es.X.Pos()
	e := ann.resExpr(ctx, &es.X)
	ctx3 := ctx.withInsertStmtAfter(true)
	ann.insertDebugLine(ctx3, xPos, e)
}

func (ann *Annotator) VisAssignStmt(ctx *Ctx, as *ast.AssignStmt) {
	ctx = ctx.withFixedDebugIndex()

	asPos := as.Pos()
	asTok := as.Tok

	// right hand side
	ctx3 := ctx
	if len(as.Rhs) == 1 {
		// ex: a:=b
		// ex: a,ok:= m[b]
		// ex: a,b,c:=f()
		ctx3 = ctx3.withNResults(len(as.Lhs))
	} else {
		// ex: a,b,c:=d,e,f // each of {d,e,f} return only 1 result
		ctx3 = ctx3.withNResults(1)
	}
	rhs := ann.resExprs(ctx3, &as.Rhs)

	// assign right hand side to a var before the main assign stmt
	ctx5 := ctx.withInsertStmtAfter(false)
	rhs2 := ann.assignToNewIdent(ctx5, rhs)

	// simplify (don't show lhs since it's similar to rhs)
	allSimple := false // ex: a++, a+=1
	switch asTok {
	case token.DEFINE, token.ASSIGN:
		allSimple = true
		if len(as.Rhs) == 1 {
			if _, ok := as.Rhs[0].(*ast.TypeAssertExpr); ok {
				allSimple = false
			}
		}
	}
	if allSimple {
		for _, e := range as.Lhs {
			if !isIdentOrSelectorOfIdents(e) {
				allSimple = false
				break
			}
		}
	}
	if allSimple {
		ann.insertDebugLine(ctx, asPos, rhs2)
		return
	}

	// left hand side
	// ex: a[i] // returns 1 result
	// ex: a,b,c:= // each expr returns 1 result to be debugged
	ctx4 := ctx
	ctx4 = ctx.withInsertStmtAfter(true)
	ctx4 = ctx4.withNResults(1)
	ctx4 = ctx4.withBoolean(ctxIdExprInLhs, true)
	lhs := ann.resExprs(ctx4, &as.Lhs)

	ctx = ctx.withInsertStmtAfter(true)
	opbl := basicLitInt(int(asTok))
	ce3 := ann.newDebugI(ctx, "IA", lhs, opbl, rhs2)
	ann.insertDebugLine(ctx, asPos, ce3)
}

func (ann *Annotator) VisReturnStmt(ctx *Ctx, rs *ast.ReturnStmt) {
	ft, ok := ctx.funcType()
	if !ok {
		return
	}

	// functype number of results to return
	nres := ft.Results.NumFields()
	if nres == 0 {
		// show debug step
		ce := ann.newDebugI(ctx, "ISt")
		ann.insertDebugLine(ctx, rs.Pos(), ce)
		return
	}

	// naked return (have nres>0), use results ids
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
	pos := rs.Pos()
	ctx2 := ctx
	ctx2 = ctx2.withFixedDebugIndex()
	if len(rs.Results) > 1 { // each return expr returns 1 result
		ctx2 = ctx2.withNResults(1)
	} else {
		ctx2 = ctx2.withNResults(nres)
	}
	e2 := ann.resExprs(ctx2, &rs.Results)

	ann.insertDebugLine(ctx2, pos, e2)
}

func (ann *Annotator) VisTypeSwitchStmt(ctx *Ctx, tss *ast.TypeSwitchStmt) {
	ctx = ctx.withFixedDebugIndex()
	if ok := ann.wrapInitInBlockAndVisit(ctx, tss, &tss.Init); ok {
		return
	}

	// visiting tss.assign stmt would enter into the stmt exprstmt that always inserts the stmt after; this case needs the stmt insert before
	e2 := (*ast.Expr)(nil)
	switch t := tss.Assign.(type) {
	case *ast.ExprStmt:
		e2 = &t.X
	case *ast.AssignStmt:
		if len(t.Rhs) == 1 {
			e2 = &t.Rhs[0]
		}
	}
	if e2 != nil {
		pos := (*e2).Pos()
		ctx2 := ctx
		ctx2 = ctx2.withFixedDebugIndex()
		ctx2 = ctx2.withInsertStmtAfter(false)
		e := ann.resExpr(ctx2, e2)
		ann.insertDebugLine(ctx2, pos, e)
	}

	ctx3 := ctx.withNilFixedDebugIndex()
	ann.vis(ctx3, tss.Body)
}

func (ann *Annotator) VisSwitchStmt(ctx *Ctx, ss *ast.SwitchStmt) {
	ctx = ctx.withFixedDebugIndex()
	if ok := ann.wrapInitInBlockAndVisit(ctx, ss, &ss.Init); ok {
		return
	}

	if ss.Tag != nil {
		tagPos := ss.Tag.Pos()
		ctx3 := ctx
		ctx3 = ctx3.withNResults(1) // ex: switch f1() // f1 has 1 result
		e := ann.resExpr(ctx3, &ss.Tag)
		ann.insertDebugLine(ctx3, tagPos, e)
	}

	// reset fixed debug index to visit the body since it has several stmts
	ctx2 := ctx.withNilFixedDebugIndex()
	ann.vis(ctx2, ss.Body)
}

func (ann *Annotator) VisIfStmt(ctx *Ctx, is *ast.IfStmt) {
	ctx = ctx.withFixedDebugIndex()
	if ok := ann.wrapInitInBlockAndVisit(ctx, is, &is.Init); ok {
		return
	}

	// condition
	isPos := is.Cond.Pos()
	ctx2 := ctx
	ctx2 = ctx2.withNResults(1)
	ctx2 = ctx2.withInsertStmtAfter(false)
	e := ann.resExpr(ctx2, &is.Cond)
	ann.insertDebugLine(ctx2, isPos, e)

	// reset fixed debug index to visit the body since it has several stmts
	ctx3 := ctx.withNilFixedDebugIndex()
	ann.vis(ctx3, is.Body)

	switch is.Else.(type) {
	case *ast.IfStmt: // "else if ..."
		// wrap in block stmt to visit "if" stmt
		bs := &ast.BlockStmt{List: []ast.Stmt{is.Else}}
		is.Else = bs
	}
	ann.vis(ctx3, is.Else)
}

func (ann *Annotator) VisForStmt(ctx *Ctx, fs *ast.ForStmt) {
	ctx = ctx.withFixedDebugIndex()
	if ok := ann.wrapInitInBlockAndVisit(ctx, fs, &fs.Init); ok {
		return
	}

	// visit the body first to avoid annotating inserted stmts of cond/post later
	ctx7 := ctx.withNilFixedDebugIndex()
	ann.vis(ctx7, fs.Body)

	if fs.Cond != nil {
		fsBodyCtx := ctx.withStmts(&fs.Body.List)
		condPos := fs.Cond.Pos()

		e := ann.resExpr(fsBodyCtx, &fs.Cond)
		ann.insertDebugLine(fsBodyCtx, condPos, e)

		// create ifstmt to break the loop
		ue := &ast.UnaryExpr{Op: token.NOT, X: fs.Cond} // negate
		is := &ast.IfStmt{If: fs.Pos(), Cond: ue, Body: &ast.BlockStmt{}}
		fs.Cond = nil // clear forstmt condition
		fsBodyCtx.insertStmt(is)
		isBodyCtx := ctx.withStmts(&is.Body.List)

		// insert break inside ifstmt
		brk := &ast.BranchStmt{Tok: token.BREAK}
		isBodyCtx.insertStmt(brk)
	}

	if fs.Post != nil {
		fsBodyCtx := ctx.withStmts(&fs.Body.List) // TODO: review

		// init flag var to run post stmt
		flagVar := ann.newIdent(ctx)
		as4 := ann.newAssignStmt11(flagVar, &ast.Ident{Name: "false"})
		ctx.insertStmt(as4)

		// inside the forloop: ifstmt to know if the post stmt can run
		is := &ast.IfStmt{Body: &ast.BlockStmt{}}
		is.Cond = flagVar
		fsBodyCtx.insertStmt(is)
		isBodyCtx := ctx.withStmts(&is.Body.List)

		// move fs.Post to inside the ifstmt
		isBodyCtx.insertStmt(fs.Post)
		fs.Post = nil

		// visit ifstmt body that now contains the post stmt
		ann.vis(ctx, is.Body)

		// insert stmt that sets the flag to true
		as5 := ann.newAssignStmt11(flagVar, &ast.Ident{Name: "true"})
		as5.Tok = token.ASSIGN
		fsBodyCtx.insertStmt(as5)
	}
}

func (ann *Annotator) VisRangeStmt(ctx *Ctx, rs *ast.RangeStmt) {
	ctx = ctx.withFixedDebugIndex()

	xPos := rs.X.Pos()
	rsTok := rs.Tok

	// range stmt
	ctx3 := ctx.withNResults(1)
	x := ann.resExpr(ctx3, &rs.X)
	//ann.insertDebugLine(ctx3, xPos, x)

	// length of x (insert before for stmt)
	e5 := &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{rs.X}}
	e6 := ann.newDebugI(ctx, "IVr", e5)
	e7 := ann.assignToNewIdent(ctx, e6) // var is of type item

	// first debug line (entering the loop)
	e11 := ann.newDebugIL(ctx, e7, x)
	ann.insertDebugLine(ctx, xPos, e11)

	// visit the body first to avoid annotating inserted stmts
	ctx6 := ctx.withNilFixedDebugIndex()
	ann.vis(ctx6, rs.Body)

	// key and value
	kvPos := token.Pos(0)
	rsBodyCtx := ctx.withStmts(&rs.Body.List)
	kves := []ast.Expr{}
	if rs.Key != nil {
		kvPos = rs.Key.Pos()
		e2 := ann.resExpr(rsBodyCtx, &rs.Key)
		kves = append(kves, e2)
	}
	if rs.Value != nil {
		if kvPos == 0 {
			kvPos = rs.Value.Pos()
		}
		e2 := ann.resExpr(rsBodyCtx, &rs.Value)
		kves = append(kves, e2)
	}

	if len(kves) > 0 {
		ctx5 := rsBodyCtx
		opbl := basicLitInt(int(rsTok))
		rhs := ann.newDebugI(ctx5, "IL", []ast.Expr{e7}...)
		lhs := ann.newDebugI(ctx5, "IL", kves...)
		e9 := ann.newDebugI(ctx5, "IA", lhs, opbl, rhs)
		ann.insertDebugLine(ctx5, kvPos, e9)
	}
}

func (ann *Annotator) VisIncDecStmt(ctx *Ctx, ids *ast.IncDecStmt) {
	ctx = ctx.withFixedDebugIndex()
	idsPos := ids.X.Pos()
	idsTok := ids.Tok

	// value before
	e1 := ann.resExpr(ctx, &ids.X)
	l1 := ann.newDebugI(ctx, "IL", e1)
	ctx3 := ctx.withInsertStmtAfter(false)
	l1before := ann.assignToNewIdent(ctx3, l1)

	// value after
	ctx2 := ctx.withInsertStmtAfter(true)
	e2 := ann.resExpr(ctx2, &ids.X)
	l2 := ann.newDebugI(ctx, "IL", e2)

	opbl := basicLitInt(int(idsTok))
	ce3 := ann.newDebugI(ctx2, "IA", l2, opbl, l1before)
	ann.insertDebugLine(ctx2, idsPos, ce3)
}

func (ann *Annotator) VisLabeledStmt(ctx *Ctx, ls *ast.LabeledStmt) {
	// Problem:
	// -	label1: ; // inserting empty stmt breaks compilation
	// 	for { break label1 } // compile error: invalid break label
	// -	using block stmts won't work
	// 	label1:
	// 	{ for { break label1} } // compile error
	// No way to insert debug stmts between the label and the stmt.
	// Just make a debug step with "label" where a warning can be shown.

	if ls.Stmt == nil {
		return
	}

	switch ls.Stmt.(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		// can't insert stmts between the label and the stmt (alters program)

		ctx = ctx.withLabeledStmt(ls)
		ann.vis(ctx, ls.Stmt)
	default:
		// use empty stmt to insert stmts between label and stmt
		stmt := ls.Stmt
		ls.Stmt = &ast.EmptyStmt{}

		// insert stmt before, consider stmt done
		ctx3 := ctx.withInsertStmtAfter(false)
		ctx3.insertStmt(ls)

		// replace stmt and continue visit
		ctx.replaceStmt(stmt)
		ann.vis(ctx, stmt)
	}
}

func (ann *Annotator) VisBranchStmt(ctx *Ctx, bs *ast.BranchStmt) {
	ce := ann.newDebugI(ctx, "IBr")
	ann.insertDebugLine(ctx, bs.Pos(), ce)
}

func (ann *Annotator) VisDeferStmt(ctx *Ctx, ds *ast.DeferStmt) {
	ann.visDeferStmt2(ctx, &ds.Call)
}
func (ann *Annotator) VisGoStmt(ctx *Ctx, gs *ast.GoStmt) {
	ann.visDeferStmt2(ctx, &gs.Call)
}
func (ann *Annotator) visDeferStmt2(ctx *Ctx, cep **ast.CallExpr) {
	ctx = ctx.withCallExpr(cep)
	ce := *cep

	funPos := ce.Fun.Pos()

	// assign arguments to tmp variables
	if len(ce.Args) > 0 {
		//args2 := make([]ast.Expr, len(ce.Args))
		//copy(args2, ce.Args)
		//ids := ann.assignToNewIdents2(ctx, len(ce.Args), args2...)
		for i, e := range ce.Args {
			if ann.canAssignToVar(ctx, e) {
				id := ann.assignToNewIdent(ctx, e)
				ce.Args[i] = id
			}
			//else {
			//ce.Args[i] = ids[i]
			//}
		}
	}

	ctx = ctx.withFixedDebugIndex()

	// handle funclit
	ceFunIsFuncLit := false
	switch ce.Fun.(type) {
	case *ast.FuncLit:
		ceFunIsFuncLit = true
		ctx3 := ctx.withNResults(1)
		fun := ann.resExpr(ctx3, &ce.Fun)
		ann.insertDebugLine(ctx, funPos, fun)
	}

	// replace func call with wrapped function
	fl2 := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{&ast.ExprStmt{X: ce}},
		},
	}
	ce2 := &ast.CallExpr{Fun: fl2}
	ctx.replaceCallExpr(ce2)

	// temporary switch names for better debug line
	if ceFunIsFuncLit {
		if id, ok := ce.Fun.(*ast.Ident); ok {
			name := id.Name
			id.Name = "f"
			defer func() { id.Name = name }()
		}
	}

	// visit block with the func call to insert debug stmts inside
	ann.vis(ctx, fl2.Body)
}

func (ann *Annotator) VisValueSpec(ctx *Ctx, vs *ast.ValueSpec) {
	// ex: var a, b int = 1, 2
	// ex: var a, b = f()
	if len(vs.Values) > 0 {
		pos := vs.Pos()
		nres := len(vs.Names)
		if len(vs.Values) >= 2 {
			nres = 1
		}
		ctx2 := ctx
		ctx2 = ctx2.withFixedDebugIndex()
		ctx2 = ctx2.withNResults(nres)
		e := ann.resExprs(ctx2, &vs.Values)
		ann.insertDebugLine(ctx2, pos, e)
	}
}

func (ann *Annotator) VisSelectStmt(ctx *Ctx, ss *ast.SelectStmt) {
	// debug step to show it has entered the select statement
	e1 := ann.newDebugI(ctx, "ISt")
	ann.insertDebugLine(ctx, ss.Pos(), e1)

	ann.vis(ctx, ss.Body)
}

func (ann *Annotator) VisCaseClause(ctx *Ctx, cc *ast.CaseClause) {
	// visit body
	ann.vis(ctx, &cc.Body)

	// debug step showing the case was entered
	ctx2 := ctx.withStmts(&cc.Body)
	ce := ann.newDebugI(ctx2, "ISt")
	ann.insertDebugLine(ctx2, cc.Case, ce)
}

func (ann *Annotator) VisCommClause(ctx *Ctx, cc *ast.CommClause) {
	// visit body first to avoid annotating inserted stmts
	ann.vis(ctx, &cc.Body)

	// debug step showing the case was entered
	ctx2 := ctx.withStmts(&cc.Body)
	e1 := ann.newDebugI(ctx2, "ISt")
	ann.insertDebugLine(ctx2, cc.Case, e1)
}

func (ann *Annotator) VisSendStmt(ctx *Ctx, ss *ast.SendStmt) {
	pos := ss.Pos()

	ctx2 := ctx.withNResults(1)
	val := ann.resExpr(ctx2, &ss.Value)

	ctx3 := ctx.withInsertStmtAfter(true)
	ch := ann.resExpr(ctx3, &ss.Chan)

	ce := ann.newDebugI(ctx, "IS", ch, val)
	ann.insertDebugLine(ctx3, pos, ce)
}

//----------

func (ann *Annotator) visStmts(ctx *Ctx, stmts *[]ast.Stmt) {
	ctx2 := ctx.withStmts(stmts)
	for stmt := ctx2.curStmt(); stmt != nil; stmt = ctx2.nextStmt() {
		// handle here to enabled/disable next stmts in this list
		ctx2 = ann.annotationBlockCtx(ctx2, stmt)

		ann.vis(ctx2, stmt)
	}
}

// NOTE: mostly used for params (fixed debug index)
func (ann *Annotator) visFieldList(ctx *Ctx, fl *ast.FieldList, body *ast.BlockStmt) {
	if len(fl.List) == 0 {
		return
	}
	ctx2 := ctx
	ctx2 = ctx2.withStmts(&body.List)
	ctx2 = ctx2.withFixedDebugIndex()
	e1 := ann.resFieldList(ctx2, fl)
	ann.insertDebugLine(ctx2, fl.Opening, e1)
}

//----------
//----------
//----------

func (ann *Annotator) ResCallExpr(ctx *Ctx, ce *ast.CallExpr) ast.Expr {
	cePos := ce.Pos()

	// handle fun expr
	isFuncLit := false
	isPanic := false
	isFirstArgType := false
	switch t := ce.Fun.(type) {
	case *ast.Ident:
		// handle builtin funcs
		switch t.Name {
		case "panic":
			if ann.isBuiltin(t) {
				isPanic = true
			}
		case "new", "make":
			if ann.isBuiltin(t) {
				isFirstArgType = true
			}
		}
	case *ast.FuncLit:
		isFuncLit = true
	}

	// visit fun expr
	ctx5 := ctx.withNResults(1)
	ctx5 = ctx5.withBoolean(ctxIdNameInsteadOfValue, true)
	fx := ann.resExpr(ctx5, &ce.Fun)
	if isFuncLit {
		ann.insertDebugLine(ctx, cePos, fx) // show step
		fx = ann.newDebugIVsBl(ctx, "f")    // continue with simple name
	}

	// visit args
	ctx2 := ctx
	ctx2 = ctx2.withNResults(1) // each arg returns 1 result
	ctx2 = ctx2.withBoolean(ctxIdFirstArgIsType, isFirstArgType)
	args := ann.resExprs(ctx2, &ce.Args)

	// show stepping in (insert before func call)
	args2 := append([]ast.Expr{fx}, args)
	e4 := ann.newDebugI(ctx, "ICe", args2...)
	ann.insertDebugLine(ctx, cePos, e4)

	// avoid "line unreachable" compiler errors
	if isPanic {
		return afterPanicIdent()
	}

	result := ann.solveNResults(ctx, ce)
	//if ctx.boolean(ctxIdNameInsteadOfValue) {
	//	result = nilIdent()
	//}

	return ann.newDebugI(ctx, "IC", e4, result)
}

func (ann *Annotator) ResBasicLit(ctx *Ctx, bl *ast.BasicLit) ast.Expr {
	return ann.newDebugIVi(ctx, bl)
}

func (ann *Annotator) ResIdent(ctx *Ctx, id *ast.Ident) ast.Expr {
	if id.Name == "_" {
		return ann.newDebugI(ctx, "IAn")
	}
	if ctx.boolean(ctxIdInTypeArg) {
		return ann.resType(ctx, id)
	}
	if ctx.boolean(ctxIdNameInsteadOfValue) {
		return ann.newDebugIVsBl(ctx, id.Name)
	}
	return ann.newDebugIVi(ctx, id)
}

func (ann *Annotator) ResUnaryExpr(ctx *Ctx, ue *ast.UnaryExpr) ast.Expr {
	xPos := ue.X.Pos()

	ctx2 := ctx
	if ue.Op == token.AND {
		ctx2 = ctx2.withTakingVarAddress(ue.X)
	}
	x := ann.resExpr(ctx2, &ue.X)

	opbl := basicLitInt(int(ue.Op))
	e4 := ann.newDebugI(ctx, "IUe", opbl, x)
	if ue.Op == token.ARROW {
		ann.insertDebugLine(ctx, xPos, e4)
	}

	result := ann.solveNResults(ctx, ue)

	return ann.newDebugI(ctx, "IU", e4, result)
}

func (ann *Annotator) ResFuncLit(ctx *Ctx, fl *ast.FuncLit) ast.Expr {
	ctx0 := ctx // for the last return
	ctx = ctx.withResetForFuncLit()

	// visit body first to avoid annotating params insertions
	ctx3 := ctx
	ctx3 = ctx3.withFuncType(fl.Type)
	ann.vis(ctx3, fl.Body)

	// visit params
	ann.visFieldList(ctx, fl.Type.Params, fl.Body)

	return ann.solveNResults(ctx0, fl)
}

func (ann *Annotator) ResSelectorExpr(ctx *Ctx, se *ast.SelectorExpr) ast.Expr {
	if ctx.boolean(ctxIdInTypeArg) {
		return ann.resType(ctx, se)
	}

	// simplify
	if isIdentOrSelectorOfIdents(se) {
		if ctx.boolean(ctxIdNameInsteadOfValue) {
			return ann.newDebugIVsBl(ctx, se.Sel.Name)
		}
		return ann.newDebugIVi(ctx, se)
	}

	//xPos := se.X.Pos()
	ctx4 := ctx.withNResults(1) //.withNameInsteadOfValue(false)
	x := ann.resExpr(ctx4, &se.X)
	//ann.insertDebugLine(ctx4, xPos, x)

	sel := ast.Expr(nil)
	if ctx.boolean(ctxIdNameInsteadOfValue) {
		sel = ann.newDebugIVsBl(ctx, se.Sel.Name)
	} else {
		sel = ann.newDebugIVi(ctx, se)
	}
	return ann.newDebugI(ctx, "ISel", x, sel)

	//ctx3 := ctx.withExpr(&se.X).withNResults(1)
	//result := ann.solveNResults(ctx3, se.X)

	//return ann.newDebugItem(ctx, "ISel", result, sel)
}

func (ann *Annotator) ResCompositeLit(ctx *Ctx, cl *ast.CompositeLit) ast.Expr {
	// NOTE: not doing cl.Type
	e := ann.resExprs(ctx, &cl.Elts)
	return ann.newDebugI(ctx, "ILit", e)
}

func (ann *Annotator) ResKeyValueExpr(ctx *Ctx, kv *ast.KeyValueExpr) ast.Expr {
	k := ast.Expr(nil)
	if id, ok := kv.Key.(*ast.Ident); ok {
		bl := basicLitStringQ(id.Name)
		k = ann.newDebugI(ctx, "IVs", bl)
	} else {
		k = ann.resExpr(ctx, &kv.Key)
	}
	v := ann.resExpr(ctx, &kv.Value)
	return ann.newDebugI(ctx, "IKV", k, v)
}

func (ann *Annotator) ResTypeAssertExpr(ctx *Ctx, tae *ast.TypeAssertExpr) ast.Expr {
	// tae.Type!=nil is "t,ok:=X.(<type>)"
	// tae.Type==nil is "switch X.(type)"
	isSwitch := tae.Type == nil

	ctx2 := ctx
	ctx2 = ctx2.withNResults(1) // ex: f().(type) // f() returns 1 result
	x := ann.resExpr(ctx2, &tae.X)

	// simplify
	if !isSwitch {
		return x
	}

	xt := ann.newDebugI(ctx, "IVt", tae.X)
	return ann.newDebugI(ctx, "ITA", x, xt)
}

func (ann *Annotator) ResParenExpr(ctx *Ctx, pe *ast.ParenExpr) ast.Expr {
	x := ann.resExpr(ctx, &pe.X)
	return ann.newDebugI(ctx, "IP", x)
}

func (ann *Annotator) ResIndexExpr(ctx *Ctx, ie *ast.IndexExpr) ast.Expr {
	// ex: a,b=c[i],d[j]
	// ex: a,ok:=m[f1()] // map access, more then 1 result

	isSimple := isIdentOrSelectorOfIdents(ie.X)

	ctx2 := ctx.withBoolean(ctxIdNameInsteadOfValue, true)
	x := ann.resExpr(ctx2, &ie.X)

	// wrap in parenthesis
	if !isSimple {
		x = ann.newDebugIP(ctx, x)
	}

	// Index expr
	ctx3 := ctx
	ctx3 = ctx3.withNResults(1)                    // ex: a[f()] // f() returns 1 result
	ctx3 = ctx3.withInsertStmtAfter(false)         // ex: a[f()] // f() must be before
	ctx3 = ctx3.withBoolean(ctxIdExprInLhs, false) // ex: a[f()] // allow to replace f()
	ix := ann.resExpr(ctx3, &ie.Index)

	result := ann.solveNResults(ctx, ie)

	return ann.newDebugI(ctx, "II", x, ix, result)
}

func (ann *Annotator) ResSliceExpr(ctx *Ctx, se *ast.SliceExpr) ast.Expr {
	isSimple := isIdentOrSelectorOfIdents(se.X)

	ctx2 := ctx.withBoolean(ctxIdNameInsteadOfValue, true)
	x := ann.resExpr(ctx2, &se.X)

	// wrap in parenthesis
	if !isSimple {
		x = ann.newDebugIP(ctx, x)
	}

	// index expr
	ix := []ast.Expr{}
	for _, e := range []*ast.Expr{&se.Low, &se.High, &se.Max} {
		r := ann.resExpr(ctx, e)
		ix = append(ix, r)
	}

	result := ann.solveNResults(ctx, se)

	// slice3: 2 colons present
	s := "false"
	if se.Slice3 {
		s = "true"
	}
	slice3Bl := basicLitString(s)

	return ann.newDebugI(ctx, "II2", x, ix[0], ix[1], ix[2], slice3Bl, result)
}

func (ann *Annotator) ResStarExpr(ctx *Ctx, se *ast.StarExpr) ast.Expr {
	if ann.isType(se) {
		return ann.resType(ctx, se)
	}

	// ex: *a=1
	ctx = ctx.withNResults(1)

	x := ann.resExpr(ctx, &se.X)
	//return x

	result := ann.solveNResults(ctx, se)

	opbl := basicLitInt(int(token.MUL))
	e2 := ann.newDebugI(ctx, "IUe", opbl, x)
	return ann.newDebugI(ctx, "IU", e2, result)
}

func (ann *Annotator) ResMapType(ctx *Ctx, mt *ast.MapType) ast.Expr {
	return ann.resType(ctx, mt)
}
func (ann *Annotator) ResChanType(ctx *Ctx, ct *ast.ChanType) ast.Expr {
	return ann.resType(ctx, ct)
}
func (ann *Annotator) ResArrayType(ctx *Ctx, at *ast.ArrayType) ast.Expr {
	return ann.resType(ctx, at)
}
func (ann *Annotator) ResInterfaceType(ctx *Ctx, it *ast.InterfaceType) ast.Expr {
	return ann.resType(ctx, it)
}

//----------

func (ann *Annotator) ResBinaryExpr(ctx *Ctx, be *ast.BinaryExpr) ast.Expr {
	ctx = ctx.withNResults(1)
	switch be.Op {
	case token.LAND, token.LOR:
		return ann.resBinaryExprAndOr(ctx, be)
	default:
		return ann.resBinaryExpr2(ctx, be)
	}
}
func (ann *Annotator) resBinaryExpr2(ctx *Ctx, be *ast.BinaryExpr) ast.Expr {
	x := ann.resExpr(ctx, &be.X)
	y := ann.resExpr(ctx, &be.Y)
	result := ann.solveNResults(ctx, be)
	opbl := basicLitInt(int(be.Op))
	return ann.newDebugI(ctx, "IB", x, opbl, y, result)
}
func (ann *Annotator) resBinaryExprAndOr(ctx *Ctx, be *ast.BinaryExpr) ast.Expr {
	// ex: f1() || f2() // f2 should not be called if f1 is true
	// ex: f1() && f2() // f2 should not be called if f1 is false

	x := ann.resExpr(ctx, &be.X)

	// init result var with be.X
	resVar := ast.Expr(ann.newIdent(ctx))
	as4 := ann.newAssignStmt11(resVar, be.X)
	as4.Tok = token.DEFINE
	ctx.insertStmt(as4)
	// replace expr with result var
	ctx.replaceExpr(resVar)

	// init y result var (in case be.Y doesn't run)
	ybl := ann.newDebugIVsBl(ctx, "?")
	yRes := ann.assignToNewIdent(ctx, ybl)

	// x condition based on being "&&" or "||"
	xCond := resVar
	if be.Op == token.LOR {
		xCond = &ast.UnaryExpr{Op: token.NOT, X: xCond}
	}

	// ifstmt to test x result to decide whether to run y
	is := &ast.IfStmt{If: be.Pos(), Body: &ast.BlockStmt{}}
	is.Cond = xCond
	ctx.insertStmt(is)
	isBodyCtx := ctx.withStmts(&is.Body.List)

	y := ann.resExpr(isBodyCtx, &be.Y)

	// (inside ifstmt) assign debug result to y
	as3 := ann.newAssignStmt11(yRes, y)
	as3.Tok = token.ASSIGN
	isBodyCtx.insertStmt(as3)

	// inside ifstmt: assign be.Y to result var
	as2 := ann.newAssignStmt11(resVar, be.Y)
	as2.Tok = token.ASSIGN
	isBodyCtx.insertStmt(as2)

	opbl := basicLitInt(int(be.Op))
	resVar2 := ann.newDebugIVi(ctx, resVar)
	return ann.newDebugI(ctx, "IB", x, opbl, yRes, resVar2)
}

//----------
//----------
//----------

// Mostly gets the expression into a var(s) to be able to use the result without calculating it twice. Ex: don't double call f().
func (ann *Annotator) solveNResults(ctx *Ctx, e ast.Expr) ast.Expr {
	// ex: a, b = c[i], d[j] // 1 result for each expr
	// ex: a, ok := c[f1()] // map access, more then 1 result
	// ex: a[i]=b // can't replace a[i]
	// ex: a:=&b[i] -> d0:=b[i];a:=&d0 // d0 wrong address

	nres := ctx.nResults()
	if nres == 0 {
		return nilIdent() // ex: f1() // with no return value
	}

	if !ann.canAssignToVar(ctx, e) {
		return ann.newDebugIVi(ctx, e)
	}

	if ce, ok := e.(*ast.CallExpr); ok {
		if typ, ok := ann.typev(ce); ok {
			switch t := typ.Type.(type) {
			case *types.Tuple:
				nres = t.Len()
			}
		}
	}

	if nres >= 2 {
		ids := ann.assignToNewIdents2(ctx, nres, e)
		ctx.replaceExprs(ids)
		ids2 := ann.wrapInIVi(ctx, ids...)
		return ann.newDebugIL(ctx, ids2...)
	}

	ctx = ctx.withInsertStmtAfter(false)

	// TODO: review
	//if ctx.exprOnLhs() {
	//	// get address of the expression to replace
	//	ue := &ast.UnaryExpr{Op: token.AND, X: e}
	//	e2 := ann.assignToNewIdent2(ctx, ue)
	//	se := &ast.StarExpr{X: e2}
	//	ctx.replaceExpr(se)
	//	return ann.newDebugItem(ctx, "IVi", e2)
	//}

	e2 := ann.assignToNewIdent(ctx, e)
	ctx.replaceExpr(e2)
	return ann.newDebugIVi(ctx, e2)
}

//----------

func (ann *Annotator) newDebugIVi(ctx *Ctx, e ast.Expr) ast.Expr {
	if s, ok := ann.isBigConst(e); ok {
		return ann.newDebugIVsBl(ctx, s)
	}
	return ann.newDebugI(ctx, "IVi", e)
}
func (ann *Annotator) newDebugIVsBl(ctx *Ctx, s string) ast.Expr {
	return ann.newDebugI(ctx, "IVs", basicLitStringQ(s))
}
func (ann *Annotator) newDebugIL(ctx *Ctx, es ...ast.Expr) ast.Expr {
	return ann.newDebugI(ctx, "IL", es...)
}
func (ann *Annotator) newDebugIP(ctx *Ctx, e ast.Expr) ast.Expr {
	return ann.newDebugI(ctx, "IP", e)
}

func (ann *Annotator) wrapInIVi(ctx *Ctx, es ...ast.Expr) []ast.Expr {
	ivs := []ast.Expr{}
	for _, e := range es {
		e2 := ann.newDebugIVi(ctx, e)
		ivs = append(ivs, e2)
	}
	return ivs
}

func (ann *Annotator) newDebugI(ctx *Ctx, fname string, es ...ast.Expr) ast.Expr {
	se := &ast.SelectorExpr{
		X:   ast.NewIdent(ann.debugPkgName),
		Sel: ast.NewIdent(fname),
	}
	ce := &ast.CallExpr{Fun: se, Args: es}

	// TODO: handle this where they are called?
	// assign to var because these are called more then once
	assign := false
	switch fname {
	case "ICe", "IUe":
		assign = true
	}
	if assign {
		return ann.assignToNewIdent(ctx, ce)
	}

	return ce
}

//----------

func (ann *Annotator) insertDebugLine(ctx *Ctx, pos token.Pos, expr ast.Expr) {
	if isAfterPanicIdent(expr) {
		return
	}
	stmt := ann.newDebugLine(ctx, pos, expr)
	ctx.insertStmt(stmt)
}
func (ann *Annotator) newDebugLine(ctx *Ctx, pos token.Pos, expr ast.Expr) ast.Stmt {
	se := &ast.SelectorExpr{
		X:   ast.NewIdent(ann.debugPkgName),
		Sel: ast.NewIdent("Line"),
	}
	args := []ast.Expr{
		basicLitInt(ann.fileIndex),
		basicLitInt(ctx.nextDebugIndex()),
		basicLitInt(ann.fset.Position(pos).Offset),
		expr,
	}
	es := &ast.ExprStmt{X: &ast.CallExpr{Fun: se, Args: args}}
	ann.builtDebugLineStmt = true
	return es
}

//----------

func (ann *Annotator) assignToNewIdent(ctx *Ctx, expr ast.Expr) ast.Expr {
	return ann.assignToNewIdents2(ctx, 1, expr)[0]
}
func (ann *Annotator) assignToNewIdents2(ctx *Ctx, nIds int, exprs ...ast.Expr) []ast.Expr {
	ids := []ast.Expr{}
	for i := 0; i < nIds; i++ {
		id := ann.newIdent(ctx)

		// have id use expr position
		k := i
		if len(exprs) == 1 {
			k = 0
		}
		id.NamePos = exprs[k].Pos()

		ids = append(ids, id)
	}
	as := ann.newAssignStmt(ids, exprs)
	ctx.insertStmt(as)
	return ids
}

//----------

func (ann *Annotator) newAssignStmt11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return ann.newAssignStmt([]ast.Expr{lhs}, []ast.Expr{rhs})
}
func (ann *Annotator) newAssignStmt(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.DEFINE, Lhs: lhs, Rhs: rhs}
}

//----------

func (ann *Annotator) newIdent(ctx *Ctx) *ast.Ident {
	return &ast.Ident{Name: ann.newVarName(ctx)}
}
func (ann *Annotator) newVarName(ctx *Ctx) string {
	s := fmt.Sprintf("%s%d", ann.debugVarPrefix, ann.debugVarNameIndex)
	ann.debugVarNameIndex++
	return s
}

//----------

// assigning the expr to a var could create a var of different type,
// if the expr is on the left-hand-side, it could alter program behavior if the original destination doesn't get the value
func (ann *Annotator) canAssignToVar(ctx *Ctx, e ast.Expr) bool {
	if ann.isType(e) {
		return false
	}
	if ann.isConst(e) {
		return false
	}
	if isNilIdent(e) {
		return false
	}

	// TODO: review
	if e2, ok := ctx.takingVarAddress(); ok {
		if e2 == e {
			return false
		}
	}

	switch t := e.(type) {
	case *ast.BasicLit:
		return false
	case *ast.CallExpr:
		return true
	case *ast.FuncLit:
		return true
	case *ast.TypeAssertExpr:
		return true
	case *ast.ParenExpr:
		return ann.canAssignToVar(ctx, t.X)
	case *ast.StarExpr:
		return !ctx.boolean(ctxIdExprInLhs) && !ann.isType(t.X)
	case *ast.SelectorExpr:
		// could be a const in a pkg (pkg.const1)
		//return !ann.isType(t.Sel) && !ann.isConst(t.Sel)
		return ann.canAssignToVar(ctx, t.Sel)
	case *ast.Ident:
		//return !ann.isType(t) && !ann.isConst(t)
		return true
	case *ast.IndexExpr, *ast.SliceExpr:
		// ex: a[i]=b // can't replace a[i] // TODO: it could with an address but that can't be done here
		return !ctx.boolean(ctxIdExprInLhs)

	case *ast.BinaryExpr:
		switch t.Op {
		// depend on one being assignable (both could be consts)
		case
			token.ADD, // +
			token.SUB, // -
			token.MUL, // *
			token.QUO, // /
			token.REM, // %

			token.AND, // &
			token.OR,  // |
			token.XOR, // ^
			//token.SHL,     // << // depend on left arg only
			//token.SHR,     // >> // depend on left arg only
			token.AND_NOT: // &^
			return ann.canAssignToVar(ctx, t.X) || ann.canAssignToVar(ctx, t.Y)

		// depend on the left arg
		case token.SHL, // <<
			token.SHR: // >>
			return ann.canAssignToVar(ctx, t.X)

		default:
			// ex: a>b
			return true
		}

	case *ast.UnaryExpr:
		switch t.Op {
		case token.AND, // ex: &a
			token.ARROW: // ex: <-a
			return true
		case token.ADD, // ex: 1+(+2)
			token.SUB, // ex: 1+(-2)
			token.XOR, // ex: ^a
			token.NOT: // ex: !a
			return ann.canAssignToVar(ctx, t.X) // can be a const
		}

	default:
		fmt.Printf("todo: canassigntovar: %T", e)
	}
	return false
}

//----------

func (ann *Annotator) wrapInitInBlockAndVisit(ctx *Ctx, stmt ast.Stmt, initStmt *ast.Stmt) bool {
	if *initStmt == nil {
		return false
	}

	if _, ok := ctx.labeledStmt(); ok {
		// setup debug msg
		bls := basicLitStringQ("init not annotated due to label stmt")
		ce := ann.newDebugI(ctx, "ILa", bls)
		ann.insertDebugLine(ctx, stmt.Pos(), ce)
		return false
	}

	// wrap in blockstmt to have vars that belong only to init
	bs := &ast.BlockStmt{}
	bs.List = append(bs.List, *initStmt, stmt)
	//ctx.nilifyStmt(initStmt)
	*initStmt = nil
	ctx.replaceStmt(bs)

	ann.vis(ctx, bs)

	return true
}

//func (ann *Annotator) wrapInBlockIfInit(ctx *Ctx, stmt ast.Stmt, initStmt *ast.Stmt) (*Ctx, bool) {
//	if *initStmt == nil {
//		return nil, false
//	}

//	//if _, ok := ctx.labeledStmt(); ok {
//	//	// setup debug msg
//	//	bls := basicLitStringQ("init not annotated due to label stmt")
//	//	ce := ann.newDebugItem(ctx, "ILa", bls)
//	//	ann.insertDebugLine(ctx, stmt.Pos(), ce)
//	//	return false
//	//}

//	// wrap in blockstmt to have vars that belong only to init
//	bs := &ast.BlockStmt{}
//	bs.List = append(bs.List, *initStmt, stmt)
//	*initStmt = nil
//	ctx.replaceStmt(bs)

//	ctx2 := ctx.withStmts(&bs.List)
//	ann.vis(ctx2, bs)

//	return true
//}

//----------

//func (ann *Annotator) castStringToItem(ctx *Ctx, v string) ast.Expr {
//	se := &ast.SelectorExpr{
//		X:   ast.NewIdent(ann.debugPkgName),
//		Sel: ast.NewIdent("Item"),
//	}
//	bl := basicLitStringQ(v)
//	return &ast.CallExpr{Fun: se, Args: []ast.Expr{bl}}
//}

func (ann *Annotator) resType(ctx *Ctx, e ast.Expr) ast.Expr {
	//str := "type"
	//switch t := e.(type) {
	//case *ast.InterfaceType:
	//case *ast.MapType:
	//	str = "map"
	//case *ast.ArrayType:
	//	str = "array"
	//	if t.Len == nil {
	//		str = "slice"
	//	}
	//case *ast.SelectorExpr:
	//	str = "type"
	//}

	return ann.newDebugIVsBl(ctx, "T")
}

//----------

func (ann *Annotator) splitGenDecl(ctx *Ctx, gd *ast.GenDecl) {
	// keep to reset later
	si := ctx.stmtsIter()
	after := si.after

	ctx2 := ctx.withInsertStmtAfter(true)
	for _, spec := range gd.Specs {
		gd2 := &ast.GenDecl{Tok: gd.Tok, Specs: []ast.Spec{spec}}
		stmt := &ast.DeclStmt{Decl: gd2}
		ctx2.insertStmt(stmt)
	}
	// reset counter to have the other specs be visited
	si.after = after
	// clear gd stmt, will output as "var ()"
	gd.Specs = nil
}

//----------

func (ann *Annotator) annotationBlockCtx(ctx *Ctx, n ast.Node) *Ctx {
	// TODO: catches "godebug" directives surrounded with blank lines?

	// catch "godebug" directive at the top of this node
	on, ok := ann.annotationBlockOn(n)

	// if annotating only blocks, start with no annotations
	// TODO: confusing? there is no other way having only a block being annotated and don't annotate the rest
	if ok && on {
		if _, ok2 := n.(*ast.File); ok2 {
			on = false
		}
	}

	if ok && ctx.boolean(ctxIdNoAnnotations) != !on {
		return ctx.withBoolean(ctxIdNoAnnotations, !on)
	}
	return ctx
}

// Returns (on/off, ok)
func (ann *Annotator) annotationBlockOn(n ast.Node) (bool, bool) {
	if ann.nodeAnnTypes == nil {
		return false, false
	}
	at, ok := ann.nodeAnnTypes[n]
	if !ok {
		return false, false
	}
	switch at {
	case AnnotationTypeOff:
		return false, true
	case AnnotationTypeBlock:
		return true, true
	default:
		return false, false
	}
}

//----------

func (ann *Annotator) isType(e ast.Expr) bool {
	if ann.simpleTestMode {
		return false
	}
	tv, ok := ann.typev(e)
	return ok && tv.IsType()
}

func (ann *Annotator) isBuiltin(e ast.Expr) bool {
	if ann.simpleTestMode {
		return true
	}
	tv, ok := ann.typev(e)
	return ok && tv.IsBuiltin()
}

//// ex: fn() is not addressable (can't do "&fn()", only a:=fn(); &a)
//func (ann *Annotator) isAddressable(e ast.Expr) bool {
//	tv, ok := ann.typev(e)
//	return ok && tv.Addressable()
//}

func (ann *Annotator) isConst(e ast.Expr) bool {
	if ann.simpleTestMode {
		//if id, ok := e.(*ast.Ident); ok {
		//	isVar := strings.HasPrefix(id.Name, ann.debugVarPrefix)
		//	return !isVar
		//}
		return false
	}

	tv, ok := ann.typev(e)
	return ok && tv.Value != nil

	//if !ok || tv.Value == nil {
	//	return false
	//}
	//switch tv.Value.Kind() {
	//case constant.Int, constant.Float, constant.Complex:
	//	return true
	//default:
	//	return false
	//}
}

func (ann *Annotator) isBigConst(e ast.Expr) (string, bool) {
	// handles big constants:
	// _=uint64(1<<64 - 1)
	// _=uint64(math.MaxUint64)
	// the annotator would generate IV(1<<64), which will give a compile error since "1<<64" overflows an int (consts are assigned to int by default)

	tv, ok := ann.typev(e)
	if !ok || tv.Value == nil {
		return "", false
	}
	u := constant.Val(tv.Value)
	switch t := u.(type) {
	case *big.Int, *big.Float, *big.Rat:
		return fmt.Sprintf("%s", t), true
	}
	return "", false
}

func (ann *Annotator) typev(e ast.Expr) (types.TypeAndValue, bool) {
	if ann.typesInfo == nil {
		return types.TypeAndValue{}, false
	}
	tv, ok := ann.typesInfo.Types[e]
	return tv, ok
}

//----------

// Correct debugindexes to have the numbers attributed in order at compile time. Allows assuming an ordered slice (debugindex/textindex) in the editor.
func (ann *Annotator) correctDebugIndexes(n ast.Node) int {

	type lineInfo struct {
		lineCE        *ast.CallExpr
		oldDebugIndex int
		byteIndex     int // fset position offset (for sort)
		seenCount     int
	}
	seenCount := 0
	lines := []*lineInfo{}

	// collect all calls to debug.Line()
	ast.Inspect(n, func(n2 ast.Node) bool {
		// find line callexpr
		ce := (*ast.CallExpr)(nil)
		switch t2 := n2.(type) {
		case *ast.CallExpr:
			ce2 := t2
			if se, ok := ce2.Fun.(*ast.SelectorExpr); ok {
				if id, ok := se.X.(*ast.Ident); ok {
					if id.Name == ann.debugPkgName && se.Sel.Name == "Line" {
						ce = ce2
					}
				}
			}
		}
		if ce == nil {
			return true // continue
		}

		// debugindex
		di, err := strconv.Atoi(ce.Args[1].(*ast.BasicLit).Value)
		if err != nil {
			panic(err)
		}
		// fset offset position (byteindex)
		byteIndex, err := strconv.Atoi(ce.Args[2].(*ast.BasicLit).Value)
		if err != nil {
			panic(err)
		}
		// keep
		li := &lineInfo{
			lineCE:        ce,
			oldDebugIndex: di,
			byteIndex:     byteIndex,
			seenCount:     seenCount,
		}
		seenCount++

		lines = append(lines, li)

		return true
	})

	// sort debug lines by byteindex (fset offset)
	sort.Slice(lines, func(a, b int) bool {
		va, vb := lines[a], lines[b]
		if va.byteIndex == vb.byteIndex {
			// the one seen first while visiting the ast
			return va.seenCount < vb.seenCount
		}
		return va.byteIndex < vb.byteIndex
	})

	// setup new debug indexes
	di := 0
	m := map[int]int{}         // [textIndex]debugIndex
	for _, li := range lines { // visited by byteindex order
		di2 := di
		// check if this debugIndex was already seen
		if di3, ok := m[li.oldDebugIndex]; ok {
			di2 = di3
		} else {
			// assign new debug index
			di2 = di
			m[li.oldDebugIndex] = di
			di++
		}

		// assign final debug index
		li.lineCE.Args[1] = basicLitInt(di2)
	}

	return di
}

//----------

func (ann *Annotator) removeInnerFuncComments(astFile *ast.File) {
	// ensure comments are not in between stmts in the middle of declarations inside functions (solves test100)
	// Other comments stay in place since they might be needed (build comments, "c" package comments, ...)
	u := astFile.Comments[:0]             // use already allocated mem
	for _, cg := range astFile.Comments { // all comments
		keep := true

		// check if inside func decl
		for _, d := range astFile.Decls {
			if _, ok := d.(*ast.FuncDecl); !ok {
				continue
			}
			if d.Pos() > cg.End() { // passed comment
				break
			}
			in := cg.Pos() >= d.Pos() && cg.Pos() < d.End()
			if in {
				keep = false
				break
			}
		}

		if keep {
			u = append(u, cg)
		}
	}
	astFile.Comments = u
}

//----------

func (ann *Annotator) sprintNode(n ast.Node) string {
	return astut.SprintNode(ann.fset, n)
}
func (ann *Annotator) printNode(n ast.Node) {
	fmt.Println(ann.sprintNode(n))
}

//----------

//func (ann *Annotator) wrapInParenIfNotSimple(ctx *Ctx, e ast.Expr) ast.Expr {
//	if !isSimple(e) {
//		return ann.newDebugIP(ctx, e)
//	}
//	return e
//}

//----------
//----------
//----------

//func isSimple(e ast.Expr) bool {
//	return isIdentOrSelectorOfIdents(e)
//}

func isIdentOrSelectorOfIdents(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isIdentOrSelectorOfIdents(t.X)
	}
	// ex: (a+b).Fn()
	return false
}

//----------

var _emptyExpr = &ast.Ident{Name: "*emptyExpr*"}

func emptyExpr() ast.Expr { return _emptyExpr }

//----------

func nilIdent() *ast.Ident {
	return &ast.Ident{Name: "nil"}
}
func isNilIdent(e ast.Expr) bool {
	return isIdentWithName(e, "nil")
}

func anonIdent() *ast.Ident {
	return &ast.Ident{Name: "_"}
}
func isAnonIdent(e ast.Expr) bool {
	return isIdentWithName(e, "_")
}

func afterPanicIdent() *ast.Ident {
	return &ast.Ident{Name: "after panic"}
}
func isAfterPanicIdent(e ast.Expr) bool {
	return isIdentWithName(e, "after panic")
}

func isIdentWithName(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

//----------

func basicLitString(v string) *ast.BasicLit {
	s := strings.ReplaceAll(v, "%", "%%")
	return &ast.BasicLit{Kind: token.STRING, Value: s}
}
func basicLitStringQ(v string) *ast.BasicLit { // quoted
	s := strings.ReplaceAll(v, "%", "%%")
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", s)}
}
func basicLitInt(v int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}
