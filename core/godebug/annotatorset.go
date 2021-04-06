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

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, f *SrcFile) error {
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
		annset.updateOsExitCalls(astFile)
	}

	// set to write (even if it might endup not getting annotations)
	// can't default to copy as it will lose src line references
	f.action = FAWrite

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

func (annset *AnnotatorSet) InsertDebugExitInMain(fd *ast.FuncDecl, astFile *ast.File, f *SrcFile) {
	annset.insertDebugExitInFuncDecl(fd)
	annset.updateImports(astFile, f)
	annset.updateOsExitCalls(astFile)
}

func (annset *AnnotatorSet) insertDebugExitInFuncDecl(fd *ast.FuncDecl) {
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
}

//----------

func (annset *AnnotatorSet) updateImports(astFile *ast.File, f *SrcFile) {
	annset.insertedImports.Lock()
	defer annset.insertedImports.Unlock()
	if _, ok := annset.insertedImports.m[f.filename]; !ok {
		annset.insertedImports.m[f.filename] = true
		annset.insertImportDebug(astFile)
	}
}

//----------

func (annset *AnnotatorSet) insertImportDebug(astFile *ast.File) {
	annset.insertImport(astFile, annset.debugPkgName, debugPkgPath)
}

func (annset *AnnotatorSet) insertImport(astFile *ast.File, name, path string) {
	astutil.AddNamedImport(annset.FSet, astFile, name, path)
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

	if count == 0 {
		_ = astutil.DeleteImport(annset.FSet, astFile, "os")
	}
}

//----------

func (annset *AnnotatorSet) annotatorFileData(f *SrcFile) (*debug.AnnotatorFileData, error) {
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

//func (annset *AnnotatorSet) ConfigContent(serverNetwork, serverAddr string, syncSend, acceptOnlyFirstClient bool) []byte {
//	src := `package godebugconfig
//import "` + DebugPkgPath + `"
//func init(){
//	debug.ServerNetwork = "` + serverNetwork + `"
//	debug.ServerAddress = "` + serverAddr + `"
//	debug.SyncSend = ` + strconv.FormatBool(syncSend) + `
//	debug.AcceptOnlyFirstClient = ` + strconv.FormatBool(acceptOnlyFirstClient) + `
//	debug.AnnotatorFilesData = []*debug.AnnotatorFileData{
//		` + annset.buildConfigContentEntries() + `
//	}
//	debug.StartServer()
//}
//`
//	return []byte(src)
//}

func (annset *AnnotatorSet) buildDebugConfigEntries() string {
	// build map data
	var u []string
	for _, afd := range annset.afds.a {
		// sanity check
		if afd.FileIndex >= len(annset.afds.m) {
			panic(fmt.Sprintf("file index doesn't fit map len: %v vs %v", afd.FileIndex, len(annset.afds.m)))
		}

		s := fmt.Sprintf("&AnnotatorFileData{%v,%v,%q,%v,[]byte(%q)}",
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

func lookupFuncDeclWithBody(astFile *ast.File, name string) (*ast.FuncDecl, bool) {
	obj := astFile.Scope.Lookup(name)
	if obj != nil && obj.Kind == ast.Fun {
		fd, ok := obj.Decl.(*ast.FuncDecl)
		if ok && fd.Body != nil {
			return fd, true
		}
	}
	return nil, false
}
