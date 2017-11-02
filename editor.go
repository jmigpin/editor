// Source code editor in pure Go.
package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/jmigpin/editor/core"
)

func main() {
	log.SetFlags(0)
	//log.SetFlags(log.Llongfile)

	// flags
	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")
	fontFlag := flag.String("font", "regular", "ttf font filename or: regular, medium, mono")
	fontSizeFlag := flag.Float64("fontsize", 12, "")
	dpiFlag := flag.Float64("dpi", 72, "monitor dots per inch")
	scrollbarWidth := flag.Int("scrollbarwidth", 12, "textarea scrollbar width")
	colorTheme := flag.String("colortheme", "light", "available: light, dark, acme")
	wrapLineRune := flag.Int("wraplinerune", 8594, "code for wrap line rune, can be set to zero for a space of half line height")
	tabWidth := flag.Int("tabwidth", 8, "")
	scrollbarLeft := flag.Bool("scrollbarleft", true, "set scrollbars on the left side")
	sessionName := flag.String("sessionname", "", "open existing session")

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
		Font:           *fontFlag,
		FontSize:       *fontSizeFlag,
		DPI:            *dpiFlag,
		ScrollbarWidth: *scrollbarWidth,
		ColorTheme:     *colorTheme,
		WrapLineRune:   *wrapLineRune,
		TabWidth:       *tabWidth,
		ScrollbarLeft:  *scrollbarLeft,
		SessionName:    *sessionName,
		Filenames:      flag.Args(),
	}
	_, err := core.NewEditor(eopt)
	if err != nil {
		log.Fatal(err)
	}
}
