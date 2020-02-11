package godebug

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"golang.org/x/tools/go/ast/astutil"
)

const DebugPkgPath = "github.com/jmigpin/editor/core/godebug/debug"
const GodebugconfigPkgPath = "github.com/jmigpin/editor/core/godebug/godebugconfig"

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
	afds struct { // TODO: rename afds
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
	annset.afds.m = make(map[string]*debug.AnnotatorFileData)
	annset.debugPkgName = "d" + string(rune(931)) // uncommon  rune to avoid clashes
	annset.debugVarPrefix = annset.debugPkgName   // will have integer appended
	return annset
}

//----------

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, filename string, files *Files) error {

	afd, err := annset.annotatorFileData(filename, files)
	if err != nil {
		return err
	}

	ann := NewAnnotator(annset.FSet, files.NodeAnnType)
	ann.debugPkgName = annset.debugPkgName
	ann.debugVarPrefix = annset.debugVarPrefix
	ann.fileIndex = afd.FileIndex

	typ := files.annTypes[filename]
	ann.AnnotateAstFile(astFile, typ)

	// n debug stmts inserted
	afd.DebugLen = ann.debugIndex

	// insert imports if debug stmts were inserted
	if ann.builtDebugLineStmt {
		annset.insertImportDebug(astFile)

		// insert in all files to ensure inner init function runs
		annset.insertImport(astFile, "_", GodebugconfigPkgPath)

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
	annset.insertImport(astFile, annset.debugPkgName, DebugPkgPath)
}

func (annset *AnnotatorSet) insertImport(astFile *ast.File, name, path string) {
	astutil.AddNamedImport(annset.FSet, astFile, name, path)
}

//----------

func (annset *AnnotatorSet) annotatorFileData(filename string, files *Files) (*debug.AnnotatorFileData, error) {
	annset.afds.Lock()
	defer annset.afds.Unlock()

	afd, ok := annset.afds.m[filename]
	if ok {
		return afd, nil
	}

	// create new afd
	fafd, ok := files.annFileData[filename]
	if !ok {
		return nil, fmt.Errorf("annset: annotatorfiledata: file not found: %v", filename)
	}
	afd = &debug.AnnotatorFileData{
		FileIndex: annset.afds.index,
		Filename:  filename,
		FileHash:  fafd.FileHash,
		FileSize:  fafd.FileSize,
	}
	annset.afds.m[filename] = afd

	annset.afds.a = append(annset.afds.a, afd) // keep order
	annset.afds.index++

	return afd, nil
}

//----------

func (annset *AnnotatorSet) ConfigContent(network, addr string) string {
	entriesStr := annset.buildConfigContentEntries()

	syncSendStr := "false"
	if debug.SyncSend {
		syncSendStr = "true"
	}

	src := `package godebugconfig
import "` + DebugPkgPath + `"
func init(){
	debug.ServerNetwork = "` + network + `"
	debug.ServerAddress = "` + addr + `"
	debug.SyncSend = ` + syncSendStr + `
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
	for _, afd := range annset.afds.a {
		// sanity check
		if afd.FileIndex >= len(annset.afds.m) {
			panic(fmt.Sprintf("file index doesn't fit map len: %v vs %v", afd.FileIndex, len(annset.afds.m)))
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

//func (annset *AnnotatorSet) ConfigGoModuleContent() string {
//	return "module " + GodebugconfigPkgPath + "\n"
//}

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
import ` + annset.debugPkgName + ` "` + DebugPkgPath + `"
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

type TestMainSrc struct {
	Dir string
	Src string
}

//----------
