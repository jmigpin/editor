/*
Build with:
$ go build -buildmode=plugin autocomplete.go
Start gocode
*/

package main

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func AutoComplete(ed *core.Editor, cfb *ui.ContextFloatBox) {
	ta, ok := cfb.FindTextAreaUnderPointer()
	if !ok {
		cfb.Hide()
		return
	}

	erow, ok := findERow(ed, ta)
	if ok {
		ok = autoCompleteERow(ed, cfb, erow)
		if ok {
			return
		}
	}

	cfb.SetRefPointToTextAreaCursor(ta)
	cfb.TextArea.SetStr("no results")
	return

}

func autoCompleteERow(ed *core.Editor, cfb *ui.ContextFloatBox, erow *core.ERow) bool {
	if erow.Info.IsFileButNotDir() && path.Ext(erow.Info.Name()) == ".go" {
		autoCompleteERowGolang(ed, cfb, erow)
		return true
	}
	return false
}

//----------

func autoCompleteERowGolang(ed *core.Editor, cfb *ui.ContextFloatBox, erow *core.ERow) {
	// timeout for the cmd to run
	timeout := 8000 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// gocode args
	filename := erow.Info.Name()
	offset := erow.Row.TextArea.TextCursor.Index()
	args := []string{"gocode", "autocomplete", fmt.Sprintf("%v", offset)}

	// textarea bytes
	bin, err := erow.Row.TextArea.Bytes()
	if err != nil {
		ed.Error(err)
		return
	}
	in := bytes.NewBuffer(bin)

	// execute external cmd
	dir := filepath.Dir(filename)
	bout, err := core.ExecCmdStdin(ctx, dir, in, args...)
	if err != nil {
		ed.Error(err)
		return
	}

	//// decode json
	//out := bytes.NewBuffer(bout)
	//dec := json.NewDecoder(out)
	//log.Println(string(bout))

	cfb.SetRefPointToTextAreaCursor(erow.Row.TextArea)
	cfb.TextArea.SetStr(string(bout))
	cfb.TextArea.ClearPos()
}

//----------

func findERow(ed *core.Editor, node widget.Node) (*core.ERow, bool) {
	for p := node.Embed().Parent; p != nil; p = p.Parent {
		if r, ok := p.Wrapper.(*ui.Row); ok {
			for _, erow := range ed.ERows() {
				if r == erow.Row {
					return erow, true
				}
			}
		}
	}
	return nil, false
}
