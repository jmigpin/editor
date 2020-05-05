package rwedit

type Cursor interface {
	Set(c SimpleCursor)
	Get() SimpleCursor

	Index() int
	SetIndex(int)
	SelectionIndex() int
	SetSelection(si, ci int)
	SetSelectionOff()
	SetIndexSelectionOff(i int)
	HaveSelection() bool
	UpdateSelection(on bool, ci int)
	SelectionIndexes() (int, int, bool)
	SelectionIndexesUnsorted() (int, int, bool)
}

//----------

type SimpleCursor struct {
	index int
	sel   struct { // selection
		on    bool
		index int
	}
}

func (c *SimpleCursor) Set(c2 SimpleCursor) {
	*c = c2
}
func (c *SimpleCursor) Get() SimpleCursor {
	return *c
}

//----------

func (c *SimpleCursor) Index() int {
	return c.index
}
func (c *SimpleCursor) SetIndex(i int) {
	c.index = i
}
func (c *SimpleCursor) SelectionIndex() int {
	return c.sel.index
}
func (c *SimpleCursor) SetSelection(si, ci int) { // start/finish
	c.sel.on = true
	c.sel.index = si
	c.index = ci
}
func (c *SimpleCursor) SetSelectionOff() {
	c.sel.on = false
	c.sel.index = 0
}
func (c *SimpleCursor) SetIndexSelectionOff(i int) {
	c.index = i
	c.sel.on = false
	c.sel.index = 0
}

//----------

func (c *SimpleCursor) HaveSelection() bool {
	return c.sel.on && c.sel.index != c.index
}

//----------

func (c *SimpleCursor) UpdateSelection(on bool, ci int) {
	if on {
		si := c.sel.index
		if !c.sel.on {
			si = c.index
		}
		c.SetSelection(si, ci)
	} else {
		c.SetIndexSelectionOff(ci)
	}
}

//----------

// Values returned are sorted
func (c *SimpleCursor) SelectionIndexes() (int, int, bool) {
	if !c.HaveSelection() {
		return 0, 0, false
	}
	a, b := c.index, c.sel.index
	if a > b {
		a, b = b, a
	}
	return a, b, true
}

func (c *SimpleCursor) SelectionIndexesUnsorted() (int, int, bool) {
	if !c.HaveSelection() {
		return 0, 0, false
	}
	return c.sel.index, c.index, true // start/finish (can be finish<start)
}

//----------

type TriggerCursor struct {
	*SimpleCursor
	c        *SimpleCursor
	onChange func()
}

func NewTriggerCursor(onChange func()) *TriggerCursor {
	tc := &TriggerCursor{onChange: onChange}
	c := &SimpleCursor{}
	tc.SimpleCursor = c
	tc.c = c
	return tc
}

//----------

func (tc *TriggerCursor) Set(c SimpleCursor) {
	tmp := tc.copy()
	*tc.SimpleCursor = c
	tc.changed(tmp)
}
func (tc *TriggerCursor) SetIndex(i int) {
	tmp := tc.copy()
	tc.c.SetIndex(i)
	tc.changed(tmp)
}
func (tc *TriggerCursor) SetSelection(si, ci int) { // start/finish
	tmp := tc.copy()
	tc.c.SetSelection(si, ci)
	tc.changed(tmp)
}
func (tc *TriggerCursor) SetSelectionOff() {
	tmp := tc.copy()
	tc.c.SetSelectionOff()
	tc.changed(tmp)
}
func (tc *TriggerCursor) SetIndexSelectionOff(i int) {
	tmp := tc.copy()
	tc.c.SetIndexSelectionOff(i)
	tc.changed(tmp)
}
func (tc *TriggerCursor) UpdateSelection(on bool, ci int) {
	tmp := tc.copy()
	tc.c.UpdateSelection(on, ci)
	tc.changed(tmp)
}

//----------

func (tc *TriggerCursor) copy() SimpleCursor {
	return *tc.SimpleCursor
}
func (tc *TriggerCursor) changed(c SimpleCursor) {
	if tc.onChange == nil {
		return
	}
	if c != *tc.c {
		tc.onChange()
	}
}
