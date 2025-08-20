package termemu

import (
	"fmt"
	"io"
	"sync"

	"github.com/jmigpin/editor/util/iout"
)

//godebug:annotatefile
//godebug:annotatefile:vtparser.go
//godebug:annotatefile:screen.go
////godebug:annotatefile:../textareareader.go

//----------

// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html

//----------

// NOTES
// tput smcup - enter alt screen
// tput rmcup - return
// tput -Tvt100 clear
// infocmp vt100
// infocmp -1 "vt100" | grep -E 'smcup|rmcup'
// infocmp -1 "$TERM" | grep -E 'smcup|rmcup'

const TermEnv = "TERM=vt100"

const vt100 = "\x1b[?1;0c" //
// const vt101NoOpt = vt100
// const vt100WithAVO = "\x1b[?1;2c"
// const vt102 = "\x1b[?6c"
// const vt420 = "\x1b[?64c"
// const vt420Sixel = "\x1b[?6;4;4c" // ?
const termSeqReply = vt100

//----------

// TODO: pty size (getsize/setsize/...)

type Emu struct {
	userCons ConsoleConn // user side (ex: editor textarea)

	execRwc   io.ReadWriteCloser
	execPipes struct {
		r io.Reader
		w io.Writer
	}

	parser     *VTParser // parser->emu->screen
	parserDone sync.WaitGroup

	mu  sync.Mutex
	scr *Screen

	opts Opts
}

// emu itself is a read/write to be passed to the executable, as well as read/writing from the user (ex: textarea)
func NewEmu(userCons ConsoleConn, opts Opts) *Emu {
	if opts.W <= 0 {
		opts.W = 80
	}
	if opts.H <= 0 {
		opts.H = 24
	}

	emu := &Emu{userCons: userCons, opts: opts}

	emu.scr = NewScreen(opts.W, opts.H)
	emu.userCons.SetSize(opts.W, opts.H)

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
		*rd = io.TeeReader(*rd, emu.userCons)
	}

	// auto read from user to exec
	go func() {
		if emu.opts.Debug {
			rd := iout.FnReader(func(p []byte) (int, error) {
				n, err := emu.userCons.Read(p)
				s := fmt.Sprintf("\n*RCV: %q\n", string(p[:n]))
				emu.sendToUser(s)
				return n, err
			})
			_, _ = io.Copy(emu.execRwc, rd)
		} else {
			_, _ = io.Copy(emu.execRwc, emu.userCons)
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
		emu.userCons.Close()  // flush
		emu.parserDone.Wait() // after user close, parse should stop
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
	_, _ = emu.userCons.Write([]byte(s))
}

//----------

func (emu *Emu) plainMode() bool {
	return emu.opts.Mode == ModePlain
}

//----------

func (emu *Emu) Snapshot() *Screen {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.Clone()
}

func (emu *Emu) ScrMode() *Modes {
	return &emu.scr.modes
}

//----------
//----------

// called from the parser
func (emu *Emu) applyEmit(op *TermOp) {
	emu.mu.Lock()
	defer emu.mu.Unlock()

	//fmt.Printf("op %v: cursor %v\n", op.kind, emu.scr.Cursor)

	switch op.kind {
	case "aln":
		emu.scr.escAln_screenAlignment()
		if emu.plainMode() {
			// optional: nothing to send; UI mode repaints
		}
	case "bell": // TODO
	case "bs":
		emu.scr.backspace()
	case "cr":
		emu.scr.carriageReturn()
	case "csi":
		emu.applyEmitCsi(op)
	case "fnkey": // TODO
	case "ht":
		emu.scr.escHt_tab(1)
	case "hts":
		emu.scr.escHts_horizontalTabSet()
	case "ind":
		emu.scr.escInd_index()
	case "lf":
		emu.scr.lineFeed()
		if emu.plainMode() {
			emu.sendToUser("\n")
		}
	case "nel":
		emu.scr.carriageReturn()
		emu.scr.lineFeed()
	case "rc":
		emu.scr.escRc_restoreCursor()
	case "ri":
		emu.scr.escRi_reverseIndex()
	case "sc":
		emu.scr.escSc_saveCursor()

	//----------

	case "print":
		for _, ru := range op.s {
			emu.scr.putRune(ru)
		}
		if emu.plainMode() {
			emu.sendToUser(op.s)
		}

	default:
		err := fmt.Errorf("emu.applyemit: %q", op.kind)
		//fmt.Println(err)
		panic(err)
	}

	if emu.opts.Mode == ModeUI {
		emu.userCons.Repaint()
	}
}

func (emu *Emu) applyEmitCsi(op *TermOp) {
	switch op.csi.final {
	case 'A': // CUU: Cursor Up (n rows, default 1)
		emu.scr.csiCuu_cursorUp(op.csiADef(1))
	case 'B': // CUD: Cursor Down
		emu.scr.csiCud_cursorDown(op.csiADef(1))
	case 'C': // CUF: Cursor Forward (right)
		emu.scr.csiCuf_cursorForward(op.csiADef(1))
	case 'D': // CUB: Cursor Backward (left)
		emu.scr.csiCub_cursorBackward(op.csiADef(1))

		//E  CNL – Cursor Next Line (down n rows, col 1)
		//F  CPL – Cursor Previous Line (up n rows, col 1)

	case 'G': // CHA: Cursor Horizontal Absolute (to col n, same row)
		emu.scr.csiCha_cursorHorizontalAbsolute(op.csiADef(1))
	case 'H', 'f': // cup: Cursor Position (row n, col m, default 1,1)
		emu.scr.csiCup_cursorPosition(op.csiADef(1), op.csiBDef(1))
	case 'I': // CHT: cursor horizontal tabulation
		emu.scr.csiCht_cursorHorizontalTabulation(op.csiADef(1))
	case 'J': // ed: Erase in Display
		emu.scr.csiEd_eraseInDisplay(op.csiA())
	case 'K': // el: Erase in Line
		emu.scr.csiEl_eraseInLine(op.csiA())
	case 'L': // IL: Insert Lines
		emu.scr.csiIl_insertLines(op.csiADef(1))
	case 'M': // DL: Delete Lines
		emu.scr.csiDl_deleteLines(op.csiADef(1))
	case 'P': //  DCH: Delete Characters
		emu.scr.csiDch_deleteChars(op.csiADef(1))
	case 'S': // SU: Scroll Up
		emu.scr.csiSu_scrollUp(op.csiADef(1))
	case 'T': // SD: Scroll Down
		emu.scr.csiSd_scrollDown(op.csiADef(1))
	case 'X': //  ECH: Erase Characters
		emu.scr.csiEch_eraseChars(op.csiADef(1))
	case 'Z': // CBT: Cursor Backward Tab
		emu.scr.csiCbt_cursorBackwardTab(op.csiADef(1))

	//@  ICH – Insert Characters
	//`  HPA – Horizontal Position Absolute (same as CHA, but 0-based in some terms)
	//a  HPR – Horizontal Position Relative (right n cols)

	case 'c': // DA: Device Attributes
		// primary
		if !op.csi.hasPriv && op.csiA() == 0 {
			emu.sendToExec(termSeqReply)
		}
		// secondary
		if op.csiPrivIs('>') && op.csiA() == 0 {
			emu.sendToExec("\x1b[>0;1;1c")
		}
		// tertiary
		if op.csiPrivIs('=') && op.csiA() == 0 {
			emu.sendToExec("\x1b[>0;1;1c")
		}

	case 'd': //  vpa: Vertical Position Absolute (to row n)
		emu.scr.moveToRow(op.csiA())

	//e  VPR – Vertical Position Relative (down n rows)
	//f  HVP – Horizontal and Vertical Position (same as CUP)

	case 'g': // TBC: Tabulation Clear
		emu.scr.csiTbc_tabClear(op.csiADef(0))

	case 'h', 'l': // h:sm: Set Mode; l:rm: Reset Mode
		on := op.csi.final == 'h'
		if op.csiPrivIs('?') {
			emu.scr.modes.set(op.csiA(), on)
			switch op.csiA() {
			case 3:
				if needResize := emu.scr.csiColm_column132Mode(); needResize {
					emu.userCons.SetSize(emu.scr.W, emu.scr.H)
				}
			case 6:
				emu.scr.moveToOrigin()
			case 69:

			}
		}

	case 'm': // SGR: Select Graphic Rendition (colors, bold, etc.)
		emu.scr.csiSgr_selectGraphicRendition(op.csi.params)

	case 'n': // DSR: Device Status Report
		switch op.csiA() {
		case 5: // "are you ok?"
			emu.sendToExec("\x1b[0n") // "OK"
		case 6: // cursor position report
			row1, col1 := emu.scr.csiCpr_cursorPositionReport()
			s := fmt.Sprintf("\x1b[%d;%dR", row1, col1)
			emu.sendToExec(s)
		}
	case 'q': // DECLL: Load LEDs
		switch op.csiA() {
		case 0: // 	clear all leds
		case 1: // light nums lock
		case 2:
			switch op.csiB() {
			case 0: // light caps lock
			case 1: // extinguish num lock
			case 2: // extinguish caps lock
			case 3: // extinguish scroll lock
			}
		case 3: // light scroll lock
		}
	case 'r': // DECSTBM: Set Scrolling Region
		top1, bot1 := op.csiADef(1), op.csiBDef(emu.scr.H)
		emu.scr.setScrollRegion(top1, bot1)
		emu.scr.moveToOrigin()
	case 's':
		// SLRM: set left right margins
		if len(op.csi.params) == 2 {
			left1, right1 := op.csiADef(1), op.csiBDef(1)
			emu.scr.csiSlrm_lrmmSetMargins(left1, right1)
			return
		}
		// SCP: Save Cursor Position
		emu.scr.csiScp_saveCursorPos()
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

		if !op.csi.hasPriv {
			emu.scr.csiRcp_restoreCursorPos()
		}

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

//----------

// bidirectional UI endpoint (kbd/mouse + draw)
type ConsoleConn interface {
	io.ReadWriteCloser
	SetSize(w, h int)
	Repaint()
	Error(error)
}
