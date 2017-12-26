package gosource

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"
)

func CodeCompletion(filename string, src interface{}, index int) (int, []types.Object, error) {
	info := NewInfo()

	// parse main file
	filename = info.AddPathFile(filename)
	astFile := info.ParseFile(filename, src)
	if astFile == nil {
		return 0, nil, fmt.Errorf("unable to parse file")
	}

	if Debug {
		info.PrintIdOffsets(astFile)
	}

	// index node
	tokenFile := info.FSet.File(astFile.Package)
	if tokenFile == nil {
		return 0, nil, fmt.Errorf("unable to get token file")
	}
	// avoid panic from a bad index
	if index > tokenFile.Size() {
		return 0, nil, fmt.Errorf("index bigger than file size")
	}
	indexNode := info.PosNode(tokenFile.Pos(index))

	// must be an id
	id, ok := indexNode.(*ast.Ident)
	if !ok {
		return 0, nil, fmt.Errorf("index not at an id node")
	}

	idPos := info.FSet.Position(id.Pos())

	Logf("id=%v", id)

	cobjs := _getCandidates(info, id)
	objs := _searchCandidates(info, id, index, cobjs)
	return idPos.Offset, objs, nil
}

func _getCandidates(info *Info, id *ast.Ident) []types.Object {
	var objs []types.Object

	astFile := info.PosAstFile(id.Pos())
	path := info.PosFilePath(astFile.Package)

	// resolve
	res := NewResolver(info, path, id)
	_ = res.ResolveType(id)

	//// search scopes
	//if scope, ok := info.Info.Scopes[astFile]; ok {
	//	if s2 := scope.Innermost(id.Pos()); s2 != nil {
	//		for ; s2 != nil; s2 = s2.Parent() {
	//			for _, name := range s2.Names() {
	//				o := s2.Lookup(name)
	//				objs = append(objs, o)
	//			}
	//		}
	//	}
	//}

	// selector of SelectorExpr
	if pn, ok := info.Parents[id]; ok {
		if se, ok := pn.(*ast.SelectorExpr); ok && id == se.Sel {
			//u := res.ResolveType(se.Sel)
			//Logf("TODO 5")
			//Logf("%v", info.FSet.Position(n2.Pos()))
			//Dump(n2)

			n2 := res.ResolveType(se.X)

			switch t2 := n2.(type) {
			case *ast.ImportSpec:
				path2 := res.importSpecPath(t2)
				pkg := res.info.Pkgs[path2]
				if pkg != nil {
					scope := pkg.Scope()
					for _, name := range scope.Names() {
						o := scope.Lookup(name)
						objs = append(objs, o)
					}
				}
			case *ast.StructType:
				// solve each field to help the checker
				for _, f := range t2.Fields.List {
					_ = res.ResolveType(f)
				}

				// TODO: solve gen declarations from all astFiles? need to solve the package to get all the methods

				if tv, ok := info.Info.Types[t2]; ok {
					switch t3 := tv.Type.(type) {
					case *types.Struct:
						for i := 0; i < t3.NumFields(); i++ {
							o := t3.Field(i)
							objs = append(objs, o)
						}
						for _, t4 := range []types.Type{t3, types.NewPointer(t3)} {
							Dump(t4)
							mset := types.NewMethodSet(t4)
							for i := 0; i < mset.Len(); i++ {
								o := mset.At(i).Obj()
								Dump(o)
								objs = append(objs, o)
							}
						}
					}
				}
			default:
				Logf("TODO t2")
				Dump(t2)
			}
		}
	}

	//Logf("TODO pn")
	//Dump(pn)

	return objs
}

func _searchCandidates(info *Info, id *ast.Ident, index int, candidates []types.Object) []types.Object {
	// get id string up to index
	idStr := id.Name
	shortIdStr := ""
	p := info.FSet.Position(id.Pos())
	diff := index - p.Offset
	if diff > 0 {
		shortIdStr = id.Name[:diff]
	}

	Logf("searching for %q (%q)", shortIdStr, idStr)

	shortIdStrLow := strings.ToLower(shortIdStr)

	type entry struct {
		obj    types.Object
		index1 int
		index2 int
	}

	var entries []entry
	for _, obj := range candidates {
		if !obj.Exported() {
			continue
		}

		i1 := strings.Index(obj.Name(), idStr)
		nameLow := strings.ToLower(obj.Name())
		i2 := strings.Index(nameLow, shortIdStrLow)
		if i2 >= 0 {
			entries = append(entries, entry{obj, i1, i2})
		}
	}

	sort.Slice(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[a]
		if ea.index1 >= 0 && eb.index1 >= 0 && ea.index1 < eb.index1 {
			return true
		}
		if ea.index2 < eb.index2 {
			return true
		}
		na, nb := ea.obj.Name(), eb.obj.Name()
		return na < nb
	})

	var objs []types.Object
	for _, e := range entries {
		objs = append(objs, e.obj)
	}

	return objs
}

func FormatObjs__(objs []types.Object) string {
	var u []string
	for _, o := range objs {
		var buf bytes.Buffer
		buf.WriteString(o.String())
		u = append(u, buf.String())
	}
	return strings.Join(u, "\n")
}

func FormatObjs(objs []types.Object) string {
	var u []string
	for _, o := range objs {
		var buf bytes.Buffer
		ws := buf.WriteString

		switch t := o.(type) {
		case *types.Func:
			ws("func ")
			ws(t.Name())
			switch t2 := t.Type().(type) {
			case *types.Signature:
				ws("(")
				w := []string{}
				if tuple := t2.Params(); tuple != nil {
					for i := 0; i < tuple.Len(); i++ {
						v := tuple.At(i)
						w = append(w, v.Name())
						w = append(w, " ")
						w = append(w, v.Type().String())
					}
					ws(strings.Join(w, ","))
				}
				ws(")")
			default:
				Logf("TODO 2")
				Dump(t2)
			}
		case *types.Const:
			ws("const ")
			ws(t.Name())
			//ws(t.Val().String())
		case *types.TypeName:
			ws("type ")
			ws(t.Name())
			switch t2 := t.Type().Underlying().(type) {
			case *types.Interface:
				ws(" interface")
			case *types.Struct:
				ws(" struct")
			case *types.Slice:
				ws(" []")
				//ws(t2.Underlying().String())
			case *types.Basic:
				//ws(t2.Name())
				//ws(" basic")
			default:
				Logf("TODO 3")
				Dump(t2)
			}
		case *types.Var:
			ws("var ")
			ws(t.Name())
			ws(" ")
			ws(t.Type().String())
		case *types.Builtin:
			ws("builtin ")
			ws(t.Name())
			//ws(t.Type().String())
		case *types.PkgName:
			ws("package ")
			ws(t.Name())
		case *types.Nil:
			//ws(t.Name())
			//ws(" <nil>")
		default:
			Logf("TODO 1")
			Dump(o)
			continue
		}

		u = append(u, buf.String())
	}
	return strings.Join(u, "\n")
}
