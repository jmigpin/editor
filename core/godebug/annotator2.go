package godebug

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/astut"
	"github.com/jmigpin/editor/util/goutil"
	"golang.org/x/tools/go/ast/astutil"
)

// continuation of the "annotator.go" file to separate some util funcs

//----------

func (ann *Annotator) newDebugCE(fname string, des ...DebugExpr) DebugExpr {
	// ex: fname="IAn" has no args, need to use newDebugCE2 directly
	if len(des) == 0 {
		panic("len(des)=0, use newDebugCE2")
	}

	return ann.newDebugCE2(fname, des[0].Pos(), des...)
}
func (ann *Annotator) newDebugCE2(fname string, pos token.Pos, des ...DebugExpr) DebugExpr {
	// transform to ast.Expr
	w := make([]ast.Expr, 0, len(des))
	for _, de := range des {
		w = append(w, de)
	}

	se := &ast.SelectorExpr{
		X:   &ast.Ident{Name: ann.dopt.PkgName, NamePos: pos},
		Sel: ast.NewIdent(fname),
	}
	return &ast.CallExpr{Fun: se, Args: w}
}

//----------

func (ann *Annotator) newDebugIVi(e ast.Expr) DebugExpr {
	return ann.newDebugCE("IVi", DebugExpr(e))
}
func (ann *Annotator) newDebugIVs(s string, pos token.Pos) DebugExpr {
	return ann.newDebugCE("IVs", basicLitStringQ(s, pos))
}
func (ann *Annotator) newDebugISt(pos token.Pos) DebugExpr {
	return ann.newDebugCE2("ISt", pos)
}
func (ann *Annotator) newDebugIL(exprs ...DebugExpr) ast.Expr {
	return ann.newDebugCE("IL", exprs...)
}
func (ann *Annotator) newDebugILOrNilIdent(pos token.Pos, exprs ...DebugExpr) DebugExpr {
	switch len(exprs) {
	case 0:
		return nilIdent(pos)
	default:
		return ann.newDebugIL(exprs...)
	}
}

//----------

func (ann *Annotator) insertDebugLineStmt(ctx *Ctx, de DebugExpr) {
	stmt := (ast.Stmt)(nil)
	if ctx.boolean(cidbNoBlockAnnotations) {
		stmt = newAssignToAnons(de)
	} else {
		stmt = ann.newDebugLineStmt(ctx, de)
	}
	ctx.insertStmt(stmt)
}
func (ann *Annotator) newDebugLineStmt(ctx *Ctx, de DebugExpr) ast.Stmt {
	se := &ast.SelectorExpr{
		X:   &ast.Ident{Name: ann.dopt.PkgName, NamePos: de.Pos()},
		Sel: ast.NewIdent("L"),
	}
	args := []ast.Expr{
		basicLitInt(ann.fileIndex, token.NoPos),
		basicLitInt(ctx.getDebugIndex(), token.NoPos),
		basicLitInt(ann.fset.Position(de.Pos()).Offset, token.NoPos),
		ast.Expr(de),
	}
	return &ast.ExprStmt{X: &ast.CallExpr{Fun: se, Args: args}}
}

//----------

func (ann *Annotator) newTType(node ast.Node) (*TType, error) {
	tt, ok := ann.newTType2(node)
	if ok {
		return tt, nil
	}
	s := ann.sprintNode(node)
	return nil, goutil.TodoErrorSkipf(1, "missing type: %T: %s", node, s)
}
func (ann *Annotator) newTType2(node ast.Node) (*TType, bool) {
	tt, ok := newTType(node, ann.typesInfo)
	if ok {
		return tt, true
	}

	// special case (type not defined, debug pkg not imported yet)
	if ann.isDebugPkgCallExpr(node) {
		tt := &TType{node: node}
		tt.Type = types.NewInterfaceType(nil, nil) // mostly debug.Item
		return tt, true
	}

	return nil, false
}

//----------

func (ann *Annotator) exprsTypesExpanded(exprs ...ast.Expr) ([]types.Type, error) {
	w := []types.Type{}
	for _, e := range exprs {
		tt, err := ann.newTType(e)
		if err != nil {
			return nil, err
		}

		// special case: expand // ex: a,ok:=m[1]
		if len(exprs) == 1 {
			return tt.typeTypes(false), nil
		}

		w = append(w, tt.Type)
	}
	return w, nil
}

//----------

func (ann *Annotator) resultDE(ctx *Ctx, expr ast.Expr) (DebugExpr, error) {
	if ctx.valueMatch2(cidnResNil, expr) {
		return nilIdent(expr.Pos()), nil
	}

	expr2, ok, err := ann.resConstantValue(ctx, expr)
	if err != nil {
		return nil, err
	}
	if ok {
		return expr2, nil
	}

	if ctx.valueMatch2(cidnResIsForAddress, expr) {
		return ann.resBasic(ctx, expr, expr)
	}

	if ctx.valueMatch2(cidnResNotReplaceable, expr) {
		return ann.resBasic(ctx, expr, expr)
	}

	if ctx.valueMatch2(cidnResReplaceWithVar, expr) {
		e2, err := ann.resReplaceWithVar(ctx, expr)
		if err != nil {
			return nil, err
		}
		return ann.resBasic(ctx, expr, e2)
	}

	return ann.resBasic(ctx, expr, expr)
}
func (ann *Annotator) resBasic(ctx *Ctx, expr, res ast.Expr) (DebugExpr, error) {
	// build debugexpr
	de := (DebugExpr)(nil)
	if te, ok := res.(*tupleExpr); ok {
		des := []DebugExpr{}
		for _, e := range te.w {
			de2 := ann.newDebugIVi(e)
			des = append(des, de2)
		}
		de = ann.newDebugIL(des...)
	} else {
		de = ann.newDebugIVi(res)
	}
	// operate on debugexpr
	if ctx.valueMatch2(cidnResAssignDebugToVar, expr) {
		e2, err := ann.insertAssignToIdent(ctx, de)
		if err != nil {
			return nil, err
		}
		de = e2
	}
	return de, nil
}
func (ann *Annotator) resConstantValue(ctx *Ctx, expr ast.Expr) (DebugExpr, bool, error) {
	tt, ok := ann.newTType2(expr)
	if !ok {
		return nil, false, nil
	}
	switch {
	case tt.nResults() == 0: // ex: callexpr with no results "f()"
		return nilIdent(expr.Pos()), true, nil
	case tt.isNil():
		de, err := ann.resBasic(ctx, expr, expr)
		return de, true, err
	case tt.isType():
		//return ann.nodeStrDebugExpr(ctx, expr) // ex: "(func(*int))(nil)" redundant, already visible in the code

		//tstr := "T"
		tstr := "_" // simpler, also used in stringifyitem
		return ann.newDebugIVs(tstr, expr.Pos()), true, nil
	}

	//	if tb, ok := tt.Type.(*types.Basic); ok {
	//		// ex: untyped int: "1<<64" (must replace with const value)
	//		// ex: "!true" // untyped bool
	//		// ex: "1<2" // untyped bool
	//		// ex: "2*10" // untyped int
	//		// NOTE: info:types.IsUntyped, kind=types.UntypedBool
	//		if isTypesBasicInfo(tb, types.IsUntyped|types.IsNumeric) {
	//			if v, ok := tt.constValue(); ok {
	//				// TODO: possible to want a type (IVt) of a const value? would need to call basic(...)

	//				val := fmt.Sprintf("%v", v)
	//				return ann.newDebugIVs(val, expr.Pos()), true, nil
	//			}
	//		}
	//	}

	if v, ok := tt.constValue(); ok {
		val := fmt.Sprintf("%v", v)
		return ann.newDebugIVs(val, expr.Pos()), true, nil
	}

	return nil, false, nil
}

//----------

func (ann *Annotator) resReplaceWithVar(ctx *Ctx, expr ast.Expr) (ast.Expr, error) {
	tt, err := ann.newTType(expr)
	if err != nil {
		return nil, err
	}

	if e, ok := ann.castExpr(ctx, expr, tt); ok {
		expr = e
	}

	types := tt.typeTypes(false)
	poss := []token.Pos{expr.Pos()} // can use one pos
	ids := ann.newIncIdentsWithTypes(true, types, poss)
	as := newAssignStmtD(ids, []ast.Expr{expr})

	resultNotAssigned := ctx.valueMatch2(cidnIsExprStmtExpr, expr)
	if resultNotAssigned {
		// TODO: check if can declare variables
		// TODO: check detectjumps

		ctx.replaceStmt(as) // ex: "f(1)" -> "v1:=f(1)"
	} else {
		ctx.insertStmt(as)
		ctx.replaceExprs(ids...)
	}

	if len(ids) == 1 {
		return ids[0], nil
	}
	return &tupleExpr{w: ids}, nil
}
func (ann *Annotator) castExpr(ctx *Ctx, expr ast.Expr, tt *TType) (ast.Expr, bool) {

	needCast := false
	switch t := expr.(type) {
	case *ast.BinaryExpr:
		// For each binary operation, inspect the types of the operands involved. If they are of different types and neither is an untyped constant, then you will likely need to cast. Otherwise, the operation will probably succeed without requiring a cast.

		be := t

		if tt.isBasicInfo(types.IsBoolean) {
			break
		}

		tt1, ok := ann.newTType2(be.X)
		if !ok {
			break
		}
		tt2, ok := ann.newTType2(be.Y)
		if !ok {
			break
		}

		sameType := func(u1, u2 *TType) bool {
			return types.Identical(u1.Type, u2.Type)
		}
		if sameType(tt1, tt2) && sameType(tt, tt1) {
			break
		}

		isUntypedConst := func(tt3 *TType) bool {
			return tt3.isBasicInfo(types.IsUntyped | types.IsConstType)
		}
		if !(isUntypedConst(tt1) || isUntypedConst(tt2)) {
			break
		}

		needCast = true
	}
	if !needCast {
		return nil, false
	}

	fun := &ast.Ident{Name: ann.typeString(tt.Type), NamePos: expr.Pos()}
	ce := &ast.CallExpr{Fun: fun, Args: []ast.Expr{expr}}
	return ce, true
}

//----------

func (ann *Annotator) insertAssignToIdent(ctx *Ctx, expr ast.Expr) (ast.Expr, error) {
	ids, err := ann.insertAssignToIdents(ctx, expr)
	if err != nil {
		return nil, err
	}
	if len(ids) != 1 {
		return nil, goutil.TodoError("len(ids)=%v", len(ids))
	}
	return ids[0], nil
}
func (ann *Annotator) insertAssignToIdents(ctx *Ctx, exprs ...ast.Expr) ([]ast.Expr, error) {
	as, err := ann.insertAssignToIdents2(ctx, exprs...)
	return as.Lhs, err
}
func (ann *Annotator) insertAssignToIdents2(ctx *Ctx, exprs ...ast.Expr) (*ast.AssignStmt, error) {
	as, err := ann.newAssignToIdents(exprs...)
	if err != nil {
		return nil, err
	}
	ctx.insertStmt(as)
	return as, err
}
func (ann *Annotator) newAssignToIdents(exprs ...ast.Expr) (*ast.AssignStmt, error) {
	ids, err := ann.newIncIdentsFromExprs(exprs...)
	if err != nil {
		return nil, err
	}

	as := newAssignStmtD(ids, exprs)
	//setNodePos(as, exprs[0].Pos())

	// TODO: review ***
	//// wrap expr in type if needed for assign
	//for i, e := range as.Rhs {
	//	e2, ok := ann.wrapInTypeForNewVarIfNeeded(e) //***
	//	if !ok {
	//		continue
	//	}
	//	as.Rhs[i] = e2
	//}

	return as, nil
}

func (ann *Annotator) newIncIdentsFromExprs(exprs ...ast.Expr) ([]ast.Expr, error) {
	types, err := ann.exprsTypesExpanded(exprs...)
	if err != nil {
		return nil, err
	}

	// positions for each id to be built
	poss := []token.Pos{}
	for _, e := range exprs {
		poss = append(poss, e.Pos())
	}

	return ann.newIncIdentsWithTypes(true, types, poss), nil
}
func (ann *Annotator) newIncIdentsWithTypes(def bool, ts []types.Type, poss []token.Pos) []ast.Expr {
	w := []ast.Expr{}
	for i, t := range ts {
		// position: assume last given position if len(ts)!=len(poss)
		k := i
		if k >= len(poss) {
			k = len(poss) - 1
		}
		pos := poss[k]

		w = append(w, ann.newIncIdentWithType(def, t, pos))
	}
	return w
}
func (ann *Annotator) newIncIdentWithType(define bool, t types.Type, pos token.Pos) *ast.Ident {
	//id := ann.newIncIdent(pos)

	name := fmt.Sprintf("%s%d", ann.dopt.VarPrefix, ann.debugVarNameIndex)
	ann.debugVarNameIndex++ // increment identifier
	id := &ast.Ident{Name: name, NamePos: pos}

	pkg := (*types.Package)(nil) // TODO: ann.pkg***

	// set type
	v := types.NewVar(id.Pos(), pkg, id.Name, t)
	if define {
		ann.typesInfo.Defs[id] = v
	} else {
		ann.typesInfo.Uses[id] = v
	}
	return id
}

//func (ann *Annotator) newIncIdent(pos token.Pos) *ast.Ident {
//	name := fmt.Sprintf("%s%d", ann.debugVarPrefix, ann.debugVarNameIndex)
//	ann.debugVarNameIndex++ // increment identifier
//	return &ast.Ident{NamePos: pos, Name: name}
//}

//----------

func (ann *Annotator) nameMissingFieldListNames(fl *ast.FieldList) error {
	for _, f := range fl.List {
		if err := ann.nameMissingFieldNames(f); err != nil {
			return err
		}
	}
	return nil
}

//func (ann *Annotator) nameAnonFieldListNames(fl *ast.FieldList) error {
//	for _, f := range fl.List {
//		if err := ann.nameAnonFieldNames(f); err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (ann *Annotator) fieldListNames(fl *ast.FieldList) []ast.Expr {
	w := []ast.Expr{}
	for _, f := range fl.List {
		for _, id := range f.Names {
			w = append(w, id)
		}
	}
	return w
}
func (ann *Annotator) fieldListTypeExprs(fl *ast.FieldList) []ast.Expr {
	w := []ast.Expr{}
	for _, f := range fl.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for k := 0; k < n; k++ {
			w = append(w, f.Type)
		}
	}
	return w
}

//----------

func (ann *Annotator) nameMissingFieldNames(f *ast.Field) error {
	if len(f.Names) == 0 {
		f.Names = []*ast.Ident{anonIdent(f.Pos())}
	}
	return ann.nameAnonFieldNames(f)
}
func (ann *Annotator) nameAnonFieldNames(f *ast.Field) error {
	for i, id := range f.Names {
		if isAnonIdent(id) {
			ids, err := ann.newIncIdentsFromExprs(f.Type)
			if err != nil {
				return err
			}
			if len(ids) != 1 {
				return fmt.Errorf("len(ids)!=1: %v", len(ids))
			}
			f.Names[i] = ids[0].(*ast.Ident)
		}
	}
	return nil
}

//----------

func (ann *Annotator) newFuncLitWithType(resultsTypes []ast.Expr) *ast.FuncLit {
	resultsFields := []*ast.Field{}
	resultsVars := []*types.Var{}
	variadic := false
	pkg := (*types.Package)(nil) // TODO
	for _, t := range resultsTypes {
		f := &ast.Field{Type: t}
		resultsFields = append(resultsFields, f)

		t2 := (types.Type)(nil)
		if tt, ok := ann.newTType2(t); ok {
			t2 = tt.Type
			if tt.isSignatureVariadic() {
				variadic = true
			}
		}
		v := types.NewVar(t.Pos(), pkg, "", t2)
		resultsVars = append(resultsVars, v)
	}
	fl := newFuncLit()
	fl.Type.Results.List = resultsFields

	resultsTuple := types.NewTuple(resultsVars...)
	sig := types.NewSignature(nil, nil, resultsTuple, variadic)
	ann.typesInfo.Types[fl] = types.TypeAndValue{Type: sig}

	return fl
}

//----------
//----------

func (ann *Annotator) sprintNode(n ast.Node) string {
	return astut.SprintNode(ann.fset, n)
}
func (ann *Annotator) printNode(n ast.Node) {
	fmt.Println(ann.sprintNode(n))
}

//----------

func (ann *Annotator) debug(item any) {
	goutil.LogSkipf(1, ann.debug2(item))
}
func (ann *Annotator) debug2(item any) string {
	s := ""
	switch t := item.(type) {
	case ast.Expr:
		s += ann.sprintNode(t)
	case []ast.Expr:
		s += "[0]: " + ann.sprintNode(t[0])
	case ast.Node:
		if t.Pos() != token.NoPos {
			s += ann.posSrc(t.Pos()) + "\n"
		}
		s += ann.sprintNode(t)
	default:
		s = fmt.Sprintf("<debugsrc:%T>", t)
	}
	return "DEBUGSRC:\n" + s
}

//----------

func (ann *Annotator) posSrc(pos token.Pos) string {
	p := ann.fset.Position(pos)
	return fmt.Sprintf("%v:%v:%v", p.Filename, p.Line, p.Column)
}
func (ann *Annotator) posSrcError(pos token.Pos, err error) error {
	return fmt.Errorf("%v: %v", ann.posSrc(pos), err)
}

//----------

func (ann *Annotator) isIdentWithDebugVarPrefix(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && strings.HasPrefix(id.Name, ann.dopt.VarPrefix)
}
func (ann *Annotator) isDebugPkgCallExpr(node ast.Node) bool {
	if ce, ok := node.(*ast.CallExpr); ok {
		if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
			if id, ok := se.X.(*ast.Ident); ok {
				if id.Name == ann.dopt.PkgName {
					return true
				}
			}
		}
	}
	return false
}

//----------

// Returns (on/off, ok)
func (ann *Annotator) nodeAnnotationBlockOn(n ast.Node) (bool, bool) {
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

func (ann *Annotator) addImports(astFile *ast.File) {
	addImp := func(name, path string) {
		_ = astutil.AddNamedImport(ann.fset, astFile, name, path)
	}

	done := false
	ast.Inspect(astFile, func(n2 ast.Node) bool {
		switch t := n2.(type) {
		case *ast.SelectorExpr:
			if id, ok := t.X.(*ast.Ident); ok {
				if id.Name == ann.dopt.PkgName {
					// used
					addImp(ann.dopt.PkgName, ann.dopt.PkgPath)
					done = true
				}
			}
		}
		return !done
	})

	// TODO: other imports?
	//done := false
	//ast.Inspect(astFile, func(n2 ast.Node) bool {
	//	switch t := n2.(type) {
	//	case *ast.SelectorExpr:
	//		if id, ok := t.X.(*ast.Ident); ok {
	//			if pkg, ok := ann.imported.pkgs[id.Name]; ok {
	//				// used
	//				addImp(id.Name, pkg.Path())
	//				done = true
	//			}
	//		}
	//	}
	//	return !done
	//})
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
				if x, ok := se.X.(*ast.Ident); ok {
					if x.Name == ann.dopt.PkgName && se.Sel.Name == "L" {
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
		li.lineCE.Args[1] = basicLitInt(di2, token.NoPos)
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

func (ann *Annotator) detectJumps(ctx *Ctx, node0 ast.Node) (string, bool) {
	detected := ""

	// detect forward jumps
	bm := map[string]*ast.BranchStmt{}
	ast.Inspect(node0, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.BranchStmt: // break,continue,goto,fallthrough
			if t.Label != nil {
				bm[t.Label.Name] = t
			}
		case *ast.LabeledStmt:
			if bs, ok := bm[t.Label.Name]; ok {
				// forward label
				if bs.Pos() < t.Pos() {
					detected = t.Label.Name
				}

				//goutil.Logf("FORWARD LABEL: %v, %v->%v", t.Label.Name, l.Pos(), t.Pos())

				// TEMPORARY
				//if ann.nodeAnnTypes
				//ann.nodeAnnTypes[bs] = AnnotationTypeOff
				//ann.nodeAnnTypes[t] = AnnotationTypeBlock

			}
		}
		return detected == ""
	})

	return detected, detected != ""
}

//----------

func (ann *Annotator) typesPkg() *types.Package {
	for _, obj := range ann.typesInfo.Defs {
		if obj != nil {
			return obj.Pkg()
		}
	}
	return nil
}

//----------

func (ann *Annotator) insertMainClose(ctx *Ctx, fd *ast.FuncDecl) bool {
	if fd.Recv != nil { // is a method
		return false
	}

	// NOTE: case of TestMain in testmode (note from old code)
	// getting the generated file by packages.Load() that contains a "main" will not allow it to be compiled since it uses "testing/internal" packages.

	// set has main flag
	name := "main"
	if ann.testModeMainFunc {
		name = "TestMain"
	}
	if fd.Name.Name != name {
		return false
	}
	ann.hasMainFunc = true

	ds := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(ann.dopt.PkgName),
				Sel: ast.NewIdent("Close"),
			},
		},
	}
	ctx.insertStmt(ds)
	return true
}

func (ann *Annotator) insertDeferRecover(ctx *Ctx) {
	ds := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(ann.dopt.PkgName),
				Sel: ast.NewIdent("Recover"),
			},
		},
	}
	ctx.insertStmt(ds)
}

//----------

func (ann *Annotator) updateOsExitCalls(ctx *Ctx, ce *ast.CallExpr) (error, bool) {
	// verify name
	expr := ce.Fun
	if se, ok := bypassParenExpr(expr).(*ast.SelectorExpr); ok {
		expr = se.Sel
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		return nil, false
	}
	if id.Name != "Exit" {
		return nil, false
	}

	// verify package
	tt, ok := ann.newTType2(id)
	if !ok {
		return nil, false
	}
	pkg, ok := tt.objPackage()
	if !ok {
		return nil, false
	}
	if pkg.Path() != "os" {
		return nil, false
	}

	// insert empty assign to avoid dealing with imports not being used
	as := newAssignStmtD11(anonIdent(ce.Pos()), ce.Fun)
	ctx.insertStmt(as)

	// replace wtih call to debug exit
	ce.Fun = &ast.SelectorExpr{
		X:   ast.NewIdent(ann.dopt.PkgName),
		Sel: ast.NewIdent("Exit"),
	}

	return nil, true
}

//----------

func (ann *Annotator) newFuncLitRetType(node ast.Node) (*ast.FuncLit, error) {
	tt, err := ann.newTType(node)
	if err != nil {
		return nil, err
	}
	fl := newFuncLitRetType(ann.typeString(tt.Type))
	return fl, nil
}

//----------

func (ann *Annotator) typeString(typ types.Type) string {
	// TODO: external pkg private type?

	// can return "untyped bool"
	// can return "go/printer.Mode(...)"
	//name := fmt.Sprintf("%s", tt.Type)

	qf := func(pkg *types.Package) string {
		if pkg == nil || ann.pkg == nil || pkg.Path() == ann.pkg.Path() {
			return ""
		}
		//goutil.Log(pkg.Path())
		return pkg.Name()
	}
	name := types.TypeString(typ, qf)

	// TODO: a better way?
	name = strings.Replace(name, "untyped ", "", 1)

	return name
}

//----------
//----------
//----------

type DebugExpr ast.Expr

//----------
//----------
//----------

type tupleExpr struct {
	ast.Expr
	w []ast.Expr
}

func isTupleExpr(e ast.Expr) bool {
	_, ok := e.(*tupleExpr)
	return ok
}
func mustNotBeTupleExpr(e ast.Expr) {
	if isTupleExpr(e) {
		//s, _ := astut.SprintNode3(&token.FileSet{}, e.(*tupleExpr).w)
		//s := fmt.Sprintf("%v", e.(*tupleExpr).w)
		//panic(fmt.Sprintf("not expecting tuple expr: %s", s))
		panic("not expecting tuple expr")
	}
}

//----------
//----------
//----------

func newAssignStmtA11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return newAssignStmtA([]ast.Expr{lhs}, []ast.Expr{rhs})
}
func newAssignStmtD11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return newAssignStmtA([]ast.Expr{lhs}, []ast.Expr{rhs})
}

func newAssignStmtA(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.ASSIGN, Lhs: lhs, Rhs: rhs}
}
func newAssignStmtD(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.DEFINE, Lhs: lhs, Rhs: rhs}
}

func newAssignToAnons(exprs ...ast.Expr) *ast.AssignStmt {
	// build n anon idents
	ids := make([]ast.Expr, 0, len(exprs))
	for _, e := range exprs {
		ids = append(ids, anonIdent(e.Pos()))
	}

	return newAssignStmtA(ids, exprs)
}

//----------

func newFuncLit() *ast.FuncLit {
	fl := &ast.FuncLit{}
	fl.Type = &ast.FuncType{
		//TypeParams: &ast.FieldList{},
		Params:  &ast.FieldList{},
		Results: &ast.FieldList{},
	}
	fl.Body = &ast.BlockStmt{}
	return fl
}
func newFuncLitRetType(typeName string) *ast.FuncLit {
	fl := newFuncLit()
	fl.Type.Results.List = []*ast.Field{
		{Type: &ast.Ident{Name: typeName}},
	}
	return fl
}

//----------

func isTypesBasicInfo(tb *types.Basic, bi types.BasicInfo) bool {
	return tb.Info()&bi != 0
}

//----------

func isEmptyFieldList(fl *ast.FieldList) bool {
	return fl == nil || len(fl.List) == 0
}

//----------

func nilIdent(pos token.Pos) *ast.Ident {
	return &ast.Ident{Name: "nil", NamePos: pos}
}
func isNilIdent(e ast.Expr) bool {
	return isIdentWithName(e, "nil")
}

func anonIdent(pos token.Pos) *ast.Ident {
	return &ast.Ident{Name: "_", NamePos: pos}
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

// a.b.c runs fn(a) first and fn(c) last.
func identsSequenceFn(e ast.Expr, fn func(*ast.Ident)) bool {
	// ex: (a.b).c
	e = bypassParenExpr(e)

	switch t := e.(type) {
	case *ast.Ident:
		fn(t)
		return true
	case *ast.SelectorExpr:
		if isIdentsSequence(t.X) {
			fn(t.Sel)
			return true
		}
	}
	// ex: (a+b).Fn()
	// ex: a[i]
	return false
}

func isIdentsSequence(e ast.Expr) bool {
	return identsSequenceFn(e, func(*ast.Ident) {})
}

//----------

func bypassParenExpr(expr ast.Expr) ast.Expr {
	if pe, ok := expr.(*ast.ParenExpr); ok {
		return bypassParenExpr(pe.X)
	}
	return expr
}

//----------

func basicLitInt(v int, pos token.Pos) *ast.BasicLit {
	return &ast.BasicLit{
		ValuePos: pos,
		Kind:     token.INT,
		Value:    fmt.Sprintf("%d", v),
	}
}
func basicLitString(v string, pos token.Pos) *ast.BasicLit {
	return &ast.BasicLit{
		ValuePos: pos,
		Kind:     token.STRING,
		Value:    v,
	}
}
func basicLitStringQ(v string, pos token.Pos) *ast.BasicLit {
	return basicLitString(strconv.Quote(v), pos)
}

//----------

func hasChanRecvOp(ue *ast.UnaryExpr) bool {
	return ue.Op == token.ARROW
}
func hasAddressOp(ue *ast.UnaryExpr) bool {
	return ue.Op == token.AND
}

func isChanRecvExpr(expr ast.Expr) (*ast.UnaryExpr, bool) {
	if ue, ok := expr.(*ast.UnaryExpr); ok {
		return ue, hasChanRecvOp(ue)
	}
	return nil, false
}
func isAddressOfExpr(expr ast.Expr) (*ast.UnaryExpr, bool) {
	if ue, ok := expr.(*ast.UnaryExpr); ok {
		return ue, hasAddressOp(ue)
	}
	return nil, false
}

//----------
