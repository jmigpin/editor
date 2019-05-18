package contentcmds

import "github.com/jmigpin/editor/core"

func init() {
	// order matters
	core.ContentCmds.Append("gotodefinition", GoToDefinitionGolang)
	core.ContentCmds.Append("openfilename", OpenFilename)
	core.ContentCmds.Append("opensession", OpenSession)
	core.ContentCmds.Append("openurl", OpenURL)

	// NOTE: least priority since under development. Can be canceled with escape key.
	core.ContentCmds.Append("gotodefinition_lsproto", GoToDefinitionLSProto)
}
