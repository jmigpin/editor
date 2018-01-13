package cmdutil

import (
	"strings"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

func SetupLayoutHomeVars(ed Editorer) {
	_ = NewLayoutHomeVars(ed)
}

type LayoutHomeVars struct {
	ed Editorer

	entries []string
}

func NewLayoutHomeVars(ed Editorer) *LayoutHomeVars {
	lhv := &LayoutHomeVars{ed: ed}

	ed.UI().Root.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaSetStrEvent)
		lhv.update(ev)
	})

	return lhv
}

func (lhv *LayoutHomeVars) update(ev *ui.TextAreaSetStrEvent) {
	tb := lhv.ed.UI().Root.Toolbar

	entries := lhv.getEntries(tb.Str())

	// check if there are any changes
	changes := false
	if len(entries) != len(lhv.entries) {
		changes = true
	} else {
		for i, _ := range entries {
			if entries[i] != lhv.entries[i] {
				changes = true
				break
			}
		}
	}

	if !changes {
		return
	}

	// get all decoded toolbars in all rows
	m := make(map[ERower]string)
	for _, erow := range lhv.ed.ERowers() {
		td := erow.ToolbarData()
		decoded := td.StrWithPart0Arg0Decoded()
		m[erow] = decoded
	}

	hv := lhv.ed.HomeVars()

	// delete layout old home vars
	for i := 0; i < len(lhv.entries); i += 2 {
		hv.Delete(lhv.entries[i])
	}

	// append layout new home vars (the modified str)
	for i := 0; i < len(entries); i += 2 {
		hv.Append(entries[i], entries[i+1])
	}

	lhv.entries = entries

	// insert decoded toolbars and let triggers handle changes
	for erow, s := range m {
		erow.Row().Toolbar.SetStrClear(s, false, false)
	}
}
func (lhv *LayoutHomeVars) getEntries(str string) []string {
	var vars []string
	td := toolbardata.NewToolbarData(str, nil)
	for _, part := range td.Parts {
		if len(part.Args) != 1 {
			continue
		}
		str := part.Args[0].Str
		a := strings.Split(str, "=")
		if len(a) != 2 {
			continue
		}
		// var name: only 2 chars and must start with '~'
		if !(len(a[0]) == 2 && a[0][0] == '~') {
			continue
		}
		key, val := a[0], a[1]
		vars = append(vars, key, val)
	}
	return vars
}
