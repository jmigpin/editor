package main

import (
	"fmt"

	"github.com/jmigpin/editor/edit"
)

func main() {
	_, err := edit.NewEditor()
	if err != nil {
		fmt.Println(err)
	}
}
