package contentcmds

import "github.com/jmigpin/editor/core"

func init() {
	// order matters
	core.ContentCmds.Append("gotodefinition", GoToDefinitionGolang)
	core.ContentCmds.Append("gotodefinition_lsproto", GoToDefinitionLSProto)

	// opensession runs before openfilename to avoid failing if a file with that name exists in the current directory
	core.ContentCmds.Append("opensession", OpenSession)

	core.ContentCmds.Append("openfilename", OpenFilename)
	core.ContentCmds.Append("openurl", OpenURL)
}
