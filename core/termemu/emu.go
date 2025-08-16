package termemu

import (
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/util/iout"
)

//godebug:annotatefile
//godebug:annotatefile:vtparser.go
////godebug:annotatefile:../textareareader.go

//----------

// TEST
// tput smcup - enter alt screen
// tput rmcup - return
// tput -Tvt100 clear
// infocmp vt100
// infocmp -1 "vt100" | grep -E 'smcup|rmcup'
// infocmp -1 "$TERM" | grep -E 'smcup|rmcup'
const TermEnv = "TERM=vt100"

//----------

// TODO: pty size (getsize/setsize/...)

type Emu struct {
	userRwc io.ReadWriteCloser // user side (ex: editor textarea)

	execRwc   io.ReadWriteCloser
	execPipes struct {
		r io.Reader
		w io.Writer
	}

	mu  sync.Mutex
	scr *Screen

	evs chan Event

	parser     *VTParser
	parserDone sync.WaitGroup

	opts Opts
}

// emu itself is a rwc to be passed to the executable, then the emu reads and write from rwc which is the textarea input and output
func NewEmu(rwc io.ReadWriteCloser, opts Opts) *Emu {
	if opts.W <= 0 {
		opts.W = 80
	}
	if opts.H <= 0 {
		opts.H = 24
	}

	emu := &Emu{
		userRwc: rwc,
		opts:    opts,
		scr:     NewScreen(opts.W, opts.H),
		evs:     make(chan Event, 10),
	}

	emu.setupExecSideRWC()

	emu.parser = NewVTParser(emu.execRwc, emu.applyEmit)

	emu.parserDone.Add(1)
	go func() {
		defer emu.parserDone.Done()
		err := emu.parser.Run()
		_ = err // TODO: check error
		//log.Println("termemu parser error:", err)
	}()

	return emu
}

func (emu *Emu) setupExecSideRWC() {
	readPr, readPw := io.Pipe()
	writePr, writePw := io.Pipe()

	// allow concurrent writes (ex: textarea vs emu cmds)
	readPw2 := iout.NewSafeWriter(readPw)

	execRwc := &iout.RWC{}
	emu.execRwc = execRwc

	emu.execPipes.r, execRwc.Writer = readPr, readPw2  // read input
	emu.execPipes.w, execRwc.Reader = writePw, writePr // write to parser

	execRwc.Closer = iout.FnCloser(func() error {
		defer readPw.Close()
		defer writePr.Close()

		_ = readPr.Close()
		return writePw.Close()
	})

	if emu.opts.Mode == ModeRaw {
		rd := &execRwc.Reader
		*rd = io.TeeReader(*rd, emu.userRwc)
	}

	// auto read from user to exec
	go func() {
		if emu.opts.Debug {
			rd := iout.FnReader(func(p []byte) (int, error) {
				n, err := emu.userRwc.Read(p)
				s := fmt.Sprintf("\n*RCV: %q\n", string(p[:n]))
				emu.sendToUser(s)
				return n, err
			})
			_, _ = io.Copy(emu.execRwc, rd)
		} else {
			_, _ = io.Copy(emu.execRwc, emu.userRwc)
		}
	}()
}

//----------

func (emu *Emu) Read(p []byte) (int, error) {
	return emu.execPipes.r.Read(p) // read input
}
func (emu *Emu) Write(p []byte) (int, error) {
	return emu.execPipes.w.Write(p) // write to parser
}

func (emu *Emu) Close() error {
	defer func() {
		emu.userRwc.Close()   // flush
		emu.parserDone.Wait() // after user close, parse should stop
		close(emu.evs)        // after parse done, no more evs
	}()
	return emu.execRwc.Close()
}

//----------

func (emu *Emu) sendToExec(s string) {
	if emu.opts.Debug {
		s2 := fmt.Sprintf("\n*SEND: %q\n", s)
		emu.sendToUser(s2)
	}

	_, _ = emu.execRwc.Write([]byte(s))
}
func (emu *Emu) sendToUser(s string) {
	_, _ = emu.userRwc.Write([]byte(s))
}

//----------

func (emu *Emu) plainMode() bool {
	return emu.opts.Mode == ModePlain
}

//----------

func (emu *Emu) Events() <-chan Event {
	return emu.evs
}

func (emu *Emu) push(ev Event) {
	// TODO: review.. should not be skipping any, need cache then push
	select {
	case emu.evs <- ev:
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
//----------

// called from the parser
func (emu *Emu) applyEmit(op *TermOp) {
	emu.mu.Lock()
	defer emu.mu.Unlock()

	//fmt.Printf("op %v: cursor %v\n", op.kind, emu.scr.Cursor)

	switch op.kind {
	case "cr":
		emu.scr.CR()
	case "lf":
		emu.scr.LF()
		if emu.plainMode() {
			emu.sendToUser("\n")
		}
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
		if emu.plainMode() {
			emu.sendToUser(op.s)
		}

	//case OpTitle:
	//	t.push(Event{Kind: "title", Data: op.S})

	default:
		err := fmt.Errorf("emu.applyemit: %q", op.kind)
		//fmt.Println(err)
		panic(err)
	}

	if emu.opts.Mode == ModeUI {
		emu.push(Event{Kind: "repaint"})
	}
}

func (emu *Emu) applyEmitCsi(op *TermOp) {
	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html

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
		if !op.csi.hasPriv && op.csiA() == 0 {
			const vt100 = "\x1b[?1;0c"
			//const vt101NoOpt = vt100
			//const vt100WithAVO = "\x1b[?1;2c"
			//const vt102 = "\x1b[?6c"
			//const vt420 = "\x1b[?64c"
			//const vt420Sixel = "\x1b[?6;4;4c" // ?
			emu.sendToExec(vt100)
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
			emu.sendToExec("\x1b[0n") // "OK"
		case 6: // cursor position report
			row1, col1 := emu.scr.replyCPR()
			s := fmt.Sprintf("\x1b[%d;%dR", row1, col1)
			emu.sendToExec(s)
		}

	//q  DECLL – Load LEDs

	case 'r': // DECSTBM: Set Scrolling Region
		top1, bot1 := op.csiADef(1), op.csiBDef(emu.scr.H)
		emu.scr.SetScrollRegion(top1, bot1)
		if emu.scr.modes.Origin() {
			emu.scr.MoveTo(top1, 1)
		} else {
			emu.scr.MoveTo(1, 1)
		}

	case 's': // SCP: Save Cursor Position
		emu.scr.SaveCursorPos()
	case 'u': // RCP: Restore Cursor Position
		//switch {
		//case op.csiPrivIs('>'): // kitty: push flags (default 0)
		//	//emu.kittyPush(op.csiADef(0))
		//	return
		//case op.csiPrivIs('<'): // kitty: pop N (default 1)
		//	//emu.kittyPop(op.csiADef(1))
		//	return
		//case op.csiPrivIs('?'): // kitty: query flags
		//	//fmt.Fprintf(emu.readPw, "\x1b[?%du", emu.kittyFlags)
		//	return
		//}

		emu.scr.RestoreCursorPos()

	default:
		err := fmt.Errorf("emu.csi.final: todo: %c", op.csi.final)
		fmt.Println(err)
		//panic(err) // TESTING
	}
}

//----------
//----------
//----------

type Opts struct {
	W, H  int
	Mode  Mode
	Debug bool
}

//----------

type Event struct {
	Kind string
	Data any
}
