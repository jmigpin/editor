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

	cmd(Version, "Version")
	cmd(Exit, "Exit")

	cmd(SaveSession, "SaveSession")
	cmd(OpenSession, "OpenSession")
	cmd(DeleteSession, "DeleteSession")
	cmd(ListSessions, "ListSessions")

	cmd(NewColumn, "NewColumn")
	cmd(CloseColumn, "CloseColumn")

	cmd(NewRow, "NewRow")
	cmd(ReopenRow, "ReopenRow")
	cmd(CloseRow, "CloseRow")
	cmd(MaximizeRow, "MaximizeRow")

	cmd(NewFile, "NewFile")
	cmd(Save, "Save")
	cmd(SaveAllFiles, "SaveAllFiles")

	cmd(Reload, "Reload")
	cmd(ReloadAllFiles, "ReloadAllFiles")
	cmd(ReloadAll, "ReloadAll")

	cmd(Stop, "Stop")
	cmd(Clear, "Clear")

	cmd(Find, "Find")
	cmd(Replace, "Replace")
	cmd(GotoLine, "GotoLine", "GoToLine")

	cmd(CopyFilePosition, "CopyFilePosition")
	cmd(RuneCodes, "RuneCodes")
	cmd(FontRunes, "FontRunes")

	cmd(OpenFilemanager, "OpenFilemanager", "XdgOpenDir") // TODO: deprecate XdgOpenDir
	cmd(OpenTerminal, "OpenTerminal")
	cmd(OpenExternal, "OpenExternal")

	cmd(ListDir, "ListDir")

	cmd(GoRename, "GoRename") // TODO: deprecate

	cmd(GoDebug, "GoDebug")
	cmd(GoDebugFind, "GoDebugFind")
	cmd(GoDebugTrace, "GoDebugTrace")

	cmd(LSProtoCloseAll, "LsprotoCloseAll", "LSProtoCloseAll") // TODO: deprecate LSProtoCloseAll
	cmd(LSProtoRename, "LsprotoRename")
	cmd(LSProtoReferences, "LsprotoReferences")
	cmd(LSProtoCallHierarchyIncomingCalls, "LsprotoCallers", "LsprotoCallHierarchyIncomingCalls")
	cmd(LSProtoCallHierarchyOutgoingCalls, "LsprotoCallees", "LsprotoCallHierarchyOutgoingCalls")

	cmd(ColorTheme, "ColorTheme")
	cmd(FontTheme, "FontTheme")

	cmd(CtxutilCallsState, "CtxutilCallsState")

	cmd(sortTextLines, "SortTextLines")
}
