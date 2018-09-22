package contentcmds

import "github.com/jmigpin/editor/core"

func init() {
	core.RegisterContentCmd(goDefinition)
	core.RegisterContentCmd(filename)
	core.RegisterContentCmd(openSession)
	core.RegisterContentCmd(http)
}
