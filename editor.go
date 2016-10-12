package main

import (
	"fmt"
	"jmigpin/editor/edit"
)

//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
func main() {
	//flag.Parse()
	//if *cpuprofile != "" {
	//f, err := os.Create(*cpuprofile)
	//if err != nil {
	//log.Fatal(err)
	//}
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()
	//}

	_, err := edit.NewEditor()
	if err != nil {
		fmt.Println(err)
	}
}
