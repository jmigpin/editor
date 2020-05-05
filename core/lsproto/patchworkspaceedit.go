package lsproto

import (
	"bytes"
	"io/ioutil"
	"sort"

	"github.com/jmigpin/editor/v2/util/iout/iorw"
	"github.com/jmigpin/editor/v2/util/parseutil"
)

type WorkspaceEditChange struct {
	Filename string
	Edits    []*TextEdit
}

func WorkspaceEditChanges(we *WorkspaceEdit) ([]*WorkspaceEditChange, error) {
	m := map[string]*WorkspaceEditChange{}
	for url, edits := range we.Changes {
		filename, err := parseutil.UrlToAbsFilename(string(url))
		if err != nil {
			return nil, err
		}
		m[filename] = &WorkspaceEditChange{filename, edits}
	}
	for _, tde := range we.DocumentChanges {
		filename, err := parseutil.UrlToAbsFilename(string(tde.TextDocument.Uri))
		if err != nil {
			return nil, err
		}
		m[filename] = &WorkspaceEditChange{filename, tde.Edits}
	}
	u := []*WorkspaceEditChange{}
	for _, v := range m {
		u = append(u, v)
	}
	return u, nil
}

//----------

func PatchWorkspaceEditChanges(wecs []*WorkspaceEditChange) error {
	for _, wec := range wecs {
		if err := PatchFileTextEdits(wec.Filename, wec.Edits); err != nil {
			return err
		}
	}
	return nil
}

func PatchFileTextEdits(filename string, edits []*TextEdit) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	res, err := PatchTextEdits(b, edits)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filename, res, 0644); err != nil {
		return err
	}
	return nil
}

func PatchTextEdits(src []byte, edits []*TextEdit) ([]byte, error) {
	sortTextEdits(edits)
	res := bytes.Buffer{} // resulting patched src
	rd := iorw.NewBytesReadWriterAt(src)
	start := 0
	for _, e := range edits {
		offset, n, err := RangeToOffsetLen(rd, &e.Range)
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
