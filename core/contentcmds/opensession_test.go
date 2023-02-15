package contentcmds

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestOpenSession1(t *testing.T) {
	s := "aa OpenSession bb\n cc"
	rd := iorw.NewStringReaderAt(s)
	for i := 7; i < 17; i++ {
		sn, err := sessionName(rd, i)
		if err != nil {
			t.Fatal("i=", i, "err=", err)
		}
		if sn != "bb" {
			t.Fatalf("i=%v, %q\n", i, sn)
		}
	}
}
