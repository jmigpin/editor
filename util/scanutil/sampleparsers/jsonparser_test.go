package sampleparsers

//godebug:annotatepackage
//godebug:annotatepackage:github.com/jmigpin/editor/util/smparse

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestJsonParser(t *testing.T) {
	//s := "{\"a\":{}}"
	s := jsonparserInput1
	v, err := ParseJson([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	_ = v

	spew.Config.Indent = "\t"
	spew.Dump(v)
}

func BenchmarkJsonParser(b *testing.B) {
	s := jsonparserInput1
	for i := 0; i < b.N; i++ {
		v, err := ParseJson([]byte(s))
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}
