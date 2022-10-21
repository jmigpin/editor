package godebug

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"golang.org/x/tools/go/ast/astutil"
)

// TODO: a built binary will only be able to run with this editor instance (but on the other side, it can self debug any part of the editor, including the debug pkg)
var genEncoderId string // used to build config.go

func init() {
	genEncoderId = genDigitsStr(10)
	debug.RegisterStructsForEncodeDecode(genEncoderId)
}

//----------

type AnnotatorSet struct {
	fset           *token.FileSet
	debugPkgName   string
	debugVarPrefix string
	afds           struct {
		sync.Mutex
		m     map[string]*debug.AnnotatorFileData // map[filename]afd
		order []*debug.AnnotatorFileData          // ordered
		index int                                 // counter for new files
	}
}

func NewAnnotatorSet(fset *token.FileSet) *AnnotatorSet {
	annset := &AnnotatorSet{}
	annset.fset = fset
	annset.afds.m = map[string]*debug.AnnotatorFileData{}
	annset.debugPkgName = "Σ" // uncommon rune to avoid clashes
	annset.debugVarPrefix = "Σ"
	return annset
}

//----------

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, ti *types.Info, nat map[ast.Node]AnnotationType) error {

	filename, err := nodeFilename(annset.fset, astFile)
	if err != nil {
		return err
	}

	afd, err := annset.annotatorFileData(filename)
	if err != nil {
		return err
	}

	ann := NewAnnotator(annset.fset)
	ann.debugPkgName = annset.debugPkgName
	ann.debugVarPrefix = annset.debugVarPrefix
	ann.fileIndex = afd.FileIndex
	ann.typesInfo = ti
	ann.nodeAnnTypes = nat
	ann.AnnotateAstFile(astFile)

	// n debug stmts inserted
	afd.DebugLen = ann.debugLastIndex

	// insert imports if debug stmts were inserted
	if ann.builtDebugLineStmt {
		annset.insertDebugPkgImport(astFile)
		annset.updateOsExitCalls(astFile)
	}
	return nil
}

//----------

func (annset *AnnotatorSet) setupDebugExitInFuncDecl(fd *ast.FuncDecl, astFile *ast.File) {
	// defer exit stmt
	stmt1 := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(annset.debugPkgName),
				Sel: ast.NewIdent("ExitServer"),
			},
		},
	}

	//// initial call to force server start in case of an empty main
	//stmt2 := &ast.ExprStmt{
	//	X: &ast.CallExpr{
	//		Fun: &ast.SelectorExpr{
	//			X:   ast.NewIdent(annset.debugPkgName),
	//			Sel: ast.NewIdent("StartServer"),
	//		},
	//	},
	//}

	// insert as first stmts
	stmts := []ast.Stmt{stmt1}
	//stmts:=[]ast.Stmt{stmt1, stmt2}
	fd.Body.List = append(stmts, fd.Body.List...)

	annset.insertDebugPkgImport(astFile)
	annset.updateOsExitCalls(astFile)
}

//----------

func (annset *AnnotatorSet) insertDebugPkgImport(astFile *ast.File) {
	// adds import if absent
	_ = astutil.AddNamedImport(annset.fset, astFile, annset.debugPkgName, debugPkgPath)
}

//----------

func (annset *AnnotatorSet) updateOsExitCalls(astFile *ast.File) {
	// check if "os" is imported
	osImported := false
	for _, imp := range astFile.Imports {
		v, err := strconv.Unquote(imp.Path.Value)
		if err == nil && v == "os" {
			if imp.Name == nil { // must not have been named
				osImported = true
			}
			break
		}
	}
	if !osImported {
		return
	}

	// replace os.Exit() calls with debug.Exit()
	// count other os.* calls to know if the "os" import should be removed
	count := 0
	_ = astutil.Apply(astFile, func(c *astutil.Cursor) bool {
		if se, ok := c.Node().(*ast.SelectorExpr); ok {
			if id1, ok := se.X.(*ast.Ident); ok {
				if id1.Name == "os" {
					count++
					if se.Sel.Name == "Exit" {
						count--
						se2 := &ast.SelectorExpr{
							X:   ast.NewIdent(annset.debugPkgName),
							Sel: ast.NewIdent("Exit"),
						}
						c.Replace(se2)
					}
				}
			}
		}
		return true
	}, nil)

	// if !astutil.UsesImport(astFile, "os"){
	if count == 0 {
		_ = astutil.DeleteImport(annset.fset, astFile, "os")

		// TODO: waiting for fix: vet: ./testmain64531_test.go:3:10: could not import os (can't resolve import "")
		// https://github.com/golang/go/issues/50044
		// https://github.com/golang/go/issues/44957
		// ensure "os" is imported as "_"
		_ = astutil.AddNamedImport(annset.fset, astFile, "_", "os")
	}
}

//----------

func (annset *AnnotatorSet) insertTestMain(astFile *ast.File) error {
	// TODO: detect if used imports are already imported with another name (os,testing)

	astutil.AddImport(annset.fset, astFile, "os")
	astutil.AddImport(annset.fset, astFile, "testing")

	// build ast to insert (easier to parse from text then to build the ast manually here. notice how "imports" are missing since it is just to get the ast of the funcdecl)
	src := `
		package main		
		func TestMain(m *testing.M) {
			os.Exit(m.Run())
		}
	`
	fset := token.NewFileSet()
	astFile2, err := parser.ParseFile(fset, "a.go", src, 0)
	if err != nil {
		panic(err)
	}
	//goutil.PrintNode(fset, astFile2)

	// get the only func decl for insertion
	fd := (*ast.FuncDecl)(nil)
	ast.Inspect(astFile2, func(n ast.Node) bool {
		if n2, ok := n.(*ast.FuncDecl); ok {
			fd = n2
			return false
		}
		return true
	})
	if fd == nil {
		err := fmt.Errorf("missing func decl")
		panic(err)
	}

	// insert in ast file
	astFile.Decls = append(astFile.Decls, fd)

	// DEBUG
	//goutil.PrintNode(fa.cmd.fset, astFile)

	annset.setupDebugExitInFuncDecl(fd, astFile)

	return nil
}

//----------

func (annset *AnnotatorSet) annotatorFileData(filename string) (*debug.AnnotatorFileData, error) {
	annset.afds.Lock()
	defer annset.afds.Unlock()

	afd, ok := annset.afds.m[filename]
	if ok {
		return afd, nil
	}

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("annotatorfiledata: %w", err)
	}

	// create new afd
	afd = &debug.AnnotatorFileData{
		FileIndex: annset.afds.index,
		Filename:  filename,
		FileSize:  len(src),
		FileHash:  sourceHash(src),
	}
	annset.afds.m[filename] = afd

	annset.afds.order = append(annset.afds.order, afd) // keep order
	annset.afds.index++

	return afd, nil
}

//----------

func (annset *AnnotatorSet) BuildConfigSrc(serverNetwork, serverAddr string, flags *flags) []byte {
	acceptOnlyFirstClient := flags.mode.run || flags.mode.test
	aofc := strconv.FormatBool(acceptOnlyFirstClient)
	bcce := annset.buildConfigAfdEntries()

	src := `
package debug
func init(){
	ServerNetwork = "` + serverNetwork + `"
	ServerAddress = "` + serverAddr + `"
	
	hasGenConfig = true
	encoderId = "` + genEncoderId + `"
	syncSend = ` + strconv.FormatBool(flags.syncSend) + `
	acceptOnlyFirstClient = ` + aofc + `
	stringifyBytesRunes = ` + strconv.FormatBool(flags.stringifyBytesRunes) + `
	hasSrcLines = ` + strconv.FormatBool(flags.srcLines) + `
	annotatorFilesData = []*AnnotatorFileData{` + bcce + `}
}
`
	return []byte(src)
}

func (annset *AnnotatorSet) buildConfigAfdEntries() string {
	u := []string{}
	for _, afd := range annset.afds.order {
		s := fmt.Sprintf("&AnnotatorFileData{%v,%v,%q,%v,[]byte(%q)}",
			afd.FileIndex,
			afd.DebugLen,
			afd.Filename,
			afd.FileSize,
			string(afd.FileHash),
		)
		u = append(u, s)
	}
	return strings.Join(u, ",")
}

//----------
//----------
//----------

func findFuncDeclWithBody(astFile *ast.File, name string) (*ast.FuncDecl, bool) {
	// commented: in the case of the main func being inserted in the ast, it would not be available in the scope lookup
	//obj := astFile.Scope.Lookup(name)
	//if obj != nil && obj.Kind == ast.Fun {
	//	fd, ok := obj.Decl.(*ast.FuncDecl)
	//	if ok && fd.Body != nil {
	//		return fd, true
	//	}
	//}
	//return nil, false

	fd := (*ast.FuncDecl)(nil)
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.File:
			return true
		case *ast.FuncDecl:
			if t.Name.Name == name && t.Body != nil {
				fd = t
			}
			return false
		default:
			return false
		}
	})
	return fd, fd != nil
}

//----------

func sourceHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}
