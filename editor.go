// Source code editor in pure Go.
package main

import (
	"fmt"
	"log"

	"github.com/jmigpin/editor/edit"
)

func main() {
	log.SetFlags(log.Llongfile)
	_, err := edit.NewEditor()
	if err != nil {
		fmt.Println(err)
	}
}
