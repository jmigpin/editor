package godebug

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"golang.org/x/tools/go/ast/astutil"
)

const debugPkgPath = "github.com/jmigpin/editor/core/godebug/debug"

type AnnotatorSet struct {
	FSet           *token.FileSet
	debugPkgName   string
	debugVarPrefix string
	//simpleOut      bool
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

func (annset *AnnotatorSet) AnnotateFile(filename string, src interface{}) (*ast.File, error) {
	srcb, err := goutil.ReadSource(filename, src)
	if err != nil {
		return nil, err
	}

	afd := annset.annotatorFileData(filename, srcb)

	ann := &Annotator{
		debugPkgName:   annset.debugPkgName,
		debugVarPrefix: annset.debugVarPrefix,
		fileIndex:      afd.FileIndex,
		fset:           annset.FSet,
	}

	astFile, err := ann.ParseAnnotateFile(filename, srcb)
	if err != nil {
		return nil, err
	}

	// n debug stmts inserted
	afd.DebugLen = ann.debugIndex

	// insert imports if debug stmts were inserted
	if ann.builtDebugLineStmt {
		annset.insertImportDebug(astFile)

		// insert in all files to ensure inner init function runs
		annset.insertImport(astFile, "_", "godebugconfig")

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

	return astFile, nil
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

	// insert
	//fd.Body.List = sann.insertInStmts(stmt1, 0, fd.Body.List)
	done := false
	pre := func(c *astutil.Cursor) bool {
		if !done && c.Index() >= 0 {
			// insert as first stmt (previous sibling of a node with
			c.InsertBefore(stmt1)
			done = true
		}
		return !done
	}
	astutil.Apply(fd.Body, pre, nil)

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

func (annset *AnnotatorSet) ConfigSource() (string, string) {
	// content
	entriesStr := annset.buildConfigSourceEntries()
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

	pkgFilename := "godebugconfig/config.go"

	return src, pkgFilename
}

func (annset *AnnotatorSet) buildConfigSourceEntries() string {
	// build map data
	var u []string
	for _, afd := range annset.fdata.a {
		logger.Printf("configsource: included file %v", afd.Filename)

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
