package internalcmds

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/reslocparser"
)

func Open(args *core.InternalCmdArgs) error {
	opts, err := parseOpenOptions(args.Part)
	if err != nil {
		return err
	}
	return openRun(args, opts)
}

func openRun(args *core.InternalCmdArgs, opts *openOptions) error {
	switch {
	case *opts.rowMode:
		return openRowPath(args, opts.path)
	case *opts.externalMode:
		return openExternalPath(args, opts.path)
	case *opts.filemanagerMode:
		return openFilemanagerPath(args, opts.path)
	case *opts.terminalMode:
		return openTerminalPath(args, opts.path)
	case *opts.terminalEmuMode:
		return openTerminalEmuPath(args, opts.path, opts.args)
	default:
		return fmt.Errorf("missing open mode")
	}
}

//----------

type openOptions struct {
	rowMode         *bool
	externalMode    *bool
	filemanagerMode *bool
	terminalMode    *bool
	terminalEmuMode *bool
	path            string
	args            []string
}

func newOpenOptions() *openOptions {
	return &openOptions{
		rowMode:         new(bool),
		externalMode:    new(bool),
		filemanagerMode: new(bool),
		terminalMode:    new(bool),
		terminalEmuMode: new(bool),
	}
}

func parseOpenOptions(part *toolbarparser.Part) (*openOptions, error) {
	fs := flag.NewFlagSet("Open", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opts := newOpenOptions()
	opts.rowMode = fs.Bool("row", false, "open in a new row (default)")
	opts.externalMode = fs.Bool("external", false, "open with the preferred external application")
	opts.filemanagerMode = fs.Bool("filemanager", false, "open with the external file manager")
	opts.terminalMode = fs.Bool("terminal", false, "open an external terminal at the path directory")
	opts.terminalEmuMode = fs.Bool("terminalemu", false, "open an internal terminal emulator at the path directory")
	if err := fs.Parse(part.ArgsUnquoted()[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			buf := &bytes.Buffer{}
			fs.SetOutput(buf)
			fs.Usage()
			return nil, fmt.Errorf("%w\n%v", err, buf.String())
		}
		return nil, err
	}

	if err := openValidateModes(opts); err != nil {
		return nil, err
	}

	remaining := fs.Args()
	if *opts.rowMode && len(remaining) == 0 {
		return nil, fmt.Errorf("missing filename")
	}

	if *opts.terminalEmuMode {
		if len(remaining) > 0 {
			opts.path = remaining[0]
			opts.args = remaining[1:]
		}
	} else {
		opts.path = strings.Join(remaining, " ")
	}
	return opts, nil
}

func openValidateModes(opts *openOptions) error {
	selectedModes := 0
	for _, mode := range []*bool{
		opts.rowMode,
		opts.externalMode,
		opts.filemanagerMode,
		opts.terminalMode,
		opts.terminalEmuMode,
	} {
		if *mode {
			selectedModes++
		}
	}
	if selectedModes > 1 {
		return fmt.Errorf("multiple open modes")
	}
	if selectedModes == 0 {
		*opts.rowMode = true
	}
	return nil
}

//----------

func openRowPath(args *core.InternalCmdArgs, path string) error {
	if path == "" {
		return fmt.Errorf("missing filename")
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	filePos := openFilePos(path)
	filePos.Filename = args.Ed.HomeVars.Decode(filePos.Filename)

	filename, fi, ok := core.FindFileInfo(filePos.Filename, erow.Info.Dir())
	if !ok {
		return fmt.Errorf("fileinfo not found: %q", filePos.Filename)
	}
	filePos.Filename = filename

	rowPos := erow.Row.PosBelow()
	if erow.Info.IsDir() && !fi.IsDir() {
		rowPos = args.Ed.GoodRowPos()
	}

	conf := &core.OpenFileERowConfig{
		FilePos:               filePos,
		RowPos:                rowPos,
		FlashVisibleOffsets:   true,
		NewIfNotExistent:      true,
		NewIfOffsetNotVisible: true,
		PreferedERow:          erow,
	}
	core.OpenFileERow(args.Ed, conf)
	return nil
}

func openExternalPath(args *core.InternalCmdArgs, path string) error {
	var err error
	path, err = openPathOrCurrentRowName(args, path)
	if err != nil {
		return err
	}

	filename, _, err := openResolvedPath(args, path)
	if err != nil {
		return err
	}
	return osutil.OpenExternal(filename)
}

func openFilemanagerPath(args *core.InternalCmdArgs, path string) error {
	var err error
	path, err = openPathOrCurrentDir(args, path)
	if err != nil {
		return err
	}

	filename, _, err := openResolvedPath(args, path)
	if err != nil {
		return err
	}
	return osutil.OpenFilemanager(filename)
}

func openTerminalPath(args *core.InternalCmdArgs, path string) error {
	var err error
	path, err = openPathOrCurrentDir(args, path)
	if err != nil {
		return err
	}

	filename, fi, err := openResolvedPath(args, path)
	if err != nil {
		return err
	}
	return osutil.OpenTerminal(openDirname(filename, fi))
}

func openTerminalEmuPath(args *core.InternalCmdArgs, path string, shellArgs []string) error {
	var err error
	path, err = openPathOrCurrentDir(args, path)
	if err != nil {
		return err
	}

	filename, fi, err := openResolvedPath(args, path)
	if err != nil {
		return err
	}
	return core.StartTerminalEmu(args.Ed, openDirname(filename, fi), args.Ed.GoodRowPos(), "", shellArgs)
}

func openResolvedPath(args *core.InternalCmdArgs, path string) (string, os.FileInfo, error) {
	filePos := openFilePos(path)
	filePos.Filename = args.Ed.HomeVars.Decode(filePos.Filename)

	dir, err := openCurrentDirOrWd(args)
	if err != nil {
		return "", nil, err
	}

	filename, fi, ok := core.FindFileInfo(filePos.Filename, dir)
	if !ok {
		return "", nil, fmt.Errorf("fileinfo not found: %q", filePos.Filename)
	}
	return filename, fi, nil
}

func openPathOrCurrentRowName(args *core.InternalCmdArgs, path string) (string, error) {
	if path != "" {
		return path, nil
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return "", err
	}
	if erow.Info.IsSpecial() {
		return "", fmt.Errorf("can't run on special row")
	}
	return erow.Info.Name(), nil
}

func openPathOrCurrentDir(args *core.InternalCmdArgs, path string) (string, error) {
	if path != "" {
		return path, nil
	}
	return openCurrentDirOrWd(args)
}

func openCurrentDirOrWd(args *core.InternalCmdArgs) (string, error) {
	erow, ok := args.ERow()
	if ok && !erow.Info.IsSpecial() {
		return erow.Info.Dir(), nil
	}
	return os.Getwd()
}

func openDirname(filename string, fi os.FileInfo) string {
	if fi.IsDir() {
		return filename
	}
	return filepath.Dir(filename)
}

func openFilePos(s string) *parseutil.FilePos {
	rd := iorw.NewStringReaderAt(s)
	rl, err := reslocparser.ParseResLoc2(rd, len(s))
	if err == nil && rl.Pos == 0 && rl.End == len(s) {
		fp := reslocparser.ResLocToFilePos(rl)
		fp.Filename = parseutil.RemoveFilenameEscapes(fp.Filename, rl.Escape, rl.PathSep)
		return fp
	}

	filename := parseutil.RemoveFilenameEscapes(s, osutil.EscapeRune, os.PathSeparator)
	return &parseutil.FilePos{
		Filename: filename,
		Offset:   -1,
	}
}
