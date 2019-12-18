package lsproto

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func testGoSource1() string {
	return `
		package lsproto
		import "log"
		func main(){
			log.●Printf("aaa")
		}	
	`
}

func TestManGoSrc1Definition(t *testing.T) {
	offset, src := sourceCursor(t, testGoSource1(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

func TestManGoSrc1Completion(t *testing.T) {
	offset, src := sourceCursor(t, testGoSource1(), 0)
	filename := "src.go"
	testSrcCompletion(t, filename, offset, src)
}

//----------

func testGoSource2() string {
	return `
		package lsproto
		import "log"
		func main(){
			log.P●rintf("aaa")
		}	
	`
}

func TestManGoSrc2Definition(t *testing.T) {
	offset, src := sourceCursor(t, testGoSource2(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

func TestManGoSrc2Completion(t *testing.T) {
	offset, src := sourceCursor(t, testGoSource2(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

//----------

func testCSource1() string {
	return `
		#include <iostream>
		using namespace std;
		int main() {
			co●ut << "Hello, World!";
			return 0;
		}
	`
}

func TestManCSrc1Definition(t *testing.T) {
	offset, src := sourceCursor(t, testCSource1(), 0)
	filename := "src.cpp"
	testSrcDefinition(t, filename, offset, src)
}
func TestManCSrc1Completion(t *testing.T) {
	offset, src := sourceCursor(t, testCSource1(), 0)
	filename := "src.cpp"
	testSrcCompletion(t, filename, offset, src)
}

//----------

func TestManGoCompletionF1(t *testing.T) {
	s := "/home/jorge/lib/golang_packages/src/github.com/BurntSushi/xgb/xproto/xproto.go:140:23"
	testFileLineColCompletion(t, s)
}
func TestManGoCompletionF2(t *testing.T) {
	s := "/home/jorge/lib/golang/go/src/context/context.go:242:12"
	testFileLineColCompletion(t, s)
}

//----------

func testSrcDefinition(t *testing.T, filename string, offset int, src string) {
	t.Helper()

	rd := iorw.NewStringReader(src)

	ctx := context.Background()
	//ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	//defer cancel()
	//ctx = ctx2

	man := newTestManager(t)
	defer man.Close()

	// pre-sync even thought completion might re-sync again
	//if err := man.SyncText(ctx, filename, rd); err != nil {
	//	t.Fatal(err)
	//}

	f, rang, err := man.TextDocumentDefinition(ctx, filename, rd, offset)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v %v", f, rang)
}

//----------

func testFileLineColCompletion(t *testing.T, loc string) {
	t.Helper()

	filename, l, c := parseLocation(t, loc)

	// read file to get offset
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	rw := iorw.NewBytesReadWriter(b)
	offset, err := parseutil.LineColumnIndex(rw, l, c)
	if err != nil {
		t.Fatal(err)
	}

	testSrcCompletion(t, filename, offset, string(b))
}

func testSrcCompletion(t *testing.T, filename string, offset int, src string) {
	t.Helper()

	rd := iorw.NewStringReader(src)

	ctx := context.Background()
	//ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	//defer cancel()
	//ctx = ctx2

	// start manager
	man := newTestManager(t)
	defer man.Close()

	// pre-sync even thought completion might re-sync again
	//if err := man.SyncText(ctx, filename, rd); err != nil {
	//	t.Fatal(err)
	//}

	comp, err := man.TextDocumentCompletion(ctx, filename, rd, offset)
	if err != nil {
		t.Fatal(err)
	}
	if !(len(comp) >= 1) {
		t.Fatal(comp)
	}
	t.Logf("%v", strings.Join(comp, "\n"))
}

//----------

func newTestManager(t *testing.T) *Manager {
	fnErr := func(err error) {
		//t.Log(err) // error if t.Log gets used after returning from func
		logPrintf("test: manager async error: %v", err)
	}
	man := NewManager(fnErr)

	// registrations
	u := []string{
		GoplsRegistration(logTestVerbose()),
		CLangRegistration(logTestVerbose()),
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

//----------

func sourceCursor(t *testing.T, src string, nth int) (int, string) {
	src2, index, err := goutil.SourceCursor("●", src, nth)
	if err != nil {
		t.Fatal(err)
	}
	return index, src2
}

func parseLocation(t *testing.T, loc string) (string, int, int) {
	rd := iorw.NewStringReader(loc)
	res, err := parseutil.ParseResource(rd, 0)
	if err != nil {
		t.Fatal(err)
	}
	return res.Path, res.Line, res.Column
}

func readBytesOffset(t *testing.T, filename string, line, col int) (iorw.ReadWriter, int) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	rw := iorw.NewBytesReadWriter(b)
	offset, err := parseutil.LineColumnIndex(rw, line, col)
	if err != nil {
		t.Fatal(err)
	}
	return rw, offset
}

//----------

func TestManagerGo1(t *testing.T) {
	goRoot := os.Getenv("GOROOT")
	loc := filepath.Join(goRoot, "src/context/context.go:242:12")
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

	comp, err := man.TextDocumentCompletion(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comp) == 0 {
		t.Fatal(comp)
	}

	// change content
	if err := rw.Insert(offset-11, []byte("\n\n\n")); err != nil {
		t.Fatal(err)
	}
	offset += 33 // 3 newlines

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comp, err = man.TextDocumentCompletion(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comp) != 0 {
		t.Fatal(comp)
	}
}

func TestManagerGo2(t *testing.T) {
	goRoot := filepath.Join(os.Getenv("GOROOT"), "src")
	loc := filepath.Join(goRoot, "context/context.go:242:12")
	f, l, c := parseLocation(t, loc)

	rw, offset := readBytesOffset(t, f, l, c)

	ctx0 := context.Background()
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	man := newTestManager(t)
	defer man.Close()

	// ensure the lsp server runs
	comp, err := man.TextDocumentCompletion(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comp) == 0 {
		t.Fatal(comp)
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
		goRoot := filepath.Join(os.Getenv("GOROOT"), "src")
		loc1 := filepath.Join(goRoot, "context/context.go:242:12")
		f, l, c := parseLocation(t, loc1)
		rw, offset := readBytesOffset(t, f, l, c)
		comp, err := man.TextDocumentCompletion(ctx, f, rw, offset)
		if err != nil {
			t.Fatal(err)
		}
		if len(comp) == 0 {
			t.Fatal(comp)
		}
	}

	// loc2
	{
		loc2 := "/home/jorge/lib/golang_packages/src/golang.org/x/image/vector/vector.go:115:14"
		f, l, c := parseLocation(t, loc2)
		rw, offset := readBytesOffset(t, f, l, c)
		comp, err := man.TextDocumentCompletion(ctx, f, rw, offset)
		if err != nil {
			t.Fatal(err)
		}
		if len(comp) == 0 {
			t.Fatal(comp)
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

	comp, err := man.TextDocumentCompletion(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comp) == 0 {
		t.Fatal(comp)
	}

	// change content
	if err := rw.Insert(offset-37, []byte("\n\n\n")); err != nil {
		t.Fatal(err)
	}
	offset += 3 // 3 newlines

	// pre sync text
	//if err := man.SyncText(ctx, f, rw); err != nil {
	//	t.Fatal(err)
	//}

	comp, err = man.TextDocumentCompletion(ctx, f, rw, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(comp) == 0 {
		t.Fatal(comp)
	}
}
