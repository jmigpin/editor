package core

import (
	"image/color"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

// detection and setup of syntax highlighting for strings and comments
func detectSetupSyntaxHighlight(erow *ERow) {

	// special handling for the toolbar (allow comment shortcut to work in the toolbar to easily disable cmds)
	setupToolbarCommenting(erow.Row.Toolbar.Toolbar)

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
		".toml", ".yaml", ".yml",
		".py", // python
		".pl": // perl
		setc("#")
	case ".go",
		".c", ".h",
		".cpp", ".hpp", ".cxx", ".hxx", // c++
		".cu", ".cuh", // cuda
		".java",
		".v",  // verilog
		".js", // javascript
		".rs": // rust
		setc("//", [2]string{"/*", "*/"})
	case ".zig", ".zon": // zig
		setc("//")
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
	case ".ml", ".mli":
		setc([2]string{"(*", "*)"})

	case ".txt":
		setc("#") // useful (but not correct)

	case ".json": // no comments to setup
	case ".json5", ".jsonc", ".jsonh": // json flavors
		setc("//", [2]string{"/*", "*/"})

	default: // all other file extensions
		// ex: /etc/network/interfaces (no file extension)

		// TODO: read header (ex: "#!...") but this gives "#", which is already used now for non-detected

		setc("#") // useful (but not correct)
	}
}

//----------

func setupToolbarCommenting(tb *ui.Toolbar) {
	tb.SetCommentStrings("#")
	tb.EditCtx().Fns.CommentUnitIndexes = toolbarCommentUnitIndexes
	updateToolbarImportantVariableColoring(tb)
	tb.RWEvReg.Add(iorw.RWEvIdWrite, func(any) {
		updateToolbarImportantVariableColoring(tb)
	})
}

func updateToolbarImportantVariableColoring(tb *ui.Toolbar) {
	spans := toolbarImportantVariableSpans(tb.Str())
	ops := make([]*drawutil.ColorizeOp, 0, len(spans)*2)
	procColor := func(fg, bg color.Color) (_, _ color.Color) {
		if bg == nil {
			bg = tb.TreeThemePaletteColor("text_bg")
		}
		return toolbarImportantVariableColors(fg, bg)
	}
	for _, span := range spans {
		ops = append(ops,
			&drawutil.ColorizeOp{Offset: span[0], ProcColor: procColor},
			&drawutil.ColorizeOp{Offset: span[1]},
		)
	}
	tb.SetExtraColorOps(ops)
}

var toolbarVarFgColor int
var toolbarVarBgColor int

func toolbarImportantVariableColors(fg, bg color.Color) (_, _ color.Color) {
	if toolbarVarFgColor > 1 {
		fg = imageutil.RgbaFromInt(toolbarVarFgColor)
	}
	switch {
	case toolbarVarBgColor == 0:
		bg = imageutil.TintOrShade(bg, 0.1)
	case toolbarVarBgColor > 1:
		bg = imageutil.RgbaFromInt(toolbarVarBgColor)
	}
	return fg, bg
}

func toolbarImportantVariableSpans(src string) [][2]int {
	important := map[string]bool{
		"$colorize":   true,
		"$font":       true,
		"$scrollMode": true,
		"$terminal":   true,
	}
	spans := [][2]int{}
	data := toolbarparser.Parse(src)
	for _, part := range data.Parts {
		if len(part.Args) != 1 {
			continue
		}
		arg := part.Args[0]
		s := arg.String()
		i := strings.IndexByte(s, '=')
		if i < 0 || !important[s[:i]] {
			continue
		}
		spans = append(spans, [2]int{arg.Pos(), arg.Pos() + i})
	}
	return spans
}

func toolbarCommentUnitIndexes(ctx *rwedit.Ctx) (int, int, bool, bool, error) {
	if _, _, ok := ctx.C.SelectionIndexes(); ok {
		return 0, 0, false, false, nil
	}

	src, err := iorw.ReadFastFull(ctx.RW)
	if err != nil {
		return 0, 0, false, false, err
	}

	data := toolbarparser.Parse(string(src))
	a, b, ok := data.PartCommentIndexes(ctx.C.Index())
	if !ok {
		return 0, 0, false, false, nil
	}

	lineA, lineB, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return 0, 0, false, false, err
	}
	a = max(a, lineA)
	b = min(b, lineB)
	return a, b, newline, true, nil
}
