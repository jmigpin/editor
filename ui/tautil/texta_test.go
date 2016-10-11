package tautil

type Textad struct { // Texta dummy
	Texta
	text           string
	cursorIndex    int
	selectionOn    bool
	selectionIndex int
}

func (ta *Textad) Text() string            { return ta.text }
func (ta *Textad) CursorIndex() int        { return ta.cursorIndex }
func (ta *Textad) SetCursorIndex(v int)    { ta.cursorIndex = v }
func (ta *Textad) SelectionOn() bool       { return ta.selectionOn }
func (ta *Textad) SetSelectionOn(v bool)   { ta.selectionOn = v }
func (ta *Textad) SelectionIndex() int     { return ta.selectionIndex }
func (ta *Textad) SetSelectionIndex(v int) { ta.selectionIndex = v }
func (ta *Textad) NeedPaint()              {}
