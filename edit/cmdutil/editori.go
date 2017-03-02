package cmdutil

import (
	"github.com/jmigpin/editor/edit/toolbar"
	"github.com/jmigpin/editor/ui"
)

type Editori interface {
	Error(error)

	UI() *ui.UI
	FindRowOrCreate(name string) *ui.Row
	RowToolbarStringData(*ui.Row) *toolbar.StringData
	FilepathContent(filepath string) (string, error)
}
