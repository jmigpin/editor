package core

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
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

	// file type only
	fileData struct {
		// saved/memory (keep even if file is deleted and reappears later)
		saved struct {
			size int
			hash []byte
		}
		// filesystem (reflects changes by other programs)
		fs struct {
			hash    []byte
			modTime time.Time
		}
		// not always up to date, used if the hash is being requested without the contents being changed
		edited struct {
			updated bool
			size    int
			hash    []byte
		}
	}

	cmd struct {
		sync.Mutex
		cancelCmd context.CancelFunc
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
	info.fileData.edited.updated = false
}

func (info *ERowInfo) updateEditedHash() {
	if info.fileData.edited.updated {
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
	info.fileData.edited.size = size
	info.fileData.edited.hash = hash
	info.fileData.edited.updated = true
}

func (info *ERowInfo) setSavedHash(hash []byte, size int) {
	info.fileData.saved.size = size
	info.fileData.saved.hash = hash
	info.UpdateFsDifferRowState()
}

func (info *ERowInfo) setFsHash(hash []byte) {
	if info.fi == nil {
		return
	}
	//info.fsHash.size = int(info.fi.Size()) // TODO: downgrading if 32bit system
	info.fileData.fs.hash = hash
	info.fileData.fs.modTime = info.fi.ModTime()
	info.UpdateFsDifferRowState()
}

func (info *ERowInfo) updateFsHashIfNeeded() {
	if !info.IsFileButNotDir() {
		return
	}
	if info.fi == nil {
		return
	}
	if !info.fi.ModTime().Equal(info.fileData.fs.modTime) {
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

func (info *ERowInfo) ReloadFile() error {
	b, err := info.readFsFile()
	if err != nil {
		return err
	}

	// update data
	info.setSavedHash(info.fileData.fs.hash, len(b))

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

	// run src formatters (ex: goimports)
	ctx1, cancel1 := info.newCmdCtx()
	defer cancel1()
	if b2, err := info.Ed.runPreSaveHooks(ctx1, info, b); err != nil {
		// ignore errors, can catch them when compiling
		//info.Ed.Error(err)
	} else {
		b = b2
	}

	if err := info.saveFsFile(b); err != nil {
		return err
	}

	// update all erows (including row saved states)
	info.SetRowsBytes(b)

	// editor events
	ev := &PostFileSaveEEvent{Info: info}
	info.Ed.EEvents.emit(PostFileSaveEEventId, ev)

	//// warn lsproto of file save
	//go func() {
	//	ctx2, cancel2 := info.newCmdCtx()
	// 	defer cancel2()
	//	ctx3, cancel3 := context.WithTimeout(ctx2, 3 * time.Second)
	//	defer cancel3()
	//	err := info.Ed.LSProtoMan.DidSave(ctx3, info.Name(), nil)
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
	return bytes.Equal(hash, info.fileData.edited.hash)
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
	edited := !info.EqualToBytesHash(info.fileData.saved.size, info.fileData.saved.hash)
	info.updateRowsStates(ui.RowStateEdited, edited)
}

func (info *ERowInfo) UpdateExistsRowState() {
	info.updateRowsStates(ui.RowStateNotExist, info.IsNotExist())
}

func (info *ERowInfo) UpdateFsDifferRowState() {
	if !info.IsFileButNotDir() {
		return
	}
	h1 := info.fileData.fs.hash
	h2 := info.fileData.saved.hash
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

func (info *ERowInfo) newCmdCtx() (context.Context, context.CancelFunc) {
	info.cmd.Lock()
	defer info.cmd.Unlock()
	info.cancelCmd2()
	ctx0 := context.Background() // TODO: editor ctx
	ctx, cancel := context.WithCancel(ctx0)
	info.cmd.cancelCmd = cancel
	return ctx, cancel
}
func (info *ERowInfo) CancelCmd() {
	info.cmd.Lock()
	defer info.cmd.Unlock()
	info.cancelCmd2()
}
func (info *ERowInfo) cancelCmd2() {
	if info.cmd.cancelCmd != nil {
		info.cmd.cancelCmd()
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
