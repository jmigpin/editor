package cmdutil

import (
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editori interface {
	Error(error)

	UI() *ui.UI
	FindRowOrCreate(name string) *ui.Row
	RowToolbarStringData(*ui.Row) *toolbardata.StringData
	FilepathContent(filepath string) (string, error)
}
