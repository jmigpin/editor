package cmdutil

import (
	"strings"

	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xgbutil"
)

func SetupLayoutHomeVars(ed Editorer) {
	ed.UI().Layout.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ev := ev0.(*ui.TextAreaSetStrEvent)
			updateHomeVars(ed, ev)
		}})

}
func updateHomeVars(ed Editorer, ev *ui.TextAreaSetStrEvent) {
	tb := ed.UI().Layout.Toolbar

	// get all decoded toolbars in all rows
	m := make(map[ERower]string)
	for _, erow := range ed.ERows() {
		// decode toolbar
		str := erow.Row().Toolbar.Str()
		tbsd := toolbardata.NewStringData(str)
		decoded := tbsd.StrWithPart0Arg0Decoded()

		m[erow] = decoded
	}

	// delete layout old home vars
	oldVars := getLayoutHomeVars(ev.OldStr)
	for i := 0; i < len(oldVars); i += 2 {
		toolbardata.DeleteHomeVar(oldVars[i])
	}

	// append layout new home vars (the modified str)
	vars := getLayoutHomeVars(tb.Str())
	for i := 0; i < len(vars); i += 2 {
		toolbardata.AppendHomeVar(vars[i], vars[i+1])
	}

	// insert decoded toolbars and let triggers handle changes
	for erow, s := range m {
		erow.Row().Toolbar.SetStrClear(s, false, false)
	}
}
func getLayoutHomeVars(str string) []string {
	var vars []string
	tbsd := toolbardata.NewStringData(str)
	for _, part := range tbsd.Parts {
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
