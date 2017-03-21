package cmdutil

import (
	"crypto/sha1"
	"io/ioutil"
	"os"

	"github.com/jmigpin/editor/ui"
)

type RowStatus struct {
	ed      Editorer
	rowData map[*ui.Row]*RSData
}

func NewRowStatus(ed Editorer) (*RowStatus, error) {
	rs := &RowStatus{
		ed:      ed,
		rowData: make(map[*ui.Row]*RSData),
	}
	return rs, nil
}
func (rs *RowStatus) Add(row *ui.Row) {
	_, ok := rs.rowData[row]
	if ok {
		panic("row already exists")
	}
	rd := NewRSData(rs.ed, row)
	rs.rowData[row] = rd
}
func (rs *RowStatus) Remove(row *ui.Row) {
	delete(rs.rowData, row)
}
func (rs *RowStatus) OnRowToolbarSetStr(row *ui.Row) {
	rd, ok := rs.rowData[row]
	if !ok {
		panic("row not found")
	}
	if rd.doFirstPart() {
		rd.calcNotExist()
		rd.calcDirty()
	}
}
func (rs *RowStatus) OnRowTextAreaSetStr(row *ui.Row) {
	rd, ok := rs.rowData[row]
	if !ok {
		panic("row not found")
	}
	if rd.isFile {
		row.Square.SetValue(rowDirty, true)
	}
}
func (rs *RowStatus) fileChange(name string) {
}
func (rs *RowStatus) NotDirty(row *ui.Row) {
	row.Square.SetValue(rowDirty, false)
}
func (rs *RowStatus) NotCold(row *ui.Row) {
	row.Square.SetValue(rowCold, false)
}
func (rs *RowStatus) NotDirtyOrCold(row *ui.Row) {
	row.Square.SetValue(rowDirty, false)
	row.Square.SetValue(rowCold, false)
}

const (
	rowDirty    = 0
	rowCold     = 2
	rowNotExist = 4
)

type RSData struct {
	ed         Editorer
	row        *ui.Row
	firstPart  string
	isDir      bool
	isFile     bool
	isNotExist bool
}

func NewRSData(ed Editorer, row *ui.Row) *RSData {
	rd := &RSData{ed: ed, row: row}
	_ = rd.doFirstPart()
	return rd
}
func (rd *RSData) doFirstPart() bool {
	tsd := rd.ed.RowToolbarStringData(rd.row)
	fp := tsd.FirstPartFilepath()
	if fp == rd.firstPart {
		return false
	}
	rd.firstPart = fp
	rd.doStat()
	return true
}
func (rd *RSData) doStat() {
	rd.isDir = false
	rd.isFile = false
	rd.isNotExist = false
	fi, err := os.Stat(rd.firstPart)
	if err != nil {
		if os.IsNotExist(err) {
			rd.isNotExist = true
		}
	} else {
		if fi.Mode().IsRegular() {
			rd.isFile = true
		} else if fi.Mode().IsDir() {
			rd.isDir = true
		}
	}
}
func (rd *RSData) calcNotExist() {
	rd.row.Square.SetValue(rowNotExist, rd.isNotExist)
}
func (rd *RSData) calcDirty() {
	dirty := false
	if rd.isFile {
		b, err := ioutil.ReadFile(rd.firstPart)
		if err == nil {
			s1 := string(b)
			s2 := rd.row.TextArea.Str()
			dirty = s1 != s2
		}
	}
	rd.row.Square.SetValue(rowDirty, dirty)
}

func contentShaSum(b []byte) []byte {
	hasher := sha1.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}
