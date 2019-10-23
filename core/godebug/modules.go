package godebug

import (
	"context"
	"path/filepath"

	"github.com/jmigpin/editor/util/goutil"
)

func SetupGoMods(ctx context.Context, cmd *Cmd, files *Files, mainFilename string, tests bool) error {
	dir := filepath.Dir(mainFilename)
	if tests {
		dir = mainFilename
	}

	// no go.mod defined (probably small simple file)
	if len(files.modFilenames) == 0 {
		dirAtTmp := cmd.tmpDirBasedFilename(dir)

		// create mod file at tmp
		mod := "example.com/main"
		if err := goutil.GoModCreate(dirAtTmp, mod); err != nil {
			return err
		}

		// add to go.mod the godebugconfig location
		if err := replaceGodebugconfigPkgInGoMod(ctx, cmd, dirAtTmp); err != nil {
			return err
		}
		return nil
	}

	// update main go.mod (mainFilename)
	for filename := range files.modFilenames {
		// find main go.mod
		dir2 := filepath.Dir(filename)
		if dir2 == dir {
			if err := setupGoMod(ctx, cmd, files, filename); err != nil {
				return err
			}
			// update only the main go.mod
			break
		}
	}
	return nil
}

func setupGoMod(ctx context.Context, cmd *Cmd, files *Files, filename string) error {
	dir := filepath.Dir(filename)

	// add to go.mod the godebugconfig location
	dirAtTmp := cmd.tmpDirBasedFilename(dir)
	if err := replaceGodebugconfigPkgInGoMod(ctx, cmd, dirAtTmp); err != nil {
		return err
	}

	// read go.mod
	goMod, err := goutil.ReadGoMod(ctx, dirAtTmp)
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
			if err := goutil.GoModReplace(ctx, dirAtTmp, rep.Old.Path, abs); err != nil {
				return err
			}
		}

	}

	// update/add "replaces" for the other mod files (annotated pkgs)
	for filename2 := range files.modFilenames {
		dir2 := filepath.Dir(filename2)
		if dir2 == dir { // same file
			continue
		}
		dirAtTmp2 := cmd.tmpDirBasedFilename(dir2)

		// read go.mod
		goMod2, err := goutil.ReadGoMod(ctx, dirAtTmp2)
		if err != nil {
			return err
		}

		// if gomod depends on gomod2
		for _, req := range goMod.Require {
			if req.Path == goMod2.Module.Path {
				if err := goutil.GoModReplace(ctx, dirAtTmp, req.Path, dirAtTmp2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func replaceGodebugconfigPkgInGoMod(ctx context.Context, cmd *Cmd, dirAtTmp string) error {
	gdcPkgPath := "example.com/godebugconfig"
	gdcDir := filepath.Join(cmd.tmpDir, gdcPkgPath)
	return goutil.GoModReplace(ctx, dirAtTmp, gdcPkgPath, gdcDir)
}
