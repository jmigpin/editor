package core

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type TerminalIO interface {
	Init(tf *TerminalFilter)

	Read([]byte) (int, error) // input interface
	AddToRead([]byte)         // add input internally to be read

	WriteOp(any) error // accepted types: {[]byte,string}

	Close() error
}

//----------
//----------
//----------

type ERowTermIO struct {
	erow *ERow
	tf   *TerminalFilter

	inputReg *evreg.Regist // input events

	input struct {
		sync.Mutex
		cond    *sync.Cond
		buf     bytes.Buffer
		closing bool
	}
	update struct {
		sync.Mutex
		updating bool
		ops      []any
	}
}

func NewERowTermIO(erow *ERow) *ERowTermIO {
	tio := &ERowTermIO{erow: erow}
	tio.input.cond = sync.NewCond(&tio.input)
	return tio
}

func (tio *ERowTermIO) Init(tf *TerminalFilter) {
	tio.tf = tf
	tio.initInput()
}

func (tio *ERowTermIO) Close() error {
	tio.inputReg.Unregister()

	// signal to unblock waiting for a read
	tio.input.Lock()
	tio.input.closing = true
	tio.input.cond.Signal()
	tio.input.Unlock()

	return nil
}

//----------

func (tio *ERowTermIO) Read(b []byte) (int, error) {
	tio.input.Lock()

	for tio.input.buf.Len() == 0 && !tio.input.closing {
		tio.input.cond.Wait()
	}
	defer tio.input.Unlock()
	if tio.input.closing {
		return 0, io.EOF
	}
	return tio.input.buf.Read(b)
}

func (tio *ERowTermIO) AddToRead(b []byte) {
	tio.input.Lock()
	defer tio.input.cond.Signal()
	defer tio.input.Unlock()
	tio.input.buf.Write(b)
}

//----------

func (tio *ERowTermIO) WriteOp(op any) error {
	tio.updateWriteOp(op)
	return nil
}

func (tio *ERowTermIO) updateWriteOp(op any) {
	tio.update.Lock()
	defer tio.update.Unlock()

	tio.appendOp(op)

	if tio.update.updating {
		return
	}
	tio.update.updating = true

	tio.erow.Ed.UI.RunOnUIGoRoutine(func() {
		tio.update.Lock()
		defer tio.update.Unlock()
		tio.update.updating = false
		// clear ops at the end
		defer func() { tio.update.ops = nil }()

		for _, op := range tio.update.ops {
			if err := tio.updateWriteOp2(op); err != nil {
				tio.erow.Ed.Error(err)
			}
		}
	})
}

func (tio *ERowTermIO) updateWriteOp2(op any) error {
	erow := tio.tf.erow
	ta := erow.Row.TextArea
	switch t := op.(type) {
	case []byte:
		if err := erow.AppendBytesClearHistory2(t); err != nil {
			return err
		}
	case string:
		switch t {
		case "clear":
			if err := ta.SetBytesClearHistory(nil); err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("todo: %v", t))
		}
	default:
		panic(fmt.Sprintf("todo: %v %T", t, t))
	}
	return nil
}

func (tio *ERowTermIO) appendOp(op any) {
	o := &tio.update.ops
	switch t := op.(type) {
	case []byte:
		// copy to avoid losing/overwriting content
		b := iout.CopyBytes(t)

		// performance: append to previous op if possible
		l := len(*o)
		if l > 0 {
			last := &(*o)[l-1]
			if lb, ok := (*last).([]byte); ok {
				*last = append(lb, b...)
				return
			}
		}

		*o = append(*o, b)
	default:
		*o = append(*o, op)
	}
}

//----------

func (tio *ERowTermIO) initInput() {
	ta := tio.erow.Row.TextArea
	tio.inputReg = ta.EvReg.Add(ui.TextAreaInputEventId, tio.onTextAreaInputEvent)
}

func (tio *ERowTermIO) onTextAreaInputEvent(ev0 any) {
	ev := ev0.(*ui.TextAreaInputEvent)
	b, handled := tio.eventToBytes(ev.Event)
	if len(b) > 0 {
		tio.AddToRead(b)
	}
	ev.ReplyHandled = handled
}

//----------

func (tio *ERowTermIO) eventToBytes(ev any) ([]byte, event.Handled) {
	// util funcs
	keyboardEvs := func() bool {
		return tio.erow.terminalOpt.keyEvents
	}
	byteOut := func(v byte, ru rune) []byte {
		b := []byte{v}
		// also add to output
		if ru != 0 {
			b2 := []byte(string(ru))
			//_, _ = tio.tf.Write(b2) // send back to filter
			_ = tio.WriteOp(b2) // add directly to output
		}
		return b
	}

	switch t := ev.(type) {
	case *event.KeyDown:
		if keyboardEvs() {
			var b []byte
			switch t.KeySym {
			case event.KSymReturn, event.KSymKeypadEnter:
				b = byteOut('\n', '\n')
			case event.KSymEscape:
				b = []byte{27}
			case event.KSymTab:
				b = []byte{'\t'}
			case event.KSymBackspace:
				b = []byte{'\b'}
			default:
				b = []byte(string(t.Rune))
			}
			return b, true
		}
	}
	return nil, false
}
