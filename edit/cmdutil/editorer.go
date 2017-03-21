package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editorer interface {
	Error(error)
	UI() *ui.UI

	NewERow(*ui.Column) ERower
	FindERow(string) (ERower, bool)
	FindERowOrCreate(string, *ui.Column) ERower
	ERows() []ERower

	ActiveColumn() *ui.Column
}

type ERower interface {
	Editorer() Editorer
	Row() *ui.Row

	LoadContentClear() error
	ReloadContent() error
	SaveContent(string) error

	ToolbarSD() *toolbardata.StringData

	FileInfo() (string, os.FileInfo, bool)

	TextAreaAppend(string)
}
