package event

type Cursor int

const (
	UndefinedCursor Cursor = iota // unset value used while resolving the widget tree cursor
	DefaultCursor
	HiddenCursor
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	HandCursor
	BeamCursor // text cursor
	WaitCursor
)
