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
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
)

//godebug:annotatefile

// TODO: become an interface, with file/dir/special implementations.
// TODO: centralized iorw reader/writer in info

// Editor Row Info.
type ERowInfo struct {
	Ed    *Editor
	ERows []*ERow // added order

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

func readERowInfoOrNew(ed *Editor, name string) *ERowInfo {
	name = osutil.FilepathClean(name)

	// try to update the instance already used
	info, ok := ed.ERowInfo(name)
	if ok {
		info.readFileInfo()
		return info
	}

	// new erow info
	info = &ERowInfo{Ed: ed, name: name}
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

	// ensure name
	if info.Ed.FsCaseInsensitive {
		n := fi.Name()
		info.name = info.name[:len(info.name)-len(n)] + n
	}

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
	if info.editedHash.updated {
		return
	}
	// read from one of the erows
	erow0, ok := info.FirstERow()
	if !ok {
		return
	}
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

func (info *ERowInfo) FirstERow() (*ERow, bool) {
	if len(info.ERows) > 0 {
		return info.ERows[0], true
	}
	return nil, false
}

//----------

func (info *ERowInfo) NewERow(rowPos *ui.RowPos) (*ERow, error) {
	switch {
	case info.IsSpecial():
		// there can be only one instance of a special row
		if len(info.ERows) > 0 {
			return nil, fmt.Errorf("special row already exists: %v", info.Name())
		}
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
	if erow0, ok := info.FirstERow(); ok {
		// create erow first to get it updated
		erow := NewERow(info.Ed, info, rowPos)
		// update the new erow with content
		info.setRWFromMaster(erow0)
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
	if !info.IsFileButNotDir() {
		return fmt.Errorf("not a file: %s", info.Name())
	}

	// read from one of the erows
	erow0, ok := info.FirstERow()
	if !ok {
		return nil
	}
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

	// editor events
	ev := &PostFileSaveEEvent{Info: info}
	info.Ed.EEvents.emit(PostFileSaveEEventId, ev)

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
	erow0, ok := info.FirstERow()
	if !ok {
		return false
	}
	if erow0.Row.TextArea.Len() != size {
		return false
	}
	info.updateEditedHash()
	return bytes.Equal(hash, info.editedHash.hash)
}

//----------

func (info *ERowInfo) HasRowState(st ui.RowState) bool {
	erow0, ok := info.FirstERow()
	if !ok {
		return false
	}
	return erow0.Row.HasState(st)
}

//----------

func (info *ERowInfo) UpdateEditedRowState() {
	if !info.IsFileButNotDir() {
		return
	}
	info.editedHashNeedsUpdate()
	edited := !info.EqualToBytesHash(info.savedHash.size, info.savedHash.hash)
	info.updateRowsStates(ui.RowStateEdited, edited)
}

func (info *ERowInfo) UpdateExistsRowState() {
	info.updateRowsStates(ui.RowStateNotExist, info.IsNotExist())
}

func (info *ERowInfo) UpdateFsDifferRowState() {
	if !info.IsFileButNotDir() {
		return
	}
	h1 := info.fsHash.hash
	h2 := info.savedHash.hash
	differ := !bytes.Equal(h1, h2)
	info.updateRowsStates(ui.RowStateFsDiffer, differ)
}

func (info *ERowInfo) UpdateDuplicateRowState() {
	hasDups := len(info.ERows) >= 2
	info.updateRowsStates(ui.RowStateDuplicate, hasDups)
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
	info.updateRowsStates(ui.RowStateDuplicateHighlight, hasDups && on)
}

func (info *ERowInfo) UpdateAnnotationsRowState(v bool) {
	info.updateRowsStates(ui.RowStateAnnotations, v)
}

func (info *ERowInfo) UpdateAnnotationsEditedRowState(v bool) {
	info.updateRowsStates(ui.RowStateAnnotationsEdited, v)
}

func (info *ERowInfo) UpdateActiveRowState(erow *ERow) {
	// disable first the previous active row
	for _, er := range info.Ed.ERows() {
		if er != erow {
			info.updateRowState(er, ui.RowStateActive, false)
		}
	}
	// activate row
	info.updateRowState(erow, ui.RowStateActive, true)
}

//----------

func (info *ERowInfo) updateRowsStates(state ui.RowState, v bool) {
	// update this info rows state
	for _, erow := range info.ERows {
		info.updateRowState(erow, state, v)
	}
}

func (info *ERowInfo) updateRowState(erow *ERow, state ui.RowState, v bool) {
	oldState := erow.Row.HasState(state)
	if oldState != v {
		erow.Row.SetState(state, v)

		// editor events
		ev := &RowStateChangeEEvent{ERow: erow, State: state, Value: v}
		erow.Ed.EEvents.emit(RowStateChangeEEventId, ev)
	}
}

//----------

func (info *ERowInfo) SetRowsBytes(b []byte) {
	if !info.IsFileButNotDir() {
		return
	}
	if erow0, ok := info.FirstERow(); ok {
		// gets to duplicates via callback that in practice will only set pointers to share RW and History
		erow0.Row.TextArea.SetBytes(b)
	}
}

//----------

func (info *ERowInfo) HandleRWEvWrite2(erow *ERow, ev *iorw.RWEvWrite2) {
	if !info.IsFileButNotDir() {
		return
	}
	info.setRWFromMaster(erow)
	info.handleRWsWrite2(erow, ev)
}

func (info *ERowInfo) setRWFromMaster(erow *ERow) {
	for _, e := range info.ERows {
		if e == erow {
			continue
		}
		e.Row.TextArea.SetRWFromMaster(erow.Row.TextArea.TextEdit)
	}
	info.UpdateEditedRowState()
	info.Ed.GoDebug.UpdateUIERowInfo(info)
}

func (info *ERowInfo) handleRWsWrite2(erow *ERow, ev *iorw.RWEvWrite2) {
	for _, e := range info.ERows {
		if e == erow {
			continue
		}
		e.Row.TextArea.HandleRWWrite2(ev)
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
