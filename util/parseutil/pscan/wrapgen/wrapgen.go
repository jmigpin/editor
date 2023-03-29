package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"

	"github.com/jmigpin/editor/util/astut"
)

// var input = "../scmatch.go"
// var output = "../scwrap.go"
var input = "./match.go"
var output = "./wrap.go"

func main() {
	if err := main2(); err != nil {
		fmt.Println(err)
	}
}
func main2() error {
	//src, err := ioutil.ReadFile(input)
	//if err != nil {
	//	return err
	//}
	//fmt.Printf("%s\n", src)

	fset := &token.FileSet{}
	//f, err := parser.ParseFile(fset, input, src, 0)
	f, err := parser.ParseFile(fset, input, nil, 0)
	if err != nil {
		return err
	}
	//fmt.Printf("%s\n", f)

	b, err := build(fset, f)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(output, b, 0o644); err != nil {
		return err
	}
	return nil
}

//----------

func build(fset *token.FileSet, f *ast.File) ([]byte, error) {
	buf := &bytes.Buffer{}

	// header
	//----------START
	buf.WriteString(`package pscan

// WARNING: DO NOT EDIT, THIS FILE WAS AUTO GENERATED

type Wrap struct {
	sc *Scanner
	M  *Match
}

func (w *Wrap) init(sc *Scanner) {
	w.sc = sc
	w.M = sc.M
}
`)
	//----------END

	// methods
	visitFuncs(f, "Match", func(fd *ast.FuncDecl) bool {
		switch fd.Name.Name {
		case "init":
			return true
		}
		//if fd.Type.Results != nil || len(fd.Type.Results.List) != 1 {
		//	return true
		//}
		//resNames := fd.Type.Results.List[0].Names // result
		//if len(resNames) != 1 {
		//	return true
		//}
		//if resNames[0].Name != "error" {
		//	return true
		//}

		params, results, vars := mustSprintFuncType(fset, fd.Type)

		results2 := ""
		switch results {
		//case "Pos, error":
		//	results = fmt.Sprintf("(%s)", results) // wrap
		//	results2 = "MFn"
		//case "byte, Pos, error":
		//	results = fmt.Sprintf("(%s)", results) // wrap
		//	results2 = "MFn"
		//case "any, Pos, error":
		//	results = fmt.Sprintf("(%s)", results) // wrap
		//	results2 = "VFn"

		case "int, error":
			results = fmt.Sprintf("(%s)", results) // wrap
			results2 = "MFn"
		case "byte, int, error":
			results = fmt.Sprintf("(%s)", results) // wrap
			results2 = "MFn"
		case "any, int, error":
			results = fmt.Sprintf("(%s)", results) // wrap
			results2 = "VFn"

		default:
			panic(fmt.Sprintf("type results type: %v (fname=%v)", results, fd.Name.Name))
		}

		sig := fmt.Sprintf("%s(%s) %s", fd.Name.Name, params, results2)

		//----------START
		ret := fmt.Sprintf(`return func(pos int) %s {
		return w.M.%s(%s)
	}`, results, fd.Name.Name, vars)
		if vars == "" {
			ret = fmt.Sprintf("return w.M.%s", fd.Name.Name)
		}
		//----------END

		//----------START
		fstr := fmt.Sprintf(`
func (w *Wrap) %s {
	%s
}`, sig, ret)
		//----------END

		fmt.Fprintf(buf, "%s\n", fstr)
		return true
	})

	return buf.Bytes(), nil
}

func visitFuncs(f *ast.File, recvName string, fn func(fd *ast.FuncDecl) bool) {
	ast.Inspect(f, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.FuncDecl:
			if t.Recv != nil && len(t.Recv.List) == 1 {
				if se, ok := t.Recv.List[0].Type.(*ast.StarExpr); ok {
					if id, ok2 := se.X.(*ast.Ident); ok2 {
						if id.Name == recvName {
							if !fn(t) {
								return false
							}
						}
					}
				}

			}
		}
		return true
	})
}

func mustSprintFuncType(fset *token.FileSet, ft *ast.FuncType) (string, string, string) {
	// parameters
	h := []string{"pos"} // add first arg

	p := []*ast.Field{}
	if ft.Params != nil && len(ft.Params.List) > 0 {
		p = ft.Params.List[1:] // remove first arg ("p pos")
	}

	params := MustSprintNode(fset, p, func(names string) {
		h = append(h, names)
	})

	vars := strings.Join(h, ", ")
	results := MustSprintNode(fset, ft.Results, nil)
	return params, results, vars
}

//----------

func MustSprintNode(fset *token.FileSet, node0 any, fieldNamesFn func(string)) string {

	funcTypeDepth := 0
	fn := (func(node any) string)(nil)
	fn = func(node any) string {
		switch t := node.(type) {
		case *ast.FieldList:
			if t == nil {
				return ""
			}
			return fn(t.List)
		case []*ast.Field:
			w := []string{}
			for _, field := range t {
				s := fn(field)
				w = append(w, s)
			}
			return strings.Join(w, ", ")
		case *ast.Field:
			n := fn(t.Names)
			u := fn(t.Type)
			if funcTypeDepth == 0 && fieldNamesFn != nil {
				s := n
				//fmt.Printf("+++ %T\n", t.Type)
				if _, ok := t.Type.(*ast.Ellipsis); ok {
					s += "..."
				}
				fieldNamesFn(s)
			}
			if n != "" {
				n += " "
			}
			return fmt.Sprintf("%s%s", n, u)
		case []*ast.Ident:
			w := []string{}
			for _, id := range t {
				w = append(w, fn(id))
			}
			return strings.Join(w, ", ")
		case *ast.ArrayType:
			l := fn(t.Len)
			et := fn(t.Elt)
			return fmt.Sprintf("[%s]%s", l, et)
		case *ast.Ellipsis:
			return "..." + fn(t.Elt)
		case nil:
			return ""
		case *ast.FuncType:
			funcTypeDepth++
			params := fn(t.Params)
			results := fn(t.Results)
			funcTypeDepth--
			if results != "" {
				results = " " + results
			}
			return fmt.Sprintf("func(%s)%s", params, results)
		case *ast.StarExpr:
			return fmt.Sprintf("*%s", fn(t.X))

		//----------

		case *ast.Ident:
			s, err := astut.SprintNode2(fset, node)
			if err != nil {
				panic(err)
			}
			return s
		default:
			return fmt.Sprintf("<TODO:%T>", t)
		}
	}
	return fn(node0)
}
