package internalcmds

import (
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func TestOpenFilePos(t *testing.T) {
	tests := []struct {
		src      string
		filename string
		line     int
		column   int
		offset   int
	}{
		{"a/b.txt", "a/b.txt", 0, 0, -1},
		{"a/b.txt:12:3", "a/b.txt", 12, 3, -1},
		{"a file.txt", "a file.txt", 0, 0, -1},
	}

	for _, tt := range tests {
		fp := openFilePos(tt.src)
		if fp.Filename != tt.filename {
			t.Fatalf("%q: filename: got %q, want %q", tt.src, fp.Filename, tt.filename)
		}
		if fp.Line != tt.line {
			t.Fatalf("%q: line: got %d, want %d", tt.src, fp.Line, tt.line)
		}
		if fp.Column != tt.column {
			t.Fatalf("%q: column: got %d, want %d", tt.src, fp.Column, tt.column)
		}
		if fp.Offset != tt.offset {
			t.Fatalf("%q: offset: got %d, want %d", tt.src, fp.Offset, tt.offset)
		}
	}
}

func TestParseOpenOptions(t *testing.T) {
	tests := []struct {
		src             string
		rowMode         bool
		externalMode    bool
		filemanagerMode bool
		terminalEmuMode bool
		path            string
		args            []string
	}{
		{src: "Open a/b.txt", rowMode: true, path: "a/b.txt"},
		{src: "Open -external a/b.txt", externalMode: true, path: "a/b.txt"},
		{src: "Open -external", externalMode: true},
		{src: "Open -filemanager a file.txt", filemanagerMode: true, path: "a file.txt"},
		{src: "Open -filemanager", filemanagerMode: true},
		{src: "Open -terminalemu a/b top -o x", terminalEmuMode: true, path: "a/b", args: []string{"top", "-o", "x"}},
		{src: "Open -terminalemu", terminalEmuMode: true},
	}

	for _, tt := range tests {
		part := parseOpenPart(t, tt.src)
		opts, err := parseOpenOptions(part)
		if err != nil {
			t.Fatalf("%q: %v", tt.src, err)
		}
		if *opts.rowMode != tt.rowMode {
			t.Fatalf("%q: row mode: got %v, want %v", tt.src, *opts.rowMode, tt.rowMode)
		}
		if *opts.externalMode != tt.externalMode {
			t.Fatalf("%q: external mode: got %v, want %v", tt.src, *opts.externalMode, tt.externalMode)
		}
		if *opts.filemanagerMode != tt.filemanagerMode {
			t.Fatalf("%q: filemanager mode: got %v, want %v", tt.src, *opts.filemanagerMode, tt.filemanagerMode)
		}
		if *opts.terminalEmuMode != tt.terminalEmuMode {
			t.Fatalf("%q: terminalemu mode: got %v, want %v", tt.src, *opts.terminalEmuMode, tt.terminalEmuMode)
		}
		if opts.path != tt.path {
			t.Fatalf("%q: path: got %q, want %q", tt.src, opts.path, tt.path)
		}
		if len(opts.args) != len(tt.args) {
			t.Fatalf("%q: args: got %v, want %v", tt.src, opts.args, tt.args)
		}
		for i := range opts.args {
			if opts.args[i] != tt.args[i] {
				t.Fatalf("%q: args: got %v, want %v", tt.src, opts.args, tt.args)
			}
		}
	}
}

func TestParseOpenOptionsMultipleModes(t *testing.T) {
	part := parseOpenPart(t, "Open -row -external a/b.txt")
	if _, err := parseOpenOptions(part); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseOpenOptionsMissingRowPath(t *testing.T) {
	part := parseOpenPart(t, "Open")
	if _, err := parseOpenOptions(part); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseOpenOptionsHelp(t *testing.T) {
	part := parseOpenPart(t, "Open -h")
	_, err := parseOpenOptions(part)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "-external") {
		t.Fatalf("missing usage: %v", err)
	}
}

func parseOpenPart(t *testing.T, s string) *toolbarparser.Part {
	t.Helper()
	data := toolbarparser.Parse(s)
	if len(data.Parts) != 1 {
		t.Fatalf("parts: got %d, want 1", len(data.Parts))
	}
	return data.Parts[0]
}
