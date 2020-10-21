package contentcmds

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestOpenSession1(t *testing.T) {
	s := "aa OpenSession bb cc"
	rd := iorw.NewStringReaderAt(s)
	for i := 0; i < 10; i++ {
		sn, err := sessionName(rd, 17-i)
		if err != nil {
			t.Fatal(err)
		}
		if sn != "bb" {
			t.Fatal()
		}
	}
}
