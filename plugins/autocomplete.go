package main

import (
	"path"

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
		return autoCompleteERowGolang(ed, cfb, erow)
	}
	return false
}

//----------

func autoCompleteERowGolang(ed *core.Editor, cfb *ui.ContextFloatBox, erow *core.ERow) bool {
	//cfg := &packages.Config{
	//	Dir: erow.Info.Dir(),
	//}
	//patts := []string{"file=" + erow.Info.Name()}
	//pkgs, err := packages.Load(cfg, patts...)
	//if err != nil {
	//	ed.Error(err)
	//	return true
	//}
	//_ = pkgs
	//fmt.Printf("%v\n", pkgs)

	cfb.SetRefPointToTextAreaCursor(erow.Row.TextArea)
	cfb.TextArea.SetStr("golang: (todo: work-in-progress)")
	return true
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
