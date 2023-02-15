package goutil

import (
	"testing"
)

func TestGoVersion1(t *testing.T) {
	v, err := GoVersion()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v\n", v)
}

func TestGoPath1(t *testing.T) {
	a := GoPath()
	t.Logf("%v\n", a)
}
