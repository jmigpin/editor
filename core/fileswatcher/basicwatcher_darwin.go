// TODO: this version allows compilation without fileswatcher support
// +build darwin

package fileswatcher

type BasicWatcher struct {
	Events chan interface{}
}

type Logf func(format string, args ...interface{})

func NewBasicWatcher(logf Logf) (*BasicWatcher, error) {
	w := &BasicWatcher{
		Events: make(chan interface{}),
	}
	return w, nil
}
func (w *BasicWatcher) Close() {
}
func (w *BasicWatcher) Add(name string) error {
	return nil
}
func (w *BasicWatcher) Remove(name string) error {
	return nil
}

type Event struct {
	Name     string
	Op       Op
	Filename string // full filename, Joined(name,shortFilename)
}
