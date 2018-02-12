package gosource

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type CCResult struct {
	StartIndex int
	Str        string
	Segments   [][2]int
	Objs       []types.Object
}

func CodeCompletion(filename string, src interface{}, index int) (*CCResult, error) {
	cc := &CC2{}
	res, err := cc.run(filename, src, index)
	if err != nil {
		return nil, err
	}

	// show comment if the first result is matching
	comment := ""
	if len(res.Objs) > 0 {
		obj := res.Objs[0]
		seg := res.Segments[0]
		if len(res.Objs) == 1 || (seg[0] == 0 && seg[1] == len(obj.Name())) {
			s, ok := cc.objComment(obj)
			if ok {
				comment = "\n\n" + s
			}
		}
	}

	showArgs := len(res.Objs) < 3

	FormatResult2(res, showArgs) // alters res.Str and res.Segments

	res.Str += comment

	return res, err
}

type CC2 struct {
	astFile   *ast.File
	conf      *Config
	res       *DeclResolver
	result    *CCResult
	filterStr string
	//filter    struct {
	//types bool
	//}
}

func (cc *CC2) run(filename string, src interface{}, index int) (*CCResult, error) {
	cc.result = &CCResult{StartIndex: index}
	cc.conf = NewConfig()
	cc.conf.ParserMode = parser.ParseComments

	astFile, err, ok := cc.conf.ParseFile(filename, src, 0)
	if !ok {
		return nil, err
	}
	cc.astFile = astFile

	// make package path importable and re-import (type check added astfile)
	cc.conf.MakeFilePkgImportable(filename)
	_ = cc.conf.ReImportImportables()

	// index fset position
	tf, err := cc.conf.PosTokenFile(cc.astFile.Package)
	if err != nil {
		return nil, err
	}
	if index >= tf.Size() {
		return nil, fmt.Errorf("index bigger than filesize")
	}
	ipos := token.Pos(tf.Base() + index)

	cc.res = NewDeclResolver(cc.conf)

	// DEBUG
	src2 := ""
	if s, ok := src.(string); ok {
		src2 = s
	}

	//// DEBUG
	//PrintInspect(cc.astFile)
	//ast.Inspect(cc.astFile, func(node ast.Node) bool {
	//	if node == nil {
	//		return false
	//	}
	//	log.Printf("%T %v", node, node)
	//	return true
	//})

	if err := cc.candidatesInPos(ipos, src2); err != nil {
		return nil, err
	}

	cc.filterCandidates(cc.filterStr)

	return cc.result, nil
}

func (cc *CC2) candidatesInPos(pos token.Pos, src string) error {
	// TODO: help the checker

	orPos := pos

	//posStart:

	// path in astFile
	path, _ := astutil.PathEnclosingInterval(cc.astFile, pos, pos)
	if len(path) < 2 {
		return fmt.Errorf("path len: %v", len(path))
	}
	node := path[0]

	// DEBUG
	p := cc.conf.FSet.Position(pos)
	_, _ = TokPositionStr(src, p)

switchStart:

	switch t := node.(type) {
	case *ast.Ident:
		// parent node
		switch t2 := path[1].(type) {
		case *ast.SelectorExpr:
			node = t2
			goto switchStart
		}
	case *ast.BlockStmt:
		return nil
		//pos--
		//goto posStart
	case *ast.CallExpr:
		if pos == t.Lparen {
			node = t.Fun
			goto switchStart
		} else if pos == t.Rparen {
			// ex: "fmt.Print(|)"
			return cc.candidatesInLastArg(t)
		}
	case *ast.SelectorExpr:
		pos := orPos
		if pos >= t.X.Pos() && pos <= t.X.End() {
			// ex: "fmt|."
			if id, ok := t.X.(*ast.Ident); ok {
				cc.result.StartIndex = cc.conf.FSet.Position(t.X.Pos()).Offset
				cc.filterStr = id.Name[:pos-t.X.Pos()]
				return cc.candidatesInScopeUp(t.X)
			}
		} else if pos >= t.Sel.Pos() && pos <= t.Sel.End() {
			// ex: "fmt.|P"
			cc.result.StartIndex = cc.conf.FSet.Position(t.Sel.Pos()).Offset
			cc.filterStr = t.Sel.Name[:pos-t.Sel.Pos()]
			return cc.candidatesInNode(t.X)
		}

	default:
		return fmt.Errorf("todo: %T", node)
	}

	return fmt.Errorf("unable to get candidates in pos")
}

//------------

func (cc *CC2) candidatesInNode(node ast.Node) error {
	n2, err := cc.res.ResolveType(node)
	if err != nil {
		return err
	}
	switch t := n2.(type) {
	case *ast.ImportSpec:
		return cc.candidatesInImportSpec(t, node)
	case *ast.StructType:
		return cc.candidatesInNode2(t, node)
	}
	return fmt.Errorf("todo")
}

func (cc *CC2) candidatesInImportSpec(imp *ast.ImportSpec, reqNode ast.Node) error {
	// importspec pkg
	impPath, _ := strconv.Unquote(imp.Path.Value)
	pkg, ok := cc.conf.Pkgs[impPath]
	if !ok {
		return fmt.Errorf("importspec pkg not found")
	}

	// node pkg dir
	pkgDir, err := cc.conf.PosPkgDir(reqNode.Pos())
	if err != nil {
		return err
	}

	// want not exported (private) entries if on same path
	wantNotExp := pkgDir == impPath

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if wantNotExp || obj.Exported() {
			cc.result.Objs = append(cc.result.Objs, obj)
		}
	}
	return nil
}

func (cc *CC2) candidatesInNode2(node, reqNode ast.Node) error {
	//// help the checker
	//_, err := cc.res.ResolveType(node)
	//if err != nil {
	//	return err
	//}

	astFile, err := cc.conf.PosAstFile(node.Pos())
	if err != nil {
		return err
	}

	// find name
	name := ""
	path, _ := astutil.PathEnclosingInterval(astFile, node.Pos(), node.Pos())
	switch path[0].(type) {
	case *ast.StructType:
		switch t2 := path[1].(type) {
		case *ast.TypeSpec:
			name = t2.Name.Name
		}
	}

	if name == "" {
		u := []string{}
		for _, n := range path {
			u = append(u, fmt.Sprintf("%T", n))
		}
		return fmt.Errorf("unable to get name from node path: %v", u)
	}

	// scope to search for the name
	scope, err := cc.conf.PosAstFileScope(node.Pos())
	if err != nil {
		return err
	}

	// innermost scope
	scope2 := scope.Innermost(node.Pos())
	if scope2 == nil {
		return fmt.Errorf("innermost scope not found")
	}

	// lookup name
	_, obj := scope2.LookupParent(name, node.Pos())
	if obj == nil {
		return fmt.Errorf("object not found: %v", name)
	}

	// exported
	wantNotExp := cc.sameDir(node, reqNode)

	typ := obj.Type()

	// fields
	switch t := typ.Underlying().(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			vobj := t.Field(i)
			if wantNotExp || vobj.Exported() {
				cc.result.Objs = append(cc.result.Objs, vobj)
			}
		}
	}

	// methods
	// using a pointer seems to include pointer and non-pointer methods, otherwise it includes only non-pointer or if iterating through both it can include duplicates.
	t := types.NewPointer(typ)
	mset := types.NewMethodSet(t)
	for i := 0; i < mset.Len(); i++ {
		obj := mset.At(i).Obj()
		if wantNotExp || obj.Exported() {
			cc.result.Objs = append(cc.result.Objs, obj)
		}
	}

	return nil
}

//------------

func (cc *CC2) candidatesInLastArg(ce *ast.CallExpr) error {
	tn, err := cc.res.ResolveType(ce.Fun)
	if err != nil {
		return err
	}

	ft, ok := tn.(*ast.FuncType)
	if !ok {
		return fmt.Errorf("resolved type not a functype")
	}

	// no params
	if len(ft.Params.List) == 0 {
		return nil
	}

	l := len(ft.Params.List)
	field0 := ft.Params.List[l-1]

	tn2, err := cc.res.ResolveType(field0.Type)
	if err != nil {
		return err
	}

	switch tn2.(type) {
	case *ast.InterfaceType:
		//cc.filter.types = false
		return cc.candidatesInScopeUp(ce)
	default:
		_ = fmt.Sprintf("%T", tn2)
	}
	return nil
}

//------------

func (cc *CC2) candidatesInScopeLevel(node ast.Node, reqNode ast.Node) error {

	// want not exported (private) entries if on same path
	wantNotExp := cc.sameDir(node, reqNode)

	scope, err := cc.conf.PosAstFileScope(node.Pos())
	if err != nil {
		return err
	}

	scope = scope.Innermost(node.Pos())
	if scope == nil {
		return fmt.Errorf("failed to get innermost scope")
	}

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj != nil && (wantNotExp || obj.Exported()) {
			cc.result.Objs = append(cc.result.Objs, obj)
		}
	}
	return nil
}

func (cc *CC2) candidatesInScopeUp(node ast.Node) error {
	scope, err := cc.conf.PosAstFileScope(node.Pos())
	if err != nil {
		return err
	}

	// go to innermost scope and search going up in scopes
	seen := make(map[string]bool)
	for s := scope.Innermost(node.Pos()); s != nil; s = s.Parent() {
		for _, n := range s.Names() {
			if seen[n] {
				continue
			}
			seen[n] = true
			obj := s.Lookup(n)
			cc.result.Objs = append(cc.result.Objs, obj)
		}
	}
	return nil
}

//------------

func (cc *CC2) objComment(obj types.Object) (string, bool) {
	path, _, ok := cc.conf.PosAstPath(obj.Pos())
	if !ok {
		return "", false
	}
	if len(path) < 2 {
		return "", false
	}

	s := ""
	switch t := path[1].(type) {
	case *ast.FuncDecl:
		s = t.Doc.Text()
	case *ast.TypeSpec:
		s = t.Doc.Text()
	case *ast.ImportSpec:
		s = t.Doc.Text()
	case *ast.ValueSpec:
		s = t.Doc.Text()
	default:
		log.Printf("todo: %T", t)
		return "", false
	}

	s = strings.TrimRight(s, "\n")
	if len(s) == 0 {
		return "", false
	}
	return s, true
}

//------------

func (cc *CC2) filterCandidates(str string) {
	type entry struct {
		obj   types.Object
		index int    // index where str starts in the object name
		pos   [2]int // position {start,end} of str in the object name
	}

	var entries []entry
	strLow := strings.ToLower(str)
	for _, obj := range cc.result.Objs {

		_ = obj.Type()

		name := obj.Name()
		nameLow := strings.ToLower(name)
		i1 := strings.Index(nameLow, strLow)
		if i1 >= 0 {
			pos := [2]int{i1, i1 + len(strLow)}
			entries = append(entries, entry{obj, i1, pos})
		}
	}

	sort.Slice(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[b]
		if ea.index == eb.index {
			na, nb := ea.obj.Name(), eb.obj.Name()
			return na < nb
		}
		return ea.index < eb.index
	})

	var objs []types.Object
	var positions [][2]int
	for _, e := range entries {
		objs = append(objs, e.obj)
		positions = append(positions, e.pos)
	}
	cc.result.Objs = objs
	cc.result.Segments = positions
}

//------------

func FormatResult(res *CCResult) {
	var u []string
	for _, o := range res.Objs {
		var buf bytes.Buffer
		buf.WriteString(o.String())
		u = append(u, buf.String())
	}
	res.Str = strings.Join(u, "\n")
}

func FormatResult2(res *CCResult, showArgs bool) {
	var buf bytes.Buffer
	ws := buf.WriteString

	// set string at the end
	defer func() {
		res.Str = buf.String()
	}()

	for i, o := range res.Objs {
		if i > 0 {
			ws("\n")
		}

		wsseg := func(s string) {
			l := buf.Len()
			if res.Segments != nil {
				seg := &res.Segments[i]
				seg[0] += l
				seg[1] += l
			}
			ws(s)
		}

		switch t := o.(type) {
		case *types.Const:
			wsseg(t.Name())
		case *types.Var:
			wsseg(t.Name())
		case *types.Builtin:
			wsseg(t.Name())
			ws("(...)")
		case *types.PkgName:
			wsseg(t.Name())
		case *types.Nil:
			wsseg(t.Name())
		case *types.TypeName:
			wsseg(t.Name())
		case *types.Func:
			wsseg(t.Name())
			switch t2 := t.Type().(type) {
			case *types.Signature:
				ws("(")
				if tuple := t2.Params(); tuple != nil {
					if tuple.Len() > 0 {
						if !showArgs {
							ws("...")
						} else {
							w := []string{}
							for i := 0; i < tuple.Len(); i++ {
								v := tuple.At(i)
								u := v.Name() + " " + v.Type().String()
								w = append(w, u)
							}
							ws(strings.Join(w, ", "))
						}
					}
				}
				ws(")")
			default:
			}
		default:
		}
	}
}

//------------

//------------

func (cc *CC2) sameDir(n1, n2 ast.Node) bool {
	dir1, err := cc.conf.PosPkgDir(n1.Pos())
	if err != nil {
		return false
	}
	dir2, err := cc.conf.PosPkgDir(n2.Pos())
	if err != nil {
		return false
	}
	return dir1 == dir2
}
