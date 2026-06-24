package internalcmds

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/fswatcher"
	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/iout"
)

//----------

func Version(args *core.InternalCmdArgs) error {
	args.Ed.Messagef("version: %v", core.Version())
	return nil
}

//----------

func FontsList(args *core.InternalCmdArgs) error {
	fonts := fontutil.FontsMan.LoadedFonts()
	fallbacks := fontutil.FontsMan.FallbackFonts()

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Fonts: %d\n", len(fonts))
	for _, f := range fonts {
		aliases := fontutil.FontsMan.Aliases(f.NameID())
		s := ""
		if len(aliases) > 0 {
			s = fmt.Sprintf(", aliases: %s", strings.Join(aliases, ", "))
		}
		fmt.Fprintf(&buf, "- %s (%s, %s%s)\n", f.NameID(), f.Name(), f.SrcName, s)
	}

	if len(fallbacks) > 0 {
		fmt.Fprintf(&buf, "\nFallback fonts: %d\n", len(fallbacks))
		for _, f := range fallbacks {
			aliases := fontutil.FontsMan.Aliases(f.NameID())
			s := ""
			if len(aliases) > 0 {
				s = fmt.Sprintf(", aliases: %s", strings.Join(aliases, ", "))
			}
			fmt.Fprintf(&buf, "- %s (%s, %s%s)\n", f.NameID(), f.Name(), f.SrcName, s)
		}
	}

	args.Ed.Messagef("%s", buf.String())
	return nil
}

//----------

func Exit(args *core.InternalCmdArgs) error {
	args.Ed.Close()
	return nil
}

func WindowTitle(args *core.InternalCmdArgs) error {
	args.Ed.UpdateWindowTitle()
	return nil
}

//----------

func SaveSession(args *core.InternalCmdArgs) error {
	core.SaveSession(args.Ed, args.Part)
	return nil
}
func SaveSessionFile(args *core.InternalCmdArgs) error {
	return core.SaveSessionFile(args.Ed, args.Part)
}
func OpenSession(args *core.InternalCmdArgs) error {
	core.OpenSession(args.Ed, args.Part)
	return nil
}
func OpenSessionFile(args *core.InternalCmdArgs) error {
	if len(args.Part.Args) != 2 {
		return fmt.Errorf("opensessionfile: missing session filename")
	}
	filename := args.Part.Args[1].UnquotedString()
	return core.OpenSessionFromFile(args.Ed, filename)
}
func DeleteSession(args *core.InternalCmdArgs) error {
	core.DeleteSession(args.Ed, args.Part)
	return nil
}
func ListSessions(args *core.InternalCmdArgs) error {
	core.ListSessions(args.Ed)
	return nil
}

//----------

func NewColumn(args *core.InternalCmdArgs) error {
	args.Ed.NewColumn()
	return nil
}
func CloseColumn(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	erow.Row.Col.Close()
	return nil
}

//----------

func CloseRow(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	erow.Row.Close()
	return nil
}
func ReopenRow(args *core.InternalCmdArgs) error {
	args.Ed.RowReopener.Reopen()
	return nil
}
func MaximizeRow(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	erow.Row.Maximize()
	return nil
}

//----------

func Save(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	return erow.Info.SaveFile()
}
func SaveAllFiles(args *core.InternalCmdArgs) error {
	var me iout.MultiError
	for _, info := range args.Ed.ERowInfos() {
		if info.IsFileButNotDir() {
			me.Add(info.SaveFile())
		}
	}
	return me.Result()
}

//----------

func Reload(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	return erow.Reload()
}
func ReloadAllFiles(args *core.InternalCmdArgs) error {
	me := &iout.MultiError{}
	for _, info := range args.Ed.ERowInfos() {
		if info.IsFileButNotDir() {
			me.Add(info.ReloadFile())
		}
	}
	return me.Result()
}
func ReloadAll(args *core.InternalCmdArgs) error {
	// reload all dirs erows
	me := &iout.MultiError{}
	for _, info := range args.Ed.ERowInfos() {
		if info.IsDir() {
			for _, erow := range info.ERows {
				me.Add(erow.Reload())
			}
		}
	}

	me.Add(ReloadAllFiles(args))

	return me.Result()
}

//----------

func Stop(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	erow.Exec.Stop()
	return nil
}

//----------

func Clear(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	erow.Row.TextArea.SetStrClearHistory("")
	return nil
}

//----------

func OpenFilemanager(args *core.InternalCmdArgs) error {
	opts := newOpenOptions()
	*opts.filemanagerMode = true
	return openRun(args, opts)
}

func OpenTerminalExternal(args *core.InternalCmdArgs) error {
	opts := newOpenOptions()
	*opts.terminalMode = true
	return openRun(args, opts)
}

func OpenTerminalEmu(args *core.InternalCmdArgs) error {
	opts := newOpenOptions()
	*opts.terminalEmuMode = true
	shellArgs := []string{}
	if len(args.Part.Args) > 1 {
		for _, arg := range args.Part.Args[1:] {
			shellArgs = append(shellArgs, arg.UnquotedString())
		}
	}
	opts.args = shellArgs
	return openRun(args, opts)
}

func OpenExternal(args *core.InternalCmdArgs) error {
	opts := newOpenOptions()
	*opts.externalMode = true
	return openRun(args, opts)
}

//----------

func GoDebug(args *core.InternalCmdArgs) error {
	args2 := args.Part.ArgsUnquoted()

	// special case: show help
	cmd := godebug.NewCmd()
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	if err := cmd.ParseFlagsOnce(args2[1:]); errors.Is(err, flag.ErrHelp) {
		return fmt.Errorf("%w\n%v", err, buf.String())
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	return args.Ed.GoDebug.RunAsync(args.Ctx, erow, args2)
}

func GoDebugFind(args *core.InternalCmdArgs) error {
	// TODO: erow needed?
	//erow, err := args.ERowOrErr()
	//if err != nil {
	//	return err
	//}

	a := args.Part.ArgsUnquoted()
	if len(a) < 2 {
		return fmt.Errorf("missing string to find")
	}
	s := ""
	if len(a) == 2 {
		s = a[1] // single arg, unquoted if quoted
	} else {
		s = args.Part.FromArgString(1) // verbatim
	}

	return args.Ed.GoDebug.AnnotationFind(s)
}

func GoDebugTrace(args *core.InternalCmdArgs) error {
	return args.Ed.GoDebug.Trace()
}

//----------

func ColorTheme(args *core.InternalCmdArgs) error {
	ui.ColorThemeCycler.Cycle(args.Ed.UI.Root)
	args.Ed.UI.Root.MarkNeedsLayoutAndPaint()
	return nil
}
func FontTheme(args *core.InternalCmdArgs) error {
	ui.FontThemeCycler.Cycle(args.Ed.UI.Root)
	args.Ed.UI.Root.MarkNeedsLayoutAndPaint()
	return nil
}

//----------

func FontRunes(args *core.InternalCmdArgs) error {
	var u string
	for i := 0; i < 15000; {
		start := i
		var w string
		for j := 0; j < 25; j++ {
			w += string(rune(i))
			i++
		}
		u += fmt.Sprintf("%d: %s\n", start, w)
	}
	args.Ed.Messagef("%s", u)
	return nil
}

//----------

func LSProtoCloseAll(args *core.InternalCmdArgs) error {
	man := args.Ed.LSProtoMan
	if man.NInstances() == 0 {
		return fmt.Errorf("no instances are running")
	}
	args.Ed.LSProtoMan.Stop()
	return nil
}
func CtxutilCallsState(args *core.InternalCmdArgs) error {
	s := ctxutil.CallsState()
	args.Ed.Messagef("%s", s)
	return nil
}

func WatcherState(args *core.InternalCmdArgs) error {
	gw, ok := args.Ed.Watcher.(*fswatcher.GWatcher)
	if !ok {
		return fmt.Errorf("watcher debug unavailable for type %T", args.Ed.Watcher)
	}

	args.Ed.Message(gw.DebugWatchState())
	return nil
}
