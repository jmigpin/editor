package cmdutil

import (
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editori interface {
	Error(error)
	UI() *ui.UI
	FindRowOrCreate(name string) *ui.Row
	FindRowOrCreateInColFromFilepath(filepath string, col *ui.Column) (*ui.Row, error)
	RowToolbarStringData(*ui.Row) *toolbardata.StringData
	FilepathContent(filepath string) (string, error)
	FilesWatcherAdd(filename string) error
	FilesWatcherRemove(filename string) error
}
