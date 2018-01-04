package statemach

type Itemizer struct {
	Items chan *Item
}

func (i *Itemizer) Run(s State) {
	for s != nil {
		s = s()
	}
	close(i.Items)
}

// on the statemachine?
//func (i *Itemizer) Item() *Item {
//	//sm.Items <- &Item{err, sm.Start, sm.Pos}
//	return nil
//}

type State func() State

type Item struct {
	Type  interface{}
	Pos   int
	End   int
	Value string
}
