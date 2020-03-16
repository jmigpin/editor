package internalcmds

import (
	"github.com/jmigpin/editor/core"
)

func init() {
	ic := core.InternalCmds

	ic.Set(&core.InternalCmd{"Exit", Exit, true, false})

	ic.Set(&core.InternalCmd{"SaveSession", SaveSession, true, false})
	ic.Set(&core.InternalCmd{"OpenSession", OpenSession, true, false})
	ic.Set(&core.InternalCmd{"DeleteSession", DeleteSession, true, false})
	ic.Set(&core.InternalCmd{"ListSessions", ListSessions, true, false})

	ic.Set(&core.InternalCmd{"NewColumn", NewColumn, true, false})
	ic.Set(&core.InternalCmd{"CloseColumn", CloseColumn, false, false})

	ic.Set(&core.InternalCmd{"NewRow", NewRow, true, false})
	ic.Set(&core.InternalCmd{"CloseRow", CloseRow, false, false})
	ic.Set(&core.InternalCmd{"ReopenRow", ReopenRow, true, false})
	ic.Set(&core.InternalCmd{"MaximizeRow", MaximizeRow, false, false})

	ic.Set(&core.InternalCmd{"NewFile", NewFile, false, false})
	ic.Set(&core.InternalCmd{"Save", Save, false, false})
	ic.Set(&core.InternalCmd{"SaveAllFiles", SaveAllFiles, true, false})

	ic.Set(&core.InternalCmd{"Reload", Reload, false, false})
	ic.Set(&core.InternalCmd{"ReloadAllFiles", ReloadAllFiles, true, false})
	ic.Set(&core.InternalCmd{"ReloadAll", ReloadAll, true, false})

	ic.Set(&core.InternalCmd{"Stop", Stop, false, false})
	ic.Set(&core.InternalCmd{"Clear", Clear, false, false})

	ic.Set(&core.InternalCmd{"Find", Find, false, false})
	ic.Set(&core.InternalCmd{"Replace", Replace, false, false})
	ic.Set(&core.InternalCmd{"GotoLine", GotoLine, false, false})
	ic.Set(&core.InternalCmd{"GoToLine", GotoLine, false, false})

	ic.Set(&core.InternalCmd{"CopyFilePosition", CopyFilePosition, false, false})
	ic.Set(&core.InternalCmd{"RuneCodes", RuneCodes, false, false})
	ic.Set(&core.InternalCmd{"FontRunes", FontRunes, false, false})

	// Deprecated: in favor of "OpenFilemanager"
	ic.Set(&core.InternalCmd{"XdgOpenDir", OpenFilemanager, false, false})
	ic.Set(&core.InternalCmd{"OpenFilemanager", OpenFilemanager, false, false})

	ic.Set(&core.InternalCmd{"ListDir", ListDir, false, false})

	ic.Set(&core.InternalCmd{"GoRename", GoRename, false, false})
	ic.Set(&core.InternalCmd{"GoDebug", GoDebug, false, false})

	// Deprecated: in favor of "LspCloseAll"
	ic.Set(&core.InternalCmd{"LSProtoCloseAll", LSProtoCloseAll, false, false})
	ic.Set(&core.InternalCmd{"LsprotoCloseAll", LSProtoCloseAll, false, false})
	ic.Set(&core.InternalCmd{"LsprotoRename", LSProtoRename, false, true})

	ic.Set(&core.InternalCmd{"ColorTheme", ColorTheme, false, false})
	ic.Set(&core.InternalCmd{"FontTheme", FontTheme, false, false})

	ic.Set(&core.InternalCmd{"CtxutilCallsState", CtxutilCallsState, false, false})
}
