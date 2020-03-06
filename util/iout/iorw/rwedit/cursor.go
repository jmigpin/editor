package rwedit

type Cursor struct {
	d        CursorData
	OnChange func()
}

// Allows to avoid copying the cursor (*c=*c2), since it could lead to undesired OnChange() triggers; use c.Data()/c.SetData() and pass CursorData in args when possible.
type CursorData struct {
	index int
	sel   struct { // selection
		on    bool
		index int
	}
}

//----------

func (c *Cursor) Index() int {
	return c.d.index
}
func (c *Cursor) SetIndex(v int) {
	d2 := c.d
	c.d.index = v
	c.changed(c.d != d2)
}

func (c *Cursor) SelectionIndex() int {
	return c.d.sel.index
}
func (c *Cursor) SetSelection(si, ci int) { // start/finish
	d2 := c.d
	c.d.sel.on = true
	c.d.sel.index = si
	c.d.index = ci
	c.changed(c.d != d2)
}
func (c *Cursor) SetSelectionOff() {
	d2 := c.d
	c.d.sel.on = false
	c.d.sel.index = 0
	c.changed(c.d != d2)
}
func (c *Cursor) SetIndexSelectionOff(i int) {
	d2 := c.d
	c.d.index = i
	c.d.sel.on = false
	c.d.sel.index = 0
	c.changed(c.d != d2)
}
func (c *Cursor) SetSelectionUpdate(on bool, ci int) {
	if on {
		si := c.d.sel.index
		if !c.d.sel.on {
			si = c.d.index
		}
		c.SetSelection(si, ci)
	} else {
		c.SetIndexSelectionOff(ci)
	}
}

//----------

func (c *Cursor) SetData(d CursorData) {
	d2 := c.d
	c.d = d
	c.changed(c.d != d2)
}
func (c *Cursor) Data() CursorData {
	return c.d
}

//----------

func (c *Cursor) HaveSelection() bool {
	return c.d.sel.on && c.d.sel.index != c.d.index
}

//----------

// Values returned are sorted
func (c *Cursor) SelectionIndexes() (int, int, bool) {
	if !c.HaveSelection() {
		return 0, 0, false
	}
	a, b := c.d.index, c.d.sel.index
	if a > b {
		a, b = b, a
	}
	return a, b, true
}

func (c *Cursor) SelectionIndexesUnsorted() (int, int, bool) {
	if !c.HaveSelection() {
		return 0, 0, false
	}
	return c.d.sel.index, c.d.index, true // start/finish (can be finish<start)
}

//----------

func (c *Cursor) changed(t bool) {
	if t && c.OnChange != nil {
		c.OnChange()
	}
}

func (c *Cursor) Equal(c2 *Cursor) bool {
	return c.d == c2.d
}
