package internalcmds

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
)

func TestFindFilesRecursive(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"abcd.txt", "sub/CD space.txt", "cd-folder/unrelated.txt"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	buf := &bytes.Buffer{}
	opt := &iorw.IndexOpt{IgnoreCase: true, IgnoreDiacritics: true}
	if err := findFilesContext(context.Background(), buf, dir, "cd", opt); err != nil {
		t.Fatal(err)
	}
	want := "abcd.txt\ncd-folder" + string(filepath.Separator) + "\n" + parseutil.EscapeFilename(filepath.Join("sub", "CD space.txt")) + "\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := findFilesContext(ctx, buf, dir, "cd", opt); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestFindFilesMatchOffsets(t *testing.T) {
	opt := &iorw.IndexOpt{IgnoreCase: true, IgnoreDiacritics: true}
	for _, tt := range []struct{ name, query, match string }{
		{"abcd.txt", "cd", "cd"},
		{"abCD.txt", "cd", "CD"},
		{"ab\u00e7d.txt", "cd", "\u00e7d"},
		{"ab c d.txt", "c d", "c d"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := parseutil.EscapeFilename(filepath.Join("some dir", tt.name))
			start := len(s) - len(filepath.Base(s))
			i, n, err := findFilenameMatch(context.Background(), s[start:], parseutil.EscapeFilename(tt.query), opt)
			if err != nil || i < 0 {
				t.Fatalf("match: %v, %v", i, err)
			}
			got := s[start+i : start+i+n]
			if want := parseutil.EscapeFilename(tt.match); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
			if strings.Contains(got, "txt") {
				t.Fatal("match includes filename suffix")
			}
		})
	}
}
