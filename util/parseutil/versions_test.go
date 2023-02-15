package parseutil

import (
	"testing"
)

func TestVersions(t *testing.T) {
	if VersionLessThan("1.17", "1.16") {
		t.Fail()
	}
	if !VersionLessThan("1.16", "1.17") {
		t.Fail()
	}
	if !VersionLessThan("1.17", "1.161") {
		t.Fail()
	}
	if VersionLessThan("1.17", "1.16.1") {
		t.Fail()
	}
	if !VersionLessThan("1.9", "1.16") {
		t.Fail()
	}
	if !VersionLessThan("1.90", "20.21") {
		t.Fail()
	}
}
