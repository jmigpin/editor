package drawer3

type Annotations struct {
	EExt
	Opt AnnotationsOpt
}

func Annotations1() Annotations {
	return Annotations{}
}

func (ann *Annotations) Start(r *ExtRunner) {

}

func (ann *Annotations) Iterate(r *ExtRunner) {

}

//----------

type AnnotationsOpt struct {
	Entries []*Annotation // ordered
}

//----------

type Annotation struct {
	Offset int
	Bytes  []byte
}
