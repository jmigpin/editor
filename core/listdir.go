package core

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/flagutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func ListDirERow(erow *ERow, filepath string, subs, hiddens bool) {
	opts := ListDirOptions{Subs: subs, Hiddens: hiddens}
	ListDirERowOptions(erow, filepath, opts)
}

func ListDirERowOptions(erow *ERow, filepath string, opts ListDirOptions) {
	ListDirERowOptionsAt(erow, filepath, "", opts)
}

func ListDirERowOptionsAt(erow *ERow, filepath, addedFilepath string, opts ListDirOptions) {
	sources := []ListDirSource{{Filepath: filepath, AddedFilepath: addedFilepath}}
	ListDirERowOptionsSources(erow, sources, opts)
}

func ListDirERowOptionsSources(erow *ERow, sources []ListDirSource, opts ListDirOptions) {
	erow.Exec.RunAsync(func(ctx context.Context, rw io.ReadWriter) error {
		for _, source := range sources {
			if err := ListDirContextOptionsAt(ctx, rw, source.Filepath, source.AddedFilepath, opts); err != nil {
				return err
			}
		}
		return nil
	})
}

func ListDirERowReloadFromToolbar(erow *ERow) (bool, error) {
	parsed, ok, err := lastListDirReloadCmd(&erow.TbData, ListDirCmdConfig{
		BaseDir:    erow.Info.Dir(),
		DecodePath: erow.Ed.HomeVars.Decode,
		EncodePath: erow.Ed.HomeVars.Encode,
	})
	if err != nil || !ok {
		return ok, err
	}
	ListDirERowOptionsSources(erow, parsed.Sources, parsed.Opts)
	return true, nil
}

//----------

type ListDirOptions struct {
	Subs    bool
	Hiddens bool
	Short   bool
	Rel     bool
	Filters []*regexp.Regexp
	Removes []*regexp.Regexp

	EncodePath func(string) string
	RelBase    string
}

type ListDirSource struct {
	Filepath      string
	AddedFilepath string
}

type ListDirCmdConfig struct {
	BaseDir    string
	DecodePath func(string) string
	EncodePath func(string) string
}

type ListDirCmdParsed struct {
	Opts    ListDirOptions
	Sources []ListDirSource
	Reload  bool
}

func ParseListDirCmdArgs(args []string, cfg ListDirCmdConfig) (*ListDirCmdParsed, error) {
	filters := []*regexp.Regexp{}
	removes := []*regexp.Regexp{}
	fs, flags := newListDirFlagSet(&filters, &removes, cfg.DecodePath)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	sources, err := listDirSourcesFromArgs(fs.Args(), cfg.BaseDir, cfg.DecodePath)
	if err != nil {
		return nil, err
	}

	opts := ListDirOptions{
		Subs:       *flags.sub,
		Hiddens:    *flags.hidden,
		Short:      *flags.short,
		Rel:        *flags.rel,
		Filters:    filters,
		Removes:    removes,
		EncodePath: cfg.EncodePath,
		RelBase:    cfg.BaseDir,
	}

	return &ListDirCmdParsed{Opts: opts, Sources: sources, Reload: *flags.reload}, nil
}

func ListDirFlagSetUsage(w io.Writer) {
	filters := []*regexp.Regexp{}
	removes := []*regexp.Regexp{}
	fs, _ := newListDirFlagSet(&filters, &removes, nil)
	fs.SetOutput(w)
	fs.Usage()
}

type listDirFlags struct {
	sub    *bool
	hidden *bool
	short  *bool
	rel    *bool
	reload *bool
}

func newListDirFlagSet(filters, removes *[]*regexp.Regexp, decodePath func(string) string) (*flag.FlagSet, listDirFlags) {
	fs := flag.NewFlagSet("ListDir", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // don't output to stderr
	flags := listDirFlags{}
	flags.sub = fs.Bool("sub", false, "list subdirectories/files")
	flags.hidden = fs.Bool("hidden", false, "list hidden files")
	flags.short = fs.Bool("short", true, "shorten output paths with home vars")
	flags.rel = fs.Bool("rel", true, "output paths relative to the current directory")
	flags.reload = fs.Bool("reload", false, "use this ListDir definition when reloading the row")
	fs.Var(regexpListDirFlag(filters, decodePath), "f", "filter regexp")
	fs.Var(regexpListDirFlag(removes, decodePath), "exc", "exclude regexp")
	return fs, flags
}

func ListDirContext(ctx context.Context, w io.Writer, filepath string, subs, hiddens bool) error {
	opts := ListDirOptions{Subs: subs, Hiddens: hiddens}
	return ListDirContextOptions(ctx, w, filepath, opts)
}

func ListDirContextOptions(ctx context.Context, w io.Writer, fpath string, opts ListDirOptions) error {
	return ListDirContextOptionsAt(ctx, w, fpath, "", opts)
}

func ListDirContextOptionsAt(ctx context.Context, w io.Writer, fpath, addedFilepath string, opts ListDirOptions) error {
	return listDirContext(ctx, w, fpath, addedFilepath, opts, true)
}

func listDirContext(ctx context.Context, w io.Writer, fpath, addedFilepath string, opts ListDirOptions, parentEntry bool) error {
	fp2 := filepath.Join(fpath, addedFilepath)

	out := func(s string) bool {
		_, err := w.Write([]byte(s))
		return err == nil
	}

	f, err := os.Open(fp2)
	if err != nil {
		out(err.Error())
		return nil
	}

	fis, err := f.Readdir(-1)
	f.Close() // close as soon as possible
	if err != nil {
		out(err.Error())
		return nil
	}

	// stop on context
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if parentEntry {
		// parent directory at the top
		u := listDirParentEntry(fpath, addedFilepath, opts)
		if !out(u + "\n") {
			return nil
		}
	}

	slices.SortFunc(fis, CompareFileInfos)

	for _, fi := range fis {
		// stop on context
		if ctx.Err() != nil {
			return ctx.Err()
		}

		name := fi.Name()

		if !opts.Hiddens && strings.HasPrefix(name, ".") {
			continue
		}

		relPath := filepath.Join(addedFilepath, name)
		absPath := filepath.Join(fpath, relPath)
		if fi.IsDir() {
			relPath += string(os.PathSeparator)
			absPath += string(os.PathSeparator)
		}
		write, prune := opts.filter(relPath, absPath)
		if write {
			outputPath := opts.outputPath(relPath, absPath, fi.IsDir())
			s := outputPath + "\n"
			if !out(s) {
				return nil
			}
		}
		if fi.IsDir() && opts.Subs && !prune {
			afp := filepath.Join(addedFilepath, name)
			err := listDirContext(ctx, w, fpath, afp, opts, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//----------

func listDirParentEntry(fpath, addedFilepath string, opts ListDirOptions) string {
	if addedFilepath == "" {
		return ".." + string(os.PathSeparator)
	}
	absPath := filepath.Join(fpath, addedFilepath)
	candidates := []string{}
	if opts.Rel && opts.RelBase != "" {
		if relPath, ok := listDirRelPath(opts.RelBase, absPath, true); ok {
			candidates = append(candidates, parseutil.EscapeFilename(relPath+".."+string(os.PathSeparator)))
		}
	}
	candidates = append(candidates, parseutil.EscapeFilename(addedFilepath+string(os.PathSeparator)+".."+string(os.PathSeparator)))
	if opts.Short && opts.EncodePath != nil {
		candidates = append(candidates, opts.EncodePath(absPath)+string(os.PathSeparator)+".."+string(os.PathSeparator))
		if outputPath, ok := opts.shortRelOutputPath(absPath, true); ok {
			candidates = append(candidates, outputPath+".."+string(os.PathSeparator))
		}
	}
	return shortestListDirString(candidates)
}

func (opts ListDirOptions) outputPath(relPath, absPath string, isDir bool) string {
	candidates := []string{}
	if opts.Rel && opts.RelBase != "" {
		if outputPath, ok := listDirRelPath(opts.RelBase, absPath, isDir); ok {
			candidates = append(candidates, parseutil.EscapeFilename(outputPath))
		}
	}
	candidates = append(candidates, parseutil.EscapeFilename(relPath))
	if opts.Short && opts.EncodePath != nil {
		outputPath := opts.EncodePath(absPath)
		if isDir {
			outputPath += string(os.PathSeparator)
		}
		candidates = append(candidates, outputPath)
		if outputPath, ok := opts.shortRelOutputPath(absPath, isDir); ok {
			candidates = append(candidates, outputPath)
		}
	}
	return shortestListDirString(candidates)
}

func (opts ListDirOptions) shortRelOutputPath(absPath string, isDir bool) (string, bool) {
	if opts.RelBase == "" {
		return "", false
	}
	relPath, ok := listDirRelPath(opts.RelBase, absPath, isDir)
	if !ok {
		return "", false
	}
	encodedBase := opts.EncodePath(opts.RelBase)
	return cleanListDirEncodedRelPath(encodedBase, relPath, isDir), true
}

func cleanListDirEncodedRelPath(encodedBase, relPath string, isDir bool) string {
	if _, _, ok := leadingListDirHomeVar(encodedBase); !ok {
		outputPath := filepath.Join(encodedBase, parseutil.EscapeFilename(relPath))
		if isDir {
			outputPath += string(os.PathSeparator)
		}
		return outputPath
	}

	sep := string(os.PathSeparator)
	parts := strings.Split(encodedBase+sep+parseutil.EscapeFilename(relPath), sep)
	stack := []string{}
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			} else {
				stack = append(stack, part)
			}
		default:
			stack = append(stack, part)
		}
	}
	outputPath := strings.Join(stack, sep)
	if isDir {
		outputPath += sep
	}
	return outputPath
}

func listDirRelPath(base, filename string, isDir bool) (string, bool) {
	trailingSep := strings.HasSuffix(filename, string(os.PathSeparator))
	filename = filepath.Clean(filename)
	base = filepath.Clean(base)
	relPath, err := filepath.Rel(base, filename)
	if err != nil {
		return "", false
	}
	if relPath == "." && !isDir && !trailingSep {
		relPath = ""
	}
	if isDir || trailingSep {
		relPath += string(os.PathSeparator)
	}
	return relPath, true
}

func shortestListDirString(candidates []string) string {
	shortest := ""
	for _, s := range candidates {
		if shortest == "" || len(s) < len(shortest) {
			shortest = s
		}
	}
	return shortest
}

func (opts ListDirOptions) filter(relPath, absPath string) (write bool, prune bool) {
	candidates := opts.matchCandidates(relPath, absPath)
	if matchesAny(opts.Removes, candidates) {
		return false, true
	}
	if len(opts.Filters) > 0 && !matchesAny(opts.Filters, candidates) {
		return false, false
	}
	return true, false
}

func (opts ListDirOptions) matchCandidates(relPath, absPath string) []string {
	u := []string{relPath, absPath}
	if opts.EncodePath != nil {
		encodedPath := opts.EncodePath(absPath)
		if strings.HasSuffix(absPath, string(os.PathSeparator)) {
			encodedPath += string(os.PathSeparator)
		}
		u = append(u, encodedPath)
	}
	return u
}

func matchesAny(res []*regexp.Regexp, candidates []string) bool {
	for _, re := range res {
		for _, s := range candidates {
			if re.MatchString(s) {
				return true
			}
		}
	}
	return false
}

//----------

func lastListDirReloadCmd(data *toolbarparser.Data, cfg ListDirCmdConfig) (*ListDirCmdParsed, bool, error) {
	var last *ListDirCmdParsed
	for _, part := range data.Parts {
		args := part.ArgsUnquoted()
		if len(args) == 0 || args[0] != "ListDir" {
			continue
		}
		parsed, err := ParseListDirCmdArgs(args[1:], cfg)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				continue
			}
			return nil, false, err
		}
		if parsed.Reload {
			last = parsed
		}
	}
	return last, last != nil, nil
}

//----------

func listDirSourcesFromArgs(pathArgs []string, baseDir string, decodePath func(string) string) ([]ListDirSource, error) {
	if len(pathArgs) == 0 {
		return []ListDirSource{{Filepath: baseDir}}, nil
	}
	sources := []ListDirSource{}
	for _, arg := range pathArgs {
		source, err := listDirSourceFromArg(arg, baseDir, decodePath)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func listDirSourceFromArg(arg string, baseDir string, decodePath func(string) string) (ListDirSource, error) {
	p := arg
	if decodePath != nil {
		p = decodePath(p)
	}
	p = filepath.Clean(p)
	if p == "." {
		return ListDirSource{Filepath: baseDir}, nil
	}
	if filepath.IsAbs(p) {
		return ListDirSource{AddedFilepath: p}, nil
	}
	return ListDirSource{Filepath: baseDir, AddedFilepath: p}, nil
}

func regexpListDirFlag(dst *[]*regexp.Regexp, decodePath func(string) string) flag.Value {
	return flagutil.StringFuncFlag(func(s string) error {
		patterns := []string{s}
		if decodePath != nil {
			if s2, ok := decodeLeadingListDirHomeVarPattern(s, decodePath); ok {
				patterns = append(patterns, s2)
			}
		}
		for _, pattern := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			*dst = append(*dst, re)
		}
		return nil
	})
}

func decodeLeadingListDirHomeVarPattern(s string, decodePath func(string) string) (string, bool) {
	key, suffix, ok := leadingListDirHomeVar(s)
	if !ok {
		return "", false
	}
	decoded := decodePath(key)
	if decoded == key {
		return "", false
	}
	return regexp.QuoteMeta(decoded) + suffix, true
}

func leadingListDirHomeVar(s string) (string, string, bool) {
	if s == "" || s[0] != '~' {
		return "", "", false
	}
	if len(s) == 1 {
		return "~", "", true
	}
	if s[1] < '0' || s[1] > '9' {
		if s[1] == '/' || s[1] == '\\' {
			return "~", s[1:], true
		}
		return "", "", false
	}
	i := 2
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:], true
}

//----------

func CompareFileInfos(a, b os.FileInfo) int {
	an := strings.ToLower(a.Name())
	bn := strings.ToLower(b.Name())

	cmp := func() int {
		v := strings.Compare(an, bn)
		if v == 0 {
			return strings.Compare(a.Name(), b.Name())
		}
		return v
	}

	if a.IsDir() {
		if b.IsDir() {
			return cmp()
		}
		return -1
	}
	if b.IsDir() {
		return 1
	}
	return cmp()
}
