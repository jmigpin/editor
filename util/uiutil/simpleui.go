package uiutil

import (
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type SimpleUI struct {
	*BasicUI
	Root *widget.MultiLayer

	Events chan interface{}
	close  chan struct{}

	OnError func(error)
}

func NewSimpleUI(winName string) (*SimpleUI, error) {
	sui := &SimpleUI{}
	sui.close = make(chan struct{})
	sui.Events = make(chan interface{}, 64)
	sui.OnError = func(error) {}

	sui.Root = widget.NewMultiLayer()

	bui, err := NewBasicUI(sui.Events, winName, sui.Root)
	if err != nil {
		return nil, err
	}
	sui.BasicUI = bui

	return sui, nil
}

func (sui *SimpleUI) Close() {
	close(sui.close)
}

func (sui *SimpleUI) EventLoop() {
	defer sui.BasicUI.Close()
	for {
		select {
		case <-sui.close:
			goto forEnd
		case ev := <-sui.Events:
			switch t := ev.(type) {
			case error:
				sui.OnError(t)
			case *event.WindowClose:
				sui.Close()
			default:
				sui.BasicUI.HandleEvent(ev)
			}

		}
	}
forEnd:
}
