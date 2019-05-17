package core

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/osutil"
)

// TODO: become an interface, with file/dir/special implementations.
// TODO: centralized iorw reader/writer in info

// Editor Row Info.
type ERowInfo struct {
	ERows []*ERow

	Ed *Editor

	name  string // filename, or special name
	fi    os.FileInfo
	fiErr error

	// savedHash keeps the hash known even if the file gets deleted and reappears later
	savedHash struct {
		size int
		hash []byte
	}

	// filesystem hash (reflects changes by other programs)
	fsHash struct {
		//size    int
		hash    []byte
		modTime time.Time
	}

	// not always up to date, used if the hash is being requested without the contents being changed
	editedHash struct {
		updated bool
		size    int
		hash    []byte
	}
}

// Not to be created directly. Only the editor instance will check if another info already exists.
func NewERowInfo(ed *Editor, name string) *ERowInfo {
	info := &ERowInfo{Ed: ed, name: name}
	info.readFileInfo()
	return info
}

//----------

func (info *ERowInfo) readFileInfo() {
	if isSpecialName(info.name) {
		return
	}

	defer func() {
		info.UpdateExistsRowState()
	}()

	fi, err := os.Stat(info.name)
	if err != nil {
		// keep old info.fi to allow file/dir detection
		info.fiErr = err
		return
	}
	info.fi = fi
	info.fiErr = nil

	// don't open devices, ioutil.readfile can hang the editor
	if info.fi.Mode()&os.ModeDevice > 0 {
		info.fi = nil
		info.fiErr = fmt.Errorf("file is a device")
	}
}

//----------

func (info *ERowInfo) IsSpecial() bool {
	return isSpecialName(info.name)
}

func (info *ERowInfo) HasFileinfo() bool {
	return info.fi != nil
}

func (info *ERowInfo) IsFileButNotDir() bool {
	return info.HasFileinfo() && !info.fi.IsDir()
}

func (info *ERowInfo) IsDir() bool {
	return info.HasFileinfo() && info.fi.IsDir()
}

func (info *ERowInfo) IsNotExist() bool {
	return os.IsNotExist(info.fiErr)
}

func (info *ERowInfo) FileInfoErr() error {
	return info.fiErr
}

//----------

func (info *ERowInfo) Name() string {
	return info.name
}

func (info *ERowInfo) Dir() string {
	if info.IsSpecial() {
		return ""
	}
	if info.IsDir() {
		return info.Name()
	}
	return filepath.Dir(info.Name())
}

//----------

func (info *ERowInfo) editedHashNeedsUpdate() {
	info.editedHash.updated = false
}

func (info *ERowInfo) updateEditedHash() {
	if len(info.ERows) == 0 {
		return
	}
	if info.editedHash.updated {
		return
	}
	// read from one of the erows
	erow0 := info.ERows[0]
	b, err := erow0.Row.TextArea.Bytes()
	if err != nil {
		return
	}
	info.setEditedHash(bytesHash(b), len(b))
}

//----------

func (info *ERowInfo) setEditedHash(hash []byte, size int) {
	info.editedHash.size = size
	info.editedHash.hash = hash
	info.editedHash.updated = true
}

func (info *ERowInfo) setSavedHash(hash []byte, size int) {
	info.savedHash.size = size
	info.savedHash.hash = hash
	info.UpdateFsDifferRowState()
}

func (info *ERowInfo) setFsHash(hash []byte) {
	if info.fi == nil {
		return
	}
	//info.fsHash.size = int(info.fi.Size()) // TODO: downgrading if 32bit system
	info.fsHash.hash = hash
	info.fsHash.modTime = info.fi.ModTime()
	info.UpdateFsDifferRowState()
}

func (info *ERowInfo) updateFsHashIfNeeded() {
	if !info.IsFileButNotDir() {
		return
	}
	if info.fi == nil {
		return
	}
	if !info.fi.ModTime().Equal(info.fsHash.modTime) {
		info.readFsFile()
	}
}

//----------

func (info *ERowInfo) AddERow(erow *ERow) {
	// sanity check
	for _, e := range info.ERows {
		if e == erow {
			panic("adding same erow twice")
		}
	}

	info.ERows = append(info.ERows, erow)
}

func (info *ERowInfo) RemoveERow(erow *ERow) {
	for i, e := range info.ERows {
		if e == erow {
			w := info.ERows
			copy(w[i:], w[i+1:])
			w = w[:len(w)-1]
			info.ERows = w
			return
		}
	}
	panic("erow not found")
}

func (info *ERowInfo) ERowsInUIOrder() []*ERow {
	w := []*ERow{}
	for _, col := range info.Ed.UI.Root.Cols.Columns() {
		for _, row := range col.Rows() {
			for _, erow := range info.ERows {
				if erow.Row == row {
					w = append(w, erow)
				}
			}
		}
	}

	if len(w) != len(info.ERows) {
		panic("not all erows were found")
	}

	return w
}

//----------

func (info *ERowInfo) NewERow(rowPos *ui.RowPos) (*ERow, error) {
	switch {
	case info.IsSpecial():
		erow := NewERow(info.Ed, info, rowPos)
		return erow, nil
	case info.IsDir():
		return info.NewDirERow(rowPos)
	case info.IsFileButNotDir():
		return info.NewFileERow(rowPos)
	default:
		err := fmt.Errorf("unable to open erow: %v", info.name)
		if info.fiErr != nil {
			err = fmt.Errorf("%v: %v", err, info.fiErr)
		}
		return nil, err
	}
}

func (info *ERowInfo) NewERowCreateOnErr(rowPos *ui.RowPos) (*ERow, error) {
	erow, err := info.NewERow(rowPos)
	if err != nil {
		erow = NewERow(info.Ed, info, rowPos)
		return erow, err
	}
	return erow, nil
}

//----------

func (info *ERowInfo) NewDirERow(rowPos *ui.RowPos) (*ERow, error) {
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	erow := NewERow(info.Ed, info, rowPos)
	ListDirERow(erow, erow.Info.Name(), false, true)
	return erow, nil
}

func (info *ERowInfo) ReloadDir(erow *ERow) error {
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	ListDirERow(erow, erow.Info.Name(), false, true)
	return nil
}

//----------

func (info *ERowInfo) NewFileERow(rowPos *ui.RowPos) (*ERow, error) {
	// read content from existing row
	if len(info.ERows) > 0 {
		erow0 := info.ERows[0]

		// create erow first to get it updated
		erow := NewERow(info.Ed, info, rowPos)

		// update the new erow with content
		info.SetRowsStrFromMaster(erow0)

		return erow, nil
	}

	// read file
	b, err := info.readFsFile()
	if err != nil {
		return nil, err
	}

	// update data
	info.setSavedHash(info.fsHash.hash, len(b))

	// new erow (no other rows exist)
	erow := NewERow(info.Ed, info, rowPos)
	erow.Row.TextArea.SetBytesClearHistory(b)

	return erow, nil
}

func (info *ERowInfo) ReloadFile() error {
	b, err := info.readFsFile()
	if err != nil {
		return err
	}

	// update data
	info.setSavedHash(info.fsHash.hash, len(b))

	// update all erows
	info.SetRowsBytes(b)

	return nil
}

//----------

// Save file and update rows.
func (info *ERowInfo) SaveFile() error {
	if len(info.ERows) == 0 {
		return nil
	}

	if !info.IsFileButNotDir() {
		return fmt.Errorf("not a file: %s", info.Name())
	}

	// read from one of the erows
	erow0 := info.ERows[0]
	b, err := erow0.Row.TextArea.Bytes()
	if err != nil {
		return err
	}

	// run go imports for go content, updates content
	if filepath.Ext(info.Name()) == ".go" {
		u, err := runGoImports(b, filepath.Dir(info.Name()))
		// ignore errors, can catch them when compiling
		if err == nil {
			b = u
		}
	}

	// save
	err = info.saveFsFile(b)
	if err != nil {
		return err
	}

	// update all erows (including row saved states)
	info.SetRowsBytes(b)

	//// warn lsproto of file save
	//go func() {
	//	ctx0 := context.Background()
	//	ctx, cancel := context.WithTimeout(ctx0, 2000*time.Millisecond)
	//	defer cancel()
	//	err := info.Ed.LSProtoMan.DidSave(ctx, info.Name(), nil)
	//	if err != nil {
	//		info.Ed.Error(err)
	//	}
	//}()

	return nil
}

//----------

func (info *ERowInfo) readFsFile() ([]byte, error) {
	b, err := ioutil.ReadFile(info.Name())
	if err != nil {
		return nil, err
	}

	// update data
	info.readFileInfo() // get new modtime
	h := bytesHash(b)
	info.setFsHash(h)

	return b, err
}

func (info *ERowInfo) saveFsFile(b []byte) error {
	flags := os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	f, err := os.OpenFile(info.Name(), flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	defer f.Sync() // necessary? modtime needs fsync on dir?
	_, err = f.Write(b)
	if err != nil {
		return err
	}

	// update data
	h := bytesHash(b)
	info.readFileInfo() // get new modtime
	info.setFsHash(h)
	info.setSavedHash(h, len(b))

	return nil
}

//----------

// Should be called under UI goroutine.
func (info *ERowInfo) UpdateDiskEvent() {
	info.readFileInfo()
	if info.IsFileButNotDir() {
		info.updateFsHashIfNeeded()
	}
}

//----------

func (info *ERowInfo) EqualToBytesHash(size int, hash []byte) bool {
	if len(info.ERows) == 0 {
		return false
	}
	erow0 := info.ERows[0]
	if erow0.Row.TextArea.Len() != size {
		return false
	}

	info.updateEditedHash()
	return bytes.Equal(hash, info.editedHash.hash)
}

//----------

func (info *ERowInfo) UpdateEditedRowState() {
	if !info.IsFileButNotDir() {
		return
	}
	info.editedHashNeedsUpdate()
	edited := !info.EqualToBytesHash(info.savedHash.size, info.savedHash.hash)
	info.updateRowState(ui.RowStateEdited, edited)
}

func (info *ERowInfo) UpdateExistsRowState() {
	info.updateRowState(ui.RowStateNotExist, info.IsNotExist())
}

func (info *ERowInfo) UpdateFsDifferRowState() {
	if !info.IsFileButNotDir() {
		return
	}
	h1 := info.fsHash.hash
	h2 := info.savedHash.hash
	differ := !bytes.Equal(h1, h2)
	info.updateRowState(ui.RowStateFsDiffer, differ)
}

func (info *ERowInfo) UpdateDuplicateRowState() {
	hasDups := len(info.ERows) >= 2
	info.updateRowState(ui.RowStateDuplicate, hasDups)
}

func (info *ERowInfo) UpdateDuplicateHighlightRowState() {
	on := false
	for _, e := range info.ERows {
		if e.highlightDuplicates {
			on = true
			break
		}
	}
	hasDups := len(info.ERows) >= 2
	info.updateRowState(ui.RowStateDuplicateHighlight, hasDups && on)
}

func (info *ERowInfo) UpdateAnnotationsRowState(v bool) {
	info.updateRowState(ui.RowStateAnnotations, v)
}

func (info *ERowInfo) UpdateAnnotationsEditedRowState(v bool) {
	info.updateRowState(ui.RowStateAnnotationsEdited, v)
}

//----------

func (info *ERowInfo) updateRowState(state ui.RowState, v bool) {
	for _, erow := range info.ERows {
		erow.Row.SetState(state, v)
	}
}

//----------

func (info *ERowInfo) SetRowsBytes(b []byte) {
	if !info.IsFileButNotDir() {
		return
	}
	if len(info.ERows) > 0 {
		erow0 := info.ERows[0]
		erow0.Row.TextArea.SetBytes(b) // will update other rows via callback
	}
}

func (info *ERowInfo) SetRowsStrFromMaster(erow *ERow) {
	if !info.IsFileButNotDir() {
		return
	}

	// disable/enable callback recursion
	disableCB := func(v bool) {
		for _, e := range info.ERows {
			e.disableTextAreaSetStrCallback = v
		}
	}
	disableCB(true)
	defer disableCB(false)

	info.updateDuplicatesBytes(erow)
	info.UpdateEditedRowState()

	GoDebugUpdateUIERowInfo(info)
}

//----------

func (info *ERowInfo) updateDuplicatesBytes(erow *ERow) {
	for _, e := range info.ERows {
		if e == erow {
			continue
		}
		erow.Row.TextArea.UpdateDuplicate(e.Row.TextArea.TextEdit)
	}
}

//----------

func isSpecialName(name string) bool {
	return name[0] == '+'
}

//----------

func bytesHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

//----------

func runGoImports(s []byte, dir string) ([]byte, error) {
	// timeout for the cmd to run
	timeout := 5000 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r := bytes.NewReader(s)
	return ExecCmdStdin(ctx, dir, r, osutil.ExecName("goimports"))
}
