package widget

type Cursor int

const (
	NoneCursor Cursor = iota // none means not set
	DefaultCursor
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	PointerCursor
	BeamCursor // text cursor
	WaitCursor // watch cursor
)
