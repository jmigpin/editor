package fileswatcher

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"
)

func waitForEvent(t *testing.T, evChan chan interface{}, failOnTimeout bool, fn func(interface{}) bool) {
	t.Helper()

	fatal := func(err error) {
		t.Helper()

		//err2 := errors.WithStack(err)
		//e, _ := err2.(interface {
		//	StackTrace() errors.StackTrace
		//})
		//st := e.StackTrace()
		//err3 := fmt.Sprintf("%v: %+v", err, st[3:])
		//t.Fatal(err3)

		t.Fatal(err)
	}

	tick := time.NewTicker(3000 * time.Millisecond)
	select {
	case <-tick.C:
		if failOnTimeout {
			fatal(fmt.Errorf("timeout, no event was fired"))
		}
		return
	case ev := <-evChan:
		switch ev2 := ev.(type) {
		case error:
			fatal(ev2)
		case *Event:
			ok := fn(ev2)
			if !ok {
				fatal(fmt.Errorf("failed test condition: %+v", ev2))
			}
			return
		}
	}
	panic("!")
}

func waitForBWEvent(t *testing.T, w *BasicWatcher, fn func(interface{}) bool) {
	t.Helper()
	waitForEvent(t, w.Events, true, fn)
}

func waitForPossibleBWEvent(t *testing.T, w *BasicWatcher, fn func(interface{}) bool) {
	t.Helper()
	waitForEvent(t, w.Events, false, fn)
}

func newBasicWatcherForTest(t *testing.T) *BasicWatcher {
	logf := func(f string, a ...interface{}) {
		t.Helper()
		t.Logf(f, a...)
	}

	w, err := NewBasicWatcher(logf)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestBasicWatcher1(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newBasicWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	//file1 := path.Join(dir4, "file1.txt")

	w.Add(dir)
	mkDir(t, dir4)
	//createFile(t, file1)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir && ev2.Filename == dir2 && ev2.Op.HasCreate()
	})
}

func TestBasicWatcher2(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newBasicWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")
	file2 := path.Join(dir4, "file2.txt")

	mkDir(t, dir4)
	createFile(t, file1)
	createFile(t, file2)

	err := w.Add(dir4)
	if err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(file1)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Filename == file1 && ev2.Op.HasDelete()
	})

	os.RemoveAll(dir4)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Filename == file2 && ev2.Op.HasDelete()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Op.HasDelete()
	})
}

func TestBasicWatcher3(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newBasicWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	createFile(t, file1)

	err := w.Add(dir4)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Add(file1)
	if err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(file1)
	os.RemoveAll(dir4)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasIgnored()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Filename == file1 && ev2.Op.HasDelete()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Filename == "" && ev2.Op.HasDelete()
	})
}

func TestBasicWatcher4(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newBasicWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	createFile(t, file1)

	// can add same file 2 times without error
	err := w.Add(file1)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Add(file1)
	if err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(dir2)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasIgnored()
	})

	// file was auto-removed, so unwatching again should give no error
	err = w.Remove(file1)
	if err != nil {
		t.Fatal(err)
	}

	// cannot add file that doesn't exist
	err = w.Add(file1)
	if err == nil {
		t.Fatal("there should be an error")
	}

	mkDir(t, dir4)
	createFile(t, file1)
	os.RemoveAll(dir2)
	mkDir(t, dir4)
	createFile(t, file1)

	// no events since the watch was not added after the remove

	err = w.Add(file1)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, file1)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasModify()
	})
}

func TestBasicWatcher5(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newBasicWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	createFile(t, file1)

	err := w.Add(file1)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, file1)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasModify()
	})

	os.RemoveAll(file1)

	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})
	waitForBWEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasIgnored()
	})

	//createFile(t, file1)
	//writeFile(t, file1)
	//waitForPossibleBWEvent(t, w, func(ev interface{}) bool {
	//	return false
	//})
}
