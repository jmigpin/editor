package edit

import "github.com/howeyc/fsnotify"

type FilesWatcher struct {
	filenames map[string]struct{}
	w         *fsnotify.Watcher
	OnError   func(error)
	OnEvent   func(ev *fsnotify.FileEvent)
	OnDebug   func(string)
}

func NewFilesWatcher() (*FilesWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FilesWatcher{w: w, filenames: make(map[string]struct{})}
	return fw, nil
}
func (fw *FilesWatcher) Close() {
	fw.w.Close() // will close watcher chans (Events/Errors)
}
func (fw *FilesWatcher) Filenames() []string {
	var a []string
	for k := range fw.filenames {
		a = append(a, k)
	}
	return a
}
func (fw *FilesWatcher) SetFiles(filenames []string) {
	// start watching if not watched yet
	seen := make(map[string]struct{})
	for _, f := range filenames {
		seen[f] = struct{}{}
		err := fw.Add(f)
		if err != nil {
			fw.OnError(err)
			//fw.OnDebug(err.Error())
		}
	}
	// stop watching filenames not seen
	for f, _ := range fw.filenames {
		_, ok := seen[f]
		if !ok {
			// best effort to remove, ignore errors
			err := fw.Remove(f)
			if err != nil {
				fw.OnError(err)
				//fw.OnDebug(err.Error())
			}
		}
	}
	//// debug
	//fw.OnDebug(fmt.Sprintf("%v", fw.Filenames()))
}
func (fw *FilesWatcher) Add(f string) error {
	_, ok := fw.filenames[f]
	if ok {
		return nil
	}
	err := fw.w.Watch(f)
	if err != nil {
		return err
	}
	fw.filenames[f] = struct{}{}
	return nil
}

func (fw *FilesWatcher) Remove(f string) error {
	_, ok := fw.filenames[f]
	if !ok {
		return nil
	}
	// Previously used library (github.com/fsnotify/fsnotify) was locking on this call
	err := fw.w.RemoveWatch(f)
	if err != nil {
		return err
	}
	delete(fw.filenames, f)
	return nil
}
func (fw *FilesWatcher) EventLoop() {
	for {
		select {
		case ev, ok := <-fw.w.Event:
			if !ok {
				return
			}
			fw.OnEvent(ev)
		case err, ok := <-fw.w.Error:
			if !ok {
				return
			}
			fw.OnError(err)
		}
	}
}
