package godebug

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/core/gosource"
)

const debugPkgPath = "github.com/jmigpin/editor/core/godebug/debug"

type Annotator struct {
	FSet *token.FileSet

	InsertedExitIn struct {
		Main     bool
		TestMain bool
	}

	fdata struct {
		sync.Mutex
		m     map[string]*debug.AnnotatorFileData // filename -> afd
		index int                                 // counter for new files
	}
	debugPkgName   string
	debugVarPrefix string

	simpleOut     bool
	improveAssign bool

	testFilesPkgs map[string]string // dir -> package name
}

func NewAnnotator() *Annotator {
	ann := &Annotator{
		FSet:          token.NewFileSet(),
		testFilesPkgs: make(map[string]string),
	}

	ann.fdata.m = make(map[string]*debug.AnnotatorFileData)

	ann.debugPkgName = "d" + string(rune(931)) // uncommon  rune to avoid clashes
	ann.debugVarPrefix = ann.debugPkgName      // will have integer appended

	ann.improveAssign = true

	return ann
}

func (ann *Annotator) ParseAnnotate(filename string, src interface{}) (*ast.File, error) {
	// parse
	astFile, err := parser.ParseFile(ann.FSet, filename, src, parser.Mode(0))
	if err != nil {
		return nil, err
	}
	// annotate
	if err := ann.annotate(filename, src, astFile); err != nil {
		return nil, err
	}

	// DEBUG
	//ann.PrintSimple(os.Stdout, astFile)

	return astFile, nil
}

func (ann *Annotator) annotate(filename string, src interface{}, astFile *ast.File) error {
	// don't annotate certain packages
	switch astFile.Name.Name {
	case "godebugconfig", "debug":
		logger.Printf("not annotating: %v %v", astFile.Name.Name, filename)
		return nil
	}

	logger.Printf("annotate: %v", filename)

	// fileindex and afd
	ann.fdata.Lock()
	afd, ok := ann.fdata.m[filename]
	if !ok {
		// filename content hash
		size, hash, err := ann.srcSizeHash(filename, src)
		if err != nil {
			return err
		}

		afd = &debug.AnnotatorFileData{
			FileIndex: ann.fdata.index,
			Filename:  filename,
			FileHash:  hash,
			FileSize:  size,
		}
		ann.fdata.m[filename] = afd
		ann.fdata.index++
	}
	ann.fdata.Unlock()

	sann := &SingleAnnotator{ann: ann, afd: afd}
	sann.annotate(astFile)

	if ann.simpleOut {
		return nil
	}

	// n debug stmts inserted
	afd.DebugLen = sann.debugIndex

	// insert imports if debug stmts were inserted
	if sann.insertedDebugStmt {
		sann.insertImportDebug(astFile)

		// insert in all files to ensure inner init function runs
		sann.insertImport(astFile, "_", "godebugconfig")

		// insert exit in main/TestMain
		em := sann.insertDebugExitInMain(astFile)
		if !ann.InsertedExitIn.Main {
			ann.InsertedExitIn.Main = em
		}
		etm := sann.insertDebugExitInTestMain(astFile)
		if !ann.InsertedExitIn.TestMain {
			ann.InsertedExitIn.TestMain = etm
		}

		// keep test files package names in case of need to build testmain files
		ann.keepTestPackage(filename, astFile)
	}

	return nil
}

func (ann *Annotator) srcSizeHash(filename string, src interface{}) (int, []byte, error) {
	b, err := gosource.ReadSource(filename, src)
	if err != nil {
		return 0, nil, err
	}
	h := sha1.New()
	h.Write(b)
	hash := h.Sum(nil)
	size := len(b)
	return size, hash, nil
}

func (ann *Annotator) ConfigSource() (string, string) {
	// build map data
	var u []string
	for _, afd := range ann.fdata.m {
		logger.Printf("configsource: included file %v", afd.Filename)

		// sanity check
		if afd.FileIndex >= len(ann.fdata.m) {
			panic(fmt.Sprintf("file index doesn't fit map len: %v vs %v", afd.FileIndex, len(ann.fdata.m)))
		}

		s := fmt.Sprintf("&debug.AnnotatorFileData{%v,%v,%q,%v,[]byte(%q)}",
			afd.FileIndex,
			afd.DebugLen,
			afd.Filename,
			afd.FileSize,
			string(afd.FileHash),
		)
		u = append(u, s+",")
	}
	entriesStr := strings.Join(u, "\n")

	// filename
	pkgFilename := "godebugconfig/config.go"

	// content: "+build" line needs and empty line afterwards
	src := `// +build godebug

		package godebugconfig
		import "` + debugPkgPath + `"
		func init(){
			debug.AnnotatorFilesData = []*debug.AnnotatorFileData{
				` + entriesStr + `
			}
		}
	`

	return src, pkgFilename
}

//------------

func (ann *Annotator) keepTestPackage(filename string, astFile *ast.File) {
	isTest := strings.HasSuffix(filename, "_test.go")
	if isTest {
		// keep one pkg name per dir
		dir := filepath.Dir(filename)
		ann.testFilesPkgs[dir] = astFile.Name.Name // pkg name
	}
}

type TestMainSrc struct {
	Dir string
	Src string
}

func (ann *Annotator) TestMainSources() []*TestMainSrc {
	u := []*TestMainSrc{}
	for dir, pkgName := range ann.testFilesPkgs {
		src := ann.testMainSource(pkgName)
		v := &TestMainSrc{Dir: dir, Src: src}
		u = append(u, v)
	}
	return u
}

func (ann *Annotator) testMainSource(pkgName string) string {
	return `		
		package ` + pkgName + `
		import ` + ann.debugPkgName + ` "` + debugPkgPath + `"
		import "testing"
		import "os"
		func TestMain(m *testing.M) {
			code := m.Run()
			` + ann.debugPkgName + `.Exit()
			os.Exit(code)
		}
	`
}

//------------

func (ann *Annotator) Print(w io.Writer, astFile *ast.File) error {
	// print with source positions from original file
	cfg := &printer.Config{Tabwidth: 4, Mode: printer.SourcePos}
	return cfg.Fprint(w, ann.FSet, astFile)
}

func (ann *Annotator) PrintSimple(w io.Writer, astFile *ast.File) error {
	cfg := &printer.Config{Mode: printer.RawFormat}
	return cfg.Fprint(w, ann.FSet, astFile)
}

//------------

type SingleAnnotator struct {
	ann               *Annotator
	afd               *debug.AnnotatorFileData
	debugIndex        int
	debugVarNameIndex int
	insertedDebugStmt bool
}

func (sann *SingleAnnotator) annotate(root ast.Node) {
	// DEBUG
	//gosource.PrintInspect(root)

	ctx := &saCtx{}
	ctx = ctx.WithNewExprs()
	sann.visitNode(ctx, root)
}

func (sann *SingleAnnotator) visitNode(ctx *saCtx, node ast.Node) {
	//log.Printf("visitnode %T", node)

	switch t := node.(type) {
	case *ast.File:
		for _, d := range t.Decls {
			sann.visitNode(ctx, d)
		}

	case *ast.DeclStmt:
		sann.visitNode(ctx, t.Decl)

	case *ast.GenDecl:
		//switch t.Tok {
		//case token.VAR:
		//	for _, s := range t.Specs {
		//		sann.visitNode(ctx, s)
		//	}
		//}

	//case *ast.ValueSpec:
	//// TODO: from a top gendecl? build init func?
	//if ctx.Value("stmts") == nil {
	//	return
	//}
	//// TODO: "var c, d, t uint32 = 1, 2, f1()": handle f1
	//for i, v := range t.Values {
	//	ctx2 := ctx.WithValue("pos", t.Names[i].Pos())
	//	sann.visitNode(ctx2, v)
	//}

	case *ast.FuncDecl:
		sann.visitFuncDecl(ctx, t)

	case *ast.FuncType:
		sann.visitFuncType(ctx, t)
	case *ast.MapType:
		sann.visitMapType(ctx, t)

	case *ast.Field:
		sann.visitField(ctx, t)

	case *ast.Ident:
		sann.visitIdent(ctx, t)

	case *ast.BasicLit:
		sann.visitBasicLit(ctx, t)
	case *ast.CompositeLit:
		sann.visitCompositeLit(ctx, t)
	case *ast.FuncLit:
		sann.visitFuncLit(ctx, t)

	case *ast.SelectorExpr:
		sann.visitSelectorExpr(ctx, t)
	case *ast.CallExpr:
		sann.visitCallExpr(ctx, t)
	case *ast.BinaryExpr:
		sann.visitBinaryExpr(ctx, t)
	case *ast.UnaryExpr:
		sann.visitUnaryExpr(ctx, t)
	case *ast.StarExpr:
		sann.visitStarExpr(ctx, t)
	case *ast.KeyValueExpr:
		sann.visitKeyValueExpr(ctx, t)
	case *ast.SliceExpr:
		sann.visitSliceExpr(ctx, t)
	case *ast.ParenExpr:
		sann.visitParenExpr(ctx, t)
	case *ast.IndexExpr:
		sann.visitIndexExpr(ctx, t)
	case *ast.TypeAssertExpr:
		sann.visitTypeAssertExpr(ctx, t)

	case *ast.BlockStmt:
		sann.visitStmts(ctx, &t.List)
	case *ast.CaseClause:
		sann.visitStmts(ctx, &t.Body)
	case *ast.AssignStmt:
		sann.visitAssignStmt(ctx, t)
	case *ast.ExprStmt:
		sann.visitExprStmt(ctx, t)
	case *ast.SwitchStmt:
		sann.visitSwitchStmt(ctx, t)
	case *ast.TypeSwitchStmt:
		sann.visitTypeSwitchStmt(ctx, t)
	case *ast.IfStmt:
		sann.visitIfStmt(ctx, t)
	case *ast.ForStmt:
		sann.visitForStmt(ctx, t)
	case *ast.RangeStmt:
		sann.visitRangeStmt(ctx, t)
	case *ast.ReturnStmt:
		sann.visitReturnStmt(ctx, t)
	case *ast.LabeledStmt:
		sann.visitLabeledStmt(ctx, t)
	case *ast.BranchStmt:
		sann.visitBranchStmt(ctx, t)
	case *ast.IncDecStmt:
		sann.visitIncDecStmt(ctx, t)
	case *ast.DeferStmt:
		sann.visitDeferStmt(ctx, t)
	case *ast.GoStmt:
		sann.visitGoStmt(ctx, t)

	default:
		//log.Printf("todo: visitnode: %T", node)
	}
}

//------------

func (sann *SingleAnnotator) hasNodeArg(ctx *saCtx) bool {
	v := ctx.Value("nodearg")
	return v != nil
}
func (sann *SingleAnnotator) resetNodeArg(ctx *saCtx, e ast.Expr) {
	if v := ctx.Value("nodearg"); v != nil {
		p := v.(*ast.Expr)
		*p = e
	}
}
func (sann *SingleAnnotator) withNilNodeArg(ctx *saCtx) *saCtx {
	return ctx.WithValue("nodearg", nil)
}
func (sann *SingleAnnotator) visitNodeArg(ctx *saCtx, e *ast.Expr) {
	ctx = ctx.WithValue("nodearg", e)
	ctx2 := ctx.WithNewExprs()
	sann.visitNode(ctx2, *e)
	ctx.PushExprs(ctx2.PopExprs()...)
}

//------------

func (sann *SingleAnnotator) withNilResults(ctx *saCtx) *saCtx {
	ctx = ctx.WithValue("expr_ptr", nil)
	ctx = ctx.WithValue("expr_sliceptr", nil)
	ctx = ctx.WithValue("expr_nresults", nil)
	return ctx
}

func (sann *SingleAnnotator) visitExpr(ctx *saCtx, e *ast.Expr) {
	ctx = ctx.WithValue("expr_ptr", e)
	sann.visitNodeWithNewExprs(ctx, *e)
}

func (sann *SingleAnnotator) replaceExprPtrWithVar(ctx *saCtx, e ast.Expr) ast.Expr {
	if v := ctx.Value("expr_ptr"); v != nil {
		nResults := 1
		if v2 := ctx.Value("expr_nresults"); v2 != nil {
			nResults = v2.(int)
		}

		var ids []ast.Expr
		for i := 0; i < nResults; i++ {
			ids = append(ids, sann.newIdent())
		}

		stmt1 := sann.newDefine(ids, []ast.Expr{e})
		sann.insert(ctx, 0, stmt1)

		var u []ast.Expr
		for _, id := range ids {
			u = append(u, sann.debugCallExpr("IV", id))
		}

		// replace ptr
		if nResults == 1 {
			ep := v.(*ast.Expr)
			if !isDirectExpr(*ep) { // dont replace direct expr: ex: "1*2"
				*ep = ids[0]
			}
			return u[0]
		} else if nResults > 1 {
			if v2 := ctx.Value("expr_sliceptr"); v2 != nil {
				sep := v2.(*[]ast.Expr)
				*sep = ids
			}
			return sann.debugCallExpr("IL", u...)
		} else {
			panic("!")
		}
	}

	return nilIdent()
}

//------------

func (sann *SingleAnnotator) visitNodeWithNewExprs(ctx *saCtx, node ast.Node) {
	ctx2 := ctx.WithNewExprs()
	sann.visitNode(ctx2, node)
	ctx.PushExprs(ctx2.PopExprs()...)
}

func (sann *SingleAnnotator) visitStmts(ctx *saCtx, stmts *[]ast.Stmt) {
	ni0 := 0
	if v := ctx.Value("stmts_startindex"); v != nil {
		ni0 = v.(int)
		ctx.ResetValue("stmts_startindex", nil)
	}

	ctx = ctx.WithValue("stmts", stmts)
	ctx = sann.newStmtsIndexes(ctx)
	ni, si, io := sann.stmtsIndexes(ctx)

	*ni = ni0

	for ; *ni < len(*stmts); *ni++ {
		*si = *ni
		*io = 0

		stmt := (*stmts)[*ni]
		pos := stmt.End() // multiline stmts get their debug line at the last line

		sann.visitNodeWithNewExprs(ctx, stmt)

		// debugline
		u, ok := ctx.Pop1Expr()
		if ok {
			stmt1 := sann.buildLineStmt(pos, u)
			sann.insert(ctx, *io, stmt1)
		}
	}
}

//------------

func (sann *SingleAnnotator) stmtsIndexes(ctx *saCtx) (_, _, _ *int) {
	ni := ctx.Value("stmts_nindex").(*int)
	si := ctx.Value("stmts_sindex").(*int)
	io := ctx.Value("stmts_ioffset").(*int)
	return ni, si, io
}
func (sann *SingleAnnotator) newStmtsIndexes(ctx *saCtx) *saCtx {
	// ni: next index, all inserted stmts add to this var
	// si: stmt index, inserted stmts "before" (offfset==0) add to this var
	// io: debugline insertion offset
	var ni, si, io int
	ctx = ctx.WithValue("stmts_nindex", &ni)
	ctx = ctx.WithValue("stmts_sindex", &si)
	ctx = ctx.WithValue("stmts_ioffset", &io)
	return ctx
}

//------------

func (sann *SingleAnnotator) visitFuncDecl(ctx *saCtx, fd *ast.FuncDecl) {
	if fd.Body == nil {
		return
	}

	// insertions made inside the body
	ctx2 := ctx.WithValue("stmts", &fd.Body.List)
	ctx2 = sann.newStmtsIndexes(ctx2)

	sann.visitNode(ctx2, fd.Type)

	ctx3 := ctx.WithValue("functype", fd.Type) // returnstmt needs access to functype
	sann.visitNode(ctx3, fd.Body)
}

//------------

func (sann *SingleAnnotator) visitFuncType(ctx *saCtx, ft *ast.FuncType) {
	for _, f := range ft.Params.List {
		sann.visitNodeWithNewExprs(ctx, f)
	}
	u := ctx.PopExprs()
	if len(u) > 0 {
		stmt := sann.buildLineStmtWrap(ft.Pos(), "IL", u...)
		sann.insert(ctx, 0, stmt)
	}
}

func (sann *SingleAnnotator) visitMapType(ctx *saCtx, mt *ast.MapType) {
	// TODO: other cases
	// nothing todo in this case: a:=make(map[string]string)
}

//------------

func (sann *SingleAnnotator) visitField(ctx *saCtx, f *ast.Field) {
	for _, id := range f.Names {
		sann.visitNode(ctx, id)
	}
}

//------------

func (sann *SingleAnnotator) visitIdent(ctx *saCtx, id *ast.Ident) {
	if isAnonIdent(id) {
		ce := sann.debugCallExpr("IAn")
		ctx.PushExprs(ce)
		return
	}

	ce := sann.debugCallExpr("IV", id)
	ctx.PushExprs(ce)
}

//------------

func (sann *SingleAnnotator) visitBasicLit(ctx *saCtx, bl *ast.BasicLit) {
	switch bl.Kind {
	case token.STRING:
		result := sann.replaceExprPtrWithVar(ctx, bl)
		ctx.PushExprs(result)
		return
	}

	ce := sann.debugCallExpr("IV", bl)
	ctx.PushExprs(ce)
}

func (sann *SingleAnnotator) visitCompositeLit(ctx *saCtx, cl *ast.CompositeLit) {
	ctx0 := ctx
	ctx = sann.withNilResults(ctx)

	//pos := make([]token.Pos, len(cl.Elts))
	for i, e := range cl.Elts {
		_, e = i, e

		//pos[i] = e.Pos() // keep before visitnode or it might not be available after changes

		//ctx2 := ctx.WithValue("callexpr_create_id", &cl.Elts[i])
		//sann.visitNodeWithNewExprs(ctx, e)

		sann.visitExpr(ctx, &cl.Elts[i])

		//// check if next element is on another line
		//insertNow := false
		//ep1, ep2 := cl.Pos(), pos[i]
		//if i > 0 {
		//	ep1, ep2 = pos[i-1], pos[i]
		//}

		//if ep1 != token.NoPos && ep2 != token.NoPos {
		//	p1 := sann.ann.FSet.Position(ep1)
		//	p2 := sann.ann.FSet.Position(ep2)
		//	if p1.Line != p2.Line {
		//		insertNow = true
		//	}
		//}

		//if insertNow {
		//	u := ctx.PopExprs()
		//	if len(u) > 0 {
		//		stmt1 := sann.buildLineStmt(pos[i], u...)
		//		sann.insert(ctx, 0, stmt1)
		//	}
		//}
	}

	u := ctx.PopExprs()
	e := sann.debugCallExpr("ILit", u...)
	ctx0.PushExprs(e)
}

func (sann *SingleAnnotator) visitFuncLit(ctx *saCtx, fl *ast.FuncLit) {
	if fl.Body == nil {
		return
	}

	ctx0 := ctx

	// don't inner nodes set an outer node
	ctx = sann.withNilResults(ctx)

	// insertions made inside the body
	ctx2 := ctx.WithValue("stmts", &fl.Body.List)
	ctx2 = sann.newStmtsIndexes(ctx2)
	sann.visitNode(ctx2, fl.Type)

	ctx3 := ctx.WithValue("functype", fl.Type) // returnstmt needs access to functype
	sann.visitNode(ctx3, fl.Body)

	e := sann.debugCallExpr("ILit")
	ctx0.PushExprs(e)
}

//------------

func (sann *SingleAnnotator) visitExprStmt(ctx *saCtx, es *ast.ExprStmt) {
	sann.visitNode(ctx, es.X)
}

func (sann *SingleAnnotator) visitTypeSwitchStmt(ctx *saCtx, tss *ast.TypeSwitchStmt) {
	if as, ok := tss.Assign.(*ast.AssignStmt); ok {
		// ignore assign lhs
		sann.visitNode(ctx, as.Rhs[0])
	} else {
		sann.visitNode(ctx, tss.Assign)
	}

	// debugline
	e, ok := ctx.Pop1Expr()
	if ok {
		stmt3 := sann.buildLineStmt(tss.Pos(), e)
		sann.insert(ctx, 0, stmt3)
	}

	sann.visitNode(ctx, tss.Body)
}

func (sann *SingleAnnotator) visitSwitchStmt(ctx *saCtx, ss *ast.SwitchStmt) {
	if ss.Init != nil || ss.Tag != nil {
		if sann.visitWrappedStmt(ctx, ss) {
			return
		}
	}

	if ss.Init != nil {
		sann.visitNodeWithNewExprs(ctx, ss.Init)
		sann.insert(ctx, 0, ss.Init)
		ss.Init = nil
	}

	if ss.Tag != nil {
		//id := sann.newIdent()
		//stmt1 := sann.newDefine11(id, ss.Tag)
		//sann.visitNodeWithNewExprs(ctx, stmt1)
		//sann.visitNodeWithNewExprs(ctx, ss.Tag)
		sann.visitExpr(ctx, &ss.Tag)
		//sann.insert(ctx, 0, stmt1)
		//ss.Tag = id
	}

	// debugline
	u := ctx.PopExprs()
	if len(u) > 0 {
		stmt3 := sann.buildLineStmtWrap(ss.Pos(), "IL2", u...)
		sann.insert(ctx, 0, stmt3)
	}

	sann.visitNode(ctx, ss.Body)
}

func (sann *SingleAnnotator) visitIfStmt(ctx *saCtx, is *ast.IfStmt) {
	if is.Init != nil {
		if sann.visitWrappedStmt(ctx, is) {
			return
		}
	}

	// separate init stmt from "if"
	if is.Init != nil {
		sann.visitNodeWithNewExprs(ctx, is.Init)
		sann.insert(ctx, 0, is.Init)
		is.Init = nil
	}

	// condition
	sann.visitExpr(ctx, &is.Cond)

	// debugline
	u := ctx.PopExprs()
	if len(u) > 0 {
		stmt3 := sann.buildLineStmtWrap(is.Pos(), "IL2", u...)
		sann.insert(ctx, 0, stmt3)
	}

	// TODO: review to prevent revisit inserted nodes (possible? or need to keep using detecting if the inserted notes are debuglines)
	sann.visitNode(ctx, is.Body)

	if is.Else != nil {
		// "else if"
		if is2, ok := is.Else.(*ast.IfStmt); ok {
			// wrap in blockstmt
			bs := &ast.BlockStmt{}
			bs.List = append(bs.List, is2)
			is.Else = bs
			sann.visitNode(ctx, bs)
		} else {
			sann.visitNode(ctx, is.Else)
		}
	}
}

func (sann *SingleAnnotator) visitForStmt(ctx *saCtx, fs *ast.ForStmt) {
	if fs.Cond != nil {
		// insertions made inside the body
		ctx2 := ctx.WithValue("stmts", &fs.Body.List)
		ctx2 = sann.newStmtsIndexes(ctx2)

		// new condition to break the loop
		stmt2 := &ast.IfStmt{
			If:   fs.Pos(),
			Cond: fs.Cond,
			Body: &ast.BlockStmt{},
		}

		sann.visitNode(ctx2, stmt2)
		sann.insert(ctx2, 0, stmt2)

		fs.Cond = nil

		// negate condition
		stmt2.Cond = &ast.UnaryExpr{Op: token.NOT, X: stmt2.Cond}
		// insert break
		u := &stmt2.Body.List
		*u = append(*u, &ast.BranchStmt{Tok: token.BREAK})

		// visit body without revisiting inserted nodes
		ni := ctx2.Value("stmts_nindex").(*int)
		ctx3 := ctx.WithValue("stmts_startindex", *ni)
		sann.visitNode(ctx3, fs.Body)
		return
	}

	sann.visitNode(ctx, fs.Body)
}

func (sann *SingleAnnotator) visitRangeStmt(ctx *saCtx, rs *ast.RangeStmt) {
	// assign range expression to var
	id := sann.newIdent()
	stmt1 := sann.newDefine11(id, rs.X)
	sann.insert(ctx, 0, stmt1)
	rs.X = id

	// allow defining new vars of both key/value if not set
	if (rs.Key == nil || isAnonIdent(rs.Key)) && (rs.Value == nil || isAnonIdent(rs.Value)) {
		rs.Tok = token.DEFINE
	}

	for _, ep := range []*ast.Expr{&rs.Key, &rs.Value} {
		if rs.Tok == token.DEFINE && isAnonIdent(*ep) {
			*ep = sann.newIdent()
		}
		sann.visitExpr(ctx, ep)
	}

	//// key
	//ctx3 := ctx
	//if rs.Tok == token.DEFINE {
	//	ctx3 = ctx.WithValue("expr_anon_ptr", &rs.Key)
	//}
	//sann.visitNode(ctx3, rs.Key)

	//// value
	//ctx4 := ctx
	//if rs.Tok == token.DEFINE {
	//	ctx4 = ctx.WithValue("expr_anon_ptr", &rs.Value)
	//}
	//sann.visitNode(ctx4, rs.Value)

	{
		// inner loop stmts for insertion
		ctx2 := ctx.WithValue("stmts", &rs.Body.List)
		ctx2 = sann.newStmtsIndexes(ctx2)

		w := ctx2.PopExprs()

		// len of range expr
		ce2 := callExpr("len", id)
		ce3 := sann.debugCallExpr("IV", ce2)

		// build as assign: k, v <- len
		ce4 := sann.debugCallExpr("IL", w...)
		ce5 := sann.debugCallExpr("IL", ce3)
		ce6 := sann.debugCallExpr("IA", ce4, ce5)

		stmt3 := sann.buildLineStmt(rs.Pos(), ce6)
		sann.insert(ctx2, 0, stmt3)

		// visit body without revisiting inserted nodes
		ni := ctx2.Value("stmts_nindex").(*int)
		ctx3 := ctx.WithValue("stmts_startindex", *ni)
		sann.visitNode(ctx3, rs.Body)
	}
}

func (sann *SingleAnnotator) visitAssignStmt(ctx *saCtx, as *ast.AssignStmt) {
	// ex: a, b, _ = 1, c, b
	//	V0, V1 := c, b // because "c" could have been used on lhs
	// 	v0:=debug(1, c, b)
	// 	a, b, _  = 1, c, b

	for i, _ := range as.Rhs {
		ctx2 := ctx
		if len(as.Rhs) == 1 {
			ctx2 = ctx2.WithValue("expr_nresults", len(as.Lhs))
			ctx2 = ctx2.WithValue("expr_sliceptr", &as.Rhs)
		}
		sann.visitExpr(ctx2, &as.Rhs[i])
	}
	rhs := ctx.PopExprs()

	// rhs debugline var
	rhsId := sann.newIdent()
	ce4 := sann.debugCallExpr("IL", rhs...)
	stmt2 := sann.newDefine11(rhsId, ce4)
	sann.insert(ctx, 0, stmt2)

	for i, _ := range as.Lhs {
		// not setting "expr_ptr" (ex: "a[i]=b" would replace a[i] with a d0)
		sann.visitNodeWithNewExprs(ctx, as.Lhs[i])
	}
	lhs := ctx.PopExprs()

	ce1 := sann.debugCallExpr("IL", lhs...)
	//ce2 := sann.debugCallExpr("IL", rhs...)
	ce3 := sann.debugCallExpr("IA", ce1, rhsId)
	ctx.PushExprs(ce3)

	// if inserted, should be with this index offset (at the end of all insertions)
	ni, si, io := sann.stmtsIndexes(ctx)
	*io = *ni - *si + 1
}

func (sann *SingleAnnotator) visitReturnStmt(ctx *saCtx, rs *ast.ReturnStmt) {
	// functype
	ft := ctx.Value("functype").(*ast.FuncType)
	if ft.Results == nil {
		return
	}

	// functype number of results to return
	ftNResults := ft.Results.NumFields()
	if ftNResults == 0 {
		return
	}

	if len(rs.Results) == 0 { // naked return
		// TODO: setup name for anon var in functype results
		for _, f := range ft.Results.List {
			for _, id := range f.Names {
				sann.visitNode(ctx, id)
			}
		}
	} else if len(rs.Results) == ftNResults {
		for i, _ := range rs.Results {
			sann.visitExpr(ctx, &rs.Results[i])
		}
	} else if len(rs.Results) == 1 {
		var lhs []ast.Expr
		for i := 0; i < ftNResults; i++ {
			id := sann.newIdent()
			lhs = append(lhs, id)
		}
		stmt1 := sann.newDefine(lhs, rs.Results)
		sann.visitNode(ctx, stmt1)
		sann.insert(ctx, 0, stmt1)
		rs.Results = lhs
	}

	// debugline
	u := ctx.PopExprs()
	if len(u) > 0 {
		stmt1 := sann.buildLineStmtWrap(rs.Pos(), "IL", u...)
		sann.insert(ctx, 0, stmt1)
	}
}

func (sann *SingleAnnotator) visitLabeledStmt(ctx *saCtx, ls *ast.LabeledStmt) {
	if ls.Stmt == nil {
		return
	}

	if _, ok := ls.Stmt.(*ast.EmptyStmt); !ok {
		// move inner stmt to list of stmts and assign empty stmt
		sann.insert(ctx, 1, ls.Stmt)
		ls.Stmt = &ast.EmptyStmt{}

		// ensure the moved stmt is visited by ajusting the nindex
		ni := ctx.Value("stmts_nindex").(*int)
		(*ni)--

		return
	}

	sann.visitNode(ctx, ls.Stmt)
}

func (sann *SingleAnnotator) visitBranchStmt(ctx *saCtx, bs *ast.BranchStmt) {
	e := sann.debugCallExpr("IBr")
	stmt1 := sann.buildLineStmt(bs.Pos(), e)
	sann.insert(ctx, 0, stmt1)
}

func (sann *SingleAnnotator) visitIncDecStmt(ctx *saCtx, ids *ast.IncDecStmt) {
	sann.visitNode(ctx, ids.X)
	e, ok := ctx.Pop1Expr()
	if ok {
		stmt := sann.buildLineStmt(ids.Pos(), e)
		sann.insert(ctx, 1, stmt)
	}
}

func (sann *SingleAnnotator) visitDeferStmt(ctx *saCtx, ds *ast.DeferStmt) {
	sann.visitNode(ctx, ds.Call)
}

func (sann *SingleAnnotator) visitGoStmt(ctx *saCtx, gs *ast.GoStmt) {
	sann.visitNode(ctx, gs.Call)
}

//------------

func (sann *SingleAnnotator) visitSelectorExpr(ctx *saCtx, se *ast.SelectorExpr) {
	ce := sann.debugCallExpr("IV", se)
	ctx.PushExprs(ce)
}

func (sann *SingleAnnotator) visitTypeAssertExpr(ctx *saCtx, tae *ast.TypeAssertExpr) {
	ce := sann.debugCallExpr("IVt", tae.X)
	ctx.PushExprs(ce)
}

func (sann *SingleAnnotator) visitBinaryExpr(ctx *saCtx, be *ast.BinaryExpr) {
	switch be.Op {
	case token.LAND, token.LOR:
		sann.visitBinaryExpr3(ctx, be)
	default:
		sann.visitBinaryExpr2(ctx, be)
	}
}

func (sann *SingleAnnotator) visitBinaryExpr2(ctx *saCtx, be *ast.BinaryExpr) {
	sann.visitExpr(ctx, &be.X)
	x, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	sann.visitExpr(ctx, &be.Y)
	y, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	result := sann.replaceExprPtrWithVar(ctx, be)

	opbl := basicLitInt(int(be.Op))
	ce3 := sann.debugCallExpr("IB", result, opbl, x, y)
	id2 := sann.newIdentDefineInsert(ctx, ce3)
	ctx.PushExprs(id2)
}

func (sann *SingleAnnotator) visitBinaryExpr3(ctx *saCtx, be *ast.BinaryExpr) {
	// ex: f1() || f2() // f2 should not be called if f1 is true
	// ex: f1() && f2() // f2 should not be called if f1 is false

	sann.visitExpr(ctx, &be.X)
	x, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	// create var with final result and assign X to it
	resultId := sann.newIdentDefineInsert(ctx, be.X)

	// create variable to hold Y debug data
	q := sann.debugCallExpr("IVs", basicLitString("?"))
	yId := sann.newIdentDefineInsert(ctx, q)

	var cond ast.Expr
	switch be.Op {
	case token.LAND:
		cond = be.X
	case token.LOR:
		cond = &ast.UnaryExpr{Op: token.NOT, X: be.X}
	default:
		panic("!")
	}

	// ifstmt with X true
	ifStmt := &ast.IfStmt{If: be.Pos(), Cond: cond, Body: &ast.BlockStmt{}}
	sann.insert(ctx, 0, ifStmt)

	{
		// inner stmts for inserts
		ctx2 := ctx.WithValue("stmts", &ifStmt.Body.List)
		ctx2 = sann.newStmtsIndexes(ctx2)

		// visit Y inside ctx with X true
		sann.visitExpr(ctx2, &be.Y)
		y, ok := ctx2.Pop1Expr()
		if !ok {
			return
		}

		// assign Y debug data to var
		stmt6 := sann.newAssign11(yId, y)
		sann.insert(ctx2, 0, stmt6)

		// assign Y to final result
		stmt5 := sann.newAssign11(resultId, be.Y)
		sann.insert(ctx2, 0, stmt5)
	}

	//result := sann.replaceExprPtrWithVar(ctx, resultId)
	//opbl := basicLitInt(int(be.Op))
	//ce3 := sann.debugCallExpr("IB", result, opbl, x, yId)
	//ctx.PushExprs(ce3)

	if v := ctx.Value("expr_ptr"); v != nil {
		ep := v.(*ast.Expr)
		*ep = resultId
	}

	opbl := basicLitInt(int(be.Op))
	ce2 := sann.debugCallExpr("IV", resultId)
	ce3 := sann.debugCallExpr("IB", ce2, opbl, x, yId)
	ctx.PushExprs(ce3)
}

func (sann *SingleAnnotator) visitUnaryExpr(ctx *saCtx, ue *ast.UnaryExpr) {
	sann.visitExpr(ctx, &ue.X)
	x, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	result := sann.replaceExprPtrWithVar(ctx, ue)

	opbl := basicLitInt(int(ue.Op))
	ce3 := sann.debugCallExpr("IU", result, opbl, x)
	id := sann.newIdentDefineInsert(ctx, ce3)
	ctx.PushExprs(id)
}

func (sann *SingleAnnotator) visitStarExpr(ctx *saCtx, ue *ast.StarExpr) {
	sann.visitExpr(ctx, &ue.X)
	x, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	result := sann.replaceExprPtrWithVar(ctx, ue)

	opbl := basicLitInt(int(token.MUL))
	ce3 := sann.debugCallExpr("IU", result, opbl, x)
	ctx.PushExprs(ce3)
}

func (sann *SingleAnnotator) visitKeyValueExpr(ctx *saCtx, kv *ast.KeyValueExpr) {
	// TODO: kv.Key - need new item type

	if kv.Value != nil {
		sann.visitExpr(ctx, &kv.Value)
	}
}

func (sann *SingleAnnotator) visitParenExpr(ctx *saCtx, pe *ast.ParenExpr) {
	sann.visitNode(ctx, pe.X)
	x, ok := ctx.Pop1Expr()
	if !ok {
		return
	}

	ce := sann.debugCallExpr("IP", x)
	ctx.PushExprs(ce)
}

//------------

func (sann *SingleAnnotator) visitIndexExpr(ctx *saCtx, ie *ast.IndexExpr) {
	ctx0 := ctx

	// ex: a, b := c[f1()] // if "expr_nresults" is set, it will generate "d0,d1:=f1()"
	ctx = sann.withNilResults(ctx)

	var x ast.Expr
	switch ie.X.(type) {
	case *ast.Ident,
		*ast.SelectorExpr:
		x = nilIdent()
	default:
		sann.visitExpr(ctx, &ie.X)
		u, ok := ctx.Pop1Expr()
		if !ok {
			return
		}
		x = u
	}

	sann.visitExpr(ctx, &ie.Index)
	ix, ok := ctx.Pop1Expr()
	if !ok {
		ix = nilIdent()
	}

	result := sann.replaceExprPtrWithVar(ctx0, ie)

	ce3 := sann.debugCallExpr("II", result, x, ix)
	ctx0.PushExprs(ce3)
}

func (sann *SingleAnnotator) visitSliceExpr(ctx *saCtx, se *ast.SliceExpr) {
	var x ast.Expr
	switch se.X.(type) {
	case *ast.Ident,
		*ast.SelectorExpr:
		x = nilIdent()
	default:
		sann.visitExpr(ctx, &se.X)
		u, ok := ctx.Pop1Expr()
		if !ok {
			return
		}
		x = u
	}

	for _, e := range []*ast.Expr{&se.Low, &se.High, &se.Max} {
		if *e == nil {
			ctx.PushExprs(nilIdent())
			continue
		}
		sann.visitExpr(ctx, e)
	}
	ix := ctx.PopExprs()

	result := sann.replaceExprPtrWithVar(ctx, se)

	ce := sann.debugCallExpr("II2", result, x, ix[0], ix[1], ix[2])
	ctx.PushExprs(ce)
}

func (sann *SingleAnnotator) visitCallExpr(ctx *saCtx, ce *ast.CallExpr) {
	//// don't annotate these
	//if id, ok := ce.Fun.(*ast.Ident); ok {
	//	switch id.Name {
	//	}
	//}

	// TODO: review - where is this happening
	// don't annotate debug pkg stmts
	if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
		if id, ok := se.X.(*ast.Ident); ok {
			if id.Name == sann.ann.debugPkgName {
				return
			}
		}
	}

	// don't let subargs get an upper option that is handled to get the result here later
	ctx2 := sann.withNilResults(ctx)

	switch ce.Fun.(type) {
	case *ast.Ident:
	case *ast.SelectorExpr:
	default:
		sann.visitExpr(ctx2, &ce.Fun)
	}

	for i, _ := range ce.Args {
		sann.visitExpr(ctx2, &ce.Args[i])
	}
	args := ctx.PopExprs()

	result := sann.replaceExprPtrWithVar(ctx, ce)

	ce3 := sann.debugCallExpr("IC", append([]ast.Expr{result}, args...)...)
	varId := sann.newIdentDefineInsert(ctx, ce3)
	ctx.PushExprs(varId)

}

// Helper for stmts that need to be wrapped in a blockstmt to insert/declare variables.
func (sann *SingleAnnotator) visitWrappedStmt(ctx *saCtx, stmt ast.Stmt) bool {
	stmts := ctx.Value("stmts").(*[]ast.Stmt)
	if len(*stmts) == 1 {
		return false
	}
	// wrap in blockstmt
	bs := &ast.BlockStmt{}
	bs.List = append(bs.List, stmt)

	// replace stmt
	si := ctx.Value("stmts_sindex").(*int)
	(*stmts)[*si] = bs

	sann.visitNode(ctx, bs)
	return true
}

//------------

func (sann *SingleAnnotator) newVarName() string {
	defer func() { sann.debugVarNameIndex++ }()
	return fmt.Sprintf(sann.ann.debugVarPrefix+"%d", sann.debugVarNameIndex)
}

func (sann *SingleAnnotator) newIdent() *ast.Ident {
	return &ast.Ident{Name: sann.newVarName()}
}

func (sann *SingleAnnotator) newAssign(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.ASSIGN, Lhs: lhs, Rhs: rhs}
}

func (sann *SingleAnnotator) newDefine(lhs, rhs []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.DEFINE, Lhs: lhs, Rhs: rhs}
}

func (sann *SingleAnnotator) newAssign11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{lhs}, Rhs: []ast.Expr{rhs}}
}

func (sann *SingleAnnotator) newDefine11(lhs, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{lhs}, Rhs: []ast.Expr{rhs}}
}

func (sann *SingleAnnotator) newIdentDefineInsert(ctx *saCtx, e ast.Expr) ast.Expr {
	id := sann.newIdent()
	stmt1 := sann.newDefine11(id, e)
	sann.insert(ctx, 0, stmt1)
	return id
}

//------------

func (sann *SingleAnnotator) insert(ctx *saCtx, offset int, u ...ast.Stmt) {
	v1 := ctx.Value("stmts")
	if v1 == nil {
		return
	}
	stmts := v1.(*[]ast.Stmt)

	ni, si, _ := sann.stmtsIndexes(ctx)

	for i, stmt := range u {
		*stmts = sann.insertInStmts(stmt, (*si)+offset+i, *stmts)
	}

	if offset == 0 {
		(*si) += len(u)
	}
	(*ni) += len(u)
}

func (sann *SingleAnnotator) insertInStmts(stmt ast.Stmt, i int, stmts []ast.Stmt) []ast.Stmt {
	list := make([]ast.Stmt, 0, len(stmts)+1)
	list = append(list, stmts[:i]...)
	list = append(list, stmt)
	list = append(list, stmts[i:]...)
	return list
}

//------------

func (sann *SingleAnnotator) insertImportDebug(astFile *ast.File) {
	sann.insertImport(astFile, sann.ann.debugPkgName, debugPkgPath)
}

func (sann *SingleAnnotator) insertImport(astFile *ast.File, name, path string) {
	// pkg quoted path
	qpath := fmt.Sprintf("%q", path)

	// check if it is being imported already
	for _, imp := range astFile.Imports {
		if name != "" {
			if imp.Name != nil && imp.Name.Name == name {
				return
			}
		} else {
			if imp.Path.Value == qpath {
				return
			}
		}
	}

	// pkg name
	var nameId *ast.Ident
	if name != "" {
		nameId = ast.NewIdent(name)
	}

	// import decl
	imp := &ast.ImportSpec{
		Name: nameId,
		Path: &ast.BasicLit{Kind: token.STRING, Value: qpath},
	}
	decl := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{imp}}
	// prepend to decls
	astFile.Decls = append([]ast.Decl{decl}, astFile.Decls...)
}

func (sann *SingleAnnotator) insertDebugExitInMain(astFile *ast.File) bool {
	return sann.insertDebugExitInFunction(astFile, "main")
}

func (sann *SingleAnnotator) insertDebugExitInTestMain(astFile *ast.File) bool {
	return sann.insertDebugExitInFunction(astFile, "TestMain")
}

func (sann *SingleAnnotator) insertDebugExitInFunction(astFile *ast.File, name string) bool {
	obj := astFile.Scope.Lookup(name)
	if obj == nil || obj.Kind != ast.Fun {
		return false
	}
	fd, ok := obj.Decl.(*ast.FuncDecl)
	if !ok || fd.Body == nil {
		return false
	}

	// defer exit stmt
	stmt1 := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(sann.ann.debugPkgName),
				Sel: ast.NewIdent("Exit"),
			},
		},
	}
	// insert
	fd.Body.List = sann.insertInStmts(stmt1, 0, fd.Body.List)
	return true
}

//------------

func (sann *SingleAnnotator) buildLineStmtWrap(pos token.Pos, wrapFuncName string, u ...ast.Expr) ast.Stmt {
	switch len(u) {
	case 0:
		//panic("len=0")
		e := basicLitString("[ERROR:len=0]")
		return sann.buildLineStmt(pos, e)
	case 1:
		return sann.buildLineStmt(pos, u[0])
	default:
		// wrap: "IL", "IL2"
		e := sann.debugCallExpr(wrapFuncName, u...)
		return sann.buildLineStmt(pos, e)
	}
}

func (sann *SingleAnnotator) buildLineStmt(pos token.Pos, e ast.Expr) ast.Stmt {
	sann.insertedDebugStmt = true
	defer func() { sann.debugIndex++ }()

	position := sann.ann.FSet.Position(pos)
	lineOffset := position.Offset

	args := []ast.Expr{
		basicLitInt(sann.afd.FileIndex),
		basicLitInt(sann.debugIndex),
		basicLitInt(lineOffset),
		e,
	}

	se := &ast.SelectorExpr{
		X:   ast.NewIdent(sann.ann.debugPkgName),
		Sel: ast.NewIdent("Line"),
	}
	es := &ast.ExprStmt{X: &ast.CallExpr{Fun: se, Args: args}}
	return es
}

func (sann *SingleAnnotator) debugCallExpr(fname string, u ...ast.Expr) ast.Expr {
	se := &ast.SelectorExpr{
		X:   ast.NewIdent(sann.ann.debugPkgName),
		Sel: ast.NewIdent(fname),
	}
	return &ast.CallExpr{Fun: se, Args: u}
}

func (sann *SingleAnnotator) debugIVsAnon() ast.Expr {
	return sann.debugCallExpr("IVs", basicLitString("_"))
}

//------------

type saCtx struct {
	parent *saCtx
	name   string
	value  interface{}
}

func (ctx *saCtx) WithValue(name string, v interface{}) *saCtx {
	return &saCtx{ctx, name, v}
}
func (ctx *saCtx) Value(name string) interface{} {
	for c := ctx; c != nil; c = c.parent {
		if c.name == name {
			return c.value
		}
	}
	return nil
}
func (ctx *saCtx) ResetValue(name string, v interface{}) {
	for c := ctx; c != nil; c = c.parent {
		if c.name == name {
			c.value = v
			return
		}
	}
}

//------------

// Should be used if the function is popping, to prevent previous functions expressions to pop.
func (ctx *saCtx) WithNewExprs() *saCtx {
	u := []ast.Expr{}
	return ctx.WithValue("exprs", &u)
}

func (ctx *saCtx) PushExprs(e ...ast.Expr) {
	v := ctx.Value("exprs")
	if v == nil {
		return
	}
	u := v.(*[]ast.Expr)
	*u = append(*u, e...)
}

func (ctx *saCtx) PopExprs() []ast.Expr {
	v := ctx.Value("exprs")
	if v == nil {
		return nil
	}
	u := v.(*[]ast.Expr)
	r := *u
	*u = []ast.Expr{}
	return r
}

func (ctx *saCtx) Pop1Expr() (ast.Expr, bool) {
	u := ctx.PopExprs()
	if len(u) == 0 {
		return nil, false
	}
	if len(u) == 1 {
		return u[0], true
	}

	// DEBUG
	log.Printf("---")
	for _, e := range u {
		log.Printf("%T", e)
	}
	s := fmt.Sprintf("expecting 1 expr: len(u)=%v", len(u))
	log.Printf(s)

	return nil, false
}

//------------

func clearNilExprs(u []ast.Expr) []ast.Expr {
	w := []ast.Expr{}
	for _, e := range u {
		if e != nil {
			w = append(w, e)
		}
	}
	return w
}

func anonCount(u []ast.Expr) int {
	c := 0
	for _, e := range u {
		if isAnonIdent(e) {
			c++
		}
	}
	return c
}

func anonIdent() *ast.Ident {
	return &ast.Ident{Name: "_"}
}
func isAnonIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

var _nilIdent = &ast.Ident{Name: "nil"}

func nilIdent() *ast.Ident {
	//return &ast.Ident{Name: "nil"}
	return _nilIdent
}
func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

func callExpr(fname string, u ...ast.Expr) ast.Expr {
	return &ast.CallExpr{Fun: ast.NewIdent(fname), Args: u}
}
func basicLitString(v string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", v)}
}
func basicLitInt(v int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}

// Can't create vars from the expr or it could create a var of different type.
func isDirectExpr(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.BasicLit:
		switch t.Kind {
		case token.INT, token.FLOAT:
			return true
		}
	case *ast.Ident:
		switch t.Name {
		case "nil":
			return true
		}
	case *ast.ParenExpr:
		return isDirectExpr(t.X)
	case *ast.BinaryExpr:
		// ex: var a float =1*2 // if assigned to tmp var, it would be an int that if used without casts when assigned to "a" would not compile
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

//// Has no inner nodes. Useful to detect if the expr will output something.
//func isFlat(e ast.Expr) bool {
//	switch e.(type) {
//	case *ast.BasicLit,
//		*ast.Ident,
//		*ast.SelectorExpr:
//		return true
//	}
//	return false
//}

// Can't create vars from the expr or it could create a var of different type. Includes other expressions without the requirement but are used as direct to improve code generation.
//func mustBeDirect(e ast.Expr) bool {
//	switch t := e.(type) {
//	case *ast.BasicLit:
//		return t.Kind != token.STRING // avoid double print of long strings
//	case *ast.SelectorExpr,
//		*ast.Ident:
//		return true
//	case *ast.BinaryExpr:
//		switch t.Op {
//		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
//			return mustBeDirect(t.X) && mustBeDirect(t.Y)
//		}
//	case *ast.UnaryExpr:
//		switch t.Op {
//		case token.ADD, token.SUB:
//			return mustBeDirect(t.X)
//		}
//	}
//	//return sann.isDirect(e)
//	return false
//}
