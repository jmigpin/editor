package fswatcher

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

// Watches files even if they don't exist yet.
type TargetWatcher struct {
	Watcher

	events chan interface{}

	m struct {
		sync.Mutex
		names   map[string]*Name
		watches map[string]*Watch
	}
}

func NewTargetWatcher(w Watcher) *TargetWatcher {

	tw := &TargetWatcher{Watcher: w}
	tw.events = make(chan interface{}, 0)

	tw.m.names = map[string]*Name{}
	tw.m.watches = map[string]*Watch{}

	go tw.eventLoop()

	return tw
}

//----------

func (tw *TargetWatcher) Events() <-chan interface{} {
	return tw.events
}

//----------

func (tw *TargetWatcher) Add(name string) error {
	name = filepath.Clean(name)

	tw.m.Lock()
	defer tw.m.Unlock()

	_, ok := tw.m.names[name]
	if ok {
		return nil
	}

	p, err := tw.addCloseWatch(name)
	if err != nil {
		return err
	}

	tw.link(p, name)

	return nil
}

//----------

func (tw *TargetWatcher) Remove(name string) error {
	name = filepath.Clean(name)

	tw.m.Lock()
	defer tw.m.Unlock()

	n, ok := tw.m.names[name]
	if !ok {
		return fmt.Errorf("name not being watched: %v", name)
	}

	// name
	delete(tw.m.names, n.Str)

	// watch
	w := n.Watch
	delete(w.Names, n.Str)
	if len(w.Names) == 0 {
		delete(tw.m.watches, w.Str)
		_ = tw.Watcher.Remove(w.Str)
	}

	return nil
}

//----------

func (tw *TargetWatcher) link(watch, name string) {
	// watch
	w, ok := tw.m.watches[watch]
	if !ok {
		w = &Watch{Str: watch, Names: map[string]*Name{}}
		tw.m.watches[watch] = w
	}
	// name
	n, ok := tw.m.names[name]
	if !ok {
		n = &Name{Str: name}
		tw.m.names[name] = n
	}
	n.Watch = w
	w.Names[n.Str] = n
}

//----------

func (tw *TargetWatcher) addCloseWatch(name string) (string, error) {
	s, err := tw.addCloseWatch2(name)

	//// DEBUG
	//log.Printf("addclosewatch: %v -> %v (err=%v)", name, s, err)

	return s, err
}
func (tw *TargetWatcher) addCloseWatch2(name string) (string, error) {
	//// debug
	//log.Printf("close watch start: %v", name)
	//defer log.Printf("close watch done")

	up := true             // parent directories (shorter paths)
	downList := []string{} // child directories (longer paths)

	p := name
	best := ""
	for {
		p = filepath.Clean(p)

		if up {
			if p == "/" || p == "." {
				err := fmt.Errorf("bad parent dir: %v, %v", p, name)
				return "", errors.Wrap(err, "addclosewatch")
			}
		}

		//// debug
		//log.Printf("trying close watch: %v", p)

		err := tw.Watcher.Add(p)
		if err != nil {
			//log.Printf("close watch add fail: %v", p)
			if up {
				// try to add next parent dir
				dir, file := filepath.Split(p)
				downList = append([]string{file}, downList...)
				p = dir
				continue
			} else {
				// can't do better then this, got error while trying child dir, use best parent
				return best, nil
			}
		}

		// clear previous best if not being watched by others
		if best != "" {
			_, ok := tw.m.watches[best]
			if !ok {
				if err := tw.Watcher.Remove(best); err != nil {
					return "", errors.Wrap(err, "addclosewatch rm")
				}
			}
		}

		best = p
		up = false

		if len(downList) > 0 {
			// try to re-add next child dir
			p = filepath.Join(best, downList[0])
			downList = downList[1:]
			continue
		}

		return best, nil
	}
}

//----------

func (tw *TargetWatcher) reviewWatch(watch string, ev *Event) []*Event {
	tw.m.Lock()
	defer tw.m.Unlock()

	//log.Printf("review watch: %v", ev)

	w, ok := tw.m.watches[watch]
	if !ok {
		return nil
	}

	evs := []*Event{}
	for _, n := range w.Names {
		ev2 := tw.reviewWatchName(w, n, ev)
		if ev2 != nil {
			evs = append(evs, ev2)
		}
	}
	return evs
}

func (tw *TargetWatcher) reviewWatchName(w *Watch, n *Name, ev *Event) *Event {
	// emit event
	if w.Str == n.Str {
		return ev
	}

	p, err := tw.addCloseWatch(n.Str)
	if err != nil {
		err2 := errors.Wrap(err, "review watch name")
		log.Print(err2)
		return nil
	}

	tw.link(p, n.Str)

	// clear old watch
	if w.Str != p {
		delete(w.Names, n.Str)
		if len(w.Names) == 0 {
			delete(tw.m.watches, w.Str)
			_ = tw.Watcher.Remove(w.Str)
		}
	}

	// NOTE: the ev name is not n.Str

	var ev2 *Event

	// emit event if it refers to name
	if ev2 == nil {
		jn := ev.JoinNames()
		refersToName := jn == n.Str
		if refersToName {
			switch {
			case ev.Op.HasAny(Create):
				ev2 = &Event{Create, n.Str, ""}
			case ev.Op.HasAny(Rename):
				ev2 = &Event{Rename, n.Str, ""}
			case ev.Op.HasAny(Modify):
				ev2 = &Event{Modify, n.Str, ""}
			}
		}
	}

	// emit event if it is watching now
	if ev2 == nil {
		watchingNow := p == n.Str
		if watchingNow {
			ev2 = &Event{Create, n.Str, ""}
		}
	}

	if ev2 != nil {
		log.Printf("review watch emit: %v, watch %v", ev2, p)
	} else {
		log.Printf("review watch not emitted: %v, watch %#v", ev, p)
	}

	return ev2
}

//----------

func (tw *TargetWatcher) eventLoop() {
	defer close(tw.events)

	for {
		ev, ok := <-tw.Watcher.Events()
		if !ok {
			break
		}

		switch t := ev.(type) {
		case error:
			tw.events <- t
		case *Event:
			evs := tw.reviewWatch(t.Name, t)
			for _, e := range evs {
				tw.events <- e
			}
		}
	}
}

//----------

func (tw *TargetWatcher) LogNames() {
	for k := range tw.m.names {
		log.Printf("tw names %v", k)
	}
}

//----------

type Name struct {
	Str   string
	Watch *Watch
}
type Watch struct {
	Str   string
	Names map[string]*Name
}
