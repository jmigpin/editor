package godebug

import (
	"context"
	"path/filepath"

	"github.com/jmigpin/editor/util/goutil"
)

// debug pkg as its own module: can't use "debug" as a module (with a go.mod) in the main src tree because a default replace to use the tree version won't be honored. Other modules will try to fetch it from the web if not explicitly declared in their go.mod. The editor itself won't find the module in the tree if not explicitly in the go.mod as well.

func SetupGoMods(ctx context.Context, cmd *Cmd, files *Files) error {
	// create missing go.mods
	for f := range files.modMissings {
		dir := filepath.Dir(f)
		pkgPath, ok := files.progDirPkgPaths[dir]
		if !ok {
			continue
		}
		// create go.mod file at tmp
		dirAtTmp := cmd.tmpDirBasedFilename(dir)
		if err := goutil.GoModInit(ctx, dirAtTmp, pkgPath, cmd.env); err != nil {
			return err
		}
	}

	// update all found go.mods, only the main one will be used
	mods := files.modFilenamesAndMissing()
	for filename := range mods {
		// update go.mod
		dir2 := filepath.Dir(filename)
		if err := setupGoMod(ctx, cmd, mods, dir2, files); err != nil {
			return err
		}
	}
	return nil
}

func setupGoMod(ctx context.Context, cmd *Cmd, mods map[string]struct{}, dir string, files *Files) error {
	// add godebug dependencies
	dirAtTmp := cmd.tmpDirBasedFilename(dir)
	if err := setupGodebugGoMod(ctx, cmd, dirAtTmp, files); err != nil {
		return err
	}

	goMod, err := goutil.ReadGoMod(ctx, dirAtTmp, cmd.env)
	if err != nil {
		return err
	}

	// update existing "replaces" relative dirs
	for _, rep := range goMod.Replace {
		np := rep.New.Path
		if !filepath.IsAbs(np) {
			abs, err := filepath.Abs(filepath.Join(dir, np))
			if err != nil {
				return err
			}
			if err := goutil.GoModReplace(ctx, dirAtTmp, rep.Old.Path, abs, cmd.env); err != nil {
				return err
			}
		}

	}
	// add "replaces" of all mods
	for filename2 := range mods {
		dir2 := filepath.Dir(filename2)
		if dir2 == dir {
			continue // itself
		}
		pkgPath, ok := files.progDirPkgPaths[dir2]
		if !ok {
			// get path from the go.mod file
			goMod2, err := goutil.ReadGoMod(ctx, dir2, cmd.env)
			if err != nil {
				return err
			}
			pkgPath = goMod2.Module.Mod.Path
		}
		dirAtTmp2 := cmd.tmpDirBasedFilename(dir2)
		if err := goutil.GoModReplace(ctx, dirAtTmp, pkgPath, dirAtTmp2, cmd.env); err != nil {
			return err
		}
	}

	// run tidy
	if err := goutil.GoModTidy(ctx, dirAtTmp, cmd.env); err != nil {
		return err
	}

	return nil
}

func setupGodebugGoMod(ctx context.Context, cmd *Cmd, dir string, files *Files) error {
	// require godebugconfig
	p := GodebugconfigPkgPath + "@v0.0.0" // version needed
	if err := goutil.GoModRequire(ctx, dir, p, cmd.env); err != nil {
		return err
	}
	// require debug
	p = DebugPkgPath + "@v0.0.0" // version needed
	if err := goutil.GoModRequire(ctx, dir, p, cmd.env); err != nil {
		return err
	}

	// replace godebugconfig (point to tmp dir)
	oldPath := GodebugconfigPkgPath
	newPath := filepath.Join(cmd.tmpDir, files.GodebugconfigPkgFilename(""))
	if err := goutil.GoModReplace(ctx, dir, oldPath, newPath, cmd.env); err != nil {
		return err
	}
	// replace debug (point to tmp dir)
	oldPath = DebugPkgPath
	newPath = filepath.Join(cmd.tmpDir, files.DebugPkgFilename(""))
	if err := goutil.GoModReplace(ctx, dir, oldPath, newPath, cmd.env); err != nil {
		return err
	}
	return nil
}
