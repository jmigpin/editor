package godebug

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/goutil"
)

type Annotator struct {
	fset *token.FileSet

	typesInfo    *types.Info
	nodeAnnTypes map[ast.Node]AnnotationType

	fileIndex int

	dopt              *AnnSetDebugOpt
	debugVarNameIndex int
	debugNIndexes     int // n indexes were used

	testModeMainFunc bool
	hasMainFunc      bool

	simplify bool // document simplifications

	pkg *types.Package

	ctxData struct {
		debugIndex int
		visited    map[ast.Stmt]struct{}
	}
}

func NewAnnotator(fset *token.FileSet, ti *types.Info, dopt *AnnSetDebugOpt) *Annotator {
	ann := &Annotator{fset: fset, typesInfo: ti, dopt: dopt}
	ann.simplify = true // always true
	ann.ctxData.visited = map[ast.Stmt]struct{}{}
	ann.nodeAnnTypes = map[ast.Node]AnnotationType{}
	ann.pkg = ann.typesPkg()
	return ann
}

//----------

func (ann *Annotator) AnnotateAstFile(astFile *ast.File) {
	defer func() { // always run, even on error
		ann.debugNIndexes = ann.correctDebugIndexes(astFile)

		// fix issues like "//go:embed" comments staying in place
		//ann.removeInnerFuncComments(astFile) // failing
		astFile.Comments = nil // seems to work!
	}()

	ctx := newCtx(ann)
	if err := ann.visFile(ctx, astFile); err != nil {
		ann.logf("annotate error: %v", err)
	}

	ann.addImports(astFile)
}

//----------

func (ann *Annotator) log(args ...any) {
	//log.Print(args...)
	s := fmt.Sprint(args...)
	s = strings.TrimRight(s, "\n") + "\n"
	fmt.Print(s)
}
func (ann *Annotator) logf(f string, args ...any) {
	ann.log(fmt.Sprintf(f, args...))
}

//----------
//----------

func (ann *Annotator) visFile(ctx *Ctx, file *ast.File) error {
	ctx = ctx.withNoAnnotationsUpdated(file)
	for _, decl := range file.Decls {
		if err := ann.visDecl(ctx, decl); err != nil {
			return err
		}
	}
	return nil
}

func (ann *Annotator) visDecl(ctx *Ctx, decl ast.Decl) error {
	ctx = ctx.withNoAnnotationsUpdated(decl)
	switch t := decl.(type) {
	case *ast.BadDecl:
		return nil
	case *ast.GenDecl:
		return ann.visGenDecl(ctx, t)
	case *ast.FuncDecl:
		return ann.visFuncDecl(ctx, t)
	default:
		return goutil.TodoErrorType(decl)
	}
}

func (ann *Annotator) visStmt(ctx *Ctx, stmt ast.Stmt) error {

	if ctx.stmtVisited(stmt) {
		return nil
	}
	ctx.setStmtVisited(stmt, true)

	ctx = ctx.withFixedDebugIndex(true) // each stmt uses a fixed index
	ctx = ctx.withNoAnnotationsUpdated(stmt)

	//----------

	switch t := stmt.(type) {
	case *ast.AssignStmt:
		return ann.visAssignStmt(ctx, t)
	case *ast.BadStmt:
		return nil
	case *ast.BlockStmt:
		return ann.visStmts(ctx, &t.List)
	case *ast.BranchStmt:
		return ann.visBranchStmt(ctx, t)
	case *ast.CaseClause:
		return ann.visCaseClause(ctx, t)
	case *ast.CommClause:
		return ann.visCommClause(ctx, t)
	case *ast.DeclStmt:
		return ann.visDecl(ctx, t.Decl)
	case *ast.DeferStmt:
		return ann.visAsyncStmt(ctx, &t.Call)
	case *ast.EmptyStmt:
		return nil
	case *ast.ExprStmt:
		return ann.visExprStmt(ctx, t)
	case *ast.ForStmt:
		return ann.visForStmt(ctx, t)
	case *ast.GoStmt:
		return ann.visAsyncStmt(ctx, &t.Call)
	case *ast.IfStmt:
		return ann.visIfStmt(ctx, t)
	case *ast.IncDecStmt:
		return ann.visIncDecStmt(ctx, t)
	case *ast.LabeledStmt:
		return ann.visLabeledStmt(ctx, t)
	case *ast.RangeStmt:
		return ann.visRangeStmt(ctx, t)
	case *ast.ReturnStmt:
		return ann.visReturnStmt(ctx, t)
	case *ast.SelectStmt:
		return ann.visSelectStmt(ctx, t)
	case *ast.SendStmt:
		return ann.visSendStmt(ctx, t)
	case *ast.SwitchStmt:
		return ann.visSwitchStmt(ctx, t)
	case *ast.TypeSwitchStmt:
		return ann.visTypeSwitchStmt(ctx, t)
	default:
		return goutil.TodoErrorType(stmt)
	}
}

func (ann *Annotator) visExpr(ctx *Ctx, expr0 *ast.Expr) (DebugExpr, error) {
	ctx = ctx.withValue(cidnExpr, expr0)
	expr := *expr0
	switch t := expr.(type) {
	case *ast.ArrayType:
		return ann.resultDE(ctx, t)
	case *ast.BadExpr:
		return nil, nil
	case *ast.BasicLit:
		return ann.visBasicLit(ctx, t)
	case *ast.BinaryExpr:
		return ann.visBinaryExpr(ctx, t)
	case *ast.CallExpr:
		return ann.visCallExpr(ctx, t)
	case *ast.ChanType:
		return ann.resultDE(ctx, t)
	case *ast.CompositeLit:
		return ann.visCompositeLit(ctx, t)
	case *ast.Ellipsis:
		return ann.visExpr(ctx, &t.Elt) // TODO: review
	case *ast.FuncLit:
		return ann.visFuncLit(ctx, t)
	case *ast.FuncType:
		return ann.visFuncType(ctx, t)
	case *ast.Ident:
		return ann.visIdent(ctx, t)
	case *ast.IndexExpr:
		return ann.visIndexExpr(ctx, t)
	case *ast.InterfaceType:
		return ann.resultDE(ctx, t)
	case *ast.KeyValueExpr:
		return ann.visKeyValueExpr(ctx, t)
	case *ast.MapType:
		return ann.resultDE(ctx, t)
	case *ast.ParenExpr:
		return ann.visParenExprs(ctx, t)
	case *ast.SelectorExpr:
		return ann.visSelectorExpr(ctx, t)
	case *ast.SliceExpr:
		return ann.visSliceExpr(ctx, t)
	case *ast.StarExpr:
		return ann.visStarExpr(ctx, t)
	case *ast.StructType:
		return ann.resultDE(ctx, t)
	case *ast.TypeAssertExpr:
		return ann.visTypeAssertExpr(ctx, t)
	case *ast.UnaryExpr:
		return ann.visUnaryExpr(ctx, t)
	default:
		return nil, goutil.TodoErrorType(expr)
	}
}

func (ann *Annotator) visSpec(ctx *Ctx, spec ast.Spec) error {
	// NOTE: a spec can have initial values

	switch t := spec.(type) {
	case *ast.ImportSpec:
		return nil
	case *ast.TypeSpec:
		return nil
	case *ast.ValueSpec:
		// inside a func node
		if _, _, ok := ctx.value(cidnFuncNode); ok {
			de, err := ann.visExprs(ctx, &t.Values, t.Pos())
			if err != nil {
				return err
			}
			// TODO: insert after?
			ann.insertDebugLineStmt(ctx, de)
			return nil
		}
		return nil
	default:
		_ = t
		return goutil.TodoErrorType(spec)
	}
}

//----------
//----------

func (ann *Annotator) visGenDecl(ctx *Ctx, gd *ast.GenDecl) error {
	// split into individual decls to be able to insert debug lines
	// TODO: can't split CONST
	canSplit := gd.Tok == token.VAR
	if _, _, ok := ctx.value(cidnFuncNode); ok && canSplit {
		for _, spec := range gd.Specs {
			gd := &ast.GenDecl{Tok: gd.Tok, Specs: []ast.Spec{spec}}
			ds := &ast.DeclStmt{Decl: gd}
			ctx.insertStmt(ds)
			ctx.setStmtVisited(ds, false)
		}
		ctx.replaceStmt(&ast.EmptyStmt{})
		return nil
	}

	for i := range gd.Specs {
		// TODO: can't visit CONST
		canVisit := gd.Tok == token.VAR
		if !canVisit {
			continue
		}
		ctx2 := ctx
		//if gd.Tok == token.CONST {
		//	ctx2 = ctx2.withValue(cidnIsConstSpec, gd.Specs[i])
		//}
		if err := ann.visSpec(ctx2, gd.Specs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (ann *Annotator) visFuncDecl(ctx *Ctx, fd *ast.FuncDecl) error {
	if fd.Body == nil {
		return nil
	}

	ctx = ctx.withValue(cidnFuncNode, fd) // ex: returnstmt needs this

	ctx2 := ctx.withStmts(&fd.Body.List)

	ann.insertDeferRecover(ctx2)
	_ = ann.insertMainClose(ctx2, fd)

	if name, ok := ann.detectJumps(ctx2, fd); ok {
		// insert a not annotated step
		s := fmt.Sprintf("TODO: forward jump label detected: %s", name)
		de := ann.newDebugCE("INAnn", basicLitStringQ(s, fd.Pos()))
		ann.insertDebugLineStmt(ctx2, de)
		return nil
	}

	u := (ast.Expr)(fd.Type)
	de, err := ann.visExpr(ctx2, &u)
	if err != nil {
		return err
	}
	ann.insertDebugLineStmt(ctx2, de)

	return ann.visStmt(ctx, fd.Body)
}

//----------
//----------

func (ann *Annotator) visStmts(ctx *Ctx, stmts *[]ast.Stmt) error {
	ctx = ctx.withNoAnnotationsInstance()
	ctx = ctx.withFixedDebugIndex(false)

	si := newStmtsIter(ctx, stmts)
	ctx = ctx.withValue(cidStmtsIter, si)
	return si.iterate(func(stmt ast.Stmt) error {
		return ann.visStmt(ctx, stmt)
	})
}

//----------

func (ann *Annotator) visAssignStmt(ctx *Ctx, as *ast.AssignStmt) error {
	// ex: a,b=c,d
	// ex:"switch a:=b.(type)"

	rhsOnly := false
	if ctx.valueMatch2(cidnIsTypeSwitchStmtAssign, as) {
		rhsOnly = true
	}
	if !rhsOnly && ann.simplify {
		allowedTok := func() bool {
			return as.Tok == token.DEFINE ||
				as.Tok == token.ASSIGN
		}
		allIds := func() bool {
			for _, e := range as.Lhs {
				if !isIdentsSequence(e) {
					return false
				}
			}
			return true
		}

		rhsOnly = allowedTok() && allIds()
	}

	lhs := (DebugExpr)(nil)
	if !rhsOnly {
		ctx2 := ctx.withValue(cidnResNotReplaceable, &as.Lhs)
		//ctx2 = ctx2.withValue(cidnResAssignDebugToVar, &as.Lhs) // commented: cannot be used, won't work for all cases
		lhs2, err := ann.visExprs(ctx2, &as.Lhs, as.Lhs[0].Pos())
		if err != nil {
			return err
		}
		lhs = lhs2
	}

	ctx3 := ctx
	ctx3 = ctx3.withValue(cidnResAssignDebugToVar, &as.Rhs)
	rhs, err := ann.visExprs(ctx3, &as.Rhs, as.Rhs[0].Pos())
	if err != nil {
		return err
	}
	if rhsOnly {
		ann.insertDebugLineStmt(ctx, rhs)
		return nil
	}

	opbl := basicLitInt(int(as.Tok), as.TokPos)
	e3 := ann.newDebugCE("IA", lhs, opbl, rhs)
	ctx4 := ctx.withValue(cidbInsertStmtAfter, true)
	ann.insertDebugLineStmt(ctx4, e3)
	return nil
}

func (ann *Annotator) visBranchStmt(ctx *Ctx, bs *ast.BranchStmt) error {
	// show step in
	de := ann.newDebugISt(bs.Pos())
	ann.insertDebugLineStmt(ctx, de)
	return nil
}

func (ann *Annotator) visCaseClause(ctx *Ctx, cc *ast.CaseClause) error {
	for i := range cc.List {
		expr := &cc.List[i]

		//// decide what to wrap
		//expr2 := bypassParenExpr(*expr)
		//wrap := false
		//switch expr2.(type) {
		//// need to run for each case before actually matching
		//case *ast.CallExpr, *ast.BinaryExpr:
		//	wrap = true
		//}
		//if !wrap {
		//	continue
		//}

		// wrap in funclit
		tt, ok := ann.newTType2(*expr)
		if ok &&
			//!func() bool { _, ok := tt.constValue(); return ok }() &&
			!tt.isType() &&
			!tt.isBasicInfo(types.IsUntyped) &&
			tt.isBasicInfo(types.IsOrdered|
				types.IsBoolean|
				types.IsInteger|
				types.IsUnsigned|
				types.IsFloat|
				types.IsString|
				types.IsComplex) {
			fl := newFuncLitRetType(ann.typeString(tt.Type))
			rs := &ast.ReturnStmt{Results: []ast.Expr{*expr}}
			fl.Body.List = append(fl.Body.List, rs)
			if err := ann.visStmt(ctx, fl.Body); err != nil {
				return err
			}
			*expr = &ast.CallExpr{Fun: fl}
		}
	}

	// show debug step entering the clause
	ctx2 := ctx.withStmts(&cc.Body)
	ann.insertStepInStmt(ctx2, cc.Colon)

	return ann.visStmts(ctx, &cc.Body)
}

func (ann *Annotator) visCommClause(ctx *Ctx, cc *ast.CommClause) error {
	// TODO
	//cc.Comm

	// show debug step entering the clause
	ctx2 := ctx.withStmts(&cc.Body)
	ann.insertStepInStmt(ctx2, cc.Pos())

	return ann.visStmts(ctx, &cc.Body)
}

func (ann *Annotator) visAsyncStmt(ctx *Ctx, ce0 **ast.CallExpr) error {
	// used by: ast.DeferStmt and ast.GoStmt

	ce := *ce0

	// funclit that will run now
	runFl := newFuncLit()
	retType := &ast.FuncType{Params: &ast.FieldList{}}
	runFl.Type.Results.List = []*ast.Field{
		{Type: retType},
	}
	runFlCtx := ctx.withStmts(&runFl.Body.List)

	// visit fun
	if !isIdentsSequence(ce.Fun) {
		de, err := ann.visExpr(runFlCtx, &ce.Fun)
		if err != nil {
			return err
		}
		ann.insertDebugLineStmt(runFlCtx, de)
	}

	// visit args
	de, err := ann.visExprs(runFlCtx, &ce.Args, ce.Pos())
	if err != nil {
		return err
	}
	if !isNilIdent(de) {
		ann.insertDebugLineStmt(runFlCtx, de)
	}

	// funclit that will run later
	asyncFl := newFuncLit()
	asyncFlCtx := ctx.withStmts(&asyncFl.Body.List)
	rs := &ast.ReturnStmt{Results: []ast.Expr{asyncFl}}
	runFlCtx.insertStmt(rs)

	es := &ast.ExprStmt{X: ce}
	asyncFlCtx.insertStmt(es)
	asyncFlCtx.setStmtVisited(es, false)
	if err := ann.visStmt(ctx, asyncFl.Body); err != nil {
		return err
	}

	// replace
	ce2 := &ast.CallExpr{Fun: runFl}
	*ce0 = &ast.CallExpr{Fun: ce2}
	return nil
}

func (ann *Annotator) visExprStmt(ctx *Ctx, es *ast.ExprStmt) error {
	ctx = ctx.withValue(cidnIsExprStmtExpr, es.X)
	de, err := ann.visExpr(ctx, &es.X)
	if err != nil {
		return err
	}

	// special case: don't insert after if panic (won't compile)
	if ce, ok := bypassParenExpr(es.X).(*ast.CallExpr); ok {
		tt, ok := ann.newTType2(ce.Fun)
		if ok && tt.isBuiltinWithName("panic") {
			as := newAssignToAnons(de)
			ctx.insertStmt(as)
			return nil
		}
	}

	ctx2 := ctx
	if ctx2.valueMatch2(cidnIsTypeSwitchStmtAssign, es) {
		// ex: "switch a.(int)"
	} else {
		ctx = ctx.withValue(cidbInsertStmtAfter, true)
	}
	ann.insertDebugLineStmt(ctx, de)
	return nil
}

func (ann *Annotator) visForStmt(ctx *Ctx, fs *ast.ForStmt) error {
	// wrap in blockstmt to have vars valid only inside the stmt
	canInit := !ctx.valueMatch2(cidnIsLabeledStmtStmt, fs)
	if canInit && fs.Init != nil {
		stmt, init := fs, &fs.Init

		bs := &ast.BlockStmt{}
		bs.List = append(bs.List, *init, stmt)
		ctx2 := ctx.withStmts(&bs.List)
		if err := ann.visStmt(ctx2, *init); err != nil {
			return err
		}
		*init = nil
		ctx.replaceStmt(bs)
	}

	// wrap in funclit that returns bool
	if fs.Cond != nil {
		fl, err := ann.newFuncLitRetType(fs.Cond)
		if err != nil {
			return err
		}
		rs := &ast.ReturnStmt{Results: []ast.Expr{fs.Cond}}
		fl.Body.List = append(fl.Body.List, rs)
		if err := ann.visStmt(ctx, fl.Body); err != nil {
			return err
		}
		fs.Cond = &ast.CallExpr{Fun: fl}
	}

	// wrap in funclit, there are no new variables
	if fs.Post != nil {
		fl := newFuncLit()
		fl.Body.List = append(fl.Body.List, fs.Post)
		if err := ann.visStmt(ctx, fl.Body); err != nil {
			return err
		}
		fs.Post = &ast.ExprStmt{X: &ast.CallExpr{Fun: fl}}
	}

	return ann.visStmt(ctx, fs.Body)
}

func (ann *Annotator) visIfStmt(ctx *Ctx, is *ast.IfStmt) error {
	// wrap in blockstmt to have vars that belong only to the forstmt
	if is.Init != nil {
		bs := &ast.BlockStmt{}
		bs.List = append(bs.List, is.Init, is)
		ctx2 := ctx.withStmts(&bs.List)
		if err := ann.visStmt(ctx2, is.Init); err != nil {
			return err
		}
		is.Init = nil
		ctx.replaceStmt(bs) // replaces is with bs in stmts
	}

	// wrap in funclit with bool return value
	if is.Cond != nil {
		fl, err := ann.newFuncLitRetType(is.Cond)
		if err != nil {
			return err
		}
		rs := &ast.ReturnStmt{Results: []ast.Expr{is.Cond}}
		fl.Body.List = append(fl.Body.List, rs)
		if err := ann.visStmt(ctx, fl.Body); err != nil {
			return err
		}
		is.Cond = &ast.CallExpr{Fun: fl}
	}

	if err := ann.visStmt(ctx, is.Body); err != nil {
		return err
	}

	if is.Else != nil {
		ctx2 := ctx.withStmt(&is.Else)
		return ann.visStmt(ctx2, is.Else)
	}
	return nil
}

func (ann *Annotator) visIncDecStmt(ctx *Ctx, rs *ast.IncDecStmt) error {
	// ex: a++
	// ex: *f()++

	ctx2 := ctx.withValue(cidnResNotReplaceable, rs.X)

	// commented: not showing the value before
	//de, err := ann.visExpr(ctx2, &rs.X)
	//if err != nil {
	//	return err
	//}
	//ann.insertDebugLineStmt(ctx2, de)

	result, err := ann.resultDE(ctx2, rs.X)
	if err != nil {
		return err
	}

	ctx3 := ctx2.withValue(cidbInsertStmtAfter, true) // TODO: review
	ann.insertDebugLineStmt(ctx3, result)
	return nil
}

func (ann *Annotator) visLabeledStmt(ctx *Ctx, ls *ast.LabeledStmt) error {
	ctx = ctx.withValue(cidnIsLabeledStmtStmt, ls.Stmt)

	// these can't detach the label or the program could be altered
	// ex: setting labelstmt.stmt as empty will fail to compile continue/break labels ("invalid break label X")
	//*ast.ForStmt, 		// visStmtWithInit, !stepin
	//*ast.RangeStmt,     	// no init, !stepin
	//*ast.SwitchStmt,     	// visStmtWithInit, stepin
	//*ast.TypeSwitchStmt, 	// visStmtWithInit, stepin by assignstmt
	//*ast.SelectStmt:     	// no init, stepin

	switch ls.Stmt.(type) {
	case *ast.ForStmt,
		*ast.RangeStmt,
		*ast.SwitchStmt,
		*ast.TypeSwitchStmt,
		*ast.SelectStmt:
		return ann.visStmt(ctx, ls.Stmt)
	default:
		// detach stmt
		ctx2 := ctx.withValue(cidbInsertStmtAfter, true)
		ctx2.insertStmt(ls.Stmt)
		ctx2.setStmtVisited(ls.Stmt, false)
		ls.Stmt = &ast.EmptyStmt{}
		return nil
	}
}

func (ann *Annotator) visRangeStmt(ctx *Ctx, rs *ast.RangeStmt) error {
	canInit := !ctx.valueMatch2(cidnIsLabeledStmtStmt, rs)
	if canInit {
		// range expr
		de, err := ann.visExpr(ctx, &rs.X)
		if err != nil {
			return err
		}
		ann.insertDebugLineStmt(ctx, de)

		// TODO: check if type is countable (slice,int,...)
		//// lenght of x
		//id := &ast.Ident{Name: "len", NamePos: rs.X.Pos()}
		//ce2 := &ast.CallExpr{Fun: id, Args: []ast.Expr{rs.X}}
		//de2 := ann.newDebugCE("IVr", ce2)
		//ann.insertDebugLineStmt(ctx, de2)
	} else {
		ann.insertStepInStmt(ctx, rs.Range)
	}

	// key and value inside the range body
	rsBodyCtx := ctx.withStmts(&rs.Body.List)
	kv := []DebugExpr{}
	if rs.Key != nil {
		ctx2 := rsBodyCtx
		ctx2 = ctx2.withValue(cidnResNotReplaceable, rs.Key)
		de, err := ann.visExpr(ctx2, &rs.Key)
		if err != nil {
			return err
		}
		kv = append(kv, de)
	}
	if rs.Value != nil {
		ctx2 := rsBodyCtx
		ctx2 = ctx2.withValue(cidnResNotReplaceable, rs.Value)
		de, err := ann.visExpr(ctx2, &rs.Value)
		if err != nil {
			return err
		}
		kv = append(kv, de)
	}
	if len(kv) > 0 {
		de := ann.newDebugIL(kv...)
		ann.insertDebugLineStmt(rsBodyCtx, de)
	}

	return ann.visStmt(ctx, rs.Body)
}

func (ann *Annotator) visReturnStmt(ctx *Ctx, rs *ast.ReturnStmt) error {
	fn, ft, _ := ctx.funcNode()

	tt, err := ann.newTType(fn)
	if err != nil {
		return err
	}

	if len(rs.Results) == 0 {
		// just show debug step
		if tt.nResults2(true) == 0 {
			ann.insertStepInStmt(ctx, rs.Pos())
			return nil
		}

		// naked return, nresults>=1
		if err := ann.nameMissingFieldListNames(ft.Results); err != nil {
			return err
		}
		rs.Results = ann.fieldListNames(ft.Results)
	} else {
		//// TODO:***
		//// fix "return nil" has no type
		//tes := ann.fieldListTypeExprs(ft.Results)
		//if err := ann.setNilsTypes(rs.Results, tes); err != nil {
		//	return err
		//}
	}

	de, err := ann.visExprs(ctx, &rs.Results, rs.Pos())
	if err != nil {
		return err
	}
	ann.insertDebugLineStmt(ctx, de)
	return nil
}

func (ann *Annotator) visSelectStmt(ctx *Ctx, ss *ast.SelectStmt) error {
	ann.insertStepInStmt(ctx, ss.Pos())
	return ann.visStmt(ctx, ss.Body)
}

func (ann *Annotator) visSendStmt(ctx *Ctx, ss *ast.SendStmt) error {
	ch, err := ann.visExpr(ctx, &ss.Chan)
	if err != nil {
		return err
	}

	val, err := ann.visExpr(ctx, &ss.Value)
	if err != nil {
		return err
	}

	de := ann.newDebugCE("IS", ch, val)
	ann.insertDebugLineStmt(ctx, de)
	return nil
}

func (ann *Annotator) visSwitchStmt(ctx *Ctx, ss *ast.SwitchStmt) error {
	// show debug step entering the switch
	if ss.Init == nil && ss.Tag == nil {
		ann.insertStepInStmt(ctx, ss.Pos())
	}

	ctx3 := ctx // used in ss.tag

	// wrap in blockstmt to have vars valid only inside the stmt
	canInit := !ctx.valueMatch2(cidnIsLabeledStmtStmt, ss)
	if canInit && ss.Init != nil {
		stmt, init := ss, &ss.Init

		bs := &ast.BlockStmt{}
		bs.List = append(bs.List, *init, stmt)
		ctx2 := ctx.withStmts(&bs.List)
		if err := ann.visStmt(ctx2, *init); err != nil {
			return err
		}
		*init = nil
		ctx.replaceStmt(bs)

		ctx3 = ctx2
		ctx3.stmtsIter().index++
	}

	if canInit && ss.Tag != nil {
		// TODO: only possible with funclit+returntypes?
		de, err := ann.visExpr(ctx3, &ss.Tag)
		if err != nil {
			return err
		}
		//ctx2 := ctx.withValue(cidbInsertStmtAfter, true)
		ann.insertDebugLineStmt(ctx3, de)
	}

	return ann.visStmt(ctx, ss.Body)
}

func (ann *Annotator) visTypeSwitchStmt(ctx *Ctx, tss *ast.TypeSwitchStmt) error {
	// wrap in blockstmt to have vars valid only inside the stmt
	canInit := !ctx.valueMatch2(cidnIsLabeledStmtStmt, tss)
	if canInit && tss.Init != nil {
		stmt, init := tss, &tss.Init

		bs := &ast.BlockStmt{}
		bs.List = append(bs.List, *init, stmt)
		ctx2 := ctx.withStmts(&bs.List)
		if err := ann.visStmt(ctx2, *init); err != nil {
			return err
		}
		*init = nil
		ctx.replaceStmt(bs)
	}

	ctx2 := ctx.withValue(cidnIsTypeSwitchStmtAssign, tss.Assign)
	ctx2 = ctx2.withValue(cidnResNotReplaceable, tss.Assign)
	if err := ann.visStmt(ctx2, tss.Assign); err != nil {
		return err
	}

	return ann.visStmt(ctx, tss.Body)
}

//----------
//----------

func (ann *Annotator) visExprs(ctx *Ctx, exprs0 *[]ast.Expr, noExprsPos token.Pos) (DebugExpr, error) {
	ctx = ctx.withValue(cidnExprs, exprs0)
	exprs := *exprs0

	// performance: avoid lengthy run times
	limit := 30
	if l, ok := ctx.integer(cidiSliceExprsLimit); ok {
		limit = l
	}

	// inherit ctxs
	cids := []ctxId{
		cidnResNotReplaceable,
		cidnResAssignDebugToVar,
	}
	cids2 := ctx.valueMatch3(cids, exprs0)

	w := []DebugExpr{}
	for i := range exprs {
		ctx2 := ctx.withValue2(cids2, exprs[i]) // inherit ctxs
		de, err := ann.visExpr(ctx2, &exprs[i])
		if err != nil {
			return nil, err
		}
		w = append(w, de)

		// simplification/performance/usability: annotated executable becomes unusable if there is no limit. Ex: files with big number of constants will be really slow.
		if limit > 0 && i+1 >= limit {
			rest := len(exprs) - (i + 1)
			if rest > 0 {
				s := fmt.Sprintf("(+%v elems)", rest)
				de2 := ann.newDebugIVs(s, exprs[0].Pos())
				w = append(w, de2)
				break
			}
		}
	}

	return ann.newDebugILOrNilIdent(noExprsPos, w...), nil
}

//----------

func (ann *Annotator) visBasicLit(ctx *Ctx, bl *ast.BasicLit) (DebugExpr, error) {
	return ann.resultDE(ctx, bl)
	//return ann.newDebugCE("IVi", t), nil
}
func (ann *Annotator) visBinaryExpr(ctx *Ctx, be *ast.BinaryExpr) (DebugExpr, error) {
	switch be.Op {
	case token.LAND, token.LOR:
		return ann.visBinaryExprAndOr(ctx, be)
	default:
		return ann.visBinaryExpr2(ctx, be)
	}
}
func (ann *Annotator) visBinaryExpr2(ctx *Ctx, be *ast.BinaryExpr) (DebugExpr, error) {

	// NOTE: the need for cidnAssignDebugToVar for x and y is because in the case of the assign stmt, a var in x/y might be changed in the lhs, and getting the current value in the inserted line after the assignment will be wrong

	ctx2 := ctx.withValue(cidnResAssignDebugToVar, be.X)
	x, err := ann.visExpr(ctx2, &be.X)
	if err != nil {
		return nil, err
	}

	ctx3 := ctx.withValue(cidnResAssignDebugToVar, be.Y)
	y, err := ann.visExpr(ctx3, &be.Y)
	if err != nil {
		return nil, err
	}

	//// TODO: cast to the correct type?
	// in some cases a cast is needed
	// ex: b |= 1<<a, b is byte, 1<<a will be int if assigned to a new var

	ctx4 := ctx
	ctx4 = ctx4.withValue(cidnResReplaceWithVar, be) // needs casting in some cases
	//ctx4 = ctx4.withValue(cidnResNotReplaceable, be) // need to be careful about being able to then not debug the value directly if changed in an assign lhs
	//ctx4 = ctx4.withValue(cidnResAssignDebugToVar, be) // must keep result in case of assign lhs
	result, err := ann.resultDE(ctx4, be)
	if err != nil {
		return nil, err
	}

	opbl := basicLitInt(int(be.Op), be.Pos())
	de := ann.newDebugCE("IB", x, opbl, y, result)
	return de, nil
}
func (ann *Annotator) visBinaryExprAndOr(ctx *Ctx, be *ast.BinaryExpr) (DebugExpr, error) {
	// ex: f1() || f2() // f2 should not be called if f1 is true
	// ex: f1() && f2() // f2 should not be called if f1 is false

	x, err := ann.visExpr(ctx, &be.X)
	if err != nil {
		return nil, err
	}

	// init value var with be.X
	value, err := ann.insertAssignToIdent(ctx, be.X)
	if err != nil {
		return nil, err
	}
	ctx.replaceExprs(value)

	// init y result var (in case be.Y doesn't run)
	y, err := ann.insertAssignToIdent(ctx, ann.newDebugIVs("?", be.Y.Pos()))
	if err != nil {
		return nil, err
	}

	// build ifstmt to test x result to decide whether to run y
	ifs := &ast.IfStmt{If: be.Pos(), Body: &ast.BlockStmt{}}
	ifs.Cond = be.X
	if be.Op == token.LOR {
		// negate
		ifs.Cond = &ast.UnaryExpr{Op: token.NOT, X: ifs.Cond}
	}
	ctx.insertStmt(ifs)

	// inside ifstmt: walk be.Y inside ifstmt
	ifsBodyCtx := ctx.withStmts(&ifs.Body.List)
	y2, err := ann.visExpr(ifsBodyCtx, &be.Y)
	if err != nil {
		return nil, err
	}
	// inside ifstmt: assign debug result to y
	as2 := newAssignStmtA11(y, y2)
	ifsBodyCtx.insertStmt(as2)
	// inside ifstmt: assign be.Y to result var
	as3 := newAssignStmtA11(value, be.Y)
	ifsBodyCtx.insertStmt(as3)

	result := ann.newDebugIVi(value)

	opbl := basicLitInt(int(be.Op), be.Pos())
	de := ann.newDebugCE("IB", x, opbl, y, result)
	return de, nil
}

//----------

func (ann *Annotator) visCallExpr(ctx *Ctx, ce *ast.CallExpr) (DebugExpr, error) {

	ctx2 := ctx.withValue(cidnNameInsteadOfValue, ce.Fun)
	ctx2 = ctx2.withValue(cidnIsCallExprFun, ce.Fun)
	fun, err := ann.visExpr(ctx2, &ce.Fun)
	if err != nil {
		return nil, err
	}
	args, err := ann.visExprs(ctx, &ce.Args, ce.Pos())
	if err != nil {
		return nil, err
	}

	// don't show stepin/result if type casting or calling a builtin
	stepIn := true
	if ann.simplify {
		//u := bypassParenExpr(ce.Fun) // TODO needed?
		u := ce.Fun // detects type casts
		if tt, ok := ann.newTType2(u); ok {
			switch {
			case tt.isType():
				stepIn = false
				if tt.isBasicInfo(types.IsNumeric) {
					// show result
				} else {
					ctx = ctx.withValue(cidnResNil, ce)
				}
			case tt.isBuiltinWithName("len"):
				stepIn = false
			case tt.isBuiltinWithName("panic"):
				// show step in
			case tt.isBuiltin():
				stepIn = false
				ctx = ctx.withValue(cidnResNil, ce)
			}
		}
	}

	// show stepping in (insert before func call)
	e4 := ann.newDebugCE("ICe", fun, args)
	if stepIn {
		u, err := ann.insertAssignToIdent(ctx, e4) // avoid double call
		if err != nil {
			return nil, err
		}
		ann.insertDebugLineStmt(ctx, u)
		e4 = u
	}

	// update possible os.exit calls before replacing with var, but after the debug stmts being done
	if err, ok := ann.updateOsExitCalls(ctx, ce); ok && err != nil {
		return nil, err
	}

	ctx4 := ctx.withValue(cidnResReplaceWithVar, ce) // avoid double call
	result, err := ann.resultDE(ctx4, ce)
	if err != nil {
		return nil, err
	}

	de := ann.newDebugCE("IC", e4, result)
	return de, nil
}

func (ann *Annotator) visCompositeLit(ctx *Ctx, cl *ast.CompositeLit) (DebugExpr, error) {
	ctx2 := ctx.withValue(cidiSliceExprsLimit, 10)
	de, err := ann.visExprs(ctx2, &cl.Elts, cl.Pos())
	if err != nil {
		return nil, err
	}
	return ann.newDebugCE2("ILit", cl.Pos(), de), nil
}

func (ann *Annotator) visFuncLit(ctx *Ctx, fl *ast.FuncLit) (DebugExpr, error) {

	ctx2 := ctx.withResetForFuncLit()
	ctx2 = ctx2.withValue(cidnFuncNode, fl) // ex: returnstmt

	ctx3 := ctx2.withStmts(&fl.Body.List)

	ann.insertDeferRecover(ctx3)

	// visit type inside the body
	u := (ast.Expr)(fl.Type)
	de, err := ann.visExpr(ctx3, &u)
	if err != nil {
		return nil, err
	}
	ann.insertDebugLineStmt(ctx3, de)

	// visit body
	if err := ann.visStmt(ctx2, fl.Body); err != nil {
		return nil, err
	}

	ctx4 := ctx.withValue(cidnResReplaceWithVar, fl) // avoid double call
	return ann.resultDE(ctx4, fl)
}

func (ann *Annotator) visFuncType(ctx *Ctx, ft *ast.FuncType) (DebugExpr, error) {
	// example src code
	// ex: _=func(){}
	// ex: _=a.(func())

	// ex: _=(func(int))(nil)
	if ctx.valueMatch2(cidnIsCallExprFun, ft) {
		return ann.resultDE(ctx, ft)
	}

	w := []DebugExpr{}

	de1, ok1, err := ann.visFieldList(ctx, ft.TypeParams)
	if err != nil {
		return nil, err
	}
	if ok1 {
		w = append(w, de1)
	}

	de2, ok2, err := ann.visFieldList(ctx, ft.Params)
	if err != nil {
		return nil, err
	}
	if ok2 {
		w = append(w, de2)
	}

	switch len(w) {
	case 0:
		// show step in
		return ann.newDebugISt(ft.Pos()), nil
	default:
		return ann.newDebugIL(w...), nil
	}
}

func (ann *Annotator) visIdent(ctx *Ctx, id *ast.Ident) (DebugExpr, error) {
	if isAnonIdent(id) {
		return ann.newDebugCE2("IAn", id.Pos()), nil
	}
	if ctx.valueMatch2(cidnNameInsteadOfValue, id) {
		return ann.newDebugIVs(id.Name, id.Pos()), nil
	}
	return ann.resultDE(ctx, id)
}

func (ann *Annotator) visIndexExpr(ctx *Ctx, ie *ast.IndexExpr) (DebugExpr, error) {
	ctx2 := ctx
	ctx2 = ctx2.withValue(cidnResNil, ie.X) // TODO: review
	ctx2 = ctx2.withValue(cidnNameInsteadOfValue, ie.X)
	x, err := ann.visExpr(ctx2, &ie.X)
	if err != nil {
		return nil, err
	}

	ctx3 := ctx.withValue(cidnResReplaceWithVar, ie.Index)
	ix, err := ann.visExpr(ctx3, &ie.Index)
	if err != nil {
		return nil, err
	}

	ctx4 := ctx.withValue(cidnResReplaceWithVar, ie)
	result, err := ann.resultDE(ctx4, ie)
	if err != nil {
		return nil, err
	}
	return ann.newDebugCE("II", x, ix, result), nil
}

func (ann *Annotator) visKeyValueExpr(ctx *Ctx, kve *ast.KeyValueExpr) (DebugExpr, error) {

	// TODO: compositelit: allow replacing the key
	//allow := false
	// *ast.CompositeLit

	ctx2 := ctx
	//ctx2 = ctx2.withValue(cidnNameInsteadOfValue, kve.Key)
	ctx2 = ctx2.withValue(cidnResNotReplaceable, kve.Key)
	ctx2 = ctx2.withValue(cidnResNil, kve.Key)
	k, err := ann.visExpr(ctx2, &kve.Key)
	if err != nil {
		return nil, err
	}

	v, err := ann.visExpr(ctx, &kve.Value)
	if err != nil {
		return nil, err
	}

	de := ann.newDebugCE("IKV", k, v)
	return de, nil
}

func (ann *Annotator) visParenExprs(ctx *Ctx, pe *ast.ParenExpr) (DebugExpr, error) {
	// inherit ctxs
	cids := []ctxId{
		cidnResNotReplaceable,
		cidnResIsForAddress,
		cidnResAssignDebugToVar,
		cidnIsCallExprFun,
		cidnIsExprStmtExpr,
	}
	ctx = ctx.withValueMatch(cids, pe, pe.X)

	x, err := ann.visExpr(ctx, &pe.X)
	if err != nil {
		return nil, err
	}
	return ann.newDebugCE("IP", x), nil
}

func (ann *Annotator) visSelectorExpr(ctx *Ctx, se *ast.SelectorExpr) (DebugExpr, error) {
	doX := true
	if isIdentsSequence(se.X) {
		doX = false
	}

	doSel := doX // false ex: "a.b.c" => "5"
	//doSel := true // true ex: "a.b.c" => "5=(_.c)"
	doResult := true
	tt, ok := ann.newTType2(se)
	if ok {
		if _, ok2 := tt.Type.(*types.Signature); ok2 {
			doResult = false
			doSel = true
		}
	}

	//----------

	x := DebugExpr(nilIdent(se.X.Pos()))
	if doX {
		ctx2 := ctx
		ids := []ctxId{
			cidnResIsForAddress,
		}
		ctx2 = ctx2.withValueMatch(ids, se, se.X)

		x2, err := ann.visExpr(ctx2, &se.X)
		if err != nil {
			return nil, err
		}
		x = x2
	}

	sel := DebugExpr(nilIdent(se.Sel.Pos()))
	if doSel {
		ctx2 := ctx.withValue(cidnNameInsteadOfValue, se.Sel)
		u := (ast.Expr)(se.Sel)
		sel2, err := ann.visExpr(ctx2, &u)
		if err != nil {
			return nil, err
		}
		sel = sel2
	}

	result := DebugExpr(nilIdent(se.Pos()))
	if doResult {
		result2, err := ann.resultDE(ctx, se)
		if err != nil {
			return nil, err
		}
		result = result2
	}

	if !doX && !doSel && doResult {
		return result, nil
	}
	de := ann.newDebugCE("ISel", x, sel, result)
	return de, nil
}

func (ann *Annotator) visSliceExpr(ctx *Ctx, se *ast.SliceExpr) (DebugExpr, error) {
	ctx2 := ctx
	ctx2 = ctx2.withValue(cidnResNil, se.X) // TODO: review
	ctx2 = ctx2.withValue(cidnNameInsteadOfValue, se.X)
	x, err := ann.visExpr(ctx2, &se.X)
	if err != nil {
		return nil, err
	}

	w := []*ast.Expr{&se.Low, &se.High, &se.Max}
	ix := make([]DebugExpr, len(w))
	for i := range w {
		if *w[i] == nil {
			ix[i] = nilIdent(se.Lbrack)
			continue
		}

		ctx3 := ctx.withValue(cidnResReplaceWithVar, w[i])
		de, err := ann.visExpr(ctx3, w[i])
		if err != nil {
			return nil, err
		}
		ix[i] = de
	}

	result, err := ann.resultDE(ctx, se)
	if err != nil {
		return nil, err
	}

	slice3Bl := basicLitString(strconv.FormatBool(se.Slice3), se.Lbrack)
	de := ann.newDebugCE("II2", x, ix[0], ix[1], ix[2], slice3Bl, result)
	return de, nil
}

func (ann *Annotator) visStarExpr(ctx *Ctx, se *ast.StarExpr) (DebugExpr, error) {
	x, err := ann.visExpr(ctx, &se.X)
	if err != nil {
		return nil, err
	}

	if ann.simplify {
		// simplify "_=(*A)(nil)": "((*A=*A))(nil)" -> "(*A)(nil)"
		if tt, ok := ann.newTType2(se.X); ok {
			if tt.isType() {
				ctx = ctx.withValue(cidnResNil, se)
			}
		}
	}

	result, err := ann.resultDE(ctx, se)
	if err != nil {
		return nil, err
	}

	opbl := basicLitInt(int(token.MUL), se.Star)
	e2 := ann.newDebugCE("IUe", opbl, x)
	de := ann.newDebugCE("IU", e2, result)
	return de, nil
}

func (ann *Annotator) visTypeAssertExpr(ctx *Ctx, tae *ast.TypeAssertExpr) (DebugExpr, error) {
	// ex: a.b.(*C).d
	// ex: "a,ok:=b.(int)"
	// ex: "switch t:=b.(type)"

	isSwitch := tae.Type == nil // type switch, as opposed to type assert

	doTyp := true
	doResult := !isSwitch
	if ann.simplify {
		tt, ok := ann.newTType2(tae)
		if ok && tt.nResults() <= 1 { // ex: _=a+b.(int)+c
			doTyp = false
		}
	}

	de, err := ann.visExpr(ctx, &tae.X)
	if err != nil {
		return nil, err
	}

	typ := DebugExpr(nilIdent(tae.Pos()))
	if doTyp {
		typ = ann.newDebugCE("IVt", tae.X)
	}

	result := DebugExpr(nilIdent(tae.Pos()))
	if doResult {
		ctx2 := ctx.withValue(cidnResReplaceWithVar, tae)
		result2, err := ann.resultDE(ctx2, tae)
		if err != nil {
			return nil, err
		}
		result = result2
	}

	isSwitch2 := basicLitString(strconv.FormatBool(isSwitch), tae.Pos())
	return ann.newDebugCE("ITA", de, typ, result, isSwitch2), nil
}

func (ann *Annotator) visUnaryExpr(ctx *Ctx, ue *ast.UnaryExpr) (DebugExpr, error) {
	opbl := basicLitInt(int(ue.Op), ue.Pos())

	ctx2 := ctx
	//ctx2 = ctx2.withValue(cidnAssignDebugToVar, ue.X)
	if hasAddressOp(ue) { // ex: _=&a; _=&A{1,2}
		//ctx2 = ctx2.withValue(cidnResNotReplaceable, ue.X)
		ctx2 = ctx2.withValue(cidnResIsForAddress, ue.X)
	}
	x, err := ann.visExpr(ctx2, &ue.X)
	if err != nil {
		return nil, err
	}

	stepIn := false
	if hasChanRecvOp(ue) {
		stepIn = true
	}

	e2 := ann.newDebugCE("IUe", opbl, x)
	if stepIn {
		u, err := ann.insertAssignToIdent(ctx, e2) // avoid double call
		if err != nil {
			return nil, err
		}
		ann.insertDebugLineStmt(ctx, u)
		e2 = u
	}

	ctx3 := ctx2
	if hasChanRecvOp(ue) || hasAddressOp(ue) {
		ctx3 = ctx3.withValue(cidnResReplaceWithVar, ue) // avoid double call
	}
	result, err := ann.resultDE(ctx3, ue)
	if err != nil {
		return nil, err
	}

	de := ann.newDebugCE("IU", e2, result)
	return de, nil
}

//----------
//----------

func (ann *Annotator) visFieldList(ctx *Ctx, fl *ast.FieldList) (DebugExpr, bool, error) {
	if fl == nil { // ex: functype.typeparams
		return nil, false, nil
	}
	if err := ann.nameMissingFieldListNames(fl); err != nil {
		return nil, false, err
	}
	w := []DebugExpr{}
	for _, f := range fl.List {
		for i := range f.Names {
			ctx2 := ctx
			ctx2 = ctx2.withValue(cidnResNotReplaceable, f.Names[i])
			e := (ast.Expr)(f.Names[i]) // local var
			de, err := ann.visExpr(ctx2, &e)
			if err != nil {
				return nil, false, err
			}
			w = append(w, de)
		}
	}
	if len(w) == 0 {
		return nil, false, nil
	}
	de := ann.newDebugIL(w...)
	return de, true, nil
}
