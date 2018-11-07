package uiutil

import (
	"sync"

	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type SimpleUI struct {
	*BasicUI

	EventsQ *chanutil.ChanQ

	OnError func(error)
	OnEvent func(ev interface{})
	OnClose func()

	closeOnce sync.Once
	close     chan struct{}
}

func NewSimpleUI(winName string, root widget.Node) (*SimpleUI, error) {
	sui := &SimpleUI{
		close:   make(chan struct{}),
		EventsQ: chanutil.NewChanQ(16, 16),
		OnError: func(error) {},
		OnClose: func() {},
	}

	bui, err := NewBasicUI(sui.EventsQ.In(), winName, root)
	if err != nil {
		return nil, err
	}
	sui.BasicUI = bui

	sui.OnEvent = sui.BasicUI.HandleEvent

	return sui, nil
}

func (sui *SimpleUI) Close() {
	sui.closeOnce.Do(func() {
		sui.OnClose()
		close(sui.close)
	})
}

func (sui *SimpleUI) EventLoop() {
	defer sui.BasicUI.Close()
	evQOut := sui.EventsQ.Out()
	for {
		select {
		case <-sui.close:
			return
		case ev := <-evQOut:
			switch t := ev.(type) {
			case error:
				sui.OnError(t)
			case *event.WindowClose:
				sui.Close()
			default:
				sui.OnEvent(ev)
			}
		}
	}
}
