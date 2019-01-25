package fswatcher

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

//----------

func tmpDir() string {
	name, err := ioutil.TempDir("", "watcher_test")
	if err != nil {
		panic(err)
	}
	return name
}

func mustMkdirAll(t *testing.T, name string) {
	err := os.MkdirAll(name, 0755)
	if err != nil {
		t.Fatal(err)
	}
}

//----------

func readEvent(t *testing.T, w Watcher, failOnTimeout bool, fn func(*Event) bool) {
	t.Helper()
	tick := time.NewTicker(3000 * time.Millisecond)
	defer tick.Stop()
	select {
	case <-tick.C:
		if failOnTimeout {
			t.Fatal("event timeout")
		}
	case ev := <-w.Events():
		if err, ok := ev.(error); ok {
			t.Fatal(err)
		} else if !fn(ev.(*Event)) {
			t.Fatal(ev)
		}
	}
}

//----------

func mustNew(t *testing.T) Watcher {
	t.Helper()
	w, err := NewFsnWatcher()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func mustAdd(t *testing.T, w Watcher, name string) {
	t.Helper()
	if err := w.Add(name); err != nil {
		t.Fatal(err)
	}
}

func mustRemove(t *testing.T, w Watcher, name string) {
	t.Helper()
	if err := w.Remove(name); err != nil {
		t.Fatal(err)
	}
}

//----------

func mustCreateFile(t *testing.T, name string) {
	t.Helper()
	f, err := os.OpenFile(name, os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, name string) {
	t.Helper()
	f, err := os.OpenFile(name, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte(fmt.Sprintf("%v", rand.Int()))

	n, err := f.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n < len(data) {
		t.Fatal("short write")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustRemoveAllFile(t *testing.T, name string) {
	t.Helper()
	if err := os.RemoveAll(name); err != nil {
		t.Fatal(err)
	}
}

func mustRenameFile(t *testing.T, name, name2 string) {
	t.Helper()
	if err := os.Rename(name, name2); err != nil {
		t.Fatal(err)
	}
}

//----------

func TestFsWatcher1(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNew(t)
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")

	mustAdd(t, w, dir)
	mustMkdirAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == dir2 && ev.Op.HasAny(Create)
	})
}

func TestFsWatcher2(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNew(t)
	defer w.Close()
	*w.OpMask() = Create | Remove | Modify | Rename

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)

	mustAdd(t, w, dir4)
	mustAdd(t, w, file1)
	mustAdd(t, w, file2)
	mustRemoveAllFile(t, file1)

	// two equal events, one from dir4, one from file1
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == file1 && ev.Op.HasAny(Remove)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == file1 && ev.Op.HasAny(Remove)
	})

	mustRemoveAllFile(t, dir4)

	// 2 equal events, 1 from dir4, 1 from file1
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == file2 && ev.Op.HasAny(Remove)
	})
}

func TestFsWatcher3(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNew(t)
	defer w.Close()
	*w.OpMask() = Create | Remove | Modify | Rename

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	// can add same file 2 times without error
	mustAdd(t, w, file1)
	mustAdd(t, w, file1)

	mustRemoveAllFile(t, dir2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Remove)
	})

	// file remove was updated, can't remove what isn't watched anymore
	if err := w.Remove(file1); err == nil {
		t.Fatal("there should be an error")
	}

	// cannot add file that doesn't exist
	if err := w.Add(file1); err == nil {
		t.Fatal("there should be an error")
	}
}

func TestFsWatcher4(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNew(t)
	defer w.Close()
	*w.OpMask() = Create | Remove | Modify | Rename

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)

	mustAdd(t, w, file1)
	mustAdd(t, w, file2)

	mustWriteFile(t, file1)
	mustWriteFile(t, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Modify)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Modify)
	})

	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Rename)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
	})
}

func TestFsWatcher5(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNew(t)
	defer w.Close()
	*w.OpMask() = Create | Remove | Modify | Rename

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)
	mustAdd(t, w, dir4)
	mustAdd(t, w, file1)
	mustAdd(t, w, file2)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == file1 && ev.Op.HasAny(Rename)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.JoinNames() == file2 && ev.Op.HasAny(Create)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Rename)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
	})
}
