package lsproto

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

//----------

var logger0 = log.New(os.Stdout, "", log.Lshortfile)

func logTestVerbose() bool {
	f := flag.Lookup("test.v")
	return f != nil && f.Value.String() == "true"
}

func logPrintf(f string, args ...interface{}) {
	if !logTestVerbose() {
		return
	}
	logger0.Output(2, fmt.Sprintf(f, args...))
}

func logJson(prefix string, v interface{}) {
	if !logTestVerbose() {
		return
	}
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	logger0.Output(2, fmt.Sprintf("%v%v", prefix, string(b)))
}

//----------

func encodeJson(a interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(a)
	if err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func decodeJson(r io.Reader, a interface{}) error {
	dec := json.NewDecoder(r)
	return dec.Decode(a)
}
func decodeJsonRaw(raw json.RawMessage, a interface{}) error {
	return json.Unmarshal(raw, a)
}

//----------

func Utf16Column(rd iorw.ReaderAt, lineStartOffset, utf8Col int) (int, error) {
	b, err := rd.ReadFastAt(lineStartOffset, utf8Col)
	if err != nil {
		return 0, err
	}
	return len(utf16.Encode([]rune(string(b)))), nil
}

// Input and result is zero based.
func Utf8Column(rd iorw.ReaderAt, lineStartOffset, utf16Col int) (int, error) {
	// ensure good limits
	n := utf16Col * 2
	if lineStartOffset+n > rd.Max() {
		n = rd.Max() - lineStartOffset
	}

	b, err := rd.ReadFastAt(lineStartOffset, n)
	if err != nil {
		return 0, err
	}

	enc := utf16.Encode([]rune(string(b)))
	if len(enc) < utf16Col {
		return 0, fmt.Errorf("encoded string smaller then utf16col")
	}
	nthChar := len(enc[:utf16Col])

	return nthChar, nil
}

//----------

func OffsetToPosition(rd iorw.ReaderAt, offset int) (Position, error) {
	l, c, err := parseutil.IndexLineColumn(rd, offset)
	if err != nil {
		return Position{}, err
	}
	// zero based
	l, c = l-1, c-1

	// character offset in utf16
	c2, err := Utf16Column(rd, offset-c, c)
	if err != nil {
		return Position{}, err
	}

	return Position{Line: l, Character: c2}, nil
}

func RangeToOffsetLen(rd iorw.ReaderAt, rang *Range) (int, int, error) {
	l1, _ := rang.Start.OneBased()
	l2, _ := rang.End.OneBased()

	// line start offset
	// TODO: improve getting lso2
	lso1, err := parseutil.LineColumnIndex(rd, l1, 1)
	if err != nil {
		return 0, 0, err
	}
	lso2, err := parseutil.LineColumnIndex(rd, l2, 1)
	if err != nil {
		return 0, 0, err
	}

	// translate utf16 columns to utf8 (input and results are zero based)
	u16c1, err := Utf8Column(rd, lso1, rang.Start.Character)
	if err != nil {
		return 0, 0, err
	}
	u16c2, err := Utf8Column(rd, lso2, rang.End.Character)
	if err != nil {
		return 0, 0, err
	}

	// start/end (range)
	start := lso1 + u16c1
	end := lso2 + u16c2

	offset := start
	length := end - start

	return offset, length, nil
}

//----------

func JsonGetPath(v interface{}, path string) (interface{}, error) {
	args := strings.Split(path, ".")
	return jsonGetPath2(v, args)
}

// TODO: incomplete
func jsonGetPath2(v interface{}, args []string) (interface{}, error) {
	// handle last arg
	if len(args) == 0 {
		switch t := v.(type) {
		case bool, int, float32, float64:
			return t, nil
		}
		return nil, fmt.Errorf("unhandled last type: %T", v)
	}
	// handle args: len(args)>0
	arg, args2 := args[0], args[1:]
	switch t := v.(type) {
	case map[string]interface{}:
		if v, ok := t[arg]; ok {
			return jsonGetPath2(v, args2)
		}
		return nil, fmt.Errorf("not found: %v", arg)
	}
	return nil, fmt.Errorf("unhandled type: %T (arg=%v)", v, arg)
}

//----------

func UrlToAbsFilename(url string) (string, error) {
	return parseutil.UrlToAbsFilename(url)
}

func AbsFilenameToUrl(filename string) (string, error) {
	if runtime.GOOS == "windows" {
		// gopls requires casing to match the OS names in windows (error: case mismatch in path ...)
		if u, err := osutil.FsCaseFilename(filename); err == nil {
			filename = u
		}
	}
	return parseutil.AbsFilenameToUrl(filename)
}

//----------

type ManagerCallHierarchyCalls struct {
	item  *CallHierarchyItem
	calls []*CallHierarchyCall
}

func ManagerCallHierarchyCallsToString(mcalls []*ManagerCallHierarchyCalls, typ CallHierarchyCallType, baseDir string) (string, error) {
	res := []string{}

	// build title
	s1 := "incoming"
	if typ == OutgoingChct {
		s1 = "outgoing"
	}
	u := fmt.Sprintf("lsproto call hierarchy %s calls:", s1)
	res = append(res, u)

	for _, mcall := range mcalls {
		// build subtitle
		s2 := "to"
		if typ == OutgoingChct {
			s2 = "from"
		}
		// count results for subtitle
		nres := 0
		for _, call := range mcall.calls {
			nres += len(call.FromRanges)
		}
		s3 := fmt.Sprintf("calls %s %v: %v results", s2, mcall.item.Name, nres)
		res = append(res, s3)

		res2 := []string{}
		for _, call := range mcall.calls {
			item := call.Item()

			// item for filename
			fileItem := item
			if typ == OutgoingChct {
				fileItem = mcall.item
			}
			filename, err := UrlToAbsFilename(string(fileItem.Uri))
			if err != nil {
				return "", err
			}
			// use basedir to output filename
			if baseDir != "" {
				if u, err := filepath.Rel(baseDir, filename); err == nil {
					filename = u
				}
			}

			for _, r := range call.FromRanges {
				line, col := r.Start.OneBased()
				u := fmt.Sprintf("\t%s:%d:%d: %s", filename, line, col, item.Name)
				res2 = append(res2, u)
			}
		}
		sort.Strings(res2)
		res = append(res, res2...)
	}
	w := strings.Join(res, "\n")
	return w, nil
}

//----------

func LocationsToString(locations []*Location, baseDir string) (string, error) {
	buf := &strings.Builder{}
	for _, loc := range locations {
		filename, err := UrlToAbsFilename(string(loc.Uri))
		if err != nil {
			return "", err
		}

		// use basedir to output filename
		if baseDir != "" {
			if u, err := filepath.Rel(baseDir, filename); err == nil {
				filename = u
			}
		}

		line, col := loc.Range.Start.OneBased()
		fmt.Fprintf(buf, "\t%v:%v:%v\n", filename, line, col)
	}
	return buf.String(), nil
}

//----------

func CompletionListToString(clist *CompletionList) []string {
	res := []string{}
	for _, ci := range clist.Items {
		u := []string{}
		if ci.Deprecated {
			u = append(u, "*deprecated*")
		}
		ci.Label = strings.TrimSpace(ci.Label) // NOTE: clangd is sending with spaces
		u = append(u, ci.Label)
		if ci.Detail != "" {
			u = append(u, ci.Detail)
		}
		res = append(res, strings.Join(u, " "))
	}

	//// add documentation if there is only 1 result
	//if len(compList.Items) == 1 {
	//	doc := compList.Items[0].Documentation
	//	if doc != "" {
	//		res[0] += "\n\n" + doc
	//	}
	//}

	return res
}

//----------

func PatchTextEdits(src []byte, edits []*TextEdit) ([]byte, error) {
	sortTextEdits(edits)
	res := bytes.Buffer{} // resulting patched src
	rd := iorw.NewBytesReadWriterAt(src)
	start := 0
	for _, e := range edits {
		offset, n, err := RangeToOffsetLen(rd, e.Range)
		if err != nil {
			return nil, err
		}
		res.Write(src[start:offset])
		res.Write([]byte(e.NewText))
		start = offset + n
	}
	res.Write(src[start:]) // rest of the src
	return res.Bytes(), nil
}

func sortTextEdits(edits []*TextEdit) {
	sort.Slice(edits, func(i, j int) bool {
		p1, p2 := &edits[i].Range.Start, &edits[j].Range.Start
		return p1.Line < p2.Line ||
			(p1.Line == p2.Line && p1.Character <= p2.Character)
	})
}

//----------
