package fswatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/jmigpin/editor/util/syncutil"
)

// graph based watcher
type GWatcher struct {
	w    Watcher
	q    *syncutil.SyncedQ
	root struct {
		sync.Mutex
		n *Node
	}
}

func NewGWatcher(w Watcher) *GWatcher {
	*w.OpMask() = AllOps

	gw := &GWatcher{w: w}
	gw.q = syncutil.NewSyncedQ()

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
	err := gw.w.Close()
	gw.q.PushBack(nil)
	return err
}

//----------

func (gw *GWatcher) NextEvent() any {
	return gw.q.PopFront()
}

func (gw *GWatcher) eventLoop() {
	for {
		ev := gw.w.NextEvent()
		if ev == nil {
			break
		}
		switch t := ev.(type) {
		case error:
			gw.q.PushBack(t)
		case *Event:
			gw.handleEv(t)
		}
	}
}
func (gw *GWatcher) handleEv(ev *Event) {
	u := ev.Name
	if ev.Op.HasAny(Create | Remove | Rename) {
		if err := gw.review(u); err != nil {
			gw.q.PushBack(fmt.Errorf("gwatcher review failed (op=%v, name=%q): %w", ev.Op, u, err))
		}
		if err := gw.resync(filepath.Dir(u)); err != nil {
			gw.q.PushBack(fmt.Errorf("gwatcher parent resync failed (op=%v, name=%q): %w", ev.Op, u, err))
		}
	}
	if ev.Op.HasAny(Modify | Attrib) {
		if err := gw.modify(u); err != nil {
			gw.q.PushBack(fmt.Errorf("gwatcher modify failed (op=%v, name=%q): %w", ev.Op, u, err))
		}
		if err := gw.resync(u); err != nil {
			gw.q.PushBack(fmt.Errorf("gwatcher resync failed (op=%v, name=%q): %w", ev.Op, u, err))
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
	return gw.root.n.add(v, func(n *Node) error {
		p := n.path()
		if !n.added {
			err := gw.w.Add(p)
			if err == nil {
				n.added = true
			}
			// ignore error: allows adding watches for non-existing paths that might be created later (e.g., TestGWatcher1)
		}
		return nil
	})
}

//----------

func (gw *GWatcher) Remove(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	return gw.root.n.remove(v, func(n *Node) error {
		if n.target {
			n.target = false
			if n.added {
				p := n.path()
				if err := gw.w.Remove(p); err != nil {
					// TODO: types of errors
					return err
				}
				n.added = false
			}
		}
		if len(n.childs) == 0 {
			n.delete()
		}
		return nil
	})
}

//----------

func (gw *GWatcher) review(name string) error {
	if err := gw.normalize(&name); err != nil {
		return err
	}

	v := gw.split(name)
	gw.root.Lock()
	defer gw.root.Unlock()
	_ = gw.root.n.review(v, func(n *Node) error {
		p := n.path()
		err := gw.w.Add(p)
		wasAdded := n.added
		n.added = err == nil

		if n.target {
			if !wasAdded && n.added {
				gw.q.PushBack(&Event{Op: Create, Name: p})
			}
			if wasAdded && !n.added {
				gw.q.PushBack(&Event{Op: Remove, Name: p})
			}
		}
		return nil
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
	_ = gw.root.n.modify(v, func(n *Node) error {
		if n.target {
			p := n.path()
			gw.q.PushBack(&Event{Op: Modify, Name: p})
		}
		return nil
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

	_ = n.visit(nil, false, true, false, func(n *Node) error {
		if !n.target {
			return nil
		}

		p := n.path()
		err := gw.w.Add(p)
		wasAdded := n.added
		n.added = err == nil

		switch {
		case !wasAdded && n.added:
			gw.q.PushBack(&Event{Op: Create, Name: p})
		case wasAdded && !n.added:
			gw.q.PushBack(&Event{Op: Remove, Name: p})
		case wasAdded && n.added:
			gw.q.PushBack(&Event{Op: Resync, Name: p})
		}
		return nil
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
		path   string
		target bool
		added  bool
	}

	u := []debugWatchNode{}
	_ = gw.root.n.visit(nil, false, true, false, func(n *Node) error {
		if n.parent == nil {
			return nil
		}
		if !n.target && !n.added {
			return nil
		}
		u = append(u, debugWatchNode{
			path:   n.path(),
			target: n.target,
			added:  n.added,
		})
		return nil
	})

	sort.Slice(u, func(i, j int) bool {
		return u[i].path < u[j].path
	})
	if len(u) == 0 {
		return "watcher: no active nodes\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "watcher nodes: %d\n", len(u))
	for _, n := range u {
		flags := "--"
		if n.target {
			flags = "t-"
		}
		if n.added {
			flags = "-a"
		}
		if n.target && n.added {
			flags = "ta"
		}
		fmt.Fprintf(&b, "%s %s\n", flags, n.path)
	}
	return b.String()
}

//----------
//----------

type Node struct {
	name   string
	childs map[string]*Node
	parent *Node

	target bool // leaf node requested
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

func (n *Node) visit(v []string, create, visSubChilds, postOrder bool, fn func(*Node) error) error {
	if postOrder {
		defer fn(n)
	} else {
		fn(n)
	}
	if len(v) == 0 {
		if visSubChilds {
			for _, c := range n.childs {
				if err := c.visit(nil, create, visSubChilds, postOrder, fn); err != nil {
					return err
				}
			}
		}
		return nil
	}
	k := v[0]
	c, ok := n.childs[k]
	if !ok {
		if !create {
			return nil
		}
		c = NewNode(k, n)
	}
	if create && len(v) == 1 {
		c.target = true
	}
	return c.visit(v[1:], create, visSubChilds, postOrder, fn)
}

//----------

func (n *Node) add(v []string, fn func(*Node) error) error {
	return n.visit(v, true, false, false, fn)
}
func (n *Node) review(v []string, fn func(*Node) error) error {
	return n.visit(v, false, true, false, fn)
}
func (n *Node) remove(v []string, fn func(*Node) error) error {
	return n.visit(v, false, false, true, fn)
}
func (n *Node) modify(v []string, fn func(*Node) error) error {
	return n.visit(v, false, false, false, fn)
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
