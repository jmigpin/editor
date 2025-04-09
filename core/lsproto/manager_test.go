package lsproto

//godebug:annotatepackage

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/testutil"
)

func TestStruct1(t *testing.T) {
	{
		msg := `"abc"`
		doc := _completionItemDocumentation{}
		if err := json.Unmarshal([]byte(msg), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.str == nil || *doc.str != "abc" {
			spew.Dump(doc)
			t.Fail()
		}
	}
	{
		msg := `{"kind":"markup","value":"abc"}`
		doc := _completionItemDocumentation{}
		if err := json.Unmarshal([]byte(msg), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.mc == nil || doc.mc.Value != "abc" {
			spew.Dump(doc)
			t.Fail()
		}
	}

}

//----------
//----------
//----------

func TestScripts(t *testing.T) {
	log.SetFlags(0)
	//log.SetPrefix("lsptester: ")

	scr := testutil.NewScript(os.Args)
	scr.ScriptsDir = "testdata"
	//scr.Parallel = true // TODO: failing
	//scr.Work = true

	man := (*Manager)(nil)
	scr.ScriptStart = func(t *testing.T) error {
		man = newTestManager(t)
		return nil
	}
	scr.ScriptStop = func(t *testing.T) error {
		man.Stop()
		return nil
	}

	scr.Cmds = []*testutil.ScriptCmd{
		{"lspSourceCursor", func(st *testutil.ST, args []string) error {
			return lspSourceCursor(st, args, man)
		}},
		{"lspDefinition", func(st *testutil.ST, args []string) error {
			return lspDefinition(st, args, man)
		}},
		{"lspCompletion", func(st *testutil.ST, args []string) error {
			return lspCompletion(st, args, man)
		}},
		{"lspRename", func(st *testutil.ST, args []string) error {
			return lspRename(st, args, man)
		}},
		{"lspReferences", func(st *testutil.ST, args []string) error {
			return lspReferences(st, args, man)
		}},
		{"lspCallHierarchy", func(st *testutil.ST, args []string) error {
			return lspCallHierarchy(st, args, man)
		}},
	}

	scr.Run(t)
}

//----------

func lspSourceCursor(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 3 {
		return fmt.Errorf("sourcecursor: expecting 3 args: %v", args)
	}

	template := st.DirJoin(args[0])
	filename := st.DirJoin(args[1])
	mark := args[2]

	mark2, err := strconv.ParseInt(mark, 10, 32)
	if err != nil {
		return err
	}

	// read template
	b, err := os.ReadFile(template)
	if err != nil {
		return err
	}
	offset, src := sourceCursor(st.T, string(b), int(mark2))

	// write filename
	if err := os.WriteFile(filename, []byte(src), 0o644); err != nil {
		return err
	}

	st.Printf("%d", offset)

	return nil
}

//----------

func lspDefinition(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 2 {
		return fmt.Errorf("rename: expecting 2 args: %v", args)
	}

	filename := st.DirJoin(args[0])
	offset := args[1]

	// read offset (allow offset from env var)
	offset2, err := getIntArgPossiblyFromEnv(st, offset)
	if err != nil {
		return err
	}

	// read filename
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	rd := iorw.NewStringReaderAt(string(b))

	// full filename
	filename2, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	ctx := context.Background()
	f, rang, err := man.TextDocumentDefinition(ctx, filename2, rd, offset2)
	if err != nil {
		return err
	}
	st.Printf("%v %v", f, rang)
	return nil
}

//----------

func lspCompletion(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 2 {
		return fmt.Errorf("rename: expecting 2 args: %v", args)
	}

	filename := st.DirJoin(args[0])
	offset := args[1]

	// read offset (allow offset from env var)
	offset2, err := getIntArgPossiblyFromEnv(st, offset)
	if err != nil {
		return err
	}

	// read filename
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	rd := iorw.NewStringReaderAt(string(b))

	// full filename
	filename2, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	ctx := context.Background()
	clist, err := man.TextDocumentCompletion(ctx, filename2, rd, offset2)
	if err != nil {
		return err
	}
	w := CompletionListToString(clist)
	st.Printf("%v", w)
	return nil
}

//----------

func lspRename(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 3 {
		return fmt.Errorf("rename: expecting 3 args: %v", args)
	}

	filename := st.DirJoin(args[0])
	offsetVar := args[1]
	newName := args[2]

	// read offset (allow offset from env var)
	offset2, err := getIntArgPossiblyFromEnv(st, offsetVar)
	if err != nil {
		return err
	}

	// read filename
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	rd := iorw.NewStringReaderAt(string(b))

	// full filename
	filename2, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	ctx := context.Background()
	wecs, err := man.TextDocumentRenameAndPatch(ctx, filename2, rd, offset2, newName, nil)
	if err != nil {
		return err
	}
	for _, wec := range wecs {
		b, err := ioutil.ReadFile(wec.Filename)
		if err != nil {
			return err
		}
		st.Printf("filename: %v\n", wec.Filename)
		st.Printf("%s\n", b)
	}

	return nil
}

//----------

func lspReferences(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 2 {
		return fmt.Errorf("rename: expecting 2 args: %v", args)
	}

	filename := st.DirJoin(args[0])
	offsetVar := args[1]

	// read offset (allow offset from env var)
	offset2, err := getIntArgPossiblyFromEnv(st, offsetVar)
	if err != nil {
		return err
	}

	// read filename
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	rd := iorw.NewStringReaderAt(string(b))

	// full filename
	filename2, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	ctx := context.Background()
	locs, err := man.TextDocumentReferences(ctx, filename2, rd, offset2)
	if err != nil {
		return err
	}

	str, err := LocationsToString(locs, "")
	if err != nil {
		return err
	}
	st.Printf("%v", str)

	return nil
}

//----------

func lspCallHierarchy(st *testutil.ST, args []string, man *Manager) error {
	args = args[1:] // remove cmd string
	if len(args) != 2 {
		return fmt.Errorf("rename: expecting 2 args: %v", args)
	}

	filename := st.DirJoin(args[0])
	offsetVar := args[1]

	// read offset (allow offset from env var)
	offset2, err := getIntArgPossiblyFromEnv(st, offsetVar)
	if err != nil {
		return err
	}

	// read filename
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	rd := iorw.NewStringReaderAt(string(b))

	// full filename
	filename2, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	ctx := context.Background()
	mcalls, err := man.CallHierarchyCalls(ctx, filename2, rd, offset2, IncomingChct)
	if err != nil {
		return err
	}
	str, err := ManagerCallHierarchyCallsToString(mcalls, IncomingChct, "")
	if err != nil {
		return err
	}
	st.Printf("result: %v", str)

	return nil
}

//----------
//----------
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
		// WARNING: can't use stdio with stderr to be able to run scripts collectlog (use tcp if available)

		//GoplsRegistration(false, false,logTestVerbose()),
		GoplsRegistration(true, false, verboseLog()),

		//cLangRegistration("", false),
		//cLangRegistration("", logTestVerbose()),
		cLangRegistration("clangd-19", false),

		pylspRegistration(true, false),

		// dummy
		"dummy1,.dummy1,stdio,dummy_exec",
		"dummy2,.dummy2,tcp,dummy_exec",
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

func getIntArgPossiblyFromEnv(st *testutil.ST, v string) (int, error) {
	// read offset (allow offset from env var)
	if v2 := st.Env.Get(v); v2 != "" {
		v = strings.TrimSpace(v2)
	}

	u, err := strconv.ParseInt(v, 10, 32)
	return int(u), err
}

//----------

func sourceCursor(t *testing.T, src string, nth int) (int, string) {
	src2, index, err := testutil.SourceCursor("●", src, nth)
	if err != nil {
		t.Fatal(err)
	}
	return index, src2
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
