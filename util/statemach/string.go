package statemach

import "github.com/jmigpin/editor/util/iout/iorw"

type String struct {
	*SM
}

func NewString(input string) *String {
	r := iorw.NewBytesReadWriter([]byte(input))
	sm := NewSM(r)
	return &String{SM: sm}
}
