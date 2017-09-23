package fileswatcher

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

type BasicWatcher struct {
	fd     int
	Events chan interface{}

	watches struct {
		sync.Mutex
		m     map[string]int
		names map[int]string
	}

	Logf Logf
}

type Logf func(format string, args ...interface{})

func NewBasicWatcher(logf Logf) (*BasicWatcher, error) {
	//fd, err := unix.InotifyInit1(unix.IN_CLOEXEC)
	fd, err := unix.InotifyInit()
	if err != nil {
		return nil, err
	}

	w := &BasicWatcher{
		fd:     fd,
		Events: make(chan interface{}),
		Logf:   func(string, ...interface{}) {},
	}

	//w.log = log.New(ioutil.Discard, "", 0)
	//w.log = log.New(os.Stdout, fmt.Sprintf("%p: ", w), 0)
	//w.log = log.New(os.Stdout, fmt.Sprintf("%p: ", w), log.Lmicroseconds)

	//w.Logf = log.Printf
	if logf != nil {
		w.Logf = logf
	}

	w.watches.m = make(map[string]int)
	w.watches.names = make(map[int]string)

	go w.eventLoop()

	return w, nil
}
func (w *BasicWatcher) Close() {
	_ = unix.Close(w.fd)
}

func (w *BasicWatcher) Add(name string) error {
	w.watches.Lock()
	defer w.watches.Unlock()

	_, ok := w.watches.m[name]
	if ok {
		return nil
	}

	var flags uint32 = unix.IN_CREATE |
		unix.IN_MODIFY |
		unix.IN_DELETE_SELF | unix.IN_DELETE |
		unix.IN_MOVE_SELF | unix.IN_MOVED_TO | unix.IN_MOVED_FROM

	// allows adding a name that was already added
	wd, err := unix.InotifyAddWatch(w.fd, name, flags)
	if err != nil {
		return err
	}

	w.watches.names[wd] = name
	w.watches.m[name] = wd

	w.Logf("added %v %v", name, wd)

	return nil
}
func (w *BasicWatcher) Remove(name string) error {
	w.watches.Lock()
	defer w.watches.Unlock()

	wd, ok := w.watches.m[name]
	if !ok {
		return nil
	}

	// gives error if removing a file not watched (could've been auto-removed)
	_, err := unix.InotifyRmWatch(w.fd, uint32(wd))
	if err != nil {
		return err
	}

	// not removing here allows to receive events already sent (in the stack)
	// and not lose the corresponding wd

	// commented: removing only after receiving the ignore
	//delete(w.watches.m, name)
	//delete(w.watches.names, wd)

	w.Logf("signaled remove %v %v", name, wd)

	return nil
}

func (w *BasicWatcher) eventLoop() {
	ureader := &UnixNReader{
		fd:   w.fd,
		logf: w.Logf,
		buf:  make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1),
	}
	for {
		// read event
		var buf [unix.SizeofInotifyEvent]byte
		_, err := ureader.Read(buf[:])
		if err != nil {
			w.Events <- err
			break
		}

		//spew.Dump(buf)

		ev := (*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))

		// filename: only returned for files inside a watched directory
		filename := ""
		if ev.Len > 0 {
			tmp := make([]byte, ev.Len)
			_, err := ureader.Read(tmp)
			if err != nil {
				w.Events <- err
				break
			}
			filename = strings.TrimRight(string(tmp), "\000")
		}

		ev2 := w.handleEvent(ev, filename)
		if ev2 != nil {
			w.Logf("emit event: %+v", ev2)
			w.Events <- ev2
		}
	}
}
func (w *BasicWatcher) handleEvent(ev *unix.InotifyEvent, filename string) interface{} {
	op := Op(ev.Mask)
	wd := int(ev.Wd)

	w.Logf("inotify: wd=%v %v %v", wd, filename, op)

	if op&unix.IN_Q_OVERFLOW > 0 {
		// wd is -1
		return fmt.Errorf("event q overflow")
	}

	w.watches.Lock()
	defer w.watches.Unlock()

	name, ok := w.watches.names[wd]
	if !ok {
		// event that got in the stack for a wd already removed
		err := fmt.Errorf("name not found for wd=%d %v", wd, op)
		return err

		//return nil
	}

	if op.HasIgnored() {
		// remove from index, from this point receiving events with
		// this wd is an error
		delete(w.watches.m, name)
		delete(w.watches.names, wd)
		w.Logf("removed %v %v", name, wd)
	}

	ev2 := &Event{Name: name, Op: op}
	if filename != "" {
		ev2.Filename = path.Join(name, filename)
	}
	return ev2
}

// Blocks until it reads n bytes.
type UnixNReader struct {
	fd   int
	logf Logf
	buf  []byte

	bb bytes.Buffer
}

func (r *UnixNReader) Read(p []byte) (int, error) {
	if r.bb.Len() >= len(p) {
		return r.bb.Read(p)
	}
	for {
		//r.logf("unix read: waiting len=%v", len(p))
		n, err := unix.Read(r.fd, r.buf[:])
		//r.logf("unix read: %v %v %v", n, err, r.buf)
		if n > 0 {
			r.bb.Write(r.buf[:n])
			if r.bb.Len() >= len(p) {
				return r.bb.Read(p)
			}
		}
		if err != nil {
			if err != unix.EINTR {
				// there could be data in the buffer (less then len(p))
				// but this reader only returns ok if the requested
				// n bytes are available
				return 0, err
			}
		}
	}
}

type Event struct {
	Name     string
	Op       Op
	Filename string // full filename, Joined(name,shortFilename)
}

type Op uint32

func (op Op) HasDelete() bool {
	return op&unix.IN_DELETE_SELF+op&unix.IN_DELETE+op&unix.IN_MOVED_FROM > 0
}
func (op Op) HasCreate() bool {
	return op&unix.IN_CREATE+op&unix.IN_MOVED_TO > 0
}
func (op Op) HasModify() bool {
	return op&unix.IN_MODIFY > 0
}
func (op Op) HasIgnored() bool {
	return op&unix.IN_IGNORED > 0
}
func (op Op) HasIsDir() bool {
	return op&unix.IN_ISDIR > 0
}

func (op Op) String() string {
	var u []string
	for _, um := range unixMasks {
		if uint32(op)&um.k > 0 {
			u = append(u, um.v)
			op = Op(uint32(op) - um.k)
		}
	}
	if op > 0 {
		u = append(u, fmt.Sprintf("(%v=?)", uint32(op)))
	}
	return strings.Join(u, "|")
}

var unixMasks = []KV{
	KV{unix.IN_CREATE, "create"},
	KV{unix.IN_DELETE, "delete"},
	KV{unix.IN_DELETE_SELF, "deleteSelf"},
	KV{unix.IN_MODIFY, "modify"},
	KV{unix.IN_MOVE_SELF, "moveSelf"},
	KV{unix.IN_MOVED_FROM, "movedFrom"},
	KV{unix.IN_MOVED_TO, "movedTo"},
	KV{unix.IN_IGNORED, "ignored"},
	KV{unix.IN_ISDIR, "isDir"},
	KV{unix.IN_Q_OVERFLOW, "qOverflow"},
}

type KV struct {
	k uint32
	v string
}
