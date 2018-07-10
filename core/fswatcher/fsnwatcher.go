package fswatcher

import (
	"path/filepath"

	fsnotify "github.com/fsnotify/fsnotify"
)

type FsnWatcher struct {
	w      *fsnotify.Watcher
	events chan interface{}
	opMask Op
}

func NewFsnWatcher() (*FsnWatcher, error) {
	w0, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &FsnWatcher{
		w:      w0,
		events: make(chan interface{}),
	}

	w.opMask = AllOps

	go w.eventLoop()
	return w, nil
}

//----------

func (w *FsnWatcher) Close() error {
	return w.w.Close()
}

func (w *FsnWatcher) OpMask() *Op {
	return &w.opMask
}

//----------

func (w *FsnWatcher) Add(name string) error {
	return w.w.Add(name)
}
func (w *FsnWatcher) Remove(name string) error {
	return w.w.Remove(name)
}

//----------

func (w *FsnWatcher) Events() <-chan interface{} {
	return w.events
}

//----------

func (w *FsnWatcher) eventLoop() {
	for {
		select {
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}

			//log.Printf("got err %v", err)

			w.events <- err

		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}

			//log.Printf("fsnotify ev %v", ev)

			name := ev.Name
			subName := ""

			var op Op
			if ev.Op&fsnotify.Create > 0 {
				op.Add(Create)
				// make event name dir, with subname file
				n, sn := filepath.Split(name)
				name, subName = filepath.Clean(n), sn
			}
			if ev.Op&fsnotify.Write > 0 {
				op.Add(Modify)
			}
			if ev.Op&fsnotify.Remove > 0 {
				op.Add(Remove)
			}
			if ev.Op&fsnotify.Rename > 0 {
				op.Add(Rename)
			}
			if ev.Op&fsnotify.Chmod > 0 {
				op.Add(Attrib)
			}

			op2 := op & w.opMask
			if op2 > 0 {
				w.events <- &Event{Op: op, Name: name, SubName: subName}
			} else {
				//log.Printf("not sending event: %v", ev)
			}
		}
	}
}
