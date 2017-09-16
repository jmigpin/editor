package cmdutil

import "path"

func NewRow(ed Editorer) {
	p := "."

	erow2, ok := ed.ActiveERow()
	if ok {
		fp := erow2.DecodedPart0Arg0()
		p = path.Dir(fp) // if fp=="", dir returns "."
	}

	col, nextRow := ed.GoodColumnRowPlace()
	erow := ed.NewERowBeforeRow(p+" | ", col, nextRow)
	erow.Row().WarpPointer()
}
