package contentcmds

import (
	"github.com/jmigpin/editor/core"
)

func init() {
	// order matters
	core.ContentCmds.Append("gotoimplementation_lsproto", GoToImplementationLSProto)
	core.ContentCmds.Append("gotodefinition_lsproto", GoToDefinitionLSProto)
	

	// "gopls query" might work where lsproto might fail (no views in session)
	core.ContentCmds.Append("gotodefinition", GoToDefinitionGolang)

	// opensession runs before openfilename to avoid failing if a file with that name exists in the current directory
	core.ContentCmds.Append("opensession", OpenSession)

	core.ContentCmds.Append("openfilename", OpenFilename)
	core.ContentCmds.Append("openurl", OpenURL)
}
