package driver

import (
	"github.com/jmigpin/editor/v2/util/uiutil/event"
)

type Window interface {
	NextEvent() (_ event.Event, ok bool) // !ok = no more events
	Request(event.Request) error
}
