// Source code editor in pure Go.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/fontutil"

	// imports that can't be imported from core (cyclic import)
	_ "github.com/jmigpin/editor/core/contentcmds"
	_ "github.com/jmigpin/editor/core/internalcmds"
)

func main() {
	// allow direct access to godebug on the cmd line
	if godebugMain() {
		return
	}

	opt := &core.Options{}

	// flags
	flag.Var(&opt.Fonts, "font", "regular, medium, mono, or a filename. Can be specified multiple times to add to the font theme cycler. If a filename is provided, it will automatically register as the \"regular\" or \"mono\" alias depending on its properties (the last one provided takes precedence).")
	flag.Var(&opt.FallbackFonts, "fontfallback", "font filename. Can be specified multiple times for glyph fallbacks.")
	flag.Float64Var(&opt.FontSize, "fontsize", 12, "")
	flag.StringVar(&opt.FontHinting, "fonthinting", "full", "font hinting: none, vertical, full")
	flag.Float64Var(&opt.DPI, "dpi", 72, "monitor dots per inch")
	flag.BoolVar(&opt.StartMaximized, "startmaximized", false, "maximize window at start")
	flag.IntVar(&opt.TabWidth, "tabwidth", 8, "")
	flag.IntVar(&opt.CarriageReturnRune, "carriagereturnrune", int(fontutil.CarriageReturnRune), "replacement rune for carriage return")
	flag.IntVar(&opt.WrapLineRune, "wraplinerune", int('←'), "code for wrap line rune, can be set to zero")
	flag.Float64Var(&opt.WrapLineIndentTabs, "wraplineindenttabs", 1.5, "number of tab widths used to indent wrapped lines")
	flag.IntVar(&opt.WrapWordLimit, "wrapwordlimit", 10, "wrap at word boundaries for words up to N runes. Set to zero to disable word wrap.")
	flag.BoolVar(&opt.CursorHalfHit, "cursorhalfhit", false, "place cursor before/after a rune depending on whether the pointer is in the left/right half of the rune cell. Defaults on with -textcursor=beam unless explicitly set.")
	flag.StringVar(&opt.TextCursor, "textcursor", "default", "textarea cursor: default or beam")
	flag.StringVar(&opt.ColorTheme, "colortheme", "light", "available: "+strings.Join(ui.ColorThemeCycler.Names(), ", "))
	flag.IntVar(&opt.CommentsColor, "commentscolor", 0, "Colorize comments. Can be set to 0x1 to not colorize. Ex: 0xff0000=red.")
	flag.IntVar(&opt.StringsColor, "stringscolor", 0, "Colorize strings. Can be set to 0x1 to not colorize. Ex: 0xff0000=red.")
	flag.IntVar(&opt.ScrollBarWidth, "scrollbarwidth", 0, "Textarea scrollbar width in pixels. A value of 0 takes 3/4 of the font size.")
	flag.BoolVar(&opt.ScrollBarLeft, "scrollbarleft", true, "set scrollbars on the left side")
	flag.BoolVar(&opt.Shadows, "shadows", false, "shadow effects on some elements")
	flag.StringVar(&opt.SessionName, "sn", "", "open existing session")
	flag.StringVar(&opt.SessionName, "sessionname", "", "open existing session")
	flag.StringVar(&opt.SessionFilename, "sessionfilename", "", "open a session snapshot from a file")
	flag.BoolVar(&opt.StartTerminalEmu, "startterminalemu", false, "open the editor in the current directory with an emulated terminal running")
	// TODO: escape emuexec robustly when mirroring it into the row toolbar.
	flag.StringVar(&opt.EmuExec, "emuexec", "", "shell command to run when starting with -startterminalemu")
	flag.BoolVar(&opt.UseMultiKey, "usemultikey", false, "use multi-key to compose characters (Ex: [multi-key, ~, a] = ã)")
	flag.StringVar(&opt.Plugins, "plugins", "", "comma separated string of plugin filenames")
	flag.Var(&opt.LSProtos, "lsproto", "Language-server-protocol register options. Can be specified multiple times.\nFormat: language,fileExtensions,network{tcp|tcpclient|stdio},command,optional{stderr,nogotoimpl}\nFormat notes:\n\tif network is tcp, the command runs in a template with vars: {{.Addr}}.\n\tif network is tcpclient, the command should be an ipaddress.\nExamples:\n\t"+strings.Join(lsproto.RegistrationExamples(), "\n\t"))
	flag.Var(&opt.PreSaveHooks, "presavehook", "Run program before saving a file. Uses stdin/stdout. Can be specified multiple times. By default, a \"goimports\" entry is auto added if no entry is defined for the \"go\" language.\nFormat: language,fileExtensions,cmd\nExamples:\n"+
		"\tgo,.go,goimports\n"+
		"\tcpp,\".cpp .hpp\",\"\\\"clang-format --style={'opt1':1,'opt2':2}\\\"\"\n"+
		"\tpython,.py,python_formatter")
	flag.BoolVar(&opt.ZipSessionsFile, "zipsessionsfile", false, "Save sessions in a zip. Useful for 100+ sessions. Does not delete the plain file. Beware that the file might not be easily editable as in a plain file.")
	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")
	version := flag.Bool("version", false, "output version and exit")

	flag.Parse()
	cursorHalfHitSet := flagWasSet("cursorhalfhit")
	if !cursorHalfHitSet {
		opt.CursorHalfHit = opt.TextCursor == "beam"
	}
	args := flag.Args()
	if opt.StartTerminalEmu && len(args) > 0 {
		opt.EmuExecArgs = args
	} else {
		opt.Filenames = args
	}

	log.SetFlags(log.Lshortfile)

	if *version {
		fmt.Printf("version: %v\n", core.Version())
		return
	}

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Println(err)
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if err := core.RunEditor(opt); err != nil {
		log.Println(err) // fatal
		os.Exit(1)
	}
}

func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

//----------

func godebugMain() bool {
	args := make([]string, len(os.Args))
	copy(args, os.Args)

	if len(args) <= 1 {
		return false
	}
	if args[1] == "--" {
		args = append(args[:1], args[2:]...)
	}
	if args[1] != "godebug" {
		return false
	}
	args = args[2:]
	if err := godebugMain2(args); err != nil {
		fmt.Fprintf(os.Stderr, "godebug error: %s\n", err)
		os.Exit(1)
	}
	return true
}
func godebugMain2(args []string) error {
	cmd := godebug.NewCmd()
	cmd.CmdLineMode = true
	ctx := context.Background()
	done, err := cmd.Start(ctx, args)
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	return cmd.Wait()
}
