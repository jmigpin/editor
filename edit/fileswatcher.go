package edit

import (
	"log"
	"sync"

	"github.com/howeyc/fsnotify"
)

type FilesWatcher struct {
	w  *fsnotify.Watcher
	ed *Editor
	m  struct {
		sync.Mutex
		m map[*ERow]string
	}
}

func NewFilesWatcher(ed *Editor) (*FilesWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FilesWatcher{
		w:  w,
		ed: ed,
	}
	fw.m.m = make(map[*ERow]string)
	return fw, nil
}
func (fw *FilesWatcher) Close() {
	fw.w.Close() // will close watcher chans (Events/Errors)
}

func (fw *FilesWatcher) Add(erow *ERow, f string) {
	err := fw.add2(erow, f)
	if err != nil {
		fw.ed.Error(err)
	}
}
func (fw *FilesWatcher) add2(erow *ERow, f string) error {
	fw.m.Lock()
	defer fw.m.Unlock()

	// do nothing if same argument
	f2, ok := fw.m.m[erow]
	if ok && f2 == f {
		return nil
	}

	err := fw.remove3(erow)
	if err != nil {
		log.Println(err)
	}

	err = fw.w.Watch(f)
	if err == nil {
		fw.m.m[erow] = f
	}
	return err
}

func (fw *FilesWatcher) Remove(erow *ERow) {
	err := fw.remove2(erow)
	if err != nil {
		fw.ed.Error(err)
	}
}
func (fw *FilesWatcher) remove2(erow *ERow) error {
	fw.m.Lock()
	defer fw.m.Unlock()
	return fw.remove3(erow)
}
func (fw *FilesWatcher) remove3(erow *ERow) error {
	f, ok := fw.m.m[erow]
	if !ok {
		return nil
	}
	delete(fw.m.m, erow)
	return fw.w.RemoveWatch(f)
}

func (fw *FilesWatcher) EventLoop() {
	for {
		select {
		case err, ok := <-fw.w.Error:
			if !ok {
				return
			}
			fw.ed.Error(err)
		case ev, ok := <-fw.w.Event:
			if !ok {
				return
			}

			fw.m.Lock()
			var u []*ERow
			for erow, s := range fw.m.m {
				if ev.Name == s {
					u = append(u, erow)
				}
			}
			fw.m.Unlock()
			for _, erow := range u {
				erow.OnFilesWatcherEvent(ev)
			}
		}
	}
}
