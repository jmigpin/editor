package edit

import "github.com/howeyc/fsnotify"

type FilesWatcher struct {
	w  *fsnotify.Watcher
	m  map[*ERow]string
	ed *Editor
}

func NewFilesWatcher(ed *Editor) (*FilesWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FilesWatcher{
		w:  w,
		m:  make(map[*ERow]string),
		ed: ed,
	}
	return fw, nil
}
func (fw *FilesWatcher) Close() {
	fw.w.Close() // will close watcher chans (Events/Errors)
}
func (fw *FilesWatcher) Add(erow *ERow, f string) {
	_, ok := fw.m[erow]
	if ok {
		fw.Remove(erow)
	}
	fw.m[erow] = f
	err := fw.w.Watch(f)
	if err != nil {
		fw.ed.Error(err)
	}
}
func (fw *FilesWatcher) Remove(erow *ERow) {
	s, ok := fw.m[erow]
	if ok {
		delete(fw.m, erow)
		err := fw.w.RemoveWatch(s)
		if err != nil {
			fw.ed.Error(err)
		}
	}
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
			for erow, s := range fw.m {
				if ev.Name == s {
					erow.OnFilesWatcherEvent(ev)
				}
			}
		}
	}
}
