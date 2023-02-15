package sampleparsers

//godebug:annotatepackage
//godebug:annotatepackage:github.com/jmigpin/editor/util/parseutil

import (
	"testing"
)

func TestJsonParser(t *testing.T) {
	//s := "{\"a\":{}}"
	s := jsonparserInput1
	v, err := ParseJson([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(v)

	//spew.Config.Indent = "\t"
	//spew.Dump(v)
}

//----------

func ParseJson(src []byte) (interface{}, error) {
	//p := NewJsonParser(src)
	//return p.parseJson()
	p := NewJsonParser2()
	return p.parseJson(src)
}

//----------
//----------
//----------

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
