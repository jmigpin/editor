package goutil

import (
	"testing"

	"github.com/jmigpin/editor/util/parseutil"
)

func TestGoVersion1(t *testing.T) {
	v, err := GoVersion()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v\n", v)

	if !parseutil.VersionLessThan(v, "20.21") {
		t.Fail()
	}
}

func TestGoPath1(t *testing.T) {
	a := GoPath()
	t.Logf("%v\n", a)
}
