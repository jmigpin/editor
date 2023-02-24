package internalcmds

import (
	"github.com/jmigpin/editor/core"
)

func init() {
	ic := core.InternalCmds

	cmd := func(fn core.InternalCmdFn, names ...string) {
		for _, name := range names {
			ic.Set(&core.InternalCmd{Name: name, Fn: fn})
		}
	}
	cmdERow := func(fn core.InternalCmdFn, names ...string) {
		for _, name := range names {
			ic.Set(&core.InternalCmd{Name: name, Fn: fn, NeedsERow: true})
		}
	}

	cmd(Version, "Version")
	cmd(Exit, "Exit")

	cmd(SaveSession, "SaveSession")
	cmd(OpenSession, "OpenSession")
	cmd(DeleteSession, "DeleteSession")
	cmd(ListSessions, "ListSessions")

	cmd(NewColumn, "NewColumn")
	cmdERow(CloseColumn, "CloseColumn")

	cmd(NewRow, "NewRow")
	cmd(ReopenRow, "ReopenRow")
	cmdERow(CloseRow, "CloseRow")
	cmdERow(MaximizeRow, "MaximizeRow")

	cmdERow(NewFile, "NewFile")
	cmdERow(Save, "Save")
	cmd(SaveAllFiles, "SaveAllFiles")

	cmdERow(Reload, "Reload")
	cmd(ReloadAllFiles, "ReloadAllFiles")
	cmd(ReloadAll, "ReloadAll")

	cmdERow(Stop, "Stop")
	cmdERow(Clear, "Clear")

	cmd(Find, "Find")
	cmdERow(Replace, "Replace")
	cmdERow(GotoLine, "GotoLine", "GoToLine")

	cmdERow(CopyFilePosition, "CopyFilePosition")
	cmdERow(RuneCodes, "RuneCodes")
	cmd(FontRunes, "FontRunes")

	cmdERow(OpenFilemanager, "OpenFilemanager", "XdgOpenDir") // TODO: deprecate XdgOpenDir
	cmdERow(OpenTerminal, "OpenTerminal")
	cmdERow(OpenExternal, "OpenExternal")

	cmdERow(ListDir, "ListDir")

	cmdERow(GoRename, "GoRename") // TODO: deprecate
	cmdERow(GoDebug, "GoDebug")
	cmdERow(GoDebugFind, "GoDebugFind")

	cmd(LSProtoCloseAll, "LsprotoCloseAll", "LSProtoCloseAll") // TODO: deprecate LSProtoCloseAll
	cmdERow(LSProtoRename, "LsprotoRename")
	cmdERow(LSProtoReferences, "LsprotoReferences")
	cmdERow(LSProtoCallHierarchyIncomingCalls, "LsprotoCallers", "LsprotoCallHierarchyIncomingCalls")
	cmdERow(LSProtoCallHierarchyOutgoingCalls, "LsprotoCallees", "LsprotoCallHierarchyOutgoingCalls")

	cmd(ColorTheme, "ColorTheme")
	cmd(FontTheme, "FontTheme")

	cmd(CtxutilCallsState, "CtxutilCallsState")
}
