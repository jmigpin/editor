package core

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestListDirContextOptionsDefaultOutput(t *testing.T) {
	dir := listDirTestTree(t)

	got := listDirTestOutput(t, dir, ListDirOptions{})
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		"keep" + string(os.PathSeparator),
		"skip" + string(os.PathSeparator),
		"a.go",
		"b.txt",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsFilter(t *testing.T) {
	dir := listDirTestTree(t)

	opts := ListDirOptions{
		Subs:    true,
		Filters: listDirTestRegexps(t, `\.go$`),
	}
	got := listDirTestOutput(t, dir, opts)
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		filepath.Join("skip", "d.go"),
		"a.go",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsRemovePrunesDir(t *testing.T) {
	dir := listDirTestTree(t)

	opts := ListDirOptions{
		Subs:    true,
		Removes: listDirTestRegexps(t, `(^|`+regexp.QuoteMeta(string(os.PathSeparator))+`)skip`+regexp.QuoteMeta(string(os.PathSeparator))),
	}
	got := listDirTestOutput(t, dir, opts)
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		"keep" + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"a.go",
		"b.txt",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsRemoveTakesPriority(t *testing.T) {
	dir := listDirTestTree(t)

	opts := ListDirOptions{
		Filters: listDirTestRegexps(t, `\.go$`),
		Removes: listDirTestRegexps(t, `a\.go$`),
	}
	got := listDirTestOutput(t, dir, opts)
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAbsoluteAndEncodedFilters(t *testing.T) {
	dir := listDirTestTree(t)

	absPattern := regexp.QuoteMeta(filepath.Join(dir, "keep", "c.go"))
	got := listDirTestOutput(t, dir, ListDirOptions{
		Subs:    true,
		Filters: listDirTestRegexps(t, absPattern),
	})
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("absolute filter got:\n%s\nwant:\n%s", got, want)
	}

	encodedPattern := regexp.QuoteMeta(filepath.Join("~1", "keep")) + ".*"
	got = listDirTestOutput(t, dir, ListDirOptions{
		Subs:       true,
		Filters:    listDirTestRegexps(t, encodedPattern),
		EncodePath: listDirTestEncodePath(dir),
	})
	want = strings.Join([]string{
		".." + string(os.PathSeparator),
		"keep" + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("encoded filter got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsShortOutput(t *testing.T) {
	dir := listDirTestTree(t)

	got := listDirTestOutput(t, dir, ListDirOptions{
		Short:      true,
		EncodePath: listDirTestEncodePath(dir),
	})
	want := strings.Join([]string{
		".." + string(os.PathSeparator),
		"keep" + string(os.PathSeparator),
		"skip" + string(os.PathSeparator),
		"a.go",
		"b.txt",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAtRelativeDir(t *testing.T) {
	dir := listDirTestTree(t)

	got := listDirTestOutputAt(t, dir, "keep", ListDirOptions{})
	want := strings.Join([]string{
		"keep" + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAtShortParentEntry(t *testing.T) {
	dir := listDirTestTree(t)

	got := listDirTestOutputAt(t, dir, "keep", ListDirOptions{
		Short:      true,
		EncodePath: listDirTestEncodePath(dir),
	})
	want := strings.Join([]string{
		"keep" + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAtRelOutput(t *testing.T) {
	dir := listDirTestTree(t)
	source := filepath.Join(dir, "keep")

	got := listDirTestOutputAt(t, "", source, ListDirOptions{
		Rel:     true,
		RelBase: dir,
	})
	want := strings.Join([]string{
		"keep" + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsRelCurrentDirEntry(t *testing.T) {
	parent := t.TempDir()
	cur := filepath.Join(parent, "cur")
	if err := os.Mkdir(cur, 0o755); err != nil {
		t.Fatal(err)
	}

	got := listDirTestOutputAt(t, "", parent, ListDirOptions{
		Rel:     true,
		RelBase: cur,
	})
	want := strings.Join([]string{
		".." + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		"." + string(os.PathSeparator),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAtShortUsesShortestPath(t *testing.T) {
	dir := listDirTestTree(t)
	source := filepath.Join(dir, "keep")

	got := listDirTestOutputAt(t, "", source, ListDirOptions{
		Short:      true,
		Rel:        true,
		EncodePath: listDirTestEncodePath(dir),
		RelBase:    dir,
	})
	want := strings.Join([]string{
		"keep" + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		filepath.Join("keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirContextOptionsAtShortUsesEncodedWhenShorter(t *testing.T) {
	dir := listDirTestTree(t)
	source := filepath.Join(dir, "keep")

	got := listDirTestOutputAt(t, "", source, ListDirOptions{
		Short:      true,
		EncodePath: listDirTestEncodePath(dir),
	})
	want := strings.Join([]string{
		filepath.Join("~1", "keep") + string(os.PathSeparator) + ".." + string(os.PathSeparator),
		filepath.Join("~1", "keep", "c.go"),
		"",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestListDirOptionsShortUsesEncodedRelativeWhenShorter(t *testing.T) {
	root := t.TempDir()
	homeRoot := filepath.Join(root, "home-src")
	relBase := filepath.Join(homeRoot, "a", "b", "c")
	absPath := filepath.Join(root, "other-src", "acct") + string(os.PathSeparator)
	encodePath := func(filename string) string {
		cleanHomeRoot := filepath.Clean(homeRoot)
		cleanFilename := filepath.Clean(filename)
		if cleanFilename == cleanHomeRoot {
			return "~1"
		}
		rel, err := filepath.Rel(cleanHomeRoot, cleanFilename)
		if err != nil || strings.HasPrefix(rel, "..") {
			return filename
		}
		return filepath.Join("~1", rel)
	}

	got := (ListDirOptions{
		Short:      true,
		Rel:        true,
		RelBase:    relBase,
		EncodePath: encodePath,
	}).outputPath("", absPath, true)
	want := strings.Join([]string{"~1", "..", "other-src", "acct"}, string(os.PathSeparator)) + string(os.PathSeparator)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestListDirContextOptionsAtOpenErrorNoParentEntry(t *testing.T) {
	dir := t.TempDir()

	got := listDirTestOutputAt(t, dir, "missing", ListDirOptions{})
	if strings.Contains(got, ".."+string(os.PathSeparator)+"\n") {
		t.Fatalf("unexpected parent entry before error:\n%s", got)
	}
	if !strings.Contains(got, "no such file") {
		t.Fatalf("missing open error:\n%s", got)
	}
}

func TestParseListDirCmdArgsFlagsRepeatable(t *testing.T) {
	parsed, err := ParseListDirCmdArgs([]string{"-f", `\.go$`, "-exc=x", "a/b", "b/c"}, ListDirCmdConfig{BaseDir: "/home/a"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(parsed.Sources), 2; got != want {
		t.Fatalf("sources: got %d, want %d", got, want)
	}
	if parsed.Sources[0].AddedFilepath != "a/b" || parsed.Sources[1].AddedFilepath != "b/c" {
		t.Fatalf("sources: got %v", parsed.Sources)
	}
	if got, want := len(parsed.Opts.Filters), 1; got != want {
		t.Fatalf("filters: got %d, want %d", got, want)
	}
	if got, want := len(parsed.Opts.Removes), 1; got != want {
		t.Fatalf("removes: got %d, want %d", got, want)
	}
	if !parsed.Opts.Short {
		t.Fatalf("short default: got false, want true")
	}
	if !parsed.Opts.Rel {
		t.Fatalf("rel default: got false, want true")
	}
	if !parsed.Opts.Filters[0].MatchString("a.go") || !parsed.Opts.Removes[0].MatchString("x") {
		t.Fatalf("unexpected regexps")
	}
}

func TestParseListDirCmdArgsShortFlagCanDisable(t *testing.T) {
	parsed, err := ParseListDirCmdArgs([]string{"-short=false"}, ListDirCmdConfig{BaseDir: "/home/a"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Opts.Short {
		t.Fatalf("short: got true, want false")
	}
}

func TestParseListDirCmdArgsInvalidRegexp(t *testing.T) {
	if _, err := ParseListDirCmdArgs([]string{"-f", "["}, ListDirCmdConfig{BaseDir: "/home/a"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseListDirCmdArgsDecodesHomeVarPattern(t *testing.T) {
	decodePath := func(s string) string {
		if s == "~1" {
			return "/tmp/root"
		}
		return s
	}
	parsed, err := ParseListDirCmdArgs([]string{`-f=~1/a\.go$`}, ListDirCmdConfig{BaseDir: "/home/a", DecodePath: decodePath})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(parsed.Opts.Filters), 2; got != want {
		t.Fatalf("filters: got %d, want %d", got, want)
	}
	if !parsed.Opts.Filters[0].MatchString(`~1/a.go`) {
		t.Fatalf("original pattern did not match")
	}
	if !parsed.Opts.Filters[1].MatchString("/tmp/root/a.go") {
		t.Fatalf("decoded pattern did not match")
	}
	if parsed.Opts.Filters[1].MatchString("/tmp/root/abgo") {
		t.Fatalf("decoded pattern lost regexp escapes")
	}
}

func TestListDirSourcesFromArgs(t *testing.T) {
	decodePath := func(s string) string {
		if s == "~1" {
			return "/home/a"
		}
		return s
	}

	sources, err := listDirSourcesFromArgs([]string{"tmp"}, "/home/a", decodePath)
	if err != nil {
		t.Fatal(err)
	}
	if sources[0].Filepath != "/home/a" || sources[0].AddedFilepath != "tmp" {
		t.Fatalf("relative: got %v", sources)
	}

	sources, err = listDirSourcesFromArgs([]string{"~1/tmp"}, "/home/a", decodePath)
	if err != nil {
		t.Fatal(err)
	}
	if sources[0].Filepath != "" || sources[0].AddedFilepath != "/home/a/tmp" {
		t.Fatalf("encoded absolute: got %v", sources)
	}

	sources, err = listDirSourcesFromArgs([]string{"~1/home-src/../../"}, "/home/a/cur", decodePath)
	if err != nil {
		t.Fatal(err)
	}
	if sources[0].Filepath != "" || sources[0].AddedFilepath != "/home" {
		t.Fatalf("encoded parent traversal: got %v", sources)
	}

	sources, err = listDirSourcesFromArgs([]string{"."}, "/home/a", decodePath)
	if err != nil {
		t.Fatal(err)
	}
	if sources[0].Filepath != "/home/a" || sources[0].AddedFilepath != "" {
		t.Fatalf("dot: got %v", sources)
	}

	sources, err = listDirSourcesFromArgs([]string{"a", "b", "/tmp"}, "/home/a", decodePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(sources), 3; got != want {
		t.Fatalf("sources: got %d, want %d", got, want)
	}
	if sources[0].AddedFilepath != "a" || sources[1].AddedFilepath != "b" || sources[2].AddedFilepath != "/tmp" {
		t.Fatalf("multiple sources: got %v", sources)
	}
}

//----------

func listDirTestTree(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range []string{"keep", "skip", ".hidden"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{
		"a.go",
		"b.txt",
		filepath.Join("keep", "c.go"),
		filepath.Join("skip", "d.go"),
		filepath.Join(".hidden", "e.go"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func listDirTestOutput(t *testing.T, dir string, opts ListDirOptions) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := ListDirContextOptions(context.Background(), buf, dir, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func listDirTestOutputAt(t *testing.T, dir, addedFilepath string, opts ListDirOptions) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := ListDirContextOptionsAt(context.Background(), buf, dir, addedFilepath, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func listDirTestRegexps(t *testing.T, patterns ...string) []*regexp.Regexp {
	t.Helper()

	res := []*regexp.Regexp{}
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, re)
	}
	return res
}

func listDirTestEncodePath(root string) func(string) string {
	return func(filename string) string {
		cleanRoot := filepath.Clean(root)
		cleanFilename := filepath.Clean(filename)
		if cleanFilename == cleanRoot {
			return "~1"
		}
		rel, err := filepath.Rel(cleanRoot, cleanFilename)
		if err != nil || strings.HasPrefix(rel, "..") {
			return filename
		}
		return filepath.Join("~1", rel)
	}
}
