// Source code editor in pure Go.
package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/jmigpin/editor/edit"
)

func main() {
	log.SetFlags(log.Llongfile)

	// flags
	cpuProfileFlag := flag.String("cpuprofile", "", "profile cpu filename")
	fontFilenameFlag := flag.String("font", "", "font filename")
	fontSizeFlag := flag.Float64("fontsize", 12, "")
	dpiFlag := flag.Float64("dpi", 72, "monitor dots per inch")

	flag.Parse()

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	eopt := &edit.Options{
		FontFilename: *fontFilenameFlag,
		FontSize:     *fontSizeFlag,
		DPI:          *dpiFlag,
	}
	_, err := edit.NewEditor(eopt)
	if err != nil {
		log.Fatal(err)
	}
}
