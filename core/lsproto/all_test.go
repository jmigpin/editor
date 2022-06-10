package lsproto

//godebug:annotatepackage

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func TestGoSrc1(t *testing.T) {
	src0 := `
		package lsproto
		import "log"
		func main(){
			v1 := fn2()
			log.●Printf(v●1)
		}
		func f●n2() string {
			return "fn2"
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
	{
		src2 := `
			package lsproto
			import "log"
			func main(){
				v2 := fn2()
				log.Printf(v2)
			}
			func fn2() string {
				return "fn2"
			}
		`
		offset, src := sourceCursor(t, src0, 1)
		testSrcRename(t, "src.go", offset, src, "v2", src2)
	}
	{
		offset, src := sourceCursor(t, src0, 2)
		testSrcCallHierarchy(t, "src.go", offset, src)
	}
}

func TestGoSrc2(t *testing.T) {
	src0 := `
		package lsproto		
		func main(){
			ma●in2()
		}
		func main2() {
			println("testing")
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
	{
		src2 := `
			package lsproto		
			func main(){
				main3()
			}
			func main3() {
				println("testing")
			}
		`
		offset, src := sourceCursor(t, src0, 0)
		testSrcRename(t, "src.go", offset, src, "main3", src2)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCallHierarchy(t, "src.go", offset, src)
	}
}

//----------

func TestCSrc1(t *testing.T) {
	src0 := `
		#include <iostream>
		using namespace std;
		int main2(){
			return 3;
		}
		int main() {
			cout << "Hello, World! " << m●ain2();
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
	{
		src2 := `
			#include <iostream>
			using namespace std;
			int main3(){
				return 3;
			}
			int main() {
				cout << "Hello, World! " << main3();
				return 0;
			}
		`
		offset, src := sourceCursor(t, src0, 0)
		testSrcRename(t, "src.cpp", offset, src, "main3", src2)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCallHierarchy(t, "src.cpp", offset, src)
	}
}

//----------

func TestPythonSrc1(t *testing.T) {
	src0 := `
		#!/usr/bin/python3
		def main1(a):
			return main2(a+1)
		def main2(a):
			return a+1		
		c=m●ain1(1)
		print("val = %f" % c)
	`
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcDefinition(t, "src.py", offset, src)
	}
	{
		offset, src := sourceCursor(t, src0, 0)
		testSrcCompletion(t, "src.py", offset, src)
	}

	// TODO: failing, seems to not be implemented yet in pyls
	//{
	//	src2 := `
	//		#!/usr/bin/python3
	//		def main3(a):
	//			return main2(a+1)
	//		def main2(a):
	//			return a+1
	//		c=main3(1)
	//		print("val = %f" % c)
	//	`
	//	offset, src := sourceCursor(t, src0, 0)
	//	testSrcRename(t, "src.py", offset, src, "main3", src2)
	//}

	// TODO: not yet implemented in pylsp (method not found)
	//{
	//	offset, src := sourceCursor(t, src0, 0)
	//	testSrcCallHierarchy(t, "src.py", offset, src)
	//}
}

//----------

func TestManGoCompletionF1(t *testing.T) {
	goRoot := os.Getenv("GOROOT")
	s := filepath.Join(goRoot, "src/go/doc/doc.go:203:48")
	testFileLineColCompletion(t, s)
}
func TestManGoCompletionF2(t *testing.T) {
	goRoot := os.Getenv("GOROOT")
	s := filepath.Join(goRoot, "src/context/context.go:243:12")
	testFileLineColCompletion(t, s)
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

func testSrcRename(t *testing.T, filename string, offset int, src string, newName string, expectSrc string) {
	t.Helper()

	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename2 := tf.WriteFileInTmp2OrPanic(filename, src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	we, err := man.TextDocumentRename(ctx, filename2, rd, offset, newName)
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

		exp2 := parseutil.TrimLineSpaces(expectSrc)
		if res2 != exp2 {
			t.Fatal()
		}
	}
}

func testSrcCallHierarchy(t *testing.T, filename string, offset int, src string) {
	t.Helper()

	rd := iorw.NewStringReaderAt(src)

	tf := newTmpFiles(t)
	defer tf.RemoveAll()

	filename2 := tf.WriteFileInTmp2OrPanic(filename, src)

	man := newTestManager(t)
	defer man.Close()

	ctx := context.Background()
	mcalls, err := man.CallHierarchyCalls(ctx, filename2, rd, offset, IncomingChct)
	if err != nil {
		t.Fatal(err)
	}
	//spew.Dump(mcalls)
	if len(mcalls) == 0 {
		t.Fatal("empty mcalls")
	}

	str, err := ManagerCallHierarchyCallsToString(mcalls, IncomingChct, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %v", str)
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
