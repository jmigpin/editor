package termemu

import (
	"fmt"
	"io"
	"sync"
)

//godebug:annotatefile
////godebug:annotatefile:../textareareader.go

// TODO: pty size (getsize/setsize/...)

type Emu struct {
	rwc io.ReadWriteCloser

	writePr *io.PipeReader
	writePw *io.PipeWriter

	readPr *io.PipeReader
	readPw *io.PipeWriter

	mu   sync.Mutex
	scr  *Screen
	evCh chan Event

	parser *VTParser

	//done chan struct{}
	done2 sync.WaitGroup
}

func NewEmu(rwc io.ReadWriteCloser, opt Opts) *Emu {
	if opt.W <= 0 {
		opt.W = 80
	}
	if opt.H <= 0 {
		opt.H = 24
	}

	emu := &Emu{
		rwc:  rwc,
		scr:  NewScreen(opt.W, opt.H),
		evCh: make(chan Event, 10),
		//done: make(chan struct{}),
	}

	// write to the parse loop
	emu.writePr, emu.writePw = io.Pipe()

	// read from textarea or bytes to be sent
	emu.readPr, emu.readPw = io.Pipe()
	go func() {
		_, _ = io.Copy(emu.readPw, emu.rwc)
	}()

	emu.parser = NewVTParser(emu.writePr, emu.applyEmit)

	emu.done2.Add(1)
	go func() {
		defer emu.done2.Done()
		emu.parser.Run()
	}()

	return emu
}

//----------

func (emu *Emu) Read(p []byte) (int, error) {
	//return emu.rwc.Read(p) // ex: read textarea input
	return emu.readPr.Read(p) // ex: read textarea input or cmds
}
func (emu *Emu) Write(p []byte) (int, error) {
	return emu.writePw.Write(p) // write to the parse loop
}

func (emu *Emu) Close() error {
	defer emu.done2.Wait()
	//defer close(emu.evCh)
	//close(emu.done)

	_ = emu.writePw.Close()
	return emu.rwc.Close()
}

//----------

func (emu *Emu) Events() <-chan Event {
	return emu.evCh
}

func (emu *Emu) push(ev Event) {
	// TODO: review..
	select {
	case emu.evCh <- ev:
	default:
	}

	// TODO: review, halting without goroutine
	//go func() { emu.evCh <- ev }()
}

//----------

func (emu *Emu) Snapshot() *Screen {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.Clone()
}

//----------

func (emu *Emu) applyEmit(op *TermOp) {
	emu.mu.Lock()
	defer emu.mu.Unlock()

	//fmt.Printf("op %v: cursor %v\n", op.kind, emu.scr.Cursor)

	switch op.kind {
	case "cr":
		emu.scr.CR()
	case "lf":
		emu.scr.LF()
	case "bs":
		emu.scr.BS()
	case "csi":
		emu.applyEmitCsi(op)
	case "bell": // TODO
	case "fnkey": // TODO
	case "print":
		for _, ru := range op.s {
			emu.scr.PutRune(ru)
		}

	//case OpTitle:
	//	t.push(Event{Kind: "title", Data: op.S})

	default:
		err := fmt.Errorf("emu.applyemit: %q", op.kind)
		fmt.Println(err)
		//panic(err) // TESTING
	}
	emu.push(Event{Kind: "repaint"})
}

func (emu *Emu) applyEmitCsi(op *TermOp) {
	switch op.csi.final {
	case 'A': // cuu: Cursor Up (n rows, default 1)
		emu.scr.MoveRel(-op.csiADef(1), 0)
	case 'B': // cud: Cursor Down
		//emu.scr.MoveRel(op.csiA(), 0)
		for i := 0; i < op.csiADef(1); i++ {
			emu.scr.LF()
		}
	case 'C': // cuf: Cursor Forward (right)
		emu.scr.MoveRel(0, op.csiADef(1))
	case 'D': // cub: Cursor Backward (left)
		emu.scr.MoveRel(0, -op.csiADef(1))

	//D  CUB – Cursor Backward (left)
	//E  CNL – Cursor Next Line (down n rows, col 1)
	//F  CPL – Cursor Previous Line (up n rows, col 1)

	case 'G': // cha: Cursor Horizontal Absolute (to col n, same row)
		emu.scr.MoveToCol(op.csiA())
	case 'H', 'f': // cup: Cursor Position (row n, col m, default 1,1)
		emu.scr.MoveTo(op.csiADef(1), op.csiBDef(1))
	case 'J': // ed: Erase in Display
		emu.scr.EraseDisplay(op.csiA())
	case 'K': // el: Erase in Line
		emu.scr.EraseLine(op.csiA())
	case 'L': // IL: Insert Lines
		emu.scr.insertLines(op.csiADef(1))
	case 'M': // DL: Delete Lines
		emu.scr.deleteLines(op.csiADef(1))
	case 'P': //  DCH: Delete Characters
		emu.scr.DeleteChars(op.csiADef(1))
	case 'S': // SU: Scroll Up
		emu.scr.scrollUp(op.csiADef(1))
	case 'T': // SD: Scroll Down
		emu.scr.scrollDown(op.csiADef(1))
	case 'X': //  ECH: Erase Characters
		emu.scr.EraseChars(op.csiADef(1))

	//Z  CBT – Cursor Backward Tab
	//@  ICH – Insert Characters
	//`  HPA – Horizontal Position Absolute (same as CHA, but 0-based in some terms)
	//a  HPR – Horizontal Position Relative (right n cols)

	case 'c': // DA: Device Attributes
		//if op.csiPrivIs('?') && op.csiA() == 0 { // who are you
		//	fmt.Fprintf(emu.readPw, "\x1b[?1;2c") // VT100-like
		//}
		if op.csiPrivIs('>') { // product/version
			fmt.Fprintf(emu.readPw, "\x1b[>pv;1c")
		}

	case 'd': //  vpa: Vertical Position Absolute (to row n)
		emu.scr.MoveToRow(op.csiA())

	//e  VPR – Vertical Position Relative (down n rows)
	//f  HVP – Horizontal and Vertical Position (same as CUP)
	//g  TBC – Tab Clear

	case 'h', 'l': // h:sm: Set Mode; l:rm: Reset Mode
		if op.csiPrivIs('?') {
			on := op.csi.final == 'h'
			emu.scr.modes.set(op.csiA(), on)
			if op.csiA() == 6 {
				emu.scr.moveToOrigin(on)
			}
		}

	case 'm': // SGR: Select Graphic Rendition (colors, bold, etc.)
		emu.scr.SetSGR(op.csi.params)

	case 'n': // DSR: Device Status Report
		switch op.csiA() {
		case 5: // "are you ok?"
			fmt.Fprint(emu.readPw, "\x1b[0n") // "OK"
		case 6: // cursor position report
			row1, col1 := emu.scr.replyCPR()
			fmt.Fprintf(emu.readPw, "\x1b[%d;%dR", row1, col1)
		}

	//q  DECLL – Load LEDs

	case 'r': // DECSTBM: Set Scrolling Region
		top1, bot1 := op.csiADef(1), op.csiBDef(emu.scr.H)
		emu.scr.SetScrollRegion(top1, bot1)
		if emu.scr.modes.Origin() {
			emu.scr.MoveTo(top1, 0)
		} else {
			emu.scr.MoveTo(0, 0)
		}

	case 's': // SCP: Save Cursor Position
		emu.scr.SaveCursorPos()
	case 'u': // RCP: Restore Cursor Position
		switch {
		case op.csiPrivIs('>'): // kitty: push flags (default 0)
			//emu.kittyPush(op.csiADef(0))
			return
		case op.csiPrivIs('<'): // kitty: pop N (default 1)
			//emu.kittyPop(op.csiADef(1))
			return
		case op.csiPrivIs('?'): // kitty: query flags
			//fmt.Fprintf(emu.readPw, "\x1b[?%du", emu.kittyFlags)
			return
		}

		emu.scr.RestoreCursorPos() // ANSI/VT: restore cursor (CSI u)

	default:
		err := fmt.Errorf("emu.csi.final: todo: %c", op.csi.final)
		fmt.Println(err)
		//panic(err) // TESTING
	}
}

//----------
//----------
//----------

type Opts struct{ W, H int }

//----------

type Event struct {
	Kind string
	Data any
}
