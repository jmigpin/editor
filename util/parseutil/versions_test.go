package parseutil

import (
	"testing"
)

func TestVersions(t *testing.T) {
	if r := VersionLessThan("1.17", "1.16"); r {
		t.Fail()
	}
	if r := VersionLessThan("1.16", "1.17"); !r {
		t.Fail()
	}
	if r := VersionLessThan("1.17", "1.161"); !r {
		t.Fail()
	}
	if r := VersionLessThan("1.17", "1.16.1"); r {
		t.Fail()
	}
	if r := VersionLessThan("1.9", "1.16"); !r {
		t.Fail()
	}
}
