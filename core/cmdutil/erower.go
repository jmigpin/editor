package cmdutil

import (
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

	ToolbarData() *toolbardata.ToolbarData

	IsSpecialName() bool
	Name() string
	Filename() string
	Dir() string
	IsDir() bool
	IsRegular() bool

	TextAreaAppendAsync(string)

	UpdateState()
	UpdateDuplicates()
}
