package edit

import (
	"github.com/jmigpin/editor/edit/contentcmd"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type ERow struct {
	ed  *Editor
	Row *ui.Row
}

func NewERow(ed *Editor, col *ui.Column) *ERow {
	row := col.NewRow()
	erow := &ERow{ed: ed, Row: row}
	erow.init()
	return erow
}
func (erow *ERow) init() {
	row := erow.Row
	ed := erow.ed

	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ToolbarCmdFromRow(ed, row)
		}})
	// toolbar str change (possible filename change)
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.rowStatus.OnRowToolbarSetStr(row)
		}})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			contentcmd.Cmd(ed, row)
		}})
	// textarea error
	row.TextArea.EvReg.Add(ui.TextAreaErrorEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			err := ev0.(error)
			ed.Error(err)
		}})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.rowStatus.OnRowTextAreaSetStr(row)
		}})
	// key shortcuts
	row.EvReg.Add(ui.RowKeyPressEventId,
		&xgbutil.ERCallback{ed.onRowKeyPress})
	// close
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.rowCtx.Cancel(row)
			ed.reopenRow.Add(row)
			ed.rowStatus.Remove(row)
			ed.updateFilesWatcher()
		}})

	ed.rowStatus.Add(row)
}
