package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type ERower interface {
	Ed() Editorer
	Row() *ui.Row

	LoadContentClear() error
	ReloadContent() error
	SaveContent(string) error

	ToolbarSD() *toolbardata.StringData
	FileInfo() (string, os.FileInfo, bool)

	TextAreaAppend(string)
}
