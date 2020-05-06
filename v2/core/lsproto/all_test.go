package lsproto

//godebug:annotatepackage

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jmigpin/editor/v2/util/goutil"
	"github.com/jmigpin/editor/v2/util/iout/iorw"
	"github.com/jmigpin/editor/v2/util/osutil"
	"github.com/jmigpin/editor/v2/util/parseutil"
)

func TestGoSrc1(t *testing.T) {
	src0 := `
		package lsproto
		import "log"
		func main(){
			log.●Printf("aaa")
		}	
	`
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcDefinition(t, "src.go", offset, src)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCompletion(t, "src.go", offset, src)
	}
}

func TestGoSrc2(t *testing.T) {
	src0 := `
		package lsproto
		import "log"
		func main(){
			log.P●rintf("aaa")
		}	
	`
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcDefinition(t, "src.go", offset, src)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCompletion(t, "src.go", offset, src)
	}
}

//----------

func TestCSrc1(t *testing.T) {
	src0 := `
		#include <iostream>
		using namespace std;
		int main() {
			co●ut << "Hello, World!";
			return 0;
		}
	`
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcDefinition(t, "src.cpp", offset, src)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCompletion(t, "src.cpp", offset, src)
	}
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

func TestRenameGo(t *testing.T) {
	src0 := `
		package main
		func main() {
			a●aa := 1
			println(aaa)
		}
	`

	exp := `
		package main
		func main() {
			bbb := 1
			println(bbb)
		}
	`
	exp2 := parseutil.TrimLineSpaces(exp)

	offset, src := sourceCursor(t, src0, 0)
	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename := tf.WriteFileInTmp2OrPanic("src.go", src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	we, err := man.TextDocumentRename(ctx, filename, rd, offset, "bbb")
	if err != nil {
		t.Fatal(err)
	}

	wecs, err := WorkspaceEditChanges(we)
	if err != nil {
		t.Fatal(err)
	}

	if err := PatchWorkspaceEditChanges(wecs); err != nil {
		t.Fatal(err)
	}
	for _, wec := range wecs {
		b, err := ioutil.ReadFile(wec.Filename)
		if err != nil {
			t.Fatal(err)
		}
		res2 := parseutil.TrimLineSpaces(string(b))
		t.Log(res2)
		if res2 != exp2 {
			t.Fatal()
		}
	}
}

func TestRenameC(t *testing.T) {
	src0 := `
		#include <iostream>
		using namespace std;
		int main() {
			int a●aa = 0;
			cout<<" "<<a●aa;
			return 0;
		}
	`

	exp := `
		#include <iostream>
		using namespace std;
		int main() {
			int bbb = 0;
			cout<<" "<<bbb;
			return 0;
		}
	`
	exp2 := parseutil.TrimLineSpaces(exp)

	offset, src := sourceCursor(t, src0, 1)
	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename := tf.WriteFileInTmp2OrPanic("src.c", src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	we, err := man.TextDocumentRename(ctx, filename, rd, offset, "bbb")
	if err != nil {
		t.Fatal(err)
	}

	wecs, err := WorkspaceEditChanges(we)
	if err != nil {
		t.Fatal(err)
	}

	if err := PatchWorkspaceEditChanges(wecs); err != nil {
		t.Fatal(err)
	}
	for _, wec := range wecs {
		b, err := ioutil.ReadFile(wec.Filename)
		if err != nil {
			t.Fatal(err)
		}
		res2 := parseutil.TrimLineSpaces(string(b))
		t.Log(res2)
		if res2 != exp2 {
			t.Fatal()
		}
	}
}

//----------
//----------
//----------

func testSrcDefinition(t *testing.T, filename string, offset int, src string) {
	t.Helper()

	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename2 := tf.WriteFileInTmp2OrPanic(filename, src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	f, rang, err := man.TextDocumentDefinition(ctx, filename2, rd, offset)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v %v", f, rang)
}

func testSrcCompletion(t *testing.T, filename string, offset int, src string) {
	t.Helper()

	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename2 := tf.WriteFileInTmp2OrPanic(filename, src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	comps, err := man.TextDocumentCompletionDetailStrings(ctx, filename2, rd, offset)
	if err != nil {
		t.Fatal(err)
	}
	if !(len(comps) >= 1) {
		t.Fatal(comps)
	}
	t.Logf("%v\n", strings.Join(comps, "\n"))
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
	rw := iorw.NewBytesReadWriterAt(b)
	offset, err := parseutil.LineColumnIndex(rw, l, c)
	if err != nil {
		t.Fatal(err)
	}

	testSrcCompletion(t, filename, offset, string(b))
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
	rd := iorw.NewStringReaderAt(loc)
	res, err := parseutil.ParseResource(rd, 0)
	if err != nil {
		t.Fatal(err)
	}
	return res.Path, res.Line, res.Column
}

func readBytesOffset(t *testing.T, filename string, line, col int) (iorw.ReadWriterAt, int) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	rw := iorw.NewBytesReadWriterAt(b)
	offset, err := parseutil.LineColumnIndex(rw, line, col)
	if err != nil {
		t.Fatal(err)
	}
	return rw, offset
}

//----------

func newTmpFiles(t *testing.T) *osutil.TmpFiles {
	t.Helper()
	tf := osutil.NewTmpFiles("editor_lsproto_tests_tmpfiles")
	t.Logf("tf.Dir: %v\n", tf.Dir)
	return tf
}
