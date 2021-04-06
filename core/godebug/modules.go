package godebug

import (
	"context"
	"fmt"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

type GoMods struct {
	cmd   *Cmd
	files *Files
	mi    map[string]*ModInfo // [pkgPath]
}

func setupGoMods(ctx context.Context, cmd *Cmd, files *Files) error {
	mods := &GoMods{cmd: cmd, files: files}
	return mods.do(ctx)
}

func (mods *GoMods) do(ctx context.Context) error {
	mi, err := buildModInfos(mods.files)
	if err != nil {
		return err
	}
	mods.mi = mi

	if err := mods.updateGoMods(ctx); err != nil {
		return err
	}

	return nil
	//return goutil.GoModTidy(ctx, mods.cmd.tmpDir, mods.cmd.env)
}

//----------

func (mods *GoMods) updateGoMods(ctx context.Context) error {
	found := false
	for _, f := range mods.files.modFiles {
		if f.hasAction() && f.main {
			found = true
			if err := mods.updateMainGoMod(f); err != nil {
				return err
			}
			// update action
			f.action = FAWrite

			// updating main module only

			//dir := filepath.Dir(f.destFilename())
			//return goutil.GoModTidy(ctx, dir, mods.cmd.env)
			break
		}
	}
	if !found {
		return fmt.Errorf("main module not found")
	}
	return nil
}

//----------

func (mods *GoMods) updateMainGoMod(f1 *ModFile) error {
	// insert "requires" for all used packages (multiple versions)
	// replace them with directory (no version)

	m1, err := f1.modFile()
	if err != nil {
		return err
	}
	//godebug:annotatefile
	for _, mi := range mods.mi {
		if !mi.hasAction {
			continue
		}
		// require all versions
		for v := range mi.versions {
			if v != "" {
				// replaces previous "require" keeping only one
				if err := m1.AddRequire(mi.path, v); err != nil {
					return err
				}

				//mods.addNewRequire(m1, mi.path, v)

				// add "replace" without version
				if f2, ok := mi.modfs[v]; ok {
					if f2.hasAction() {
						dir := filepath.Dir(f2.destFilename())
						if err := mods.addReplaceDirAtTmp(m1, mi.path, "", dir, ""); err != nil {
							return err
						}
						break
					}
				}
			}
		}
	}

	// add "replace" directive of relative directories to original location
	for _, rep := range m1.Replace {
		if repOk(rep) {
			if err := mods.addReplaceWithRelativeDir(f1, m1, rep); err != nil {
				return err
			}
		}
	}

	return nil
}

//----------

func (mods *GoMods) addNewRequire(m1 *modfile.File, path, version string) {
	// prevent double lines
	for _, req := range m1.Require {
		if reqOk(req) {
			if req.Mod.Path == path && req.Mod.Version == version {
				return
			}
		}
	}

	m1.AddNewRequire(path, version, false)
}

func (mods *GoMods) addReplaceDirAtTmp(m1 *modfile.File, path, v1, dir, v2 string) error {
	dirAtTmp := mods.cmd.tmpDirBasedFilename(dir)
	return m1.AddReplace(path, v1, dirAtTmp, v2)
}

func (mods *GoMods) addReplaceWithRelativeDir(f1 *ModFile, m1 *modfile.File, rep *modfile.Replace) error {
	if !filepath.IsAbs(rep.New.Path) {
		// relative to original filename dir
		dir := filepath.Dir(f1.filename)
		rep.New.Path = filepath.Clean(filepath.Join(dir, rep.New.Path))
		if err := m1.AddReplace(rep.Old.Path, rep.Old.Version, rep.New.Path, ""); err != nil {
			return err
		}
	}
	return nil
}

//----------
//----------
//----------

type ModInfo struct {
	path      string
	hasAction bool
	modfs     map[string]*ModFile // [version]
	versions  map[string]bool     // [version]; in "require" and modfiles
}

func buildModInfos(files *Files) (map[string]*ModInfo, error) {
	m := map[string]*ModInfo{} // [pkgPath]
	// index mod files by pkgpath
	for _, f1 := range files.modFiles {
		mi, ok := m[f1.path]
		if !ok {
			mi = &ModInfo{path: f1.path}
			mi.modfs = map[string]*ModFile{}
			mi.versions = map[string]bool{}
			m[f1.path] = mi
		}
		mi.modfs[f1.version] = f1
		mi.versions[f1.version] = true
		mi.hasAction = mi.hasAction || f1.hasAction()
	}
	// add "require" versions to index
	for _, f1 := range files.modFiles {
		m1, err := f1.modFile()
		if err != nil {
			return nil, err
		}
		for _, req := range m1.Require {
			if reqOk(req) {
				mi, ok := m[req.Mod.Path]
				if ok && req.Mod.Version != "" {
					mi.versions[req.Mod.Version] = true
				}
			}
		}
	}
	return m, nil
}

//----------

func reqOk(u *modfile.Require) bool {
	return u.Syntax != nil
}
func repOk(u *modfile.Replace) bool {
	return u.Syntax != nil
}
