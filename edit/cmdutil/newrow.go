package cmdutil

func NewRow(ed Editorer) {
	row := ed.NewRow(ed.ActiveColumn())
	row.Square.WarpPointer()
}
