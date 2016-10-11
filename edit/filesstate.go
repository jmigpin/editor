package edit

import "github.com/fsnotify/fsnotify"

type FilesState struct {
	filenames map[string]struct{}
	w         *fsnotify.Watcher
	OnError   func(error)
	OnEvent   func(ev fsnotify.Event)
}

func NewFilesState() (*FilesState, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fs := &FilesState{w: w, filenames: make(map[string]struct{})}
	return fs, nil
}
func (fs *FilesState) Close() {
	fs.w.Close() // will close fs.w.{Events,Errors} chans
}
func (fs *FilesState) SetFiles(filenames []string) {
	// start watching if not watched yet
	seen := make(map[string]struct{})
	for _, f := range filenames {
		seen[f] = struct{}{}
		// best effort to add, ignore errors
		_ = fs.Add(f)
	}
	// stop watching filenames not seen
	for f, _ := range fs.filenames {
		_, ok := seen[f]
		if !ok {
			// best effort to remove, ignore errors
			_ = fs.Remove(f)
		}
	}
}
func (fs *FilesState) Add(f string) error {
	_, ok := fs.filenames[f]
	if ok {
		return nil
	}
	err := fs.w.Add(f)
	if err != nil {
		return err
	}
	fs.filenames[f] = struct{}{}
	return nil
}
func (fs *FilesState) Remove(f string) error {
	_, ok := fs.filenames[f]
	if !ok {
		return nil
	}
	err := fs.w.Remove(f)
	if err != nil {
		return err
	}
	delete(fs.filenames, f)
	return nil
}
func (fs *FilesState) EventLoop() {
	for {
		select {
		case ev, ok := <-fs.w.Events:
			if !ok {
				return
			}
			fs.OnEvent(ev)
		case err, ok := <-fs.w.Errors:
			if !ok {
				return
			}
			fs.OnError(err)
		}
	}
}
