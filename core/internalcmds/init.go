package internalcmds

import (
	"github.com/jmigpin/editor/core"
)

func init() {
	ic := core.InternalCmds

	cmd := func(name string, fn core.InternalCmdFn) {
		ic.Set(&core.InternalCmd{Name: name, Fn: fn})
	}
	cmdERow := func(name string, fn core.InternalCmdFn) {
		ic.Set(&core.InternalCmd{Name: name, Fn: fn, NeedsERow: true})
	}

	cmd("Version", Version)

	cmd("Exit", Exit)

	cmd("SaveSession", SaveSession)
	cmd("OpenSession", OpenSession)
	cmd("DeleteSession", DeleteSession)
	cmd("ListSessions", ListSessions)

	cmd("NewColumn", NewColumn)
	cmdERow("CloseColumn", CloseColumn)

	cmd("NewRow", NewRow)
	cmdERow("CloseRow", CloseRow)
	cmd("ReopenRow", ReopenRow)
	cmdERow("MaximizeRow", MaximizeRow)

	cmd("NewFile", NewFile)
	cmdERow("Save", Save)
	cmd("SaveAllFiles", SaveAllFiles)

	cmdERow("Reload", Reload)
	cmd("ReloadAllFiles", ReloadAllFiles)
	cmd("ReloadAll", ReloadAll)

	cmdERow("Stop", Stop)
	cmdERow("Clear", Clear)

	cmdERow("Find", Find)
	cmdERow("Replace", Replace)
	cmdERow("GotoLine", GotoLine)
	cmdERow("GoToLine", GotoLine)

	cmdERow("CopyFilePosition", CopyFilePosition)
	cmdERow("RuneCodes", RuneCodes)
	cmd("FontRunes", FontRunes)

	// Deprecated: in favor of "OpenFilemanager"
	cmd("XdgOpenDir", OpenFilemanager)
	cmdERow("OpenFilemanager", OpenFilemanager)
	cmdERow("OpenTerminal", OpenTerminal)

	cmdERow("ListDir", ListDir)

	cmdERow("GoRename", GoRename)
	cmdERow("GoDebug", GoDebug)
	cmdERow("GoDebugFind", GoDebugFind)

	// Deprecated: in favor of "LspCloseAll"
	cmd("LSProtoCloseAll", LSProtoCloseAll)
	cmd("LsprotoCloseAll", LSProtoCloseAll)
	cmdERow("LsprotoRename", LSProtoRename)

	cmd("ColorTheme", ColorTheme)
	cmd("FontTheme", FontTheme)

	cmd("CtxutilCallsState", CtxutilCallsState)
}
