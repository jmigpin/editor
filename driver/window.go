package driver

import (
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Window interface {
	NextEvent() (_ event.Event, ok bool) // !ok = no more events
	Request(event.Request) error
}
