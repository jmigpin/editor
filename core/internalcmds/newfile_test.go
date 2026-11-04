package internalcmds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func TestNewFileArgs(t *testing.T) {
	for _, tt := range []struct {
		input, name string
		parents     bool
		wantErr     bool
	}{
		{"NewFile file.txt", "file.txt", false, false},
		{"NewFile -p a/b/file.txt", "a/b/file.txt", true, false},
		{"NewFile", "", false, true},
		{"NewFile -p", "", false, true},
		{"NewFile -x file.txt", "", false, true},
		{"NewFile -p a b", "", false, true},
	} {
		part := toolbarparser.Parse(tt.input).Parts[0]
		name, parents, err := newFileArgs(part)
		if (err != nil) != tt.wantErr || name != tt.name || parents != tt.parents {
			t.Fatalf("%q: name=%q, parents=%v, err=%v", tt.input, name, parents, err)
		}
	}
}

func TestCreateNewFile(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "a", "b", "file.txt")
	if err := createNewFile(filename, false); err == nil || !strings.Contains(err.Error(), "directory does not exist") {
		t.Fatalf("missing parent without -p: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(filename)); !os.IsNotExist(err) {
		t.Fatalf("parent unexpectedly created: %v", err)
	}
	if err := createNewFile(filename, true); err != nil {
		t.Fatal(err)
	}
	if err := createNewFile(filepath.Join(dir, "a", "b", "other.txt"), true); err != nil {
		t.Fatalf("existing parent with -p: %v", err)
	}
	if err := os.WriteFile(filename, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := createNewFile(filename, true); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing file: %v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed: data=%q, err=%v", data, err)
	}
}
