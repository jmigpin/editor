package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

// Editor Row interface
type ERower interface {
	Ed() Editorer
	Row() *ui.Row

	LoadContentClear() error
	ReloadContent() error
	SaveContent(string) error

	DecodedPart0Arg0() string
	ToolbarSD() *toolbardata.StringData
	FileInfo() (string, os.FileInfo, error)

	TextAreaAppendAsync(string)
}
