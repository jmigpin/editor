// Source code editor in pure Go.
package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/jmigpin/editor/core"
	_ "github.com/jmigpin/editor/core/contentcmds"
)

func init() {
	log.SetFlags(0)
}

func main() {
	// flags
	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")
	fontFlag := flag.String("font", "regular", "font: regular, medium, mono, or a filename")
	fontSizeFlag := flag.Float64("fontsize", 12, "")
	fontHintingFlag := flag.String("fonthinting", "full", "font hinting: none, vertical, full")
	dpiFlag := flag.Float64("dpi", 72, "monitor dots per inch")
	scrollBarWidthFlag := flag.Int("scrollbarwidth", 0, "Textarea scrollbar width in pixels. A value of 0 takes 3/4 of the font size.")
	scrollBarLeftFlag := flag.Bool("scrollbarleft", true, "set scrollbars on the left side")
	colorThemeFlag := flag.String("colortheme", "light", "available: light, dark, acme")
	commentsColorFlag := flag.Int("commentscolor", 0, "Colorize comments. Can be set to zero to use a percentage of the font color. Ex: 0=auto, 1=Black, 0xff0000=red.")
	wrapLineRuneFlag := flag.Int("wraplinerune", 8594, "code for wrap line rune, can be set to zero")
	tabWidthFlag := flag.Int("tabwidth", 8, "")
	shadowsFlag := flag.Bool("shadows", true, "shadow effects on some elements")
	sessionNameFlag := flag.String("sessionname", "", "open existing session")
	useMultiKeyFlag := flag.Bool("usemultikey", false, "use multi-key to compose characters ([multi-key, ~, a]=ã, ...)")
	pluginsFlag := flag.String("plugins", "", "comma separated string of plugin filenames")

	flag.Parse()

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	eopt := &core.Options{
		Font:        *fontFlag,
		FontSize:    *fontSizeFlag,
		FontHinting: *fontHintingFlag,
		DPI:         *dpiFlag,

		TabWidth:     *tabWidthFlag,
		WrapLineRune: *wrapLineRuneFlag,

		ColorTheme:     *colorThemeFlag,
		CommentsColor:  *commentsColorFlag,
		ScrollBarWidth: *scrollBarWidthFlag,
		ScrollBarLeft:  *scrollBarLeftFlag,
		Shadows:        *shadowsFlag,

		SessionName: *sessionNameFlag,
		Filenames:   flag.Args(),

		UseMultiKey: *useMultiKeyFlag,

		Plugins: *pluginsFlag,
	}
	_, err := core.NewEditor(eopt)
	if err != nil {
		log.Fatal(err)
	}
}
