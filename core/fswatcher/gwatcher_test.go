package fswatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmigpin/editor/util/syncutil"
)

//----------

type fakeWatcher struct {
	q      *syncutil.SyncedQ
	opMask Op
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		q:      syncutil.NewSyncedQ(),
		opMask: AllOps,
	}
}

func (w *fakeWatcher) Add(name string) error {
	_, err := os.Stat(name)
	return err
}

func (w *fakeWatcher) Remove(name string) error {
	return nil
}

func (w *fakeWatcher) NextEvent() any {
	return w.q.PopFront()
}

func (w *fakeWatcher) OpMask() *Op {
	return &w.opMask
}

func (w *fakeWatcher) Close() error {
	w.q.PushBack(nil)
	return nil
}

//----------

func mustWriteFileContent(t *testing.T, name, s string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(s), 0644); err != nil {
		t.Fatal(err)
	}
}

//----------

func TestGWatcher1(t *testing.T) {
	dir := tmpDir()
	defer os.RemoveAll(dir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustAddWatch(t, w, dir4)
	mustMkdirAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op == Create
	})

	mustRemoveAll(t, dir2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op.HasAny(Remove|Resync)
	})

	mustRemoveWatch(t, w, dir4)
	mustAddWatch(t, w, file1)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create|Resync)
	})

	mustRemoveWatch(t, w, file1)

	s := w.root.n.SprintFlatTree()
	if s != "{/:}" {
		t.Fatal(s)
	}
}

func TestGWatcher2(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	mustMkdirAll(t, dir4)
	mustAddWatch(t, w, file2)
	mustCreateFile(t, file1)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op == Create
	})

	mustRemoveWatch(t, w, file2)

	mustRemoveAll(t, dir2)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	mustAddWatch(t, w, file2)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create|Rename|Resync)
	})

	mustRemoveWatch(t, w, file2)

	s := w.root.n.SprintFlatTree()
	if s != "{/:}" {
		t.Fatal(s)
	}
}

func TestGWatcher3(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustMkdirAll(t, dir4)
	mustAddWatch(t, w, file1)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op == Create
	})

	mustWriteFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Modify|Resync)
	})

	mustRemoveWatch(t, w, file1)

	s := w.root.n.SprintFlatTree()
	if s != "{/:}" {
		t.Fatal(s)
	}
}

func TestGWatcher4(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	dir3_ := filepath.Join(dir2, "dir3_")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)
	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file2)
	mustRenameFile(t, dir3, dir3_)

	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) && ev.Op == Remove
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) && ev.Op == Remove
	})

	mustRenameFile(t, dir3_, dir3)

	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) && ev.Op == Create
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) && ev.Op == Create
	})

	mustRemoveWatch(t, w, file1)
	mustRemoveWatch(t, w, file2)

	s := w.root.n.SprintFlatTree()
	if s != "{/:}" {
		t.Fatal(s)
	}
}

func TestGWatcherAtomicReplaceResyncsTargetFromParentDir(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	dir := filepath.Join(tmpDir, "dir")
	target := filepath.Join(dir, "file.txt")
	tmp := filepath.Join(dir, ".file.txt.tmp")

	mustMkdirAll(t, dir)
	mustWriteFileContent(t, target, "old")

	fw := newFakeWatcher()
	gw := NewGWatcher(fw)
	defer gw.Close()

	mustAddWatch(t, gw, target)

	mustWriteFileContent(t, tmp, "new")
	mustRenameFile(t, tmp, target)

	// Simulate a backend that only reports the temporary path change.
	fw.q.PushBack(&Event{Op: Rename, Name: tmp})

	readEvent(t, gw, true, func(ev *Event) bool {
		if ev.Name != target {
			return false
		}
		if ev.Op != Resync {
			t.Fatalf("expected resync on %q, got %v", ev.Name, ev.Op)
		}
		b, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "new" {
			t.Fatal(fmt.Sprintf("unexpected content: %q", string(b)))
		}
		return true
	})
}

func TestGWatcherDeadlockOnFullEvents(t *testing.T) {
	tmp := tmpDir()
	defer mustRemoveAll(t, tmp)

	fw := newFakeWatcher()
	gw := NewGWatcher(fw)
	defer gw.Close()

	// Add many watches to make resync emit many events
	for i := 0; i < 20; i++ {
		name := filepath.Join(tmp, fmt.Sprintf("file%d", i))
		mustCreateFile(t, name)
		if err := gw.Add(name); err != nil {
			t.Fatal(err)
		}
	}

	// Trigger resync that will put events into SyncedQ.
	// This should not block because SyncedQ is unlimited.
	go func() {
		_ = gw.resync(tmp)
	}()

	// Wait a bit to let resync finish filling the queue
	time.Sleep(200 * time.Millisecond)

	// Try to acquire the lock via Add().
	// It should be free because resync finished (it didn't block on events).
	addDone := make(chan bool)
	go func() {
		name := filepath.Join(tmp, "another_file")
		mustCreateFile(t, name)
		_ = gw.Add(name)
		addDone <- true
	}()

	select {
	case <-addDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Deadlock detected: could not acquire lock")
	}
}
