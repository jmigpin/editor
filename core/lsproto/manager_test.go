package lsproto

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmigpin/editor/util/iout"
)

func TestManagerGo1(t *testing.T) {
	goRoot := os.Getenv("GOROOT")
	loc := filepath.Join(goRoot, "src/context/context.go:241:1")
	f, l, c := parseLocation(t, loc)

	rw, offset := readBytesOffset(t, f, l, c)

	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)
	defer man.Close()

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comps, err := man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) == 0 {
		t.Fatal(comps)
	}

	// change content
	if err := rw.OverwriteAt(offset-11, 0, []byte("\n\n\n")); err != nil {
		t.Fatal(err)
	}
	offset += 33 // 3 newlines

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comps, err = man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) != 0 {
		t.Fatal(comps)
	}
}

func TestManagerGo2(t *testing.T) {
	goRoot := filepath.Join(os.Getenv("GOROOT"), "src")
	loc := filepath.Join(goRoot, "context/context.go:243:12")
	f, l, c := parseLocation(t, loc)

	rw, offset := readBytesOffset(t, f, l, c)

	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)
	defer man.Close()

	// ensure the lsp server runs
	comps, err := man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) == 0 {
		t.Fatal(comps)
	}

	// test closing
	if err := man.Close(); err != nil {
		t.Logf("man close err: %v", err)
	}
}

func TestManagerGo3(t *testing.T) {
	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)
	defer man.Close()

	// loc1
	{
		goRoot := os.Getenv("GOROOT")
		loc1 := filepath.Join(goRoot, "src/context/context.go:243:12")
		f, l, c := parseLocation(t, loc1)
		rw, offset := readBytesOffset(t, f, l, c)
		comps, err := man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
		if err != nil {
			t.Fatal(err)
		}
		if len(comps) == 0 {
			t.Fatal(comps)
		}
	}

	// loc2
	{
		goRoot := os.Getenv("GOROOT")
		loc2 := filepath.Join(goRoot, "src/go/doc/comment.go:128:27")
		f, l, c := parseLocation(t, loc2)
		rw, offset := readBytesOffset(t, f, l, c)
		comps, err := man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
		if err != nil {
			t.Fatal(err)
		}
		if len(comps) == 0 {
			t.Fatal(comps)
		}
	}
}

//----------

func TestManagerC1(t *testing.T) {
	loc := "/usr/include/X11/Xcursor/Xcursor.h:307:25"
	f, l, c := parseLocation(t, loc)

	rw, offset := readBytesOffset(t, f, l, c)

	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)
	defer man.Close()

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comps, err := man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) == 0 {
		t.Fatal(comps)
	}

	// change content
	if err := rw.OverwriteAt(offset-37, 0, []byte("\n\n\n")); err != nil {
		t.Fatal(err)
	}
	offset += 3 // 3 newlines

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comps, err = man.TextDocumentCompletionDetailStrings(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) == 0 {
		t.Fatal(comps)
	}
}

//----------

func newTestManager(t *testing.T) *Manager {
	t.Helper()

	msgFn := func(s string) {
		t.Helper()
		// can't use t.Log if already out of the test
		logPrintf("manager async msg: %v", s)
	}
	w := iout.FnWriter(func(p []byte) (int, error) {
		msgFn(string(p))
		return len(p), nil
	})

	man := NewManager(msgFn)
	man.serverWrapW = w

	// lang registrations
	u := []string{
		GoplsRegistration(logTestVerbose(), false, false),
		cLangRegistration(logTestVerbose()),
	}
	for _, s := range u {
		reg, err := NewRegistration(s)
		if err != nil {
			panic(err)
		}
		if err := man.Register(reg); err != nil {
			panic(err)
		}
	}

	return man
}
