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

	modF map[string]*File // [modPath]
}

func setupGoMods(ctx context.Context, cmd *Cmd, files *Files) error {
	mods := &GoMods{cmd: cmd, files: files}
	return mods.do(ctx)
}

func (mods *GoMods) do(ctx context.Context) error {
	// build map
	mods.modF = map[string]*File{}
	for _, f := range mods.files.files {
		if f.typ == FTMod {
			mods.modF[f.modulePath] = f
		}
	}

	if err := mods.updateGoMods(ctx); err != nil {
		return err
	}
	return nil
}

//----------

func (mods *GoMods) updateGoMods(ctx context.Context) error {
	for _, f := range mods.modF {
		if f.used() {
			if err := mods.updateGoMod(f); err != nil {
				return err
			}
		}
	}

	for _, f := range mods.modF {
		if f.used() {
			f.action = FAWriteMod
		}
	}
	return nil

}

//----------

func (mods *GoMods) updateGoMod(f1 *File) error {
	if err := mods.updateRequires(f1); err != nil {
		return err
	}
	if err := mods.updateReplaces(f1); err != nil {
		return err
	}
	return nil
}

//----------

func (mods *GoMods) updateRequires(f1 *File) error {
	if f1.needDebugMods {
		if err := mods.addRequireDebugMods(f1); err != nil {
			return err
		}
	}

	m1, err := f1.modFile()
	if err != nil {
		return err
	}

	// add all "requires" to main module
	if f1.mainModule {
		for _, f2 := range mods.modF {
			if f2 != f1 && f2.used() {
				if err := mods.addRequire(f1, f2); err != nil {
					return err
				}
			}
		}
	}

	// add/update used "requires"
	for _, req := range m1.Require {
		if reqOk(req) {
			f2, ok := mods.modF[req.Mod.Path]
			if ok {
				if f2 != f1 && f2.used() {
					if err := mods.addRequire(f1, f2); err != nil {
						return err
					}
				}
			}
		}
	}

	// drop "require" of editorPkg if not used
	for _, req := range m1.Require {
		if reqOk(req) {
			if req.Mod.Path == editorPkgPath {
				f2, ok := mods.modF[req.Mod.Path]
				drop := (ok && !f2.used()) || !ok
				if drop {
					if err := mods.dropRequire(f1, req.Mod.Path); err != nil {
						return err
					}
				}
				break
			}
		}
	}

	return nil
}

func (mods *GoMods) addRequireDebugPkgs(f1 *File, fmods []*File) error {
	// find files of the pkgs to insert
	w := []string{DebugPkgPath, GodebugconfigPkgPath}
	a := []*File{}
	for _, s := range w {
		f2, ok := mods.modF[s]
		if ok {
			a = append(a, f2)
		}
	}
	// must have found all
	if len(a) != len(w) {
		return fmt.Errorf("not all required debug pkgs were found: %v of %v", len(a), len(w))
	}
	// add "requires"
	for _, f2 := range a {
		if err := mods.addRequire(f1, f2); err != nil {
			return err
		}
	}
	return nil
}

//----------

func (mods *GoMods) updateReplaces(f1 *File) error {
	m1, err := f1.modFile()
	if err != nil {
		return err
	}
	// add/update "replace" directive of "required" annotated modules
	for _, req := range m1.Require {
		if reqOk(req) {
			if err := mods.addReplaceOfUsedRequired(f1, m1, req); err != nil {
				return err
			}
		}
	}
	// update "replace" directive of relative directories to original location
	for _, rep := range m1.Replace {
		if repOk(rep) {
			if err := mods.updateReplaceWithRelativeDir(f1, m1, rep); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mods *GoMods) addReplaceOfUsedRequired(f1 *File, m1 *modfile.File, req *modfile.Require) error {
	// find if the "require" is one of the used mods
	f2, ok := mods.modF[req.Mod.Path]
	if !ok || !f2.used() {
		return nil
	}

	// must match "replace" version or AddReplace() won't replace
	// could there be several replaces with diff versions? playing safe
	version := req.Mod.Version
	for _, rep := range m1.Replace {
		if repOk(rep) {
			if rep.Old.Path == req.Mod.Path {
				if rep.Old.Version == req.Mod.Version {
					version = req.Mod.Version
					break
				}
				if rep.Old.Version == "" {
					version = rep.Old.Version
				}
			}
		}
	}

	dest := filepath.Dir(mods.cmd.tmpDirBasedFilename(f2.destFilename()))
	if err := m1.AddReplace(req.Mod.Path, version, dest, ""); err != nil {
		return err
	}

	//mods.cmd.Vprintf("addReplaceOfRequired: %v: %v => %v\n", f1.shortFilename(), req.Mod.Path, dest)
	return nil
}

func (mods *GoMods) updateReplaceWithRelativeDir(f1 *File, m1 *modfile.File, rep *modfile.Replace) error {
	if !filepath.IsAbs(rep.New.Path) {
		// relative to original filename dir
		dir := filepath.Dir(f1.filename)
		rep.New.Path = filepath.Clean(filepath.Join(dir, rep.New.Path))
		if err := m1.AddReplace(rep.Old.Path, rep.Old.Version, rep.New.Path, ""); err != nil {
			return err
		}
		//mods.cmd.Vprintf("updateReplaceRelativeDir: %v: %v => %v\n", f1.shortFilename(), rep.Old.Path, rep.New.Path)
	}

	return nil
}

//----------

func (mods *GoMods) addRequireDebugMods(f1 *File) error {
	w := []string{DebugPkgPath, GodebugconfigPkgPath}
	for _, s := range w {
		f2, ok := mods.modF[s]
		if ok {
			if err := mods.addRequire(f1, f2); err != nil {
				return err
			}
		}
	}
	return nil
}

//----------

// f1 requires f2
func (mods *GoMods) addRequire(f1 *File, f2 *File) error {
	m1, err := f1.modFile()
	if err != nil {
		return err
	}
	version := f2.moduleVersion
	if version == "" {
		version = "v0.0.0"
	}
	if err := m1.AddRequire(f2.modulePath, version); err != nil {
		return err
	}

	//mods.cmd.Vprintf("addRequire: %v: %v\n", f1.shortFilename(), m2.Module.Mod.Path)
	return nil
}

// f1 does not require modPath
func (mods *GoMods) dropRequire(f1 *File, modPath string) error {
	m1, err := f1.modFile()
	if err != nil {
		return err
	}
	if err := m1.DropRequire(modPath); err != nil {
		return err
	}
	//mods.cmd.Vprintf("dropRequire: %v: %v\n", f1.shortFilename(), modPath)
	return nil
}

//----------

func reqOk(u *modfile.Require) bool {
	return u.Syntax != nil
}
func repOk(u *modfile.Replace) bool {
	return u.Syntax != nil
}
