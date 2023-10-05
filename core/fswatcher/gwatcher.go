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
	switch ev.Op {
	case Create, Remove, Rename:
		_ = gw.review(u)
	case Modify:
		_ = gw.modify(u)
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

func (n *Node) visit(v []string, create, visSubChilds, depthFirst bool, fn func(*Node)) {
	if depthFirst {
		defer fn(n)
	} else {
		fn(n)
	}
	if len(v) == 0 {
		if visSubChilds {
			for _, c := range n.childs {
				c.visit(nil, create, visSubChilds, depthFirst, fn)
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
	c.visit(v[1:], create, visSubChilds, depthFirst, fn)
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
