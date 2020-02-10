package windriver

//go:generate stringer -tags=windows -type=_wm -output zwm.go
//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zwinapi.go winapi.go

import (
	"image"
	"image/color"
	"log"

	"golang.org/x/sys/windows"
)

//----------

const (
	_CW_USEDEFAULT = 0x80000000 - 0x100000000

	_CS_VREDRAW = 0x0001 // redraw on width adjust
	_CS_HREDRAW = 0x0002 // redraw on height adjust

	_LR_DEFAULTCOLOR = 0x00000000
	_LR_DEFAULTSIZE  = 0x00000040
	_LR_SHARED       = 0x00008000

	_IMAGE_BITMAP = 0
	_IMAGE_ICON   = 1
	_IMAGE_CURSOR = 2

	_IDC_ARROW       = 32512
	_IDC_IBEAM       = 32513
	_IDC_WAIT        = 32514 // hourglass
	_IDC_UPARROW     = 32516
	_IDC_HAND        = 32649
	_IDC_SIZEALL     = 32646
	_IDC_SIZENS      = 32645
	_IDC_SIZENWSE    = 32642
	_IDC_SIZEWE      = 32644
	_IDC_CROSS       = 32515 // crosshair
	_IDC_NO          = 32648 // slashed circle
	_IDC_APPSTARTING = 32650 // standard arrow with hourglass

	_COLOR_WINDOW  = 5
	_COLOR_BTNFACE = 15

	//_IDI_APPLICATION = 32512

	_SW_SHOWDEFAULT = 10
	_SW_HIDE        = 0
	_SW_SHOW        = 5

	// redraw window
	_RDW_INTERNALPAINT = 2
	_RDW_UPDATENOW     = 256

	// bitblt
	_SRCCOPY     = 0x00CC0020
	_NOT_SRCCOPY = 0x00330008

	_ERROR_INVALID_PARAMETER = 0x57 // 87
	_ERROR_NOT_ENOUGH_MEMORY = 0x8

	_BI_RGB         = 0
	_DIB_RGB_COLORS = 0

	// related to wm_setcursor/wm_nchittest
	_HTCLIENT = 1 // In a client area.

	// clipboard format
	_CF_UNICODETEXT = 13

	// globalalloc()
	_GMEM_MOVEABLE = 2

	// error related
	// https://docs.microsoft.com/en-us/windows/win32/seccrypto/common-hresult-values
	_S_OK          = 0x0
	_E_OUTOFMEMORY = 0x8007000e
	_E_NOINTERFACE = 0x80004002 // No such interface supported
	_E_POINTER     = 0x80004003 // 	Pointer that is not valid
)

const (
	_MK_SHIFT    = 0x0004 // The SHIFT key is down.
	_MK_CONTROL  = 0x0008 // The CTRL key is down.
	_MK_LBUTTON  = 0x0001 // The left mouse button is down.
	_MK_MBUTTON  = 0x0010 // The middle mouse button is down.
	_MK_RBUTTON  = 0x0002 // The right mouse button is down.
	_MK_XBUTTON1 = 0x0020 // The first X button is down.
	_MK_XBUTTON2 = 0x0040 // The second X button is down.
)

const (
	_VK_SHIFT   = 0x10
	_VK_CONTROL = 0x11
	_VK_MENU    = 0x12 // alt
	//_VK_LMENU   = 0xa4 // alt-gr?
	//_VK_RMENU   = 0xa5 // alt-gr?
	_VK_CAPITAL = 0x14 // caps-lock

	_VK_LBUTTON  = 0x01 // mouse left button
	_VK_RBUTTON  = 0x02 // right mouse button
	_VK_MBUTTON  = 0x04 // middle mouse button
	_VK_XBUTTON1 = 0x05
	_VK_XBUTTON2 = 0x06
)

// https://docs.microsoft.com/en-us/windows/win32/menurc/wm-syscommand
const (
	_SC_CLOSE        = 0xF060 // Closes the window.
	_SC_CONTEXTHELP  = 0xF180
	_SC_DEFAULT      = 0xF160
	_SC_HOTKEY       = 0xF150
	_SC_HSCROLL      = 0xF080     // Scrolls horizontally.
	_SCF_ISSECURE    = 0x00000001 // screen saver is secure.
	_SC_KEYMENU      = 0xF100
	_SC_MAXIMIZE     = 0xF030 // Maximizes the window.
	_SC_MINIMIZE     = 0xF020 // Minimizes the window.
	_SC_MONITORPOWER = 0xF170 // need to check lparam
	_SC_MOUSEMENU    = 0xF090
	_SC_MOVE         = 0xF010 // Moves the window.
	_SC_NEXTWINDOW   = 0xF040 // Moves to the next window.
	_SC_PREVWINDOW   = 0xF050 // Moves to the previous window.
	_SC_RESTORE      = 0xF120 // Restores the window pos/size
	_SC_SCREENSAVE   = 0xF140 // Executes the screen saver
	_SC_SIZE         = 0xF000 // Sizes the window.
	_SC_TASKLIST     = 0xF130 // Activates the Start menu.
	_SC_VSCROLL      = 0xF070 // Scrolls vertically.
)

// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-mapvirtualkeyw
const (
	_MAPVK_VK_TO_VSC    = 0
	_MAPVK_VSC_TO_VK    = 1
	_MAPVK_VK_TO_CHAR   = 2
	_MAPVK_VSC_TO_VK_EX = 3
)

const (
	_WS_VISIBLE = 0x10000000
	_WS_BORDER  = 0x00800000
	_WS_SIZEBOX = 0x00040000 // same as thickframe

	_WS_OVERLAPPED       = 0x00000000
	_WS_MAXIMIZEBOX      = 0x00010000
	_WS_MINIMIZEBOX      = 0x00020000
	_WS_THICKFRAME       = 0x00040000
	_WS_SYSMENU          = 0x00080000
	_WS_CAPTION          = 0x00C00000
	_WS_OVERLAPPEDWINDOW = _WS_OVERLAPPED |
		_WS_CAPTION |
		_WS_SYSMENU |
		_WS_THICKFRAME |
		_WS_MINIMIZEBOX | _WS_MAXIMIZEBOX
)

type _wm uint32

const (
	_WM_NULL            _wm = 0x00
	_WM_CREATE          _wm = 0x01
	_WM_DESTROY         _wm = 0x02
	_WM_MOVE            _wm = 0x03
	_WM_SIZE            _wm = 0x05
	_WM_ACTIVATE        _wm = 0x06
	_WM_SETFOCUS        _wm = 0x07
	_WM_KILLFOCUS       _wm = 0x08
	_WM_ENABLE          _wm = 0x0A
	_WM_SETREDRAW       _wm = 0x0B
	_WM_SETTEXT         _wm = 0x0C
	_WM_GETTEXT         _wm = 0x0D
	_WM_GETTEXTLENGTH   _wm = 0x0E
	_WM_PAINT           _wm = 0x0F
	_WM_CLOSE           _wm = 0x10
	_WM_QUERYENDSESSION _wm = 0x11
	_WM_QUIT            _wm = 0x12
	_WM_QUERYOPEN       _wm = 0x13
	_WM_ERASEBKGND      _wm = 0x14
	_WM_SYSCOLORCHANGE  _wm = 0x15
	_WM_ENDSESSION      _wm = 0x16
	_WM_SYSTEMERROR     _wm = 0x17
	_WM_SHOWWINDOW      _wm = 0x18
	_WM_CTLCOLOR        _wm = 0x19
	_WM_WININICHANGE    _wm = 0x1A
	_WM_SETTINGCHANGE   _wm = 0x1A
	_WM_DEVMODECHANGE   _wm = 0x1B
	_WM_ACTIVATEAPP     _wm = 0x1C
	_WM_FONTCHANGE      _wm = 0x1D
	_WM_TIMECHANGE      _wm = 0x1E
	_WM_CANCELMODE      _wm = 0x1F
	_WM_SETCURSOR       _wm = 0x20
	_WM_MOUSEACTIVATE   _wm = 0x21
	_WM_CHILDACTIVATE   _wm = 0x22
	_WM_QUEUESYNC       _wm = 0x23
	_WM_GETMINMAXINFO   _wm = 0x24
	_WM_PAINTICON       _wm = 0x26
	_WM_ICONERASEBKGND  _wm = 0x27
	_WM_NEXTDLGCTL      _wm = 0x28
	_WM_SPOOLERSTATUS   _wm = 0x2A
	_WM_DRAWITEM        _wm = 0x2B
	_WM_MEASUREITEM     _wm = 0x2C
	_WM_DELETEITEM      _wm = 0x2D
	_WM_VKEYTOITEM      _wm = 0x2E
	_WM_CHARTOITEM      _wm = 0x2F

	_WM_SETFONT                _wm = 0x30
	_WM_GETFONT                _wm = 0x31
	_WM_SETHOTKEY              _wm = 0x32
	_WM_GETHOTKEY              _wm = 0x33
	_WM_QUERYDRAGICON          _wm = 0x37
	_WM_COMPAREITEM            _wm = 0x39
	_WM_COMPACTING             _wm = 0x41
	_WM_WINDOWPOSCHANGING      _wm = 0x46
	_WM_WINDOWPOSCHANGED       _wm = 0x47
	_WM_POWER                  _wm = 0x48
	_WM_COPYDATA               _wm = 0x4A
	_WM_CANCELJOURNAL          _wm = 0x4B
	_WM_NOTIFY                 _wm = 0x4E
	_WM_INPUTLANGCHANGEREQUEST _wm = 0x50
	_WM_INPUTLANGCHANGE        _wm = 0x51
	_WM_TCARD                  _wm = 0x52
	_WM_HELP                   _wm = 0x53
	_WM_USERCHANGED            _wm = 0x54
	_WM_NOTIFYFORMAT           _wm = 0x55
	_WM_CONTEXTMENU            _wm = 0x7B
	_WM_STYLECHANGING          _wm = 0x7C
	_WM_STYLECHANGED           _wm = 0x7D
	_WM_DISPLAYCHANGE          _wm = 0x7E
	_WM_GETICON                _wm = 0x7F
	_WM_SETICON                _wm = 0x80

	_WM_NCCREATE        _wm = 0x81
	_WM_NCDESTROY       _wm = 0x82
	_WM_NCCALCSIZE      _wm = 0x83
	_WM_NCHITTEST       _wm = 0x84
	_WM_NCPAINT         _wm = 0x85
	_WM_NCACTIVATE      _wm = 0x86
	_WM_GETDLGCODE      _wm = 0x87
	_WM_NCMOUSEMOVE     _wm = 0xA0
	_WM_NCLBUTTONDOWN   _wm = 0xA1
	_WM_NCLBUTTONUP     _wm = 0xA2
	_WM_NCLBUTTONDBLCLK _wm = 0xA3
	_WM_NCRBUTTONDOWN   _wm = 0xA4
	_WM_NCRBUTTONUP     _wm = 0xA5
	_WM_NCRBUTTONDBLCLK _wm = 0xA6
	_WM_NCMBUTTONDOWN   _wm = 0xA7
	_WM_NCMBUTTONUP     _wm = 0xA8
	_WM_NCMBUTTONDBLCLK _wm = 0xA9

	//_WM_KEYFIRST    _wm = 0x100
	_WM_KEYDOWN     _wm = 0x100
	_WM_KEYUP       _wm = 0x101
	_WM_CHAR        _wm = 0x102
	_WM_DEADCHAR    _wm = 0x103
	_WM_SYSKEYDOWN  _wm = 0x104
	_WM_SYSKEYUP    _wm = 0x105
	_WM_SYSCHAR     _wm = 0x106
	_WM_SYSDEADCHAR _wm = 0x107
	_WM_KEYLAST     _wm = 0x108

	_WM_IME_STARTCOMPOSITION _wm = 0x10D
	_WM_IME_ENDCOMPOSITION   _wm = 0x10E
	_WM_IME_COMPOSITION      _wm = 0x10F
	_WM_IME_KEYLAST          _wm = 0x10F

	_WM_INITDIALOG    _wm = 0x110
	_WM_COMMAND       _wm = 0x111
	_WM_SYSCOMMAND    _wm = 0x112
	_WM_TIMER         _wm = 0x113
	_WM_HSCROLL       _wm = 0x114
	_WM_VSCROLL       _wm = 0x115
	_WM_INITMENU      _wm = 0x116
	_WM_INITMENUPOPUP _wm = 0x117
	_WM_MENUSELECT    _wm = 0x11F
	_WM_MENUCHAR      _wm = 0x120
	_WM_ENTERIDLE     _wm = 0x121

	_WM_CTLCOLORMSGBOX    _wm = 0x132
	_WM_CTLCOLOREDIT      _wm = 0x133
	_WM_CTLCOLORLISTBOX   _wm = 0x134
	_WM_CTLCOLORBTN       _wm = 0x135
	_WM_CTLCOLORDLG       _wm = 0x136
	_WM_CTLCOLORSCROLLBAR _wm = 0x137
	_WM_CTLCOLORSTATIC    _wm = 0x138

	//_WM_MOUSEFIRST    _wm = 0x200
	_WM_MOUSEMOVE     _wm = 0x200
	_WM_LBUTTONDOWN   _wm = 0x201
	_WM_LBUTTONUP     _wm = 0x202
	_WM_LBUTTONDBLCLK _wm = 0x203
	_WM_RBUTTONDOWN   _wm = 0x204
	_WM_RBUTTONUP     _wm = 0x205
	_WM_RBUTTONDBLCLK _wm = 0x206
	_WM_MBUTTONDOWN   _wm = 0x207
	_WM_MBUTTONUP     _wm = 0x208
	_WM_MBUTTONDBLCLK _wm = 0x209
	_WM_MOUSEWHEEL    _wm = 0x20A
	_WM_MOUSEHWHEEL   _wm = 0x20E

	_WM_PARENTNOTIFY   _wm = 0x210
	_WM_ENTERMENULOOP  _wm = 0x211
	_WM_EXITMENULOOP   _wm = 0x212
	_WM_NEXTMENU       _wm = 0x213
	_WM_SIZING         _wm = 0x214
	_WM_CAPTURECHANGED _wm = 0x215
	_WM_MOVING         _wm = 0x216
	_WM_POWERBROADCAST _wm = 0x218
	_WM_DEVICECHANGE   _wm = 0x219

	_WM_MDICREATE      _wm = 0x220
	_WM_MDIDESTROY     _wm = 0x221
	_WM_MDIACTIVATE    _wm = 0x222
	_WM_MDIRESTORE     _wm = 0x223
	_WM_MDINEXT        _wm = 0x224
	_WM_MDIMAXIMIZE    _wm = 0x225
	_WM_MDITILE        _wm = 0x226
	_WM_MDICASCADE     _wm = 0x227
	_WM_MDIICONARRANGE _wm = 0x228
	_WM_MDIGETACTIVE   _wm = 0x229
	_WM_MDISETMENU     _wm = 0x230
	_WM_ENTERSIZEMOVE  _wm = 0x231
	_WM_EXITSIZEMOVE   _wm = 0x232
	_WM_DROPFILES      _wm = 0x233
	_WM_MDIREFRESHMENU _wm = 0x234

	_WM_IME_SETCONTEXT      _wm = 0x281
	_WM_IME_NOTIFY          _wm = 0x282
	_WM_IME_CONTROL         _wm = 0x283
	_WM_IME_COMPOSITIONFULL _wm = 0x284
	_WM_IME_SELECT          _wm = 0x285
	_WM_IME_CHAR            _wm = 0x286
	_WM_IME_KEYDOWN         _wm = 0x290
	_WM_IME_KEYUP           _wm = 0x291

	_WM_MOUSEHOVER   _wm = 0x2A1
	_WM_NCMOUSELEAVE _wm = 0x2A2
	_WM_MOUSELEAVE   _wm = 0x2A3

	_WM_CUT   _wm = 0x300
	_WM_COPY  _wm = 0x301
	_WM_PASTE _wm = 0x302
	_WM_CLEAR _wm = 0x303
	_WM_UNDO  _wm = 0x304

	_WM_RENDERFORMAT      _wm = 0x305
	_WM_RENDERALLFORMATS  _wm = 0x306
	_WM_DESTROYCLIPBOARD  _wm = 0x307
	_WM_DRAWCLIPBOARD     _wm = 0x308
	_WM_PAINTCLIPBOARD    _wm = 0x309
	_WM_VSCROLLCLIPBOARD  _wm = 0x30A
	_WM_SIZECLIPBOARD     _wm = 0x30B
	_WM_ASKCBFORMATNAME   _wm = 0x30C
	_WM_CHANGECBCHAIN     _wm = 0x30D
	_WM_HSCROLLCLIPBOARD  _wm = 0x30E
	_WM_QUERYNEWPALETTE   _wm = 0x30F
	_WM_PALETTEISCHANGING _wm = 0x310
	_WM_PALETTECHANGED    _wm = 0x311

	_WM_HOTKEY      _wm = 0x312
	_WM_PRINT       _wm = 0x317
	_WM_PRINTCLIENT _wm = 0x318

	_WM_HANDHELDFIRST  _wm = 0x358
	_WM_HANDHELDLAST   _wm = 0x35F
	_WM_PENWINFIRST    _wm = 0x380
	_WM_PENWINLAST     _wm = 0x38F
	_WM_COALESCE_FIRST _wm = 0x390
	_WM_COALESCE_LAST  _wm = 0x39F
	_WM_DDE_FIRST      _wm = 0x3E0
	_WM_DDE_INITIATE   _wm = 0x3E0
	_WM_DDE_TERMINATE  _wm = 0x3E1
	_WM_DDE_ADVISE     _wm = 0x3E2
	_WM_DDE_UNADVISE   _wm = 0x3E3
	_WM_DDE_ACK        _wm = 0x3E4
	_WM_DDE_DATA       _wm = 0x3E5
	_WM_DDE_REQUEST    _wm = 0x3E6
	_WM_DDE_POKE       _wm = 0x3E7
	_WM_DDE_EXECUTE    _wm = 0x3E8
	_WM_DDE_LAST       _wm = 0x3E8

	_WM_USER _wm = 0x400
	_WM_APP  _wm = 0x8000

	//	_WM_QUERYNEWPALETTE       _wm = 0x030F
	//	_WM_DWMNCRENDERINGCHANGED _wm = 0x031F

)

//----------
//----------
//----------

type _WndClassExW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type _Msg struct {
	HWnd     windows.Handle
	Msg      uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       _Point
	LPrivate uint32
}

type _CreateStructW struct {
	LpCreateParams uintptr
	HInstance      windows.Handle
	HMenu          windows.Handle
	HWnd           windows.Handle
	CY             int32 // h
	CX             int32 // w
	Y              int32
	X              int32
	Style          int32
	LpszName       *uint16
	LpszClass      *uint16
	DwExStyle      uint32
}

type _WindowPos struct {
	HWndInsertAfter uintptr
	HWnd            uintptr
	X               int32
	Y               int32
	CX              int32 // h
	CY              int32 // w
	Flags           uint32
}

type _MinMaxInfo struct {
	PtReserved     _Point
	PtMaxSize      _Point
	PtMaxPosition  _Point
	PtMinTrackSize _Point
	PtMaxTrackSize _Point
}

type _Paint struct {
	Hdc         windows.Handle
	FErase      bool
	RcPaint     _Rect
	FRestore    bool
	FIncUpdate  bool
	RgbReserved [32]byte
}

type _Bitmap struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

type _BitmapInfo struct {
	BmiHeader _BitmapInfoHeader
	BmColors  [1]_RgbQuad
}

type _BitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type _RgbQuad struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

//----------

type _Point struct {
	X, Y int32
}

func (p *_Point) ToImagePoint() image.Point {
	return image.Point{X: int(p.X), Y: int(p.Y)}
}

//----------

type _Rect struct {
	left, top, right, bottom int32
}

func RectFromImageRectangle(r image.Rectangle) _Rect {
	return _Rect{
		left:   int32(r.Min.X),
		right:  int32(r.Max.X),
		top:    int32(r.Min.Y),
		bottom: int32(r.Max.Y),
	}
}

func (r *_Rect) ToImageRectangle() image.Rectangle {
	return image.Rect(int(r.left), int(r.top), int(r.right), int(r.bottom))
}

//----------

type _ColorRef uint32 // hex form: 0x00bbggrr

func ColorRefFromImageColor(c color.Color) _ColorRef {
	if c2, ok := c.(color.RGBA); ok {
		return rgbToColorRef(c2.R, c2.G, c2.B)
	}
	r, g, b, _ := c.RGBA()
	return rgbToColorRef(byte(r>>8), byte(g>>8), byte(b>>8))
}

func rgbToColorRef(r, g, b byte) _ColorRef {
	return _ColorRef(r) | _ColorRef(g)<<8 | _ColorRef(b)<<16
}

//----------

func unpackLowHigh(v uint32) (int, int) {
	low := uint16(v)
	high := uint16(v >> 16)
	return int(low), int(high)
}
func packLowHigh(l, h uint16) uint32 {
	return (uint32(h) << 16) | uint32(l)
}

func UTF16PtrFromString(s string) *uint16 {
	ptr, err := windows.UTF16PtrFromString(s)
	if err != nil {
		log.Printf("error: windows UTF16PtrFromString: %v", err)
	}
	return ptr
}

//----------

// NOTES
// int -> int32
// uint -> uint32
// lpcstr -> *uint8(?)
// lpcwstr -> *uint16 // string of 16-bit unicode characters
// lpwstr -> *uint16 // string of 16-bit unicode characters
// short -> uint16
// word -> uint16
// dword -> uint32
// long -> int32 // apparently not 64(?)

//sys _GetModuleHandleW(name *uint16) (modH windows.Handle, err error) = kernel32.GetModuleHandleW
//sys _GlobalLock(h windows.Handle) (ptr uintptr, err error) = kernel32.GlobalLock
//sys _GlobalUnlock(h windows.Handle) (ok bool) = kernel32.GlobalUnlock
//sys _GlobalAlloc(uFlags uint32, dwBytes uintptr) (h windows.Handle, err error) = kernel32.GlobalAlloc
//sys _GetConsoleWindow() (cH windows.Handle) = kernel32.GetConsoleWindow
//sys _GetCurrentProcessId() (pid uint32)  = kernel32.GetCurrentProcessId

//sys _LoadImageW(hInstance windows.Handle, name uintptr, typ uint32, cx int32, cy int32, fuLoad uint32) (imgH windows.Handle, err error) = user32.LoadImageW
//sys _LoadCursorW(hInstance windows.Handle, name uint32) (cursorH windows.Handle, err error) = user32.LoadCursorW
//sys _RegisterClassExW(wcx *_WndClassExW) (atom uint16, err error) = user32.RegisterClassExW
//sys _CreateWindowExW(dwExStyle uint32,lpClassName *uint16, lpWindowName *uint16, dwStyle int32, x int32, y int32, nWidth int32, nHeight int32, hWndParent windows.Handle, hMenu windows.Handle, hInstance windows.Handle, lpParam uintptr) (wndH windows.Handle, err error) = user32.CreateWindowExW
//sys _PostMessageW(hwnd windows.Handle, msg uint32, wParam uintptr, lParam uintptr) (ok bool) =user32.PostMessageW
//sys _GetMessageW(msg *_Msg, hwnd windows.Handle, msgFilterMin uint32, msgFilterMax uint32) (res int32, err error) [failretval==-1] = user32.GetMessageW
//sys _TranslateAccelerator(hwnd windows.Handle, hAccTable windows.Handle, msg *_Msg) (ok bool) = user32.TranslateAccelerator
//sys _TranslateMessage(msg *_Msg) (translated bool) = user32.TranslateMessage
//sys _DispatchMessageW(msg *_Msg) (res int32) = user32.DispatchMessageW
//sys _DefWindowProcW(hwnd windows.Handle, msg uint32, wparam uintptr, lparam uintptr) (ret uintptr) = user32.DefWindowProcW
//sys _GetWindowRect(hwnd windows.Handle, r *_Rect) (ok bool) = user32.GetWindowRect
//sys _SetCursor(cursorH windows.Handle) (prevCursorH windows.Handle) = user32.SetCursor
//sys _DestroyWindow(hwnd windows.Handle) (ok bool) = user32.DestroyWindow
//sys _PostQuitMessage(exitCode int32) = user32.PostQuitMessage
//sys _GetCursorPos(p *_Point) (ok bool) = user32.GetCursorPos
//sys _ValidateRect(hwnd windows.Handle, r *_Rect) (ok bool) = user32.ValidateRect
//sys _InvalidateRect(hwnd windows.Handle, r *_Rect, erase bool) (ok bool) = user32.InvalidateRect
//sys _BeginPaint(hwnd windows.Handle, paint *_Paint) (dcH windows.Handle, err error) = user32.BeginPaint
//sys _EndPaint(hwnd windows.Handle, paint *_Paint) (ok bool) = user32.EndPaint
//sys _UpdateWindow(hwnd windows.Handle) (ok bool) = user32.UpdateWindow
//sys _RedrawWindow(hwnd windows.Handle, r *_Rect, region windows.Handle, flags uint) (ok bool) = user32.RedrawWindow
//sys _ShowWindow(hwnd windows.Handle, nCmdShow int) (ok bool) = user32.ShowWindow
//sys _ShowWindowAsync(hwnd windows.Handle, nCmdShow int) (ok bool) = user32.ShowWindowAsync
//sys _GetDC(hwnd windows.Handle) (dcH windows.Handle, err error) = user32.GetDC
//sys _ReleaseDC(hwnd windows.Handle, dc windows.Handle) (ok bool) = user32.ReleaseDC
//sys _MapVirtualKeyW(uCode uint32, uMapType uint32) (code uint32) = user32.MapVirtualKeyW
//sys _ToUnicode(wVirtKey uint32, wScanCode uint32, lpKeyState *[256]byte, pwszBuff *uint16, cchBuff int32, wFlags uint32) (code int32) = user32.ToUnicode
//sys _GetKeyboardState(state *[256]byte) (ok bool) = user32.GetKeyboardState
//sys _GetKeyState(vkey int32) (state uint16) = user32.GetKeyState
//sys _SetCursorPos(x int32, y int32) (ok bool) = user32.SetCursorPos
//sys _MapWindowPoints(hwndFrom windows.Handle, hwndTo windows.Handle, lpPoints *_Point, cPoints uint32) (res int32) = user32.MapWindowPoints
//sys _ClientToScreen(hwnd windows.Handle, lpPoint *_Point) (ok bool) = user32.ClientToScreen
//sys _OpenClipboard(hWndNewOwner windows.Handle) (ok bool) = user32.OpenClipboard
//sys _CloseClipboard() (ok bool) = user32.CloseClipboard
//sys _SetClipboardData(uFormat uint32, h windows.Handle) (dataH windows.Handle, err error) = user32.SetClipboardData
//sys _GetClipboardData(uFormat uint32) (dataH windows.Handle, err error) = user32.GetClipboardData
//sys _EmptyClipboard() (ok bool) = user32.EmptyClipboard
//sys _GetWindowThreadProcessId(hwnd windows.Handle, pid *uint32) (threadId uint32) = user32.GetWindowThreadProcessId
//sys _SetWindowTextW(hwnd windows.Handle, lpString *uint16) (res bool) = user32.SetWindowTextW

//sys _SelectObject(hdc windows.Handle, obj windows.Handle) (prevObjH windows.Handle, err error) = gdi32.SelectObject
//sys _CreateBitmap(w int32, h int32, planes uint32, bitCount uint32, bits uintptr) (bmH windows.Handle, err error) = gdi32.CreateBitmap
//sys _CreateCompatibleBitmap(hdc windows.Handle, w int32, h int32) (bmH windows.Handle, err error) = gdi32.CreateCompatibleBitmap
//sys _DeleteObject(obj windows.Handle) (ok bool) = gdi32.DeleteObject
//sys _CreateCompatibleDC(hdc windows.Handle) (dcH windows.Handle, err error)  = gdi32.CreateCompatibleDC
//sys _DeleteDC(dc windows.Handle) (ok bool) = gdi32.DeleteDC
//sys _BitBlt(hdc windows.Handle, x int32, y int32, w int32, h int32, hdcSrc windows.Handle, x2 int32, y2 int32, rOp uint32) (ok bool) = gdi32.BitBlt
// colorSet should be ColorRef(uint32), but the docs say it can return -1!
//sys _SetPixel(hdc windows.Handle, x int, y int, c _ColorRef) (colorSet int32, err error) [failretval==-1] = gdi32.SetPixel
//sys _CreateBitmapIndirect(bm *_Bitmap) (bmH windows.Handle, err error) = gdi32.CreateBitmapIndirect
//sys _GetObject(h windows.Handle, c int32, v uintptr) (n int) = gdi32.GetObject
//sys	_CreateDIBSection(dc windows.Handle, bmi *_BitmapInfo, usage uint32, bits **byte, section windows.Handle, offset uint32) (bmH windows.Handle, err error) = gdi32.CreateDIBSection

//sys _DragAcceptFiles(hwnd windows.Handle, fAccept bool) = shell32.DragAcceptFiles
//sys _DragQueryPoint(hDrop uintptr, ppt *_Point) (res bool) = shell32.DragQueryPoint
//sys _DragQueryFileW(hDrop uintptr, iFile uint32, lpszFile *uint16, cch uint32)(res uint32) = shell32.DragQueryFileW
//sys _DragFinish(hDrop uintptr) = shell32.DragFinish

// NOT USED
////sys _OleInitialize(pvReserved uintptr) (hRes uintptr) = ole32.OleInitialize
////sys _RegisterDragDrop(hwnd windows.Handle, pDropTarget **_IDropTarget) (resH uintptr) = ole32.RegisterDragDrop
////sys _CoLockObjectExternal(pUnk uintptr, fLock bool, fLastUnlockReleases bool) (hres uintptr) = ole32.CoLockObjectExternal
