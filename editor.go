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
	log.SetFlags(log.Llongfile)

	// flags
	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")
	fontFilenameFlag := flag.String("font", "", "ttf font filename")
	fontSizeFlag := flag.Float64("fontsize", 12, "")
	dpiFlag := flag.Float64("dpi", 72, "monitor dots per inch")
	scrollbarWidth := flag.Int("scrollbarwidth", 12, "textarea scrollbar width")
	acmeColors := flag.Bool("acmecolors", false, "acme editor color theme")

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
	}
	_, err := core.NewEditor(eopt)
	if err != nil {
		log.Fatal(err)
	}
}
