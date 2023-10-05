package fswatcher

import (
	fsnotify "github.com/fsnotify/fsnotify"
)

type FsnWatcher struct {
	w      *fsnotify.Watcher
	events chan any
	opMask Op
}

func NewFsnWatcher() (*FsnWatcher, error) {
	w0, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &FsnWatcher{
		w:      w0,
		events: make(chan any),
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

func (w *FsnWatcher) Events() <-chan any {
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
			w.events <- err

		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}
			name := ev.Name
			var op Op
			if ev.Op&fsnotify.Create > 0 {
				op.Add(Create)
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
				w.events <- &Event{Op: op, Name: name}
			} else {
				//log.Printf("not sending event: %v", ev)
			}
		}
	}
}
