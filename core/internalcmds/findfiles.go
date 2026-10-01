package internalcmds

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
)

func FindFiles(args *core.InternalCmdArgs) error {
	fs := flag.NewFlagSet("FindFiles", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "", "search directory (defaults to the row directory)")
	onrows := fs.Bool("onrows", false, "search only filenames of open rows")
	opt := &iorw.IndexOpt{}
	fs.BoolVar(&opt.IgnoreCase, "icase", true, "ignore case")
	fs.BoolVar(&opt.IgnoreDiacritics, "idiac", true, "ignore diacritics")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	words := fs.Args()
	for i, s := range words {
		if u, err := parseutil.UnquoteStringBs(s); err == nil {
			words[i] = u
		}
	}
	query := strings.Join(words, " ")
	if query == "" {
		return fmt.Errorf("missing filename search text")
	}
	base := erow.Info.Dir()
	if *dir != "" {
		base = args.Ed.HomeVars.Decode(*dir)
		if !filepath.IsAbs(base) {
			base = filepath.Join(erow.Info.Dir(), base)
		}
	}
	base, err = filepath.Abs(base)
	if err != nil {
		return err
	}
	opened := []string{}
	seen := map[string]bool{}
	for _, row := range args.Ed.ERows() {
		if !row.Info.IsFileButNotDir() {
			continue
		}
		name := row.Info.Name()
		i, _, err := findFilenameMatch(args.Ctx, filepath.Base(name), query, opt)
		if err != nil {
			return err
		}
		if i < 0 {
			continue
		}
		if !seen[name] {
			seen[name] = true
			opened = append(opened, name)
		}
		if a, ok := row.TbData.Part0Arg0(); ok {
			// Match the displayed basename so escaped toolbar paths retain byte offsets.
			s := a.String()
			start := len(s) - len(filepath.Base(s))
			if i, n, err := findFilenameMatch(args.Ctx, s[start:], parseutil.EscapeFilename(query), opt); err != nil {
				return err
			} else if i >= 0 {
				row.Row.Toolbar.FlashIndexLen(a.Pos()+start+i, n)
			}
		}
	}
	result := erow
	if !erow.Info.IsDir() {
		result = core.NewBasicERow(args.Ed.ReadERowInfo(base), erow.Row.PosBelow())
		iorw.Append(result.Row.Toolbar.RW(), []byte(" | Stop"))
	}
	outputDir := result.Info.Dir()
	result.Exec.RunAsync(func(ctx context.Context, rw io.ReadWriter) error {
		if *onrows {
			for _, name := range opened {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := writeFindFilename(rw, outputDir, name); err != nil {
					return err
				}
			}
			return nil
		}
		return findFilesContextAt(ctx, rw, base, outputDir, query, opt)
	})
	return nil
}

func findFilenameMatch(ctx context.Context, name, query string, opt *iorw.IndexOpt) (int, int, error) {
	return iorw.IndexCtx(ctx, iorw.NewBytesReadWriterAt([]byte(name)), 0, []byte(query), opt)
}

func findFilesContext(ctx context.Context, w io.Writer, base, query string, opt *iorw.IndexOpt) error {
	return findFilesContextAt(ctx, w, base, base, query, opt)
}

func findFilesContextAt(ctx context.Context, w io.Writer, base, outputDir, query string, opt *iorw.IndexOpt) error {
	return filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			_, err := fmt.Fprintf(w, "error: %v\n", walkErr)
			return err
		}
		if entry.IsDir() {
			return nil
		}
		i, _, err := findFilenameMatch(ctx, entry.Name(), query, opt)
		if err != nil || i < 0 {
			return err
		}
		return writeFindFilename(w, outputDir, path)
	})
}

func writeFindFilename(w io.Writer, base, name string) error {
	rel, err := filepath.Rel(base, name)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, parseutil.EscapeFilename(rel))
	return err
}
