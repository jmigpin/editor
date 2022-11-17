package lrparser

import (
	"fmt"
	"strconv"

	"github.com/jmigpin/editor/util/parseutil"
)

// TODO: multiple files (working for single file only)
type FileSet struct {
	Src      []byte // currently, just a single src
	Filename string // for errors only
}

func NewFileSetFromBytes(src []byte) *FileSet {
	return &FileSet{Src: src, Filename: "<bytes>"}
}

//----------

//func (fset *FileSet) SliceFrom(i int) []byte {
//	// TODO: implemented for single file only (need node arg?)
//	return fset.src[i:]
//}
//func (fset *FileSet) SliceTo(i int) []byte {
//	// TODO: implemented for single file only (need node arg?)
//	return fset.src[:i]
//}

func (fset *FileSet) NodeBytes(node PNode) []byte {
	return fset.Src[node.Pos():node.End()]
}
func (fset *FileSet) NodeString(node PNode) string {
	return string(fset.Src[node.Pos():node.End()])
}
func (fset *FileSet) NodeInt(node PNode) (int, error) {
	s := fset.NodeString(node)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

//----------

func (fset *FileSet) Error(err error) error {
	if pe, ok := err.(*PosError); ok {
		return fset.Error2(pe, pe.Pos)
	}
	return fmt.Errorf("%s: %v", fset.Filename, err)
}
func (fset *FileSet) Error2(err error, i int) error {
	line, col := parseutil.IndexLineColumn2(fset.Src, i)
	str := parseutil.SurroundingString(fset.Src, i, 20)
	return fmt.Errorf("%s:%d:%d: %v: %q", fset.Filename, line, col, err, str)
}

//----------
//----------
//----------

type PosError struct {
	err error
	Pos int
}

func (pe *PosError) Error() string {
	return pe.err.Error()
}
