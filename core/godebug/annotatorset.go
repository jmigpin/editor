package godebug

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/util/goutil"
	"golang.org/x/tools/go/ast/astutil"
)

type AnnotatorSet struct {
	fset *token.FileSet
	dopt *AnnSetDebugOpt
	afds struct {
		sync.Mutex
		m     map[string]*debug.AnnotatorFileData // map[filename]afd
		order []*debug.AnnotatorFileData          // ordered
		index int                                 // counter for new files
	}
}

func NewAnnotatorSet(fset *token.FileSet) *AnnotatorSet {
	annset := &AnnotatorSet{}
	annset.fset = fset
	annset.dopt = newAnnSetDebugOpt()
	annset.afds.m = map[string]*debug.AnnotatorFileData{}
	return annset
}

//----------

func (annset *AnnotatorSet) AnnotateAstFile(astFile *ast.File, ti *types.Info, nat map[ast.Node]AnnotationType, testModeMainFunc bool) (*Annotator, error) {

	filename, err := nodeFilename(annset.fset, astFile)
	if err != nil {
		return nil, err
	}

	afd, err := annset.annotatorFileData(filename)
	if err != nil {
		return nil, err
	}

	ann := NewAnnotator(annset.fset, ti, annset.dopt)
	ann.fileIndex = int(afd.FileIndex)
	ann.nodeAnnTypes = nat
	ann.testModeMainFunc = testModeMainFunc
	ann.AnnotateAstFile(astFile)

	// n debug stmts inserted
	afd.DebugNIndexes = debug.AfdMsgIndex(ann.debugNIndexes)

	return ann, nil
}

//----------

func (annset *AnnotatorSet) insertTestMain(astFile *ast.File) error {
	// TODO: detect if used imports are already imported with another name (os,testing)

	src := fmt.Sprintf(`
		func TestMain(m *testing.M) {
			%s.Exit(m.Run())
		}
	`, annset.dopt.PkgName)
	fd, err := goutil.ParseFuncDecl("TestMain", src)
	if err != nil {
		return err
	}

	// ensure imports
	//astutil.AddImport(annset.fset, astFile, "os")
	astutil.AddImport(annset.fset, astFile, "testing")

	astFile.Decls = append(astFile.Decls, fd)

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
		FileIndex: uint16(annset.afds.index),
		FileSize:  uint32(len(src)),
		Filename:  filename,
		FileHash:  sourceHash(src),
	}
	annset.afds.m[filename] = afd

	annset.afds.order = append(annset.afds.order, afd) // keep order
	annset.afds.index++

	return afd, nil
}

//----------

func (annset *AnnotatorSet) buildConfigAfdEntries() string {
	u := []string{}
	for _, afd := range annset.afds.order {
		s := fmt.Sprintf("&AnnotatorFileData{%v,%v,%q,%v,[]byte(%q)}",
			afd.FileIndex,
			afd.DebugNIndexes,
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

type AnnSetDebugOpt struct {
	PkgPath   string
	PkgName   string
	VarPrefix string
}

func newAnnSetDebugOpt() *AnnSetDebugOpt {
	// The godebug/debug pkg is writen to a tmp dir and used with the pkg path "godebugconfig/debug" to avoid dependencies in the target build. Annotation data is added to the aaaconfig.go. The godebug/debug pkg is included in the editor binary via //go:embed directive.
	var debugPkgPath = "godebugconfig/debug"

	return &AnnSetDebugOpt{
		PkgPath:   debugPkgPath,
		PkgName:   "Σ", // uncommon rune to avoid clashes; expected by tests
		VarPrefix: "Σ", // will have integer appended
	}
}

//----------
//----------
//----------

func sourceHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}
