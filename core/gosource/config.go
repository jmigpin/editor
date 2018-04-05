package gosource

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/ast/astutil"
)

type Config struct {
	FSet       *token.FileSet
	Info       types.Info
	Conf       types.Config
	Pkgs       map[string]*types.Package // if present, means imported
	ParserMode parser.Mode               // default parser mode

	astFiles   map[string]*ast.File
	astFilesMu sync.RWMutex

	importable    map[string]struct{}
	dirExtraFiles map[string]map[string]struct{}
}

func NewConfig() *Config {
	conf := &Config{}
	conf.FSet = token.NewFileSet()
	conf.astFiles = make(map[string]*ast.File)
	conf.Pkgs = make(map[string]*types.Package)

	conf.importable = make(map[string]struct{})
	conf.dirExtraFiles = make(map[string]map[string]struct{})

	conf.Conf.Error = func(error) {} // non-nil to avoid stopping on first error
	//conf.Conf.IgnoreFuncBodies = true
	conf.Conf.Importer = conf
	conf.Conf.DisableUnusedImportCheck = true

	conf.initInfo()

	return conf
}

func (conf *Config) initInfo() {
	conf.Info = types.Info{}
	conf.Info.Types = make(map[ast.Expr]types.TypeAndValue)
	conf.Info.Defs = make(map[*ast.Ident]types.Object)
	conf.Info.Uses = make(map[*ast.Ident]types.Object)
	conf.Info.Implicits = make(map[ast.Node]types.Object)
	conf.Info.Selections = make(map[*ast.SelectorExpr]*types.Selection)
	conf.Info.Scopes = make(map[ast.Node]*types.Scope)
}

func (conf *Config) ParseFile(filename string, src interface{}, mode parser.Mode) (*ast.File, error, bool) {
	fullFilename := FullFilename(filename)

	// add as dir extra file
	if src != nil {
		conf.dirExtraFile(fullFilename)
	}

	return conf.parseFile(fullFilename, src, mode)
}

// Can return partial ast.
func (conf *Config) parseFile(filename string, src interface{}, mode parser.Mode) (*ast.File, error, bool) {
	conf.astFilesMu.RLock()
	astFile, ok := conf.astFiles[filename]
	conf.astFilesMu.RUnlock()
	if ok {
		return astFile, nil, true
	}
	astFile, err := parser.ParseFile(conf.FSet, filename, src, conf.ParserMode|mode)
	if astFile == nil {
		return nil, err, false
	}
	conf.astFilesMu.Lock()
	conf.astFiles[filename] = astFile
	conf.astFilesMu.Unlock()
	return astFile, err, true
}

func (conf *Config) parseFiles(filenames []string, mode parser.Mode) ([]*ast.File, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var files []*ast.File
	errors := []error{}

	addFile := func(f *ast.File, err error) {
		mu.Lock()
		if f != nil {
			files = append(files, f)
		}
		if err != nil {
			errors = append(errors, err)
		}
		mu.Unlock()
	}

	for _, filename := range filenames {
		conf.astFilesMu.RLock()
		file, ok := conf.astFiles[filename]
		conf.astFilesMu.RUnlock()
		if ok {
			addFile(file, nil)
			continue
		}

		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			file, err, _ := conf.parseFile(filename, nil, mode)
			addFile(file, err)
		}(filename)
	}
	wg.Wait()
	return files, errors
}

func (conf *Config) check(path string, astFiles []*ast.File) (*types.Package, error) {
	return conf.Conf.Check(path, conf.FSet, astFiles, &conf.Info)
}

// Implements types.Importer
func (conf *Config) Import(path string) (*types.Package, error) {
	return conf.ImportPath(path)
}

// TODO: Implements types.ImporterFrom
//func (conf *Config) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {}

func (conf *Config) ImportPath(path string) (*types.Package, error) {
	pkg, ok := conf.Pkgs[path]
	if !ok {
		if _, ok := conf.importable[path]; ok {
			// Ensure bad code with a cycle import doesn't go on endless loop. Have the map entry be present to force "ok=true" in "pkg, ok := conf.Pkgs[path]"
			conf.Pkgs[path] = nil

			var err error
			pkg, err = conf.importPath2(path)
			_ = err // ignore to continue
			//pkg.MarkComplete() // allows to do more analysis? crash
		} else {
			// create empty package
			name, err := PkgName(path)
			if err != nil {
				return nil, err
				// TODO: ignore error?
				//name = filepath.Base(path)
			}
			pkg = types.NewPackage(path, name)
			pkg.MarkComplete() // allows to do more analysis
		}
		conf.Pkgs[path] = pkg
	}
	return pkg, nil
}

func (conf *Config) importPath2(path string) (*types.Package, error) {
	filenames, err := conf.PkgFilenames(path)
	if err != nil {
		return nil, err
	}
	astFiles, errors := conf.parseFiles(filenames, parser.Mode(0))

	// ignore to do conf check with possible partial ast files
	_ = errors

	return conf.check(path, astFiles)
}

func (conf *Config) PkgFilenames(path string) ([]string, error) {
	dir, _, names, err := PkgFilenames(path, true)
	if err != nil {
		// don't handle the error, the files might be in "dir extra files" (ex: provided src)
		//return nil, err
	}

	// add dir extra files
	for dir2, m := range conf.dirExtraFiles {
		if dir2 == dir {
			// mark those already added
			seen := make(map[string]bool)
			for _, name := range names {
				if _, ok := m[name]; ok {
					seen[name] = true
				}
			}
			// add extra names if not added yet
			for name, _ := range m {
				if !seen[name] {
					names = append(names, name)
				}
			}
		}
	}

	// build full filenames
	u := []string{}
	for _, n := range names {
		u = append(u, filepath.Join(dir, n))
	}
	return u, nil
}

func (conf *Config) ReImportImportables() []error {
	// reset info (needed?)
	conf.initInfo()

	// clear cached pkgs from previous imports
	conf.Pkgs = make(map[string]*types.Package)

	// import importables
	errors := []error{}
	for p := range conf.importable {
		_, err := conf.ImportPath(p)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (conf *Config) MakeFilePkgImportable(filename string) {
	fullFilename := FullFilename(filename)
	_, pkgFilename := ExtractSrcDir(fullFilename)
	pkgPath := filepath.Dir(pkgFilename)
	conf.MakeImportable(pkgPath)
}
func (conf *Config) MakeImportable(path string) {
	conf.importable[path] = struct{}{}
}

func (conf *Config) IsImportable(path string) bool {
	_, ok := conf.importable[path]
	return ok
}

func (conf *Config) dirExtraFile(filename string) {
	fullFilename := FullFilename(filename)

	path := filepath.Dir(fullFilename)
	name := filepath.Base(fullFilename)

	_, ok := conf.dirExtraFiles[path]
	if !ok {
		conf.dirExtraFiles[path] = make(map[string]struct{})
	}
	conf.dirExtraFiles[path][name] = struct{}{}
}

func (conf *Config) PosAstFile(pos token.Pos) (*ast.File, error) {
	tf, err := conf.PosTokenFile(pos)
	if err != nil {
		return nil, err
	}
	astFile, ok := conf.astFiles[tf.Name()]
	if !ok {
		return nil, fmt.Errorf("ast file not found: %v", tf.Name())
	}
	return astFile, nil
}

func (conf *Config) PosAstPath(pos token.Pos) (path []ast.Node, exact bool, ok bool) {
	astFile, err := conf.PosAstFile(pos)
	if err != nil {
		return nil, false, false
	}
	path, exact = astutil.PathEnclosingInterval(astFile, pos, pos)
	return path, exact, true
}

func (conf *Config) SurePosAstPath(pos token.Pos) []ast.Node {
	path, _, ok := conf.PosAstPath(pos)
	if !ok {
		log.Printf("unable to get sure pos ast path: %v", pos)
	}
	return path
}

// Handy for partial ASTs with bad positions.
func (conf *Config) PosTokenFile(pos token.Pos) (*token.File, error) {
	tf := conf.FSet.File(pos)
	if tf == nil {
		return nil, fmt.Errorf("unable to get pos token file")
	}
	return tf, nil
}

func (conf *Config) PosInnermostScope(pos token.Pos) (*types.Scope, error) {
	astFile, err := conf.PosAstFile(pos)
	if err != nil {
		return nil, err
	}
	scope, ok := conf.Info.Scopes[astFile]
	if !ok {
		return nil, fmt.Errorf("scope not found in info")
	}
	s2 := scope.Innermost(pos)
	if s2 == nil {
		return nil, fmt.Errorf("innermost scope not found")
	}
	return s2, nil
}

func (conf *Config) PosPkgDir(pos token.Pos) (string, error) {
	tf, err := conf.PosTokenFile(pos)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(tf.Name())
	_, pkgDir := ExtractSrcDir(dir)
	return pkgDir, nil
}

func (conf *Config) PosPkg(pos token.Pos) (*types.Package, error) {
	dir, err := conf.PosPkgDir(pos)
	if err != nil {
		return nil, err
	}
	if pkg, ok := conf.Pkgs[dir]; ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("pkg not found for pos")
}

//------------

func (conf *Config) BuiltinLookup(name string) (ast.Node, error) {
	pkg, err := conf.importBuiltin()
	if err != nil {
		return nil, err
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("builtin name not found: %v", name)
	}
	path, _, ok := conf.PosAstPath(obj.Pos())
	if !ok {
		return nil, fmt.Errorf("builtin ast path not found for pos")
	}
	return path[0], nil
}

func (conf *Config) importBuiltin() (*types.Package, error) {
	path := "builtin"
	if !conf.IsImportable(path) {
		conf.MakeImportable(path)
		_ = conf.ReImportImportables()
	}
	return conf.ImportPath(path)
}

//------------

type Builtin struct {
	Name string
}

func (b *Builtin) Pos() token.Pos { return 0 }
func (b *Builtin) End() token.Pos { return 0 }
