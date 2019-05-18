package lsproto

import (
	"context"
	"testing"
)

func TestManager1(t *testing.T) {

	loc := "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor/core/lsproto/codec.go:91:16"
	f, l, c := parseLocation(t, loc)

	rd, offset := readBytesOffset(t, f, l, c)

	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)

	_, err := man.TextDocumentCompletion(ctx, f, rd, offset)
	if err != nil {
		t.Fatal(err)
	}
}

//----------

//func newTestManager2(t *testing.T) *Manager {
//	//if testing.Verbose() {
//	//	logger = log.New(os.Stdout, "", log.Lshortfile)
//	//}

//	//var wg sync.WaitGroup
//	asyncErrors := make(chan error, 10000)
//	go func() {
//		for {
//			err, ok := <-asyncErrors
//			if !ok {
//				break
//			}
//			t.Logf("asyncerr: %v", err)
//		}
//	}()
//	man := NewManager(asyncErrors)

//	// registrations
//	u := []string{
//		GoplsRegistrationStr,
//		CLangRegistrationStr,
//	}
//	for _, s := range u {
//		if err := man.RegisterStr(s); err != nil {
//			panic(err)
//		}
//	}

//	return man
//}
