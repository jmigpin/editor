package fswatcher

import (
	"path/filepath"
)

type Watcher interface {
	Add(name string) error
	Remove(name string) error
	Events() <-chan interface{}
	OpMask() *Op
	Close() error
}

//----------

type Event struct {
	Op      Op
	Name    string
	SubName string
}

func (ev *Event) JoinNames() string {
	return filepath.Join(ev.Name, ev.SubName)
}

//----------

type Op uint16

func (op Op) HasAny(op2 Op) bool { return op&op2 != 0 }
func (op *Op) Add(op2 Op)        { *op |= op2 }
func (op *Op) Remove(op2 Op)     { *op &^= op2 }

//----------

const (
	Attrib Op = 1 << iota
	Create
	Modify // write, truncate
	Remove
	Rename

	AllOps Op = Attrib | Create | Modify | Remove | Rename
)
