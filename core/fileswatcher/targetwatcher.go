package fileswatcher

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/davecgh/go-spew/spew"
)

// Watches for target file/dir even if they don't exist but get created in the future.
type TargetWatcher struct {
	entries struct {
		sync.Mutex
		m map[string]*Entry
	}
	paths struct {
		sync.Mutex
		m map[string]int
	}

	w *BasicWatcher

	Events chan interface{}
	close  chan struct{}

	Logf Logf
}

func NewTargetWatcher(logf Logf) (*TargetWatcher, error) {
	w := &TargetWatcher{
		Events: make(chan interface{}),
		close:  make(chan struct{}),
		Logf:   func(string, ...interface{}) {},
	}
	w.entries.m = make(map[string]*Entry)
	w.paths.m = make(map[string]int)

	//w.Logf = log.Printf
	if logf != nil {
		w.Logf = logf
	}

	w2, err := NewBasicWatcher(w.Logf)
	if err != nil {
		return nil, err
	}
	w.w = w2

	go w.eventLoop()

	return w, nil
}
func (w *TargetWatcher) Close() {
	close(w.close)
}

func (w *TargetWatcher) Add(name string) {
	w.entries.Lock()
	defer w.entries.Unlock()

	e, ok := w.entries.m[name]
	if ok {
		return
	}
	e = &Entry{name: name}
	w.entries.m[name] = e

	w.addWatch(e)
}
func (w *TargetWatcher) Remove(name string) {
	w.entries.Lock()
	defer w.entries.Unlock()

	e, ok := w.entries.m[name]
	if !ok {
		return
	}
	delete(w.entries.m, name)

	w.removePath(e.path)
}

func (w *TargetWatcher) eventLoop() {
	for {
		select {
		case <-w.close:
			goto forEnd
		case ev, ok := <-w.w.Events:
			if !ok {
				goto forEnd
			}

			//w.Logf("%+v", ev)

			switch ev2 := ev.(type) {
			case error:
				w.Events <- ev2
			case *Event:
				evs := w.handleEvent(ev2)
				for _, e := range evs {
					w.Events <- e
				}
			default:
				panic(spew.Sdump(ev))
			}
		}
	}
forEnd:
	w.w.Close()
	close(w.Events)
}

func (w *TargetWatcher) handleEvent(ev *Event) (evs []interface{}) {
	w.entries.Lock()
	defer w.entries.Unlock()

	if ev.Op.HasModify() {
		e, ok := w.entries.m[ev.Name]
		if ok && e.exist {
			evs = append(evs, ev)
		}
		return
	}

	states := make(map[string]bool)
	for _, e := range w.entries.m {
		states[e.name] = e.exist
	}

	// create, delete, ignored
	for name, e := range w.entries.m {
		//w.Logf("checking with %v", name)

		// TODO: check filename?

		if strings.HasPrefix(name, ev.Name) {
			w.Logf("rechecking path %+v", e)
			w.addWatch(e)
		}
	}

	for name, exist := range states {
		e := w.entries.m[name]
		if exist != e.exist {
			if e.exist {
				u := &Event{Name: name, Op: Op(unix.IN_CREATE)}
				evs = append(evs, u)
			} else {
				u := &Event{Name: name, Op: Op(unix.IN_DELETE)}
				evs = append(evs, u)
			}
		}
	}
	return
}

// addWatch attempts to solve this case:
//	watching dir2
//	create dir2/dir3 (event)
//		create dir2/dir3/dir4 (in the meantime, this happens)
//		watching dir3 (action)
//	create file at dir4 (not watched because watching dir3 after dir4 created)
// watching dir3 with the assurance that dir4 was not created yet solves the problem

func (w *TargetWatcher) addWatch(e *Entry) {
	err := w.addWatch2(e)
	if err != nil {
		panic(err)
	}
}
func (w *TargetWatcher) addWatch2(e *Entry) error {
	e.exist = false
	if e.path != "" {
		w.removePath(e.path)
		e.path = ""
	}

	name := path.Clean(e.name)

	lastDir := func(s string) bool {
		return s == "/" || s == "."
	}

	// retries to do if structure changes while adding
	for k := 0; k < 10; k++ {

		// TODO: test addwatch func with "/" and "." args

		var prev string
		u := name
		for i := 0; i == 0 || !lastDir(u); i++ {
			err := w.addPath(u)
			if err != nil {
				prev = u
				u = path.Dir(u)
				continue
			}

			if i > 0 {
				// has a previous
				err := w.addPath(prev)
				if err == nil {
					// structure has changed
					// need to restart the process from the top and remove the
					// added name that was tought to be the one to add
					w.removePath(u)
					w.removePath(prev)
					break
				}
			}

			if i == 0 {
				e.exist = true
			}

			e.path = u

			w.Logf("added: %v for %v", u, e)
			return nil
		}
	}

	return fmt.Errorf("unable to add name: %s", name)
}

func (w *TargetWatcher) addPath(name string) error {
	err := w.w.Add(name)
	if err != nil {
		return err
	}
	w.paths.Lock()
	defer w.paths.Unlock()
	w.paths.m[name]++
	return nil
}
func (w *TargetWatcher) removePath(name string) {
	w.paths.Lock()
	defer w.paths.Unlock()
	v, ok := w.paths.m[name]
	if !ok {
		return
	}
	v--
	if v == 0 {
		delete(w.paths.m, name)
		err := w.w.Remove(name)
		if err != nil {
			// ignore error, best effort
			w.Logf("%v", err)
		}
	} else {
		if v < 0 {
			panic("v<0")
		}
		w.paths.m[name] = v
	}
}

type Entry struct {
	name  string
	exist bool
	path  string // current path used to receive events about this entry
}

//func isDirPrefix(a, b string) bool {
//	return a != b && strings.HasPrefix(b, a)
//}
//func fileExist(name string) bool {
//	_, err := os.Stat(name)
//	return err == nil || !os.IsNotExist(err)
//}
