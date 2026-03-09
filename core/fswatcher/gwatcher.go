package fswatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// graph based watcher
type GWatcher struct {
	w      Watcher
	events chan any
	root   struct {
		sync.Mutex
		n *Node
	}
}

func NewGWatcher(w Watcher) *GWatcher {
	*w.OpMask() = AllOps

	gw := &GWatcher{w: w}
	gw.events = make(chan any)

	gw.root.Lock()
	gw.root.n = NewNode(string(os.PathSeparator), nil)
	gw.root.Unlock()

	go gw.eventLoop()
	return gw
}

//----------

func (gw *GWatcher) OpMask() *Op {
	return gw.w.OpMask()
}

func (gw *GWatcher) Close() error {
	return gw.w.Close()
}

//----------

func (gw *GWatcher) Events() <-chan any {
	return gw.events
}
func (gw *GWatcher) eventLoop() {
	defer close(gw.events)
	for {
		ev, ok := <-gw.w.Events()
		if !ok {
			break
		}
		switch t := ev.(type) {
		case error:
			gw.events <- t
		case *Event:
			gw.handleEv(t)
		}
	}
}
func (gw *GWatcher) handleEv(ev *Event) {
	u := ev.Name
	if ev.Op.HasAny(Create | Remove | Rename) {
		if err := gw.review(u); err != nil {
			gw.events <- fmt.Errorf("gwatcher review failed (op=%v, name=%q): %w", ev.Op, u, err)
		}
	}
	if ev.Op.HasAny(Modify | Attrib) {
		if err := gw.modify(u); err != nil {
			gw.events <- fmt.Errorf("gwatcher modify failed (op=%v, name=%q): %w", ev.Op, u, err)
		}
		if err := gw.resync(u); err != nil {
			gw.events <- fmt.Errorf("gwatcher resync failed (op=%v, name=%q): %w", ev.Op, u, err)
		}
	}
}

//----------

func (gw *GWatcher) Add(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	gw.root.n.add(v, func(n *Node) {
		p := n.path()
		if !n.added {
			err := gw.w.Add(p)
			if err == nil {
				n.added = true
			}
		}
	})

	return nil
}

//----------

func (gw *GWatcher) Remove(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	gw.root.n.remove(v, func(n *Node) {
		if n.target {
			n.target = false
			if n.added {
				n.added = false
				p := n.path()
				_ = gw.w.Remove(p)
			}
		}
		if len(n.childs) == 0 {
			n.delete()
		}
	})

	return nil
}

//----------

func (gw *GWatcher) review(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	gw.root.n.review(v, func(n *Node) {
		p := n.path()
		err := gw.w.Add(p)
		wasAdded := n.added
		n.added = err == nil
		if n.target {
			if !wasAdded && n.added {
				gw.events <- &Event{Op: Create, Name: p}
			}
			if wasAdded && !n.added {
				gw.events <- &Event{Op: Remove, Name: p}
			}
		}
	})

	return nil
}

//----------

func (gw *GWatcher) modify(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	gw.root.n.modify(v, func(n *Node) {
		if n.target {
			p := n.path()
			gw.events <- &Event{Op: Modify, Name: p}
		}
	})

	return nil
}

// resync updates watched descendants under name and emits:
// - Create/Remove on existence transitions
// - Resync when target remains watchable and should be re-checked by callers
func (gw *GWatcher) resync(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()

	n := gw.root.n.find(v)
	if n == nil || len(n.childs) == 0 {
		return nil
	}

	n.visit(nil, false, true, false, func(n *Node) {
		if !n.target {
			return
		}

		p := n.path()
		err := gw.w.Add(p)
		wasAdded := n.added
		n.added = err == nil

		switch {
		case !wasAdded && n.added:
			gw.events <- &Event{Op: Create, Name: p}
		case wasAdded && !n.added:
			gw.events <- &Event{Op: Remove, Name: p}
		case wasAdded && n.added:
			gw.events <- &Event{Op: Resync, Name: p}
		}
	})

	return nil
}

//----------

func (gw *GWatcher) split(name string) []string {
	u := strings.Split(name, string(os.PathSeparator))
	w := []string{}
	for _, k := range u {
		if strings.TrimSpace(k) != "" {
			w = append(w, k)
		}
	}
	return w
}

func (gw *GWatcher) normalize(name *string) error {
	u, err := filepath.Abs(*name)
	if err != nil {
		return err
	}
	*name = u
	return nil
}

//----------

func (gw *GWatcher) DebugWatchState() string {
	gw.root.Lock()
	defer gw.root.Unlock()

	type debugWatchNode struct {
		Path   string
		Target bool
		Added  bool
	}

	u := []debugWatchNode{}
	gw.root.n.visit(nil, false, true, false, func(n *Node) {
		if n.parent == nil {
			return
		}
		if !n.target && !n.added {
			return
		}
		u = append(u, debugWatchNode{
			Path:   n.path(),
			Target: n.target,
			Added:  n.added,
		})
	})

	sort.Slice(u, func(i, j int) bool {
		return u[i].Path < u[j].Path
	})
	if len(u) == 0 {
		return "watcher: no active nodes\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "watcher nodes: %d\n", len(u))
	for _, n := range u {
		flags := "--"
		if n.Target {
			flags = "t-"
		}
		if n.Added {
			flags = "-a"
		}
		if n.Target && n.Added {
			flags = "ta"
		}
		fmt.Fprintf(&b, "%s %s\n", flags, n.Path)
	}
	return b.String()
}

//----------
//----------

type Node struct {
	name   string
	childs map[string]*Node
	parent *Node

	target bool
	added  bool
}

func NewNode(name string, parent *Node) *Node {
	n := &Node{name: name, childs: map[string]*Node{}}
	if parent != nil {
		n.parent = parent
		parent.childs[name] = n
	}
	return n
}

func (n *Node) delete() {
	if n.parent != nil {
		delete(n.parent.childs, n.name)
	}
}

//----------

func (n *Node) visit(v []string, create, visSubChilds, postOrder bool, fn func(*Node)) {
	if postOrder {
		defer fn(n)
	} else {
		fn(n)
	}
	if len(v) == 0 {
		if visSubChilds {
			for _, c := range n.childs {
				c.visit(nil, create, visSubChilds, postOrder, fn)
			}
		}
		return
	}
	k := v[0]
	c, ok := n.childs[k]
	if !ok {
		if !create {
			return
		}
		c = NewNode(k, n)
	}
	if create && len(v) == 1 {
		c.target = true
	}
	c.visit(v[1:], create, visSubChilds, postOrder, fn)
}

//----------

func (n *Node) add(v []string, fn func(*Node)) {
	n.visit(v, true, false, false, fn)
}
func (n *Node) review(v []string, fn func(*Node)) {
	n.visit(v, false, true, false, fn)
}
func (n *Node) remove(v []string, fn func(*Node)) {
	n.visit(v, false, false, true, fn)
}
func (n *Node) modify(v []string, fn func(*Node)) {
	n.visit(v, false, false, false, fn)
}

func (n *Node) find(v []string) *Node {
	if len(v) == 0 {
		return n
	}
	c, ok := n.childs[v[0]]
	if !ok {
		return nil
	}
	return c.find(v[1:])
}

//----------

func (n *Node) path() string {
	if n.parent == nil {
		return n.name
	}
	return filepath.Join(n.parent.path(), n.name)
}

//----------

func (n *Node) SprintFlatTree() string {
	s := fmt.Sprintf("{%s:", n.name)

	// sort childs map keys
	keys := []string{}
	for k := range n.childs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		cn := n.childs[k]
		if i > 0 {
			s += ","
		}
		s += cn.SprintFlatTree()
	}
	s += "}"
	return s
}
