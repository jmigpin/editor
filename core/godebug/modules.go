package godebug

import (
	"context"
	"path/filepath"

	"github.com/jmigpin/editor/util/goutil"
)

func SetupGoMods(ctx context.Context, cmd *Cmd, files *Files) error {
	// create go.mods on packages that have none
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
		// setup godebug dependencies
		err := setupGodebugGoMod(ctx, cmd, dirAtTmp, files)
		if err != nil {
			return err
		}

		if err := goutil.GoModTidy(ctx, dirAtTmp, cmd.env); err != nil {
			return err
		}
	}

	// updating all found go.mods, only the main one will be used
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
	// update go.mod with godebug
	dirAtTmp := cmd.tmpDirBasedFilename(dir)
	if err := setupGodebugGoMod(ctx, cmd, dirAtTmp, files); err != nil {
		return err
	}

	// read go.mod
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

	// add "replaces" for annotated modules
	for _, req := range goMod.Require {
		for filename2 := range mods {
			dir2 := filepath.Dir(filename2)
			if dir2 == dir { // same dir (same go mod file)
				continue
			}
			dirAtTmp2 := cmd.tmpDirBasedFilename(dir2)

			// read go.mod
			goMod2, err := goutil.ReadGoMod(ctx, dirAtTmp2, cmd.env)
			if err != nil {
				return err
			}

			// if gomod depends on gomod2
			if req.Path == goMod2.Module.Path {
				if err := goutil.GoModReplace(ctx, dirAtTmp, req.Path, dirAtTmp2, cmd.env); err != nil {
					return err
				}
			}
		}
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
