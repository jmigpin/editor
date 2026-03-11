package fswatcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jmigpin/editor/util/syncutil"
)

type FsnWatcher struct {
	w      *fsnotify.Watcher
	q      *syncutil.SyncedQ
	opMask Op
}

func NewFsnWatcher() (*FsnWatcher, error) {
	w0, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &FsnWatcher{
		w:      w0,
		q:      syncutil.NewSyncedQ(),
		opMask: AllOps,
	}

	go w.eventLoop()
	return w, nil
}

//----------

func (w *FsnWatcher) Close() error {
	err := w.w.Close()
	w.q.PushBack(nil)
	return err
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

func (w *FsnWatcher) NextEvent() any {
	return w.q.PopFront()
}

//----------

func (w *FsnWatcher) eventLoop() {
	for {
		select {
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			w.q.PushBack(err)

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

			if op2 := op & w.opMask; op2 != 0 {
				w.q.PushBack(&Event{Op: op2, Name: name})
			}
		}
	}
}
