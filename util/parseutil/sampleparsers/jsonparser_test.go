package sampleparsers

////godebug:annotatepackage
////godebug:annotatepackage:github.com/jmigpin/editor/util/parseutil

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestJsonParser3CompareEncodingJson(t *testing.T) {
	src := []byte(jsonparserInput1)

	v1, err := ParseJson3(src)
	if err != nil {
		t.Fatal(err)
	}

	v2 := any(nil)
	if err := json.Unmarshal(src, &v2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("values differ\nbtparser: %#v\nencoding/json: %#v", v1, v2)
	}
}

func TestJsonParser2And3Parse(t *testing.T) {
	src := []byte(jsonparserInput1)

	if _, err := ParseJson2(src); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseJson3(src); err != nil {
		t.Fatal(err)
	}
}

//----------

func BenchmarkJsonParser2(b *testing.B) {
	s := jsonparserInput1
	for i := 0; i < b.N; i++ {
		v, err := ParseJson2([]byte(s))
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}

func BenchmarkJsonParser3(b *testing.B) {
	s := jsonparserInput1
	for i := 0; i < b.N; i++ {
		v, err := ParseJson3([]byte(s))
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}
