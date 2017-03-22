package cmdutil

import "github.com/jmigpin/editor/ui"

type Editorer interface {
	Error(error)
	Errorf(string, ...interface{})
	UI() *ui.UI

	NewERow(string, *ui.Column, int) ERower
	FindERow(string) (ERower, bool)
	ERows() []ERower

	GoodColRowPlace() (col *ui.Column, rowIndex int)
}
