package lsproto

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

//----------

func testSource1() string {
	return `
		package lsproto
		import "log"
		func main(){
			log.●Printf("aaa")
		}	
	`
}

func TestManSrc1Definition(t *testing.T) {
	// TODO: fails sometimes: gopls seems to be dependent on internal parsing some data to be able to make a decision to answer a query, even though the text was send first

	offset, src := sourceCursor(t, testSource1(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

func TestManSrc1Completion(t *testing.T) {
	offset, src := sourceCursor(t, testSource1(), 0)
	filename := "src.go"
	testSrcCompletion(t, filename, offset, src)
}

//----------

func testSource2() string {
	// NOTE: uses pkg main outside gopath (currently failing)
	return `
		package main
		import "log"
		func main(){
			log.P●rintf("aaa")
		}	
	`
}

func TestManSrc2Definition(t *testing.T) {
	offset, src := sourceCursor(t, testSource2(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

func TestManSrc2Completion(t *testing.T) {
	offset, src := sourceCursor(t, testSource2(), 0)
	filename := "src.go"
	testSrcDefinition(t, filename, offset, src)
}

//----------

func testSource3() string {
	return `
		#include <iostream>
		using namespace std;
		int main() {
			co●ut << "Hello, World!";
			return 0;
		}
	`
}

func TestManSrc3Definition(t *testing.T) {
	offset, src := sourceCursor(t, testSource3(), 0)
	filename := "src.cpp"
	testSrcDefinition(t, filename, offset, src)
}
func TestManSrc3Completion(t *testing.T) {
	offset, src := sourceCursor(t, testSource3(), 0)
	filename := "src.cpp"
	testSrcCompletion(t, filename, offset, src)
}

//----------

func TestManCompletionF1(t *testing.T) {
	s := "/home/jorge/lib/golang_packages/src/github.com/BurntSushi/xgb/xproto/xproto.go:140:23"
	testFileLineColCompletion(t, s)
}
func TestManCompletionF2(t *testing.T) {
	s := "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor/core/lsproto/client.go:167:14"
	testFileLineColCompletion(t, s)
}
func TestManCompletionF3(t *testing.T) {
	// NOTE: uses pkg main outside gopath (currently failing)
	s := "/home/jorge/tmp/test2.go:28:17"
	testFileLineColCompletion(t, s)
}

//----------

func testSrcDefinition(t *testing.T, filename string, offset int, src string) {
	rw := iorw.NewBytesReadWriter([]byte(src))

	man := newTestManager(t)
	defer man.Close()

	// repeat (syncs text a 2nd time)
	for i := 0; i < 2; i++ {
		f, rang, err := man.TextDocumentDefinition(filename, rw, offset)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%v %v", f, rang)
	}
}

//----------

func testFileLineColCompletion(t *testing.T, loc string) {
	// parse location
	rd := iorw.NewBytesReadWriter([]byte(loc))
	res, err := parseutil.ParseResource(rd, 0)
	if err != nil {
		t.Fatal(err)
	}
	filename, l, c := res.Path, res.Line, res.Column

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
	rw := iorw.NewBytesReadWriter([]byte(src))

	// start manager
	man := newTestManager(t)
	defer man.Close()

	//// sync file (optional)
	//if err := man.SyncText(filename, rw); err != nil {
	//	t.Fatal(err)
	//}

	comp, err := man.TextDocumentCompletion(filename, rw, offset)
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
	if testing.Verbose() {
		logger = log.New(os.Stdout, "", log.Lshortfile)
	}
	man := NewManager()

	// registrations
	u := []string{
		GoplsRegistrationStr,
		CLangRegistrationStr,
	}
	for _, s := range u {
		if err := man.RegisterStr(s); err != nil {
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

//----------
