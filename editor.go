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
	fontFilenameFlag := flag.String("font", "", "ttf font filename")
	fontSizeFlag := flag.Float64("fontsize", 12, "")
	dpiFlag := flag.Float64("dpi", 72, "monitor dots per inch")
	scrollbarWidth := flag.Int("scrollbarwidth", 12, "textarea scrollbar width")
	acmeColors := flag.Bool("acmecolors", false, "acme editor color theme")
	wrapLineRune := flag.Int("wraplinerune", 8594, "code for wrap line rune")
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
		FontFilename:   *fontFilenameFlag,
		FontSize:       *fontSizeFlag,
		DPI:            *dpiFlag,
		ScrollbarWidth: *scrollbarWidth,
		AcmeColors:     *acmeColors,
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
