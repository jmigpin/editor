package main

import (
	//"flag"
	"jmigpin/editor/edit"
	//"log"
	//"os"
	//"runtime/pprof"
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

	edit.Main()
}
