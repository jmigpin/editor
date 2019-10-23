package godebug

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"golang.org/x/tools/go/ast/astutil"
)

const debugPkgPath = "github.com/jmigpin/editor/core/godebug/debug"
const GoDebugConfigPkgPath = "example.com/godebugconfig"

//----------

type AnnotatorSet struct {
	FSet           *token.FileSet
	debugPkgName   string
	debugVarPrefix string
	testFilesPkgs  map[string]string // map[dir]pkgname
	InsertedExitIn struct {
		Main     bool
		TestMain bool
	}
	fdata struct {
		sync.Mutex
		m     map[string]*debug.AnnotatorFileData // map[filename]afd
		a     []*debug.AnnotatorFileData          // ordered
		index int                                 // counter for new files
	}
}

func NewAnnotatorSet() *AnnotatorSet {
	annset := &AnnotatorSet{
		FSet:          token.NewFileSet(),
		testFilesPkgs: make(map[string]string),
	}
	annset.fdata.m = make(map[string]*debug.AnnotatorFileData)
	annset.debugPkgName = "d" + string(rune(931)) // uncommon  rune to avoid clashes
	annset.debugVarPrefix = annset.debugPkgName   // will have integer appended
	return annset
}

//----------

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, typ AnnotationType) error {
	filename, err := goutil.AstFileFilename(astFile, annset.FSet)
	if err != nil {
		return err
	}

	// TODO: slows down performance, extra file read
	srcb, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	afd := annset.annotatorFileData(filename, srcb)

	ann := NewAnnotator(annset.FSet)
	ann.debugPkgName = annset.debugPkgName
	ann.debugVarPrefix = annset.debugVarPrefix
	ann.fileIndex = afd.FileIndex

	ann.AnnotateAstFile(astFile, typ)

	// n debug stmts inserted
	afd.DebugLen = ann.debugIndex

	// insert imports if debug stmts were inserted
	if ann.builtDebugLineStmt {
		annset.insertImportDebug(astFile)

		// insert in all files to ensure inner init function runs
		annset.insertImport(astFile, "_", GoDebugConfigPkgPath)

		// insert exit in main
		ok := annset.insertDebugExitInFunction(astFile, "main")
		if !annset.InsertedExitIn.Main {
			annset.InsertedExitIn.Main = ok
		}

		// insert exit in testmain
		ok = annset.insertDebugExitInFunction(astFile, "TestMain")
		if !annset.InsertedExitIn.TestMain {
			annset.InsertedExitIn.TestMain = ok
		}

		// keep test files package names in case of need to build testmain files
		annset.keepTestPackage(filename, astFile)
	}

	return nil
}

//----------

func (annset *AnnotatorSet) insertDebugExitInFunction(astFile *ast.File, name string) bool {

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
				X:   ast.NewIdent(annset.debugPkgName),
				Sel: ast.NewIdent("ExitServer"),
			},
		},
	}

	// insert as first stmt
	fd.Body.List = append([]ast.Stmt{stmt1}, fd.Body.List...)

	return true
}

//----------

func (annset *AnnotatorSet) insertImportDebug(astFile *ast.File) {
	annset.insertImport(astFile, annset.debugPkgName, debugPkgPath)
}

func (annset *AnnotatorSet) insertImport(astFile *ast.File, name, path string) {
	astutil.AddNamedImport(annset.FSet, astFile, name, path)
}

//----------

func (annset *AnnotatorSet) annotatorFileData(filename string, src []byte) *debug.AnnotatorFileData {
	annset.fdata.Lock()
	defer annset.fdata.Unlock()
	afd, ok := annset.fdata.m[filename]
	if !ok {
		afd = &debug.AnnotatorFileData{
			FileIndex: annset.fdata.index,
			Filename:  filename,
			FileHash:  sourceHash(src),
			FileSize:  len(src),
		}
		annset.fdata.m[filename] = afd
		annset.fdata.a = append(annset.fdata.a, afd) // keep order
		annset.fdata.index++
	}
	return afd
}

//----------

func (annset *AnnotatorSet) Print(w io.Writer, astFile *ast.File) error {
	// TODO: without tabwidth set, it won't output the source correctly

	// print with source positions from original file
	cfg := &printer.Config{Tabwidth: 4, Mode: printer.SourcePos}
	return cfg.Fprint(w, annset.FSet, astFile)
}

//----------

func (annset *AnnotatorSet) ConfigContent() string {
	entriesStr := annset.buildConfigContentEntries()
	src := `package godebugconfig
import "` + debugPkgPath + `"
func init(){
	debug.ServerNetwork="` + debug.ServerNetwork + `"
	debug.ServerAddress="` + debug.ServerAddress + `"
	debug.AnnotatorFilesData = []*debug.AnnotatorFileData{
		` + entriesStr + `
	}
	debug.StartServer()
}
	`
	return src
}

func (annset *AnnotatorSet) buildConfigContentEntries() string {
	// build map data
	var u []string
	for _, afd := range annset.fdata.a {
		// sanity check
		if afd.FileIndex >= len(annset.fdata.m) {
			panic(fmt.Sprintf("file index doesn't fit map len: %v vs %v", afd.FileIndex, len(annset.fdata.m)))
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
	return strings.Join(u, "\n")
}

//----------

func (annset *AnnotatorSet) ConfigGoModuleContent() string {
	return "module " + GoDebugConfigPkgPath + "\n"
}

//----------

func (annset *AnnotatorSet) TestMainSources() []*TestMainSrc {
	u := []*TestMainSrc{}
	for dir, pkgName := range annset.testFilesPkgs {
		src := annset.testMainSource(pkgName)
		v := &TestMainSrc{Dir: dir, Src: src}
		u = append(u, v)
	}
	return u
}

func (annset *AnnotatorSet) testMainSource(pkgName string) string {
	return `		
package ` + pkgName + `
import ` + annset.debugPkgName + ` "` + debugPkgPath + `"
import "testing"
import "os"
func TestMain(m *testing.M) {
	var code int
	defer func(){ os.Exit(code) }()
	defer ` + annset.debugPkgName + `.ExitServer()
	code = m.Run()
}
	`
}

//----------

func (annset *AnnotatorSet) keepTestPackage(filename string, astFile *ast.File) {
	isTest := strings.HasSuffix(filename, "_test.go")
	if isTest {
		// keep one pkg name per dir
		dir := filepath.Dir(filename)
		annset.testFilesPkgs[dir] = astFile.Name.Name // pkg name
	}
}

//----------

func sourceHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

//----------

type TestMainSrc struct {
	Dir string
	Src string
}

//----------

// Src can be nil.
func ParseAnnotateFileSrc(ann *Annotator, filename string, src interface{}, typ AnnotationType) (*ast.File, error) {
	mode := parser.ParseComments // to support cgo directives on imports
	astFile, err := parser.ParseFile(ann.fset, filename, src, mode)
	if err != nil {
		return nil, err
	}
	ann.AnnotateAstFile(astFile, typ)
	return astFile, nil
}

// Src can be nil.
func ParseAnnotateFileSrcForAnnSet(annset *AnnotatorSet, filename string, src interface{}) (*ast.File, error) {
	mode := parser.ParseComments // to support cgo directives on imports
	astFile, err := parser.ParseFile(annset.FSet, filename, src, mode)
	if err != nil {
		return nil, err
	}
	err = annset.AnnotateAstFile(astFile, AnnotationTypeFile)
	if err != nil {
		return nil, err
	}
	return astFile, nil
}

//----------

func GoDebugConfigFilepathName(name string) string {
	p := filepath.FromSlash(GoDebugConfigPkgPath)
	return filepath.Join(p, name)
}
