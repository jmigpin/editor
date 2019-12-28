package internalcmds

import (
	"github.com/jmigpin/editor/core"
)

func init() {
	ic := core.InternalCmds

	ic.Set(&core.InternalCmd{"Exit", true, Exit})

	ic.Set(&core.InternalCmd{"SaveSession", true, SaveSession})
	ic.Set(&core.InternalCmd{"OpenSession", true, OpenSession})
	ic.Set(&core.InternalCmd{"DeleteSession", true, DeleteSession})
	ic.Set(&core.InternalCmd{"ListSessions", true, ListSessions})

	ic.Set(&core.InternalCmd{"NewColumn", true, NewColumn})
	ic.Set(&core.InternalCmd{"CloseColumn", false, CloseColumn})

	ic.Set(&core.InternalCmd{"NewRow", true, NewRow})
	ic.Set(&core.InternalCmd{"CloseRow", false, CloseRow})
	ic.Set(&core.InternalCmd{"ReopenRow", true, ReopenRow})
	ic.Set(&core.InternalCmd{"MaximizeRow", false, MaximizeRow})

	ic.Set(&core.InternalCmd{"Save", false, Save})
	ic.Set(&core.InternalCmd{"SaveAllFiles", true, SaveAllFiles})

	ic.Set(&core.InternalCmd{"Reload", false, Reload})
	ic.Set(&core.InternalCmd{"ReloadAllFiles", true, ReloadAllFiles})
	ic.Set(&core.InternalCmd{"ReloadAll", true, ReloadAll})

	ic.Set(&core.InternalCmd{"Stop", false, Stop})
	ic.Set(&core.InternalCmd{"Clear", false, Clear})

	ic.Set(&core.InternalCmd{"Find", false, Find})
	ic.Set(&core.InternalCmd{"Replace", false, Replace})
	ic.Set(&core.InternalCmd{"GotoLine", false, GotoLine})

	ic.Set(&core.InternalCmd{"CopyFilePosition", false, CopyFilePosition})
	ic.Set(&core.InternalCmd{"RuneCodes", false, RuneCodes})
	ic.Set(&core.InternalCmd{"FontRunes", false, FontRunes})

	ic.Set(&core.InternalCmd{"XdgOpenDir", false, XdgOpenDir})

	ic.Set(&core.InternalCmd{"ListDir", false, ListDir})

	ic.Set(&core.InternalCmd{"GoRename", false, GoRename})
	ic.Set(&core.InternalCmd{"GoDebug", false, GoDebug})

	ic.Set(&core.InternalCmd{"ColorTheme", false, ColorTheme})
	ic.Set(&core.InternalCmd{"FontTheme", false, FontTheme})

	ic.Set(&core.InternalCmd{"LSProtoCloseAll", false, LSProtoCloseAll})
	ic.Set(&core.InternalCmd{"CtxutilCallsState", false, CtxutilCallsState})
}
