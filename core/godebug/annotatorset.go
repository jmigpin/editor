package godebug

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"golang.org/x/tools/go/ast/astutil"
)

type AnnotatorSet struct {
	FSet            *token.FileSet
	debugPkgName    string
	debugVarPrefix  string
	insertedImports struct {
		sync.Mutex
		m map[string]bool
	}
	afds struct {
		sync.Mutex
		m     map[string]*debug.AnnotatorFileData // map[filename]afd
		a     []*debug.AnnotatorFileData          // ordered
		index int                                 // counter for new files
	}
}

func NewAnnotatorSet() *AnnotatorSet {
	annset := &AnnotatorSet{FSet: token.NewFileSet()}
	annset.insertedImports.m = map[string]bool{}
	annset.afds.m = map[string]*debug.AnnotatorFileData{}
	annset.debugPkgName = "d" + string(rune(931)) // uncommon rune to avoid clashes
	annset.debugVarPrefix = annset.debugPkgName   // will have integer appended
	return annset
}

//----------

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, f *File) error {
	if f.action != FAAnnotate {
		panic(fmt.Sprintf("file not set for annotation: %v", f.filename))
	}
	// default to copy (might endup not getting annotations)
	if f.action == FAAnnotate {
		f.action = FACopy
	}

	afd, err := annset.annotatorFileData(f)
	if err != nil {
		return err
	}
	ann := NewAnnotator(annset.FSet, f)
	ann.debugPkgName = annset.debugPkgName
	ann.debugVarPrefix = annset.debugVarPrefix
	ann.fileIndex = afd.FileIndex

	ann.AnnotateAstFile(astFile)

	// n debug stmts inserted
	afd.DebugLen = ann.debugIndex

	// insert imports if debug stmts were inserted
	if ann.builtDebugLineStmt {
		annset.updateImports(astFile, f)
	}

	//// DEBUG
	////godebug:annotatefile:annotator.go
	//buf := &bytes.Buffer{}
	//if err := goutil.PrintAstFile(buf, f.files.fset, astFile); err != nil {
	//	return err
	//}
	//fmt.Printf("===astfile===\n%v\n%v\n", f.filename, string(buf.Bytes()))

	return nil
}

//----------

func (annset *AnnotatorSet) InsertExitInMain(astFile *ast.File, f *File, testMode bool) bool {
	name := "main"
	if testMode {
		name = "TestMain"
	}
	ok := annset.insertDebugExitInFunction(astFile, name)
	if ok {
		annset.updateImports(astFile, f)
	}
	return ok
}

func (annset *AnnotatorSet) updateImports(astFile *ast.File, f *File) {
	annset.insertedImports.Lock()
	defer annset.insertedImports.Unlock()
	if _, ok := annset.insertedImports.m[f.filename]; !ok {
		annset.insertedImports.m[f.filename] = true
		annset.insertImportDebug(astFile)
		annset.insertImportGodebugconfig(astFile)

		// ast file changed, set for output
		f.action = FAWriteAst
	}
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

func (annset *AnnotatorSet) insertImportGodebugconfig(astFile *ast.File) {
	// should be inserted in all files to ensure inner init function runs
	annset.insertImport(astFile, "_", GodebugconfigPkgPath)
}

func (annset *AnnotatorSet) insertImport(astFile *ast.File, name, path string) {
	astutil.AddNamedImport(annset.FSet, astFile, name, path)
}

//----------

func (annset *AnnotatorSet) annotatorFileData(f *File) (*debug.AnnotatorFileData, error) {
	annset.afds.Lock()
	defer annset.afds.Unlock()

	afd, ok := annset.afds.m[f.filename]
	if ok {
		return afd, nil
	}

	// create new afd
	afd = &debug.AnnotatorFileData{
		FileIndex: annset.afds.index,
		Filename:  f.filename,
		FileHash:  f.annFileData.FileHash,
		FileSize:  f.annFileData.FileSize,
	}
	annset.afds.m[f.filename] = afd

	annset.afds.a = append(annset.afds.a, afd) // keep order
	annset.afds.index++

	return afd, nil
}

//----------

func (annset *AnnotatorSet) ConfigContent(serverNetwork, serverAddr string, syncSend, acceptOnlyFirstClient bool) string {
	src := `package godebugconfig
import "` + DebugPkgPath + `"
func init(){
	debug.ServerNetwork = "` + serverNetwork + `"
	debug.ServerAddress = "` + serverAddr + `"
	debug.SyncSend = ` + strconv.FormatBool(syncSend) + `
	debug.AcceptOnlyFirstClient = ` + strconv.FormatBool(acceptOnlyFirstClient) + `
	debug.AnnotatorFilesData = []*debug.AnnotatorFileData{
		` + annset.buildConfigContentEntries() + `
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
