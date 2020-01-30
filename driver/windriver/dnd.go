package windriver

import (
	"fmt"
	"image"
	"strings"

	"github.com/jmigpin/editor/util/uiutil/event"
	"golang.org/x/sys/windows"
)

// Drag and drop manager.
type DndMan struct {
}

func NewDndMan() *DndMan {
	return &DndMan{}
}

func (m *DndMan) HandleDrop(hDrop uintptr) (interface{}, bool, error) {
	//dropped, p := m.dropPoint(hDrop)
	//if !dropped {
	//	ev := m.buildPositionEvent(hDrop, p)
	//	return ev, true, nil
	//} else {
	//	ev := m.buildDropEvent(hDrop, p)
	//	return ev, true, nil
	//}
	//return nil, false, nil

	// always dropping
	_, p := m.dropPoint(hDrop)
	ev := m.buildDropEvent(hDrop, p)
	return ev, true, nil
}

//----------

func (m *DndMan) buildPositionEvent(hDrop uintptr, p image.Point) interface{} {
	//fmt.Printf("reply %v\n", action)
	types := []event.DndType{event.TextURLListDndT}
	appReplyFn := func(action event.DndAction) {
		// TODO: post msg to msg loop? saying the action somehow
		//m.positionReply(action)
	}
	return &event.DndPosition{p, types, appReplyFn}
}

//----------

func (m *DndMan) buildDropEvent(hDrop uintptr, p image.Point) interface{} {
	appReqFn := func(typ event.DndType) ([]byte, error) {
		u := FilesDropped(hDrop)
		if len(u) == 0 {
			return nil, fmt.Errorf("no files dropped")
		}
		b := []byte(strings.Join(u, "\n"))
		return b, nil
	}
	appReplyFn := func(v bool) {
		// TODO: true or false
		_DragFinish(hDrop)
	}
	return &event.DndDrop{p, appReplyFn, appReqFn}
}

//----------

func (m *DndMan) dropPoint(hDrop uintptr) (bool, image.Point) {
	p := _Point{}
	dropped := _DragQueryPoint(hDrop, &p)
	return dropped, p.ToImagePoint()
}

//----------

func FilesDropped(hDrop uintptr) []string {
	// http://delphidabbler.com/articles?article=11

	// find the number of files dropped
	res := _DragQueryFileW(hDrop, 0xffffffff, nil, 0)
	n := int(res)
	// find the sizes of the buffers needed
	sizes := make([]int, n)
	for i := 0; i < n; i++ {
		size := _DragQueryFileW(hDrop, uint32(i), nil, 0)
		sizes[i] = int(size)
	}
	// fetch the filenames
	names := make([]string, n)
	for i := 0; i < n; i++ {
		u := make([]uint16, sizes[i]+1) // +1 is the nil terminator
		_ = _DragQueryFileW(hDrop, uint32(i), &u[0], uint32(len(u)))
		names[i] = windows.UTF16ToString(u)
	}
	return names
}
