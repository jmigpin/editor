package core

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/howeyc/fsnotify"
	"github.com/jmigpin/editor/ui"
	"github.com/pkg/errors"
)

type FilesWatcher struct {
	w  *FSNWatcher
	ed *Editor
	m  struct {
		sync.Mutex
		m map[*ERow]*FWEntry
	}
}

func NewFilesWatcher(ed *Editor) (*FilesWatcher, error) {
	w, err := NewFSNWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FilesWatcher{
		w:  w,
		ed: ed,
	}
	fw.m.m = make(map[*ERow]*FWEntry)
	return fw, nil
}
func (fw *FilesWatcher) Close() {
	fw.w.w.Close() // will close watcher chans (Events/Errors)
}

func (fw *FilesWatcher) AddUpdate(erow *ERow, f string) {
	err := fw.add2(erow, f)
	if err != nil {
		fw.ed.Errorf("fw add: %v: %v", err, f)
	}
}
func (fw *FilesWatcher) add2(erow *ERow, f string) error {
	fw.m.Lock()
	defer fw.m.Unlock()

	e, ok := fw.m.m[erow]
	if !ok {
		e = &FWEntry{}
		fw.m.m[erow] = e
	}

	return e.updateWatch(f, fw.w)
}
func (fw *FilesWatcher) Remove(erow *ERow) {
	err := fw.remove2(erow)
	if err != nil {
		fw.ed.Errorf("fw rm: %v", err)
	}
}
func (fw *FilesWatcher) remove2(erow *ERow) error {
	fw.m.Lock()
	defer fw.m.Unlock()

	e, ok := fw.m.m[erow]
	if !ok {
		return nil
	}
	delete(fw.m.m, erow)

	return e.unwatch(fw.w)
}
func (fw *FilesWatcher) EventLoop() {
	for {
		select {
		case err, ok := <-fw.w.w.Error:
			if !ok {
				return
			}
			fw.ed.Error(err)
		case ev, ok := <-fw.w.w.Event:
			if !ok {
				return
			}
			fw.handleEvent(ev)
		}
	}
}
func (fw *FilesWatcher) handleEvent(ev *fsnotify.FileEvent) {
	//log.Printf("fw event %+v %s", *ev, ev)
	err := fw.handleEvent2(ev)
	if err != nil {
		fw.ed.Error(err)
		fw.ed.UI().RequestTreePaint()
	}
}
func (fw *FilesWatcher) handleEvent2(ev *fsnotify.FileEvent) error {
	fw.m.Lock()
	defer fw.m.Unlock()

	// update name watchers on delete/rename
	if ev.IsDelete() || ev.IsRename() {
		for _, e := range fw.m.m {
			if ev.Name == e.fp.name {
				e.fp.remove(fw.w)
			}
			if ev.Name == e.bd.name {
				e.bd.remove(fw.w)
			}
		}
	}

	if ev.IsModify() {
		for erow, e := range fw.m.m {
			if ev.Name == e.fp.name {
				erow.row.Square.SetValue(ui.SquareDiskChanges, true)
				erow.ed.UI().RequestTreePaint()
			}
		}
	}

	for erow, e := range fw.m.m {
		if ev.IsModify() {
			continue
		}
		// delete/rename/create/attrib

		// TODO: check square disk changes / edited

		if ev.Name == e.fp.name || !e.fp.watching {
			err := e.updateWatch(e.fp.name, fw.w)
			if err != nil {
				return err
			}
		}
		if ev.Name == e.fp.name {
			erow.updateFileInfo()
			erow.ed.UI().RequestTreePaint()
		}
	}
	return nil
}

type FWEntry struct {
	fp FWNameWatch // filepath
	bd FWNameWatch // basedir when filepath doesn't exist
}

func (e *FWEntry) updateWatch(name string, w *FSNWatcher) (err3 error) {
	cleanBd := true
	defer func() {
		if cleanBd {
			err := e.bd.unwatch(w)
			if err != nil {
				err3 = err
			}
		}
	}()

	err := e.fp.updateWatch(name, w)
	if err != nil {
		err2 := errors.Cause(err)
		if os.IsNotExist(err2) {

			// watch base dir
			d, ok := watchableBaseDir(name)
			if ok {
				//log.Printf("bd update watch for %v", name)
				cleanBd = false
				err := e.bd.updateWatch(d, w)
				if err != nil {
					return err
				}
				return nil
			}

		}
		return err
	}
	return nil
}
func (e *FWEntry) unwatch(w *FSNWatcher) error {
	err := e.fp.unwatch(w)
	if err != nil {
		return err
	}
	return e.bd.unwatch(w)
}

func watchableBaseDir(name string) (string, bool) {
	d := name
	for {
		// base dir
		d0 := path.Dir(d)
		if d0 == d {
			break
		}
		d = d0

		// must exist
		fi, err := os.Stat(d)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			return d, true
		}
	}
	return "", false
}

type FWNameWatch struct {
	name     string
	watching bool
}

func (nw *FWNameWatch) updateWatch(name string, w *FSNWatcher) error {
	if nw.watching {
		if name == nw.name {
			return nil
		}
		err := nw.unwatch(w)
		if err != nil {
			return err
		}
	}
	nw.name = name
	err := w.watch(name)
	if err != nil {
		nw.watching = false
		return err
	}
	//log.Printf("nw watch: %v %v", nw.name, w.m.m[name])
	nw.watching = true
	return nil
}
func (nw *FWNameWatch) unwatch(w *FSNWatcher) error {
	if !nw.watching {
		return nil
	}
	//log.Printf("nw unwatch: %v %d", nw.name, w.m.m[nw.name])
	nw.watching = false
	return w.unwatch(nw.name)
}
func (nw *FWNameWatch) remove(w *FSNWatcher) {
	if !nw.watching {
		return
	}
	//log.Printf("nw remove: %v %d", nw.name, w.m.m[nw.name])
	nw.watching = false
	w.remove(nw.name)
}

// Ensure one request notify per path, and keep track of watch count. Allows an unwatch without other watchs being turned off.
type FSNWatcher struct {
	w *fsnotify.Watcher
	m struct {
		sync.Mutex
		m map[string]int
	}
}

func NewFSNWatcher() (*FSNWatcher, error) {
	w0, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &FSNWatcher{w: w0}
	w.m.m = make(map[string]int)
	return w, nil
}
func (w *FSNWatcher) watch(path string) error {
	w.m.Lock()
	defer w.m.Unlock()

	_, ok := w.m.m[path]
	if !ok {
		err := w.w.Watch(path)
		if err != nil {
			return errors.Wrapf(err, "watch: %v", path)
		}
	}
	w.m.m[path]++
	return nil
}
func (w *FSNWatcher) unwatch(path string) error {
	w.m.Lock()
	defer w.m.Unlock()

	w.remove2(path)

	_, ok := w.m.m[path]
	if !ok {
		err := w.w.RemoveWatch(path)
		if err != nil {
			return errors.Wrapf(err, "unwatch: %v", path)
		}
	}
	return nil
}
func (w *FSNWatcher) remove(path string) {
	w.m.Lock()
	defer w.m.Unlock()
	w.remove2(path)
}
func (w *FSNWatcher) remove2(path string) {
	v, ok := w.m.m[path]
	if !ok || v <= 0 {
		panic(fmt.Sprintf("unwatch: bad count: %v", v))
	}

	// even if it won't remove, the count will be reduced
	w.m.m[path]--
	if w.m.m[path] == 0 {
		delete(w.m.m, path)
	}
}

func FWStatus(erow *ERow) {
	s := fwStatus2(erow)
	erow.row.TextArea.SetStrClear(s, false, false)
}
func fwStatus2(erow *ERow) string {
	fw := erow.ed.fw
	fw.m.Lock()
	defer fw.m.Unlock()
	s := ""
	for _, e := range fw.m.m {
		s += fmt.Sprintf("%+v\n", *e)
	}
	s += "---\n"
	fw.w.m.Lock()
	defer fw.w.m.Unlock()
	for str, c := range fw.w.m.m {
		s += fmt.Sprintf("%v:%v\n", str, c)
	}
	return s
}
