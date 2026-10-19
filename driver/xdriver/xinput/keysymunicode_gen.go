//go:build ignore

package main

import (
	"fmt"
	"go/format"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Run from xinput: go run keysymunicode_gen.go /usr/include/X11/keysymdef.h keysymunicode.go
func main() {
	if len(os.Args) != 3 {
		panic("usage: keysymunicode_gen <keysymdef.h> <output.go>")
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`(?m)^#define\s+XK_\w+\s+0x([0-9a-fA-F]+)\s+/\*\s*U\+([0-9a-fA-F]+)`)
	values := map[int]int{}
	for _, match := range re.FindAllSubmatch(src, -1) {
		ks, _ := strconv.ParseInt(string(match[1]), 16, 32)
		ru, _ := strconv.ParseInt(string(match[2]), 16, 32)
		if ks > 0xff && ks < 0xfe00 {
			values[int(ks)] = int(ru)
		}
	}
	keys := []int{}
	for ks := range values {
		keys = append(keys, ks)
	}
	sort.Ints(keys)
	b := &strings.Builder{}
	fmt.Fprintln(b, "// Code generated from X11 keysymdef.h Unicode annotations by keysymunicode_gen.go; DO NOT EDIT.")
	fmt.Fprintln(b, "package xinput\n\nvar legacyKeysymUnicode = map[uint16]rune{")
	for _, ks := range keys {
		fmt.Fprintf(b, "0x%x: 0x%x,\n", ks, values[ks])
	}
	fmt.Fprintln(b, "}")
	out, err := format.Source([]byte(b.String()))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[2], out, 0644); err != nil {
		panic(err)
	}
}
