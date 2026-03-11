package fswatcher

import (
	"os"
	"path/filepath"
	"testing"
)

//----------

func mustNewFsnWatcher(t *testing.T) Watcher {
	t.Helper()
	w, err := NewFsnWatcher()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

//----------

func TestFsWatcher1(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNewFsnWatcher(t)
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")

	mustAddWatch(t, w, dir)
	mustMkdirAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir2 && ev.Op == Create
	})
}

func TestFsWatcher2(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNewFsnWatcher(t)
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

	mustAddWatch(t, w, dir4)
	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file2)
	mustRemoveAll(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op == Remove
	})

	mustRemoveAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op == Remove
	})
}

func TestFsWatcher3(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNewFsnWatcher(t)
	defer w.Close()
	*w.OpMask() = Create | Remove | Modify | Rename

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file1)

	mustRemoveAll(t, dir2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op == Remove
	})

	if err := w.Remove(file1); err == nil {
		t.Fatal("there should be an error")
	}

	if err := w.Add(file1); err == nil {
		t.Fatal("there should be an error")
	}
}

func TestFsWatcher4(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := mustNewFsnWatcher(t)
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

	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file2)

	mustWriteFile(t, file1)
	mustWriteFile(t, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op == Modify
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op == Modify
	})

	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op == Rename
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op == Remove
	})
}
