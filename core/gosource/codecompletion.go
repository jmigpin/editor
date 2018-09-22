package gosource

import (
	"bytes"
	"fmt"
	"go/ast"
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

	//// show comment if the first result is matching
	//comment := ""
	//if len(res.Objs) > 0 {
	//	obj := res.Objs[0]
	//	seg := res.Segments[0]
	//	if len(res.Objs) == 1 || (seg[0] == 0 && seg[1] == len(obj.Name())) {
	//		s, ok := cc.objComment(obj)
	//		if ok {
	//			comment = "\n\n" + s
	//		}
	//	}
	//}

	// TODO: check args types
	showArgs := len(res.Objs) < 3

	FormatResult2(res, showArgs) // alters res.Str and res.Segments

	//res.Str += comment

	return res, err
}

//------------

type CC2 struct {
	astFile *ast.File
	conf    *Config
	res     *DeclResolver
	result  *CCResult

	filter struct {
		str     string
		typeStr string
		typeObj types.Object
	}
}

func (cc *CC2) run(filename string, src interface{}, index int) (*CCResult, error) {
	cc.result = &CCResult{StartIndex: index}
	cc.conf = NewConfig()
	//cc.conf.ParserMode = parser.ParseComments

	// insert semicolon to improve code completion
	b, err := ReadSource(filename, src)
	if err != nil {
		return nil, err
	}
	src2 := string(b)
	//src2 := InsertSemicolon(string(b), index)
	//src2 := InsertSemicolonAfterIdent(string(b), index)
	//j, src2 := BackTrackSpaceInsertSemicolon(string(b), index)
	//index -= j

	// parse and keep astfile
	astFile, err, ok := cc.conf.ParseFile(filename, src2, 0)
	if !ok {
		return nil, err
	}
	cc.astFile = astFile

	// DEBUG
	//PrintInspect(cc.astFile)

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

	if err := cc.candidatesInPos(ipos, src2); err != nil {
		return nil, err
	}

	cc.filterCandidates()

	return cc.result, nil
}

//------------

func (cc *CC2) candidatesInPos(pos token.Pos, src string) error {
	// path in astFile
	path, _ := astutil.PathEnclosingInterval(cc.astFile, pos, pos)
	if len(path) <= 1 {
		return fmt.Errorf("path len: %v", len(path))
	}
	node := path[0]

	//// DEBUG
	//p := cc.conf.FSet.Position(pos)
	//_, _ = TokPositionStr(src, p)

	switch t := node.(type) {
	case *ast.Ident:
		return cc.candidatesInIdent(t, pos)
	default:
		return cc.candidatesInPrevPos(pos)
	}

	//return fmt.Errorf("unable to get candidates in pos")
}

func (cc *CC2) candidatesInPrevPos(pos token.Pos) error {
	// path in astFile
	path, _ := astutil.PathEnclosingInterval(cc.astFile, pos-1, pos-1)
	if len(path) <= 1 {
		return fmt.Errorf("path len: %v", len(path))
	}
	node := path[0]

	switch t := node.(type) {
	case *ast.ReturnStmt,
		*ast.AssignStmt,
		*ast.StarExpr,
		*ast.BadExpr:
		return cc.candidatesInScopeUp(t.Pos())
	case *ast.BlockStmt:
		return nil
	case *ast.Ident:
		return cc.candidatesInIdent(t, pos)
	case *ast.SelectorExpr:
		if pos-1 == t.X.End() {
			return cc.candidatesInsideNode(t.X)
		}
	case *ast.CallExpr:
		// ex: fmt.Print()●
		if pos-1 == t.Rparen {
			return nil
		}
		return cc.candidatesInScopeUp(t.Pos())
	default:
		log.Printf("todo: %T", node)
	}

	return fmt.Errorf("unable to get candidates in previous pos")
}

func (cc *CC2) setStartIndexFromNode(node ast.Node) {
	cc.result.StartIndex = cc.conf.FSet.Position(node.Pos()).Offset
}
func (cc *CC2) setFilterStrFromIdent(id *ast.Ident, pos token.Pos) {
	cc.filter.str = id.Name[:pos-id.Pos()]
}

//------------

func posInNode(pos token.Pos, node ast.Node) bool {
	return pos >= node.Pos() && pos <= node.End()
}

//------------

func (cc *CC2) candidatesInIdent(id *ast.Ident, pos token.Pos) error {
	cc.setStartIndexFromNode(id)
	cc.setFilterStrFromIdent(id, pos)

	path, _ := astutil.PathEnclosingInterval(cc.astFile, id.Pos(), id.Pos())
	if len(path) <= 1 {
		return nil
	}
	// parent node
	switch t := path[1].(type) {
	case *ast.SelectorExpr:
		if id == t.Sel {
			return cc.candidatesInsideNode(t.X)
		}
	default:
	}

	return cc.candidatesInScopeUp(pos)
	//return fmt.Errorf("unable to get candidates in ident")
}

//------------

func (cc *CC2) candidatesInsideNode(node ast.Node) error {
	nt, err := cc.res.ResolveType(node)
	if err != nil {
		return err
	}
	switch t := nt.(type) {
	case *ast.ImportSpec:
		return cc.candidatesInImportSpec(t, node)
	case *ast.TypeSpec:
		return cc.candidatesInType(t, node)
	}
	return nil
}

//------------

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

//------------

func (cc *CC2) candidatesInArg(fun ast.Expr, index int) error {
	nt, err := cc.res.ResolveType(fun)
	if err != nil {
		return err
	}
	switch t := nt.(type) {
	case *ast.Ident:
		path := cc.conf.SurePosAstPath(t.Pos())
		for _, p := range path {
			switch p.(type) {
			}
		}

		//case *ast.FuncType:
		//	// index argument
		//	j := 0
		//	for _, f := range t.Params.List {
		//		for range f.Names {
		//			if j == index {
		//				// argument type
		//				return cc.candidatesOfType(f.Type, fun)
		//			}
		//			j++
		//		}
		//	}
	}
	return nil
}

//------------

//func (cc *CC2) candidatesOfType(node, reqNode ast.Node) error {
//	nt, err := cc.res.ResolveType(node)
//	if err != nil {
//		return err
//	}

//	switch t := nt.(type) {
//	case *Builtin:
//		// scope to make search
//		scope, err := cc.conf.PosInnermostScope(node.Pos())
//		if err != nil {
//			return err
//		}
//		_, obj := scope.LookupParent(t.Name, 0)
//		if obj != nil {
//			cc.filter.typeObj = obj
//		}

//		return cc.candidatesInScopeUp(reqNode)

//	case *ast.InterfaceType:
//		// astfile to find name
//		astFile, err := cc.conf.PosAstFile(t.Pos())
//		if err != nil {
//			return err
//		}

//		// find type object
//		path, _ := astutil.PathEnclosingInterval(astFile, t.Pos(), t.Pos())
//		switch t2 := path[1].(type) {
//		case *ast.Ellipsis:
//		case *ast.TypeSpec:
//			obj := cc.conf.Info.ObjectOf(t2.Name)
//			cc.filter.typeObj = obj
//		}

//		return cc.candidatesInScopeUp(reqNode)

//	case *ast.TypeSpec:
//		if obj := cc.conf.Info.ObjectOf(t.Name); obj != nil {
//			cc.filter.typeObj = obj
//		}
//		return cc.candidatesInScopeUp(reqNode)
//	}
//	return nil
//}

//------------

func (cc *CC2) candidatesInType(node, reqNode ast.Node) error {
	// astfile to find name
	astFile, err := cc.conf.PosAstFile(node.Pos())
	if err != nil {
		return err
	}

	// find name
	name := ""
	path, _ := astutil.PathEnclosingInterval(astFile, node.Pos(), node.Pos())
	switch t2 := path[1].(type) {
	case *ast.TypeSpec:
		name = t2.Name.Name
	}

	//switch path[0].(type) {
	//case *ast.InterfaceType:
	//	switch t2 := path[1].(type) {
	//	}
	//case *ast.StructType:
	//	switch t2 := path[1].(type) {
	//	case *ast.TypeSpec:
	//		name = t2.Name.Name
	//	}
	//}

	if name == "" {
		return fmt.Errorf("unable to get name from node path")
	}

	// scope to search for the name
	scope, ok := cc.conf.Info.Scopes[astFile]
	if !ok {
		return fmt.Errorf("scope not found in info")
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

		// methods
		// using a pointer seems to include pointer and non-pointer methods, otherwise it includes only non-pointer or if iterating through both it can include duplicates.
		t2 := types.NewPointer(typ)
		mset := types.NewMethodSet(t2)
		for i := 0; i < mset.Len(); i++ {
			obj := mset.At(i).Obj()
			if wantNotExp || obj.Exported() {
				cc.result.Objs = append(cc.result.Objs, obj)
			}
		}

	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			obj := t.Method(i)
			if wantNotExp || obj.Exported() {
				cc.result.Objs = append(cc.result.Objs, obj)
			}
		}
	}

	return nil
}

//------------

func (cc *CC2) candidatesInScopeUp(pos token.Pos) error {
	scope, err := cc.conf.PosInnermostScope(pos)
	if err != nil {
		return err
	}
	// search going up in scopes
	seen := make(map[string]bool)
	for s := scope; s != nil; s = s.Parent() {
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
//------------
//------------
//------------

//func (cc *CC2) candidatesInNode(node ast.Node) error {
//	nt, err := cc.res.ResolveType(node)
//	if err != nil {
//		return err
//	}
//	switch t := nt.(type) {
//	case *ast.ImportSpec:
//		return cc.candidatesInImportSpec(t, node)
//	case *ast.StructType:
//		return cc.candidatesInNode2(t, node)
//	}
//	return fmt.Errorf("todo")
//}

//func (cc *CC2) candidatesInNode2(node, reqNode ast.Node) error {
//	//// help the checker
//	//_, err := cc.res.ResolveType(node)
//	//if err != nil {
//	//	return err
//	//}

//	astFile, err := cc.conf.PosAstFile(node.Pos())
//	if err != nil {
//		return err
//	}

//	// find name
//	name := ""
//	path, _ := astutil.PathEnclosingInterval(astFile, node.Pos(), node.Pos())
//	switch path[0].(type) {
//	case *ast.StructType:
//		switch t2 := path[1].(type) {
//		case *ast.TypeSpec:
//			name = t2.Name.Name
//		}
//	}

//	if name == "" {
//		u := []string{}
//		for _, n := range path {
//			u = append(u, fmt.Sprintf("%T", n))
//		}
//		return fmt.Errorf("unable to get name from node path: %v", u)
//	}

//	// scope to search for the name
//	scope, err := cc.conf.PosInnermostScope(node.Pos())
//	if err != nil {
//		return err
//	}

//	// lookup name
//	_, obj := scope.LookupParent(name, node.Pos())
//	if obj == nil {
//		return fmt.Errorf("object not found: %v", name)
//	}

//	// exported
//	wantNotExp := cc.sameDir(node, reqNode)

//	typ := obj.Type()

//	// fields
//	switch t := typ.Underlying().(type) {
//	case *types.Struct:
//		for i := 0; i < t.NumFields(); i++ {
//			vobj := t.Field(i)
//			if wantNotExp || vobj.Exported() {
//				cc.result.Objs = append(cc.result.Objs, vobj)
//			}
//		}
//	}

//	// methods
//	// using a pointer seems to include pointer and non-pointer methods, otherwise it includes only non-pointer or if iterating through both it can include duplicates.
//	t := types.NewPointer(typ)
//	mset := types.NewMethodSet(t)
//	for i := 0; i < mset.Len(); i++ {
//		obj := mset.At(i).Obj()
//		if wantNotExp || obj.Exported() {
//			cc.result.Objs = append(cc.result.Objs, obj)
//		}
//	}

//	return nil
//}

//------------

//func (cc *CC2) candidatesInScopeLevel(node ast.Node, reqNode ast.Node) error {

//	// want not exported (private) entries if on same path
//	wantNotExp := cc.sameDir(node, reqNode)

//	scope, err := cc.conf.PosInnermostScope(node.Pos())
//	if err != nil {
//		return err
//	}

//	for _, name := range scope.Names() {
//		obj := scope.Lookup(name)
//		if obj != nil && (wantNotExp || obj.Exported()) {
//			cc.result.Objs = append(cc.result.Objs, obj)
//		}
//	}
//	return nil
//}

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

func (cc *CC2) filterCandidates() {
	type entry struct {
		obj   types.Object
		index int    // index where str starts in the object name
		pos   [2]int // position {start,end} of str in the object name
	}

	var entries []entry
	strLow := strings.ToLower(cc.filter.str)
	for _, obj := range cc.result.Objs {

		// filter objects of this type
		if cc.filter.typeObj != nil {
			if obj.Type() != cc.filter.typeObj.Type() {
				continue
			}
			// don't use the type itself
			if obj == cc.filter.typeObj {
				continue
			}
		}

		// filter name
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
