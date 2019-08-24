// Source code editor in pure Go.
package main

// build plugins (use "--plugins=<p1.so>,..." option to use these)
//go:generate go build -buildmode=plugin ./plugins/autocomplete_gocode.go
//go:generate go build -buildmode=plugin ./plugins/gotodefinition_godef.go
//go:generate go build -buildmode=plugin ./plugins/rownames.go
//go:generate go build -buildmode=plugin ./plugins/eevents.go

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/jmigpin/editor/core"
	_ "github.com/jmigpin/editor/core/contentcmds"
	"github.com/jmigpin/editor/core/lsproto"
)

func main() {
	// reset global flag var to prevent testing options from showing up if the testing package is imported (ex: using testing.Verbose())
	// TODO: probably not needed after go1.13 release, testing flags won't be added.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	opt := &core.Options{}

	// flags
	flag.StringVar(&opt.Font, "font", "regular", "font: regular, medium, mono, or a filename")
	flag.Float64Var(&opt.FontSize, "fontsize", 12, "")
	flag.StringVar(&opt.FontHinting, "fonthinting", "full", "font hinting: none, vertical, full")
	flag.Float64Var(&opt.DPI, "dpi", 72, "monitor dots per inch")
	flag.IntVar(&opt.TabWidth, "tabwidth", 8, "")
	flag.IntVar(&opt.WrapLineRune, "wraplinerune", int('←'), "code for wrap line rune, can be set to zero")
	flag.StringVar(&opt.ColorTheme, "colortheme", "light", "available: light, dark, acme")
	flag.IntVar(&opt.CommentsColor, "commentscolor", 0, "Colorize comments. Can be set to zero to use a percentage of the font color. Ex: 0=auto, 1=Black, 0xff0000=red.")
	flag.IntVar(&opt.StringsColor, "stringscolor", 0, "Colorize strings. Can be set to zero to not colorize. Ex: 0xff0000=red.")
	flag.IntVar(&opt.ScrollBarWidth, "scrollbarwidth", 0, "Textarea scrollbar width in pixels. A value of 0 takes 3/4 of the font size.")
	flag.BoolVar(&opt.ScrollBarLeft, "scrollbarleft", true, "set scrollbars on the left side")
	flag.BoolVar(&opt.Shadows, "shadows", true, "shadow effects on some elements")
	flag.StringVar(&opt.SessionName, "sessionname", "", "open existing session")
	flag.BoolVar(&opt.UseMultiKey, "usemultikey", false, "use multi-key to compose characters (Ex: [multi-key, ~, a] = ã)")
	flag.StringVar(&opt.Plugins, "plugins", "", "comma separated string of plugin filenames")

	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")

	flag.Var(&opt.LSProtos, "lsproto", "Language-server-protocol register options. Can be specified multiple times.\n"+
		"Format: language,extensions,network{tcp,tcpclient,stdio},cmd,optional{stderr}\n"+
		"Examples:\n"+lsproto.RegistrationExamples())

	flag.Parse()
	opt.Filenames = flag.Args()

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.SetFlags(log.Lshortfile)
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	_, err := core.NewEditor(opt)
	if err != nil {
		log.SetFlags(log.Lshortfile)
		log.Fatal(err)
	}
}
