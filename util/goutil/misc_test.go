package goutil

import (
	"testing"
)

func TestGoVersion(t *testing.T) {
	//a, _ := GoVersion()
	//GoVersionLessThan(a, "go1.16")

	if r := GoVersionLessThan("go1.17", "go1.16"); r {
		t.Fail()
	}
	if r := GoVersionLessThan("go1.16", "go1.17"); !r {
		t.Fail()
	}
	if r := GoVersionLessThan("go1.17", "go1.161"); !r {
		t.Fail()
	}
	if r := GoVersionLessThan("go1.17", "go1.16.1"); r {
		t.Fail()
	}
	if r := GoVersionLessThan("go1.9", "go1.16"); !r {
		t.Fail()
	}
}
