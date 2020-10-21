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
		return ev.Name == dir2 && ev.Op.HasAny(Create)
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

	// two equal events, one from dir4, one from file1
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Remove)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Remove)
	})

	mustRemoveAll(t, dir4)

	// 2 equal events, 1 from dir4, 1 from file1
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
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

	// can add same file 2 times without error
	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file1)

	mustRemoveAll(t, dir2)

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
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Rename)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Rename)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Remove)
	})
}
