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
//godebug:annotatefile:../textareareader.go
////godebug:annotatefile:../textareaconsole.go

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

// const TermEnv = "TERM=vt100" //
const TermEnv = "TERM=xterm-mono" //

// const vt100 = "\x1b[?1;0c" //
// const vt101NoOpt = vt100
const vt100WithAVO = "\x1b[?1;2c" //
// const vt102 = "\x1b[?6c"
// const vt420 = "\x1b[?64c"
// const vt420Sixel = "\x1b[?6;4;4c" // ?
const termSeqReply = vt100WithAVO

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

	// allow concurrent writes (ex: textarea input vs emu cmds)
	// commented: not needed - io.pipe doesn't interleave write calls
	//readPw2 := iout.NewSafeWriter(readPw)
	readPw2 := readPw

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
		emu.parserDone.Wait() // on exec close, parse should stop
		emu.userCons.Close()
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

func (emu *Emu) ScrMode() *PrivModes {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.pmodes.clone()
}

//----------
//----------

// called from the parser; applies lock to screen
func (emu *Emu) applyEmit(op *TermOp) {
	emu.mu.Lock()
	defer emu.mu.Unlock()

	switch op.kind {
	case "aln":
		emu.scr.escAln_screenAlignment()
	case "bell": // TODO
	case "bs":
		emu.scr.backspace()
	case "cr":
		emu.scr.carriageReturn()
	case "csi":
		emu.applyEmitCsi(op.csi)
	case "fnkey": // TODO
	case "g0", "g1":
		emu.scr.graphics.set(op.kind, op.s)
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
	case "ris":
		emu.scr.escRis_reset(true)
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

func (emu *Emu) applyEmitCsi(op *TermCsiOp) {
	switch op.final {
	case 'A': // CUU: Cursor Up (n rows, default 1)
		emu.scr.csiCuu_cursorUp(op.ADef(1))
	case 'B', 'e':
		// B: CUD: Cursor Down
		// 'e': VPR: Vertical Position Relative (down n rows)
		emu.scr.csiCud_cursorDown(op.ADef(1))
	case 'C', 'a':
		// C: CUF: Cursor Forward (right)
		// a: HPR: Horizontal Position Relative (right n cols)
		emu.scr.csiCuf_cursorForward(op.ADef(1))
	case 'D': // CUB: Cursor Backward (left)
		emu.scr.csiCub_cursorBackward(op.ADef(1))
	case 'E': // CNL: Cursor Next Line (down n rows, col 1)
		emu.scr.csiCnl_cursorNextLine(op.ADef(1))
	case 'F': // CPL: Cursor Previous Line (up n rows, col 1)
		emu.scr.csiCpl_cursorPreviousLine(op.ADef(1))
	case 'G': // G: CHA: Cursor Horizontal Absolute (to col n, same row)
		emu.scr.csiCha_cursorHorizontalAbsolute(op.ADef(1))
	case 'H', 'f':
		// H: CUP: Cursor Position (row n, col m, default 1,1)
		// f: HVP: Horizontal and Vertical Position (same as CUP)
		emu.scr.csiCup_cursorPosition(op.ADef(1), op.BDef(1))
	case 'I': // CHT: cursor horizontal tabulation
		emu.scr.csiCht_cursorHorizontalTabulation(op.ADef(1))
	case 'J': // ed: Erase in Display
		emu.scr.csiEd_eraseInDisplay(op.A())
	case 'K': // EL: Erase in Line
		emu.scr.csiEl_eraseInLine(op.A())
	case 'L': // IL: Insert Lines
		emu.scr.csiIl_insertLines(op.ADef(1))
	case 'M': // DL: Delete Lines
		emu.scr.csiDl_deleteLines(op.ADef(1))
	case 'P': //  DCH: Delete Characters
		emu.scr.csiDch_deleteChars(op.ADef(1))
	case 'S': // SU: Scroll Up
		emu.scr.csiSu_scrollUp(op.ADef(1))
	case 'T': // SD: Scroll Down
		emu.scr.csiSd_scrollDown(op.ADef(1))
	case 'X': //  ECH: Erase Characters
		emu.scr.csiEch_eraseChars(op.ADef(1))
	case 'Z': // CBT: Cursor Backward Tab
		emu.scr.csiCbt_cursorBackwardTab(op.ADef(1))

	case '@': // ICH: Insert Characters
		emu.scr.csiIch_insertChars(op.ADef(1))
	case '`': // HPA: Horizontal Position Absolute (same as CHA, but 0-based in some terms)
		emu.scr.csiCha_cursorHorizontalAbsolute(op.ADef(0) + 1)

	case 'c': // DA: Device Attributes
		switch {
		case op.isPriv(0) && op.A() == 0: // primary
			emu.sendToExec(termSeqReply)
		case op.isPriv('>') && op.A() == 0: // secondary
			emu.sendToExec("\x1b[>0;1;1c")
		case op.isPriv('=') && op.A() == 0: // tertiary
			emu.sendToExec("\x1b[>0;1;1c")
		default:
			emu.csiOpTodo(op)
		}
	case 'd': //  vpa: Vertical Position Absolute (to row n)
		emu.scr.moveToRow(op.A())
	case 'g': // TBC: Tabulation Clear
		emu.scr.csiTbc_tabClear(op.ADef(0))
	case 'h', 'l': // h:sm: Set Mode; l:rm: Reset Mode
		//// DEBUG
		//emu.csiOpTodo(op)

		on := op.final == 'h'
		emu.scr.csi_setResetMode(
			op.priv,
			op.param(0), op.param(1), op.param(2),
			on,
			emu.userCons,
		)
	case 'm': // SGR: Select Graphic Rendition (colors, bold, etc.)
		emu.scr.csiSgr_selectGraphicRendition(op.params)
	case 'n': // DSR: Device Status Report
		switch op.A() {
		case 5: // "are you ok?"
			emu.sendToExec("\x1b[0n") // "OK"
		case 6: // cursor position report
			row1, col1 := emu.scr.csiCpr_cursorPositionReport()
			s := fmt.Sprintf("\x1b[%d;%dR", row1, col1)
			emu.sendToExec(s)
		default:
			emu.csiOpTodo(op)
		}
	case 'p':
		if op.isPriv('!') {
			emu.scr.escRis_reset(false)
		} else {
			emu.csiOpTodo(op)
		}
	case 'q': // DECLL: Load LEDs
		switch op.A() {
		//case 0: // 	clear all leds
		//case 1: // light nums lock
		//case 2:
		//	switch op.B() {
		//	case 0: // light caps lock
		//	case 1: // extinguish num lock
		//	case 2: // extinguish caps lock
		//	case 3: // extinguish scroll lock
		//	}
		//case 3: // light scroll lock
		default:
			emu.csiOpTodo(op)
		}
	case 'r': // DECSTBM: Set Scrolling Region
		top1, bot1 := op.ADef(1), op.BDef(emu.scr.H)
		emu.scr.setScrollRegion(top1, bot1)
		emu.scr.moveToOrigin()
	case 's':
		// SLRM: set left right margins
		if len(op.params) == 2 {
			left1, right1 := op.ADef(1), op.BDef(1)
			emu.scr.csiSlrm_setXMargins(left1, right1)
			return
		}
		// SCP: Save Cursor Position
		if op.isPriv(0) || op.isPriv('?') {
			emu.scr.csiScp_saveCursorPos()
		}
	case 'u': // RCP: Restore Cursor Position
		//switch {
		//case op.csiPrivIs('>'): // kitty: push flags (default 0)
		//	//emu.kittyPush(op.ADef(0))
		//	return
		//case op.csiPrivIs('<'): // kitty: pop N (default 1)
		//	//emu.kittyPop(op.ADef(1))
		//	return
		//case op.csiPrivIs('?'): // kitty: query flags
		//	//fmt.Fprintf(emu.readPw, "\x1b[?%du", emu.kittyFlags)
		//	return
		//}
		if op.isPriv(0) {
			emu.scr.csiRcp_restoreCursorPos()
		} else {
			emu.csiOpTodo(op)
		}

	case 't':
		if op.isPriv(0) {
			switch op.A() {
			case 22: // xterm: save window/icon title
			case 23: // xterm: restore window/icon title
			default:
				emu.csiOpTodo(op)
			}
		} else {
			emu.csiOpTodo(op)
		}

	default:
		emu.csiOpTodo(op)
	}
}

func (emu *Emu) csiOpTodo(op *TermCsiOp) {
	err := fmt.Errorf("emu.csi.final: todo: %c, %#v", op.final, op)
	emu.userCons.Error(err)
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
