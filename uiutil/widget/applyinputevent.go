package widget

import "image"

// Auto-locks all input events into the node on the tree that handles
// a press event. The node is responsible for unlocking by handling
// the release event. Handling means OnInputEvent() returns true.

func ApplyInputEventInBounds(
	node Node,
	ev interface{},
	p image.Point,
	isPressEvent func(ev interface{}) bool,
	isReleaseEvent func(ev interface{}) bool,
) {
	aie.isPressEvent = isPressEvent
	aie.isReleaseEvent = isReleaseEvent
	if aie.pressNode != nil {
		aie.applyToPressNode(ev, p)
	} else {
		aie.apply(node, ev, p)
	}
}

var aie ApplyInputEvent

type ApplyInputEvent struct {
	pressNode      Node
	isPressEvent   func(ev interface{}) bool
	isReleaseEvent func(ev interface{}) bool
}

func (aie *ApplyInputEvent) apply(node Node, ev interface{}, p image.Point) bool {
	// Helps breaking early and avoid running siblings in the case of structure changes.
	handled := false

	// Reversed iteration for the possibility that later childs are drawn over.
	// Hidden nodes are inherently not iterated.
	for c := node.LastChild(); c != nil; c = c.Prev() {
		if p.In(c.Bounds()) {
			h := aie.apply(c, ev, p)
			if h {
				// Don't handle other siblings. The structure could have changed with this node having called CalcChildsBounds and now siblings could get the event as well.
				handled = handled || h
				break
			}
		}
	}

	h := node.OnInputEvent(ev, p)
	if h && aie.isPressEvent(ev) {
		handled = handled || h
		// the first event to handle it, locks it
		if aie.pressNode == nil {
			aie.pressNode = node
		}
	}
	return handled
}

func (aie *ApplyInputEvent) applyToPressNode(ev interface{}, p image.Point) {
	pn := aie.pressNode

	handled := aie.pressNode.OnInputEvent(ev, p)
	if handled && aie.isReleaseEvent(ev) {
		// unlock node
		aie.pressNode = nil
	}

	// call event up in the parent chain
	for u := pn.Parent(); u != nil; u = u.Parent() {
		_ = u.OnInputEvent(ev, p)
	}
}
