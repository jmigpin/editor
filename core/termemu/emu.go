package termemu

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jmigpin/editor/util/iout"
)

//godebug:annotatefile
//godebug:annotatefile:vtparser.go
//godebug:annotatefile:screen.go
////godebug:annotatefile:../erowtermemu.go

//----------

// https://vt100.net/docs/vt510-rm/contents.html
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
// const TermEnv = "TERM=xterm-mono" //
const TermEnv = "TERM=xterm" //

// const vt100 = "\x1b[?1;0c" //
// const vt101NoOpt = vt100
// const vt100WithAVO = "\x1b[?1;2c" //
// const vt102 = "\x1b[?6c" //
const vt420 = "\x1b[?64c" //
// const vt420Sixel = "\x1b[?6;4;4c" // ?
// const termSeqReply = vt100WithAVO
// const termSeqReply = vt102 //
const termSeqReply = vt420 //

//----------

type Emu struct {
	userRw io.ReadWriter
	tui    Tui

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

// emu itself is a read/write to be passed to the executable, wrapping the UI that is a rwc as well
func NewEmu(userRw io.ReadWriter, tui Tui, opts Opts) *Emu {
	emu := &Emu{userRw: userRw, tui: tui, opts: opts}

	emu.scr = NewScreen()
	emu.scr.onSizeChange = emu.tui.UpdateSize

	emu.setupExecSideRWC()

	emu.parser = NewVTParser(emu.execRwc, emu.applyEmit)
	emu.parser.ansiMode = emu.scr.privModes.AnsiNotVT52()

	emu.parserDone.Add(1)
	go func() {
		defer emu.parserDone.Done()
		err := emu.parser.Run()
		_ = err // TODO: check error
		//log.Println("termemu parser error:", err)
	}()

	return emu
}

//----------

func (emu *Emu) ClampSize(p P) P {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.clampSize(p)
}

func (emu *Emu) SetSize(p P) {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	emu.scr.setSize(p, false)
}

func (emu *Emu) GetSize() P {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.size()
}

//----------

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
		*rd = io.TeeReader(*rd, emu.userRw)
	}
	if emu.opts.Debug {
		rd := &execRwc.Reader
		*rd = io.TeeReader(*rd, iout.FnWriter(func(p []byte) (int, error) {
			s := fmt.Sprintf("%q", p)
			emu.sendForDebug("rcv from exec: " + s)
			return len(p), nil
		}))
	}

	// auto read from user to exec
	go func() {
		if emu.opts.Debug {
			rd := iout.FnReader(func(p []byte) (int, error) {
				n, err := emu.userRw.Read(p)
				s := fmt.Sprintf("rcv from user: %q\n", string(p[:n]))
				emu.sendForDebug(s)
				return n, err
			})
			_, _ = io.Copy(emu.execRwc, rd)
		} else {
			_, _ = io.Copy(emu.execRwc, emu.userRw)
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
	defer emu.parserDone.Wait() // on exec close, parse should stop
	return emu.execRwc.Close()
}

//----------

func (emu *Emu) sendToExec(s string) {
	if emu.opts.Debug {
		s2 := fmt.Sprintf("snd to exec: %q\n", s)
		emu.sendForDebug(s2)
	}

	_, _ = emu.execRwc.Write([]byte(s))
}
func (emu *Emu) sendToUser(s string) {
	_, _ = emu.userRw.Write([]byte(s))
}
func (emu *Emu) sendForDebug(s string) {
	//fmt.Print(s)
	emu.tui.Print("emu.dbg: " + s)
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

//func (emu *Emu) Snapshot2(fn func(*Screen)) {
//	emu.mu.Lock()
//	defer emu.mu.Unlock()
//	fn(emu.scr)
//}

func (emu *Emu) ScrPrivModes() *PrivModes {
	emu.mu.Lock()
	defer emu.mu.Unlock()
	return emu.scr.privModes.clone()
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
		emu.scr.escNel_nextLine()
	case "rc":
		emu.scr.escRc_restoreCursor()
	case "ri":
		emu.scr.escRi_reverseIndex()
	case "ris":
		emu.scr.escRis_reset(true)
	case "sc":
		emu.scr.escSc_saveCursor()

	case "vt52Id":
		//emu.sendToExec("\x1b/K") // vt52
		emu.sendToExec("\x1b/Z") // vt52 emulated by vt100

	//----------

	case "print":
		for _, ru := range op.s {
			emu.scr.putRune(ru)
		}
		if emu.plainMode() {
			emu.sendToUser(op.s)
		}

	case "unknownEsc":
		err := fmt.Errorf("emu.applyemit: vtparser: %q", op.s)
		emu.tui.Error(err)

	default:
		err := fmt.Errorf("emu.applyemit: %q", op.kind)
		//fmt.Println(err)
		panic(err)
	}

	if emu.opts.Mode == ModeGrid {
		if !emu.scr.privModes.SynchronizedOutput() {
			emu.tui.Paint()
		}
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
		// TODO: somehow, wait for new content for some time before showing a clear screen - avoids flicker

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
		emu.scr.csiVpa_moveToRow(op.A())
	case 'g': // TBC: Tabulation Clear
		emu.scr.csiTbc_tabClear(op.ADef(0))
	case 'h', 'l': // h:sm: Set Mode; l:rm: Reset Mode
		emu.csiSetMode(op)
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
		case 9: // CUSTOM: debug // TODO: use something else?
			emu.scr.PrintWithCursor()
			time.Sleep(100 * time.Second)
		default:
			emu.csiOpTodo(op)
		}
	case 'p':
		if op.footer == '$' && (op.isPriv('?') || op.isPriv(0)) {
			// DECRQM: request mode
			// DECRPM: report Mode
			idA := op.idA()
			on := emu.scr.privModes.isOn(idA)
			onS := "2"
			if on {
				onS = "1"
			}
			s := fmt.Sprintf("%s%s;%s$y", SeqEscCsi, idA, onS)
			emu.sendToExec(s)

			//// DEBUG
			//fmt.Println("csi report mode", op)

			break
		}

		if op.isPriv('!') {
			emu.scr.escRis_reset(false)
			break
		}

		emu.csiOpTodo(op)
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
		top1, bot1 := op.ADef(1), op.BDef(emu.scr.bounds.Max.Y)
		emu.scr.setScrollRegion(top1, bot1)
	case 's':
		// SLRM: set left right margins
		if len(op.params) == 2 {
			left1, right1 := op.ADef(1), op.BDef(1)
			emu.scr.csiSlrm_setLeftRightMargins(left1, right1)
			return
		}
		// SCP: Save Cursor Position
		//if op.isPriv(0) || op.isPriv('?') {
		if op.isPriv(0) && !emu.scr.privModes.leftRightMargin() {
			emu.scr.csiScp_saveCursorPos()
		}
	case 't':
		if op.isPriv(0) {
			switch op.A() {
			//case 14, 16: // ask for pixels?
			//case 18: // ask for cols/rows?
			case 22: // xterm: save window/icon title
			case 23: // xterm: restore window/icon title
			default:
				emu.csiOpTodo(op)
			}
		} else {
			emu.csiOpTodo(op)
		}
	case 'u':
		switch op.priv {
		case 0:
			// RCP: Restore Cursor Position
			emu.scr.csiRcp_restoreCursorPos()
		case '>': // kitty kb protocol: push flags
		case '<': // kitty kb protocol: pop n
		case '?': // kitty kb protocol: query flags
		default:
			emu.csiOpTodo(op)
		}
	case 'x': // Request Terminal Parameters (DECREQTPARM).
		if op.isPriv(0) {
			s := fmt.Sprintf("\x1b[%dx", op.A()+2)
			emu.sendToExec(s)
		} else {
			emu.csiOpTodo(op)
		}
	default:
		emu.csiOpTodo(op)
	}
}

//----------

func (emu *Emu) csiOpTodo(op *TermCsiOp) {
	err := fmt.Errorf("emu.csi.final: todo: %c, %#v", op.final, op)
	emu.tui.Error(err)
}

func (emu *Emu) csiSetMode(op *TermCsiOp) {
	//// DEBUG
	//emu.csiOpTodo(op)

	idA := op.idA()
	on := op.final == 'h'
	s := emu.scr
	s.privModes.set(idA, on)

	switch idA {
	case "2": // Keyboard Action Mode (KAM).
	case "4": // insert mode
	case "20": // Automatic Newline (LNM)

	case "?2": // ansi
		emu.parser.ansiMode = on
	case "?3": // 132 Column Mode (DECCOLM)
		s.updateSize()
	case "?6": // scroll origin mode
	case "?69": // left/right margin mode
		emu.scr.updateRegionX()

	case "?47", "?1047": // alternate screen buffer
		s.setGrid2(on)
	case "?1048": // save cursor
		if on {
			s.csiScp_saveCursorPos()
		} else {
			s.csiRcp_restoreCursorPos()
		}
	case "?1049": // save cursor, alternate screen buffer, clear
		if on {
			s.csiScp_saveCursorPos()
			s.setGrid2(true)
			//s.clearGrid()
		} else {
			s.csiRcp_restoreCursorPos()
			s.setGrid2(false)
			// commented: after an app closes, we want to see the previous content
			//s.clearGrid() // TODO: review
		}
	default:
		//emu.csiOpTodo(op)
		//emu.tui.Error(fmt.Errorf("emu.csi: todo: %v", idx))
	}
}

//----------
//----------
//----------

type Opts struct {
	Mode  Mode
	Debug bool
}

//----------

type Event struct {
	Kind string
	Data any
}

//----------

// terminal user interface
type Tui interface {
	UpdateSize()
	Paint()
	Print(any)
	Error(error)
}

//----------
