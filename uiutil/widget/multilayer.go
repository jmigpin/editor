package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ContainerEmbedNode
}

//func (ml *MultiLayer) MarkChildNeedsPaint() {

//}

func (ml *MultiLayer) Measure(hint image.Point) image.Point {
	panic("calling measure on multilayer")
}
func (ml *MultiLayer) CalcChildsBounds() {
	u := ml.Bounds()
	for _, n := range ml.Childs() {
		// all childs get full bounds
		n.SetBounds(&u)

		n.CalcChildsBounds()
	}
}

//func (ml *MultiLayer) UpperLayerCalcChildsBounds(node Node) {
//	if !ml.HasChild(node) {
//		panic("node is not a child of multilayer")
//	}
//	// give full bounds to the node for calc
//	u := ml.Bounds()
//	node.SetBounds(&u)
//}

//func (ml *MultiLayer) LayerNeedsPaint(node Node) {
//	if !ml.HasChild(node) {
//		panic("node is not a child of multilayer")
//	}

//	// check which nodes below the node will need paint
//	//nodeb := node.Bounds()
//	for _, n := range ml.Childs() {
//		n.MarkNeedsPaint()

//		// TODO
//		//var u image.Rectangle
//		//for _, c := range n.Childs() {
//		//	if !c.Bounds().Intersect(nodeb).Empty() {
//		//		c.MarkNeedsPaint()
//		//	}
//		//}
//	}
//}
