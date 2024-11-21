package core

import (
	"path/filepath"
	"strings"
)

// detection and setup of syntax highlighting for strings and comments
func detectSetupSyntaxHighlight(erow *ERow) {

	// special handling for the toolbar (allow comment shortcut to work in the toolbar to easily disable cmds)
	erow.Row.Toolbar.SetCommentStrings("#")

	// commented: allow dirs/specials to have output coloring
	//// consider only files from here (dirs and special rows are out)
	//if !erow.Info.IsFileButNotDir() {
	//	return
	//}

	//----------

	ta := erow.Row.TextArea

	// ensure syntax highlight is on (ex: strings)
	ta.EnableSyntaxHighlight(true)

	// set comments
	setc := func(a ...any) {
		ta.SetCommentStrings(a...)
	}

	name := filepath.Base(erow.Info.Name())
	// ignore "." on files starting with "."
	if len(name) >= 1 && name[0] == '.' {
		name = name[1:]
	}

	ext := strings.ToLower(filepath.Ext(name))

	//----------

	// specific names
	switch name {
	case "bashrc":
		setc("#")
		return
	case "Xresources":
		setc("!", "#")
		return
	}

	// go files specific suffixes (ex: allows "my_go.work")
	suffixes := []string{
		"go.mod", "go.sum", "go.work", "go.work.sum",
	}
	for _, suf := range suffixes {
		if strings.HasSuffix(name, suf) {
			setc("//", [2]string{"/*", "*/"})
			return
		}
	}

	// by extension
	switch ext {
	case ".sh",
		".conf", ".list",
		".py", // python
		".pl": // perl
		setc("#")
	case ".go",
		".c", ".h",
		".cpp", ".hpp", ".cxx", ".hxx", // c++
		".java",
		".jsonc", ".jsonh", // json flavors
		".v",  // verilog
		".js": // javascript
		setc("//", [2]string{"/*", "*/"})
	case ".pro": // prolog
		setc("%", [2]string{"/*", "*/"})
	case ".html", ".xml", ".svg":
		setc([2]string{"<!--", "-->"})
	case ".css":
		setc([2]string{"/*", "*/"})
	case ".s", ".asm": // assembly
		setc("//")
	case ".rb": // ruby
		setc("#", [2]string{"=begin", "=end"})
	case ".ledger":
		setc(";", "#") // ";" is main symbol for comments but is not if in the description; while "#" is not a comment in some other cases

	case ".txt":
		setc("#") // useful (but not correct)

	case ".json": // no comments to setup

	default: // all other file extensions
		// ex: /etc/network/interfaces (no file extension)

		// TODO: read header (ex: "#!...") but this gives "#", which is already used now for non-detected

		setc("#") // useful (but not correct)
	}
}
