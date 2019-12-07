package windriver

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

////godebug:annotatepackage

// Function preceded by "ost" run in the "operating-system-thread".
type Window struct {
	className   *uint16
	windowTitle *uint16
	hwnd        windows.Handle
	instance    windows.Handle

	img draw.Image
	bmH windows.Handle // bitmap handle
	bm  *_Bitmap

	events    chan interface{}
	evLoopEnd chan struct{}

	postEv struct {
		sync.Mutex
		id int
		m  map[int]interface{}
	}

	cursors struct {
		currentId int
		cache     map[int]windows.Handle
	}
}

func NewWindow() (*Window, error) {
	win := &Window{
		events:    make(chan interface{}, 8),
		evLoopEnd: make(chan struct{}),
	}
	win.cursors.cache = map[int]windows.Handle{}
	win.postEv.m = map[int]interface{}{}

	// initial size
	win.ostResizeImage(image.Rect(0, 0, 1, 1))

	if err := win.initAndSetupLoop(); err != nil {
		return nil, err
	}

	return win, nil
}

//----------

func (win *Window) initAndSetupLoop() error {
	initErr := make(chan error)

	go func() {
		// ensure OS thread
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := win.ostInitialize(); err != nil {
			initErr <- err
			return
		}
		initErr <- nil

		// run event loop in OS thread
		win.ostMsgLoop() // blocks
	}()

	return <-initErr
}

//----------

func (win *Window) ostInitialize() error {
	// handle containing the window procedure for the class.
	instance, err := _GetModuleHandleW(nil)
	if err != nil {
		return fmt.Errorf("getmodulehandle: %v", err)
	}
	win.instance = instance
	//win.instance = windows.Handle(0)

	// cursor for the window class
	win.cursors.currentId = _IDC_ARROW
	cursorH, err := win.loadCursor(win.cursors.currentId)
	if err != nil {
		return err
	}

	// window class registration
	win.className = UTF16PtrFromString("editorClass")
	wce := _WndClassExW{
		LpszClassName: win.className,
		LpfnWndProc:   windows.NewCallback(win.wndProcCallback),
		HInstance:     win.instance,
		HCursor:       cursorH,
		HbrBackground: _COLOR_WINDOW + 1,
		Style:         _CS_HREDRAW | _CS_VREDRAW,
	}
	wce.CbSize = uint32(unsafe.Sizeof(wce))
	if _, err := _RegisterClassExW(&wce); err != nil {
		return fmt.Errorf("registerclassex: %w", err)
	}

	// create window
	win.windowTitle = UTF16PtrFromString("Editor")
	hwnd, err := _CreateWindowExW(
		0,
		win.className,
		win.windowTitle,
		_WS_OVERLAPPEDWINDOW,
		_CW_USEDEFAULT, _CW_USEDEFAULT, // x,y
		// TODO: failing, giving bad rectangle with a fixed integer
		//_CW_USEDEFAULT, _CW_USEDEFAULT, // w,h
		500, 500, // w,h
		0, 0, win.instance, 0,
	)
	if err != nil {
		return fmt.Errorf("createwindow: %w", err)
	}
	win.hwnd = hwnd

	_ = _ShowWindow(win.hwnd, _SW_SHOWDEFAULT)
	_ = _UpdateWindow(win.hwnd)

	return nil
}

//----------

func (win *Window) NextEvent() interface{} {
	return <-win.events
}

//----------

func (win *Window) PostEvent(ev interface{}) error {
	win.postEv.Lock()
	defer win.postEv.Unlock()
	id := win.postEv.id
	win.postEv.m[id] = ev
	if !_PostMessageW(win.hwnd, uint32(_WM_APP), uintptr(id), 0) {
		delete(win.postEv.m, id)
		return fmt.Errorf("postevent: failed to post")
	}
	win.postEv.id++
	return nil
}

func (win *Window) getPostEventData(id int) (interface{}, error) {
	win.postEv.Lock()
	defer win.postEv.Unlock()
	ev, ok := win.postEv.m[id]
	if !ok {
		return nil, fmt.Errorf("postevent map: id not found: %v", id)
	}
	delete(win.postEv.m, id)
	return ev, nil
}

//----------

// Called from OS thread.
func (win *Window) ostMsgLoop() {
	defer func() {
		close(win.evLoopEnd)
	}()

	msg := _Msg{} // ensure it is instantiated during the loop
	for {
		res, err := _GetMessageW(&msg, win.hwnd, 0, 0) // wait for next msg
		if err != nil {
			if err2 := windows.GetLastError(); err2 != nil { // improve error
				err = fmt.Errorf("%v: %v", err, err2)
			}
			win.events <- err
			break
		}
		quit := res == 0
		if quit {
			break
		}

		// not used: virtual keys are translated ondemand (keydown/keyup)
		//_ = _TranslateMessage(&msg)

		// dispatch to hwnd.class.LpfnWndProc (runs win.wndProcCallback)
		_ = _DispatchMessageW(&msg)

		// call directly instead of dispatch? getting cpu spin
		//win.handleMsg(&msg)
	}
}

//----------

// Called by dispatchMessage and via WndClassExW.
func (win *Window) wndProcCallback(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	m := &_Msg{
		HWnd:   hwnd,
		Msg:    msg,
		WParam: wParam,
		LParam: lParam,
	}
	return win.handleMsg(m)
}

//----------

func (win *Window) handleMsg(msg *_Msg) uintptr {
	defh := func() uintptr {
		return _DefWindowProcW(msg.HWnd, msg.Msg, msg.WParam, msg.LParam)
	}

	switch _wm(msg.Msg) {
	case _WM_CREATE:
		createW := (*_CreateW)(unsafe.Pointer(msg.LParam))
		w, h := int(createW.CX), int(createW.CY)
		r := image.Rect(0, 0, w, h)
		win.events <- &event.WindowResize{Rect: r}
	case _WM_SIZE:
		w, h := unpackLowHigh(uint32(msg.LParam))
		r := image.Rect(0, 0, w, h)
		win.events <- &event.WindowResize{Rect: r}

	case _WM_PAINT:
		// validate region or it keeps sending msgs(?)
		// always validate, the paint is done by AppPutImage msg
		//_ = _ValidateRect(msg.HWnd, nil)
		win.events <- &event.WindowExpose{}
		//return 0 // return zero if processed (won't validate region!)
	//case _WM_NCPAINT:
	case _WM_ERASEBKGND: // handle to avoid flicker
		// it does not erase bg
		return 0 // return non-zero if it erases the background

	case _WM_SETCURSOR:
		l, _ := unpackLowHigh(uint32(msg.LParam))
		if l == _HTCLIENT { // set only if in the client area (not the frame)
			if err := win.loadAndSetCursor(win.cursors.currentId); err != nil {
				win.events <- err
			}
			return 1 // return TRUE to halt further processing
		}

	case _WM_CLOSE: // window close button
		win.events <- &event.WindowClose{}
	case _WM_DESTROY:
		_PostQuitMessage(0)

	case _WM_SYSCOMMAND:
		c := int(msg.WParam)
		switch c {
		case _SC_CLOSE:
			win.events <- &event.WindowClose{}
		}

	//case _WM_CHAR: // not used: making the translation at keydown

	case _WM_KEYDOWN:
		win.events <- win.keyUpDown(msg, false)
	case _WM_KEYUP:
		win.events <- win.keyUpDown(msg, true)

	case _WM_MOUSEMOVE:
		win.events <- win.mouseMove(msg)
	case _WM_LBUTTONDOWN:
		win.events <- win.mouseButton(msg, event.ButtonLeft, false)
	case _WM_LBUTTONUP:
		win.events <- win.mouseButton(msg, event.ButtonLeft, true)
	case _WM_RBUTTONDOWN:
		win.events <- win.mouseButton(msg, event.ButtonRight, false)
	case _WM_RBUTTONUP:
		win.events <- win.mouseButton(msg, event.ButtonRight, true)
	case _WM_MBUTTONDOWN:
		win.events <- win.mouseButton(msg, event.ButtonMiddle, false)
	case _WM_MBUTTONUP:
		win.events <- win.mouseButton(msg, event.ButtonMiddle, true)
	case _WM_MOUSEWHEEL:
		_, h := unpackLowHigh(uint32(msg.WParam))
		up := int16(h) > 0
		b := event.ButtonWheelDown
		if up {
			b = event.ButtonWheelUp
		}
		// TODO: necessary?
		// send two events to simulate down/up
		win.events <- win.mouseButton(msg, b, true)
		win.events <- win.mouseButton(msg, b, false)

	case _WM_APP:
		id := int(msg.WParam)
		data, err := win.getPostEventData(id)
		if err != nil {
			win.events <- err
			break
		}
		app := data.(*appMsg)
		app.Reply <- win.handleAppEvent(msg, app)
	}

	return defh()
}

//----------

func (win *Window) handleAppEvent(msg *_Msg, app *appMsg) interface{} {
	switch t := app.Event.(type) {
	case *appClose:
		if !_DestroyWindow(msg.HWnd) {
			return fmt.Errorf("destroywindow: false")
		}
		return nil // TODO: handle app reply at wm_quit msg?

	case *appResizeImage:
		return win.ostResizeImage(t.r)
	case *appPutImage:
		return win.ostPaintImg(t.r)

	case *appSetCursor:
		return win.ostSetCursor(t.Cursor)
	case *appQueryPointer:
		p, err := win.ostQueryPointer()
		if err != nil {
			return err
		}
		return p
	case *appWarpPointer:
		return win.ostWarpPointer(t.p)

	case *appSetClipboardData:
		return win.ostSetClipboardData(t.s)
	case *appGetClipboardData:
		s, err := win.ostGetClipboardData()
		if err != nil {
			return err
		}
		return s
	default:
		panic("todo: app event type")
	}
}

//----------

func (win *Window) getWindowRectangle() (image.Rectangle, error) {
	r := _Rect{}
	if !_GetWindowRect(win.hwnd, &r) {
		return image.ZR, fmt.Errorf("getwindowrect: false")
	}
	return r.ToImageRectangle(), nil
}

//----------

func (win *Window) keyUpDown(msg *_Msg, up bool) interface{} {
	p, err := win.ostQueryPointer()
	if err != nil {
		return err
	}

	// TODO: use scancode instead of regetting at virtualkeyrune?
	//kd := keyData(uint32(msg.LParam)) // has scancode

	vkey := uint32(msg.WParam)
	kstate := [256]byte{}
	_ = _GetKeyboardState(&kstate)
	ru, _ := vkeyRune(vkey, &kstate)
	ks := translateVKeyToEventKeySym(vkey, ru)
	km := translateKStateToEventKeyModifiers(&kstate)
	bs := translateKStateToEventMouseButtons(&kstate)

	var ev interface{}
	if up {
		ev = &event.KeyUp{p, ks, km, bs, ru}
	} else {
		ev = &event.KeyDown{p, ks, km, bs, ru}
	}
	return &event.WindowInput{Point: p, Event: ev}
}

func (win *Window) mouseMove(msg *_Msg) interface{} {
	p := paramToPoint(uint32(msg.LParam)) // window point

	vkey := uint32(msg.WParam)
	km := translateVKeyToEventKeyModifiers(vkey)
	bs := translateVKeyToEventMouseButtons(vkey)

	ev := &event.MouseMove{p, bs, km}
	return &event.WindowInput{Point: p, Event: ev}
}

func (win *Window) mouseButton(msg *_Msg, b event.MouseButton, up bool) interface{} {
	p := paramToPoint(uint32(msg.LParam)) // window point
	// screen point if mousewheel
	if msg.Msg == uint32(_WM_MOUSEWHEEL) {
		p2, err := win.screenToWindowPoint(p)
		if err != nil {
			return err
		}
		p = p2
	}

	vkey := uint32(msg.WParam)
	km := translateVKeyToEventKeyModifiers(vkey)
	bs := translateVKeyToEventMouseButtons(vkey)

	var ev interface{}
	if up {
		ev = &event.MouseUp{p, b, bs, km}
	} else {
		ev = &event.MouseDown{p, b, bs, km}
	}
	return &event.WindowInput{Point: p, Event: ev}
}

//----------

func (win *Window) ostPaintImg(r image.Rectangle) error {
	//return win.paintImgWithSetPixel()
	return win.paintImgWithBitmap(r)
}

func (win *Window) paintImgWithSetPixel() error {
	hdc, err := _GetDC(win.hwnd)
	if err != nil {
		return fmt.Errorf("paintimg: getdc: %w", err)
	}
	defer _ReleaseDC(win.hwnd, hdc)

	//godebug:annotateoff
	r := win.img.Bounds()
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			c := win.img.At(x, y)
			u := ColorRefFromImageColor(c)
			if _, err := _SetPixel(hdc, x, y, u); err != nil {
				return fmt.Errorf("setpixel: %w", err)
			}
		}
	}
	return nil
}

func (win *Window) paintImgWithBitmap(r image.Rectangle) error {
	// get/release dc (beginpaint/endpaint won't work here)
	hdc, err := _GetDC(win.hwnd)
	if err != nil {
		return fmt.Errorf("paintimg: getdc: %w", err)
	}
	defer _ReleaseDC(win.hwnd, hdc)

	// memory dc
	hdcMem, err := _CreateCompatibleDC(hdc)
	if err != nil {
		return err
	}
	defer _DeleteDC(hdcMem) // deleted by releasedc

	//// map image to bitmap
	//bm, err := win.buildBitmap()
	//if err != nil {
	//	return err
	//}
	//defer _DeleteObject(bm)
	bm := win.bmH

	// setup bitmap into memory dc
	prev, err := _SelectObject(hdcMem, bm)
	if err != nil {
		return err
	}
	defer _SelectObject(hdcMem, prev)

	// copy memory dc into dc
	b := win.img.Bounds()
	r2 := r.Intersect(b)
	size2 := r2.Size()
	if !_BitBlt(hdc,
		int32(r2.Min.X), int32(r2.Min.Y),
		int32(size2.X), int32(size2.Y),
		hdcMem,
		int32(r2.Min.X), int32(r2.Min.Y),
		_SRCCOPY) {
		return fmt.Errorf("bitblt: false")
	}

	return nil
}

//func (win *Window) buildBitmap_() (bm windows.Handle, _ error) {
//	// image data
//	r := win.img.Bounds()
//	size := r.Size()
//	rgba := &win.img.(*imageutil.BGRA).RGBA
//	//pixHeader := (*reflect.SliceHeader)(unsafe.Pointer(&rgba.Pix))
//	//bits := pixHeader.Data
//	bits := uintptr(unsafe.Pointer(&rgba.Pix[0]))

//	//if bits >= math.MaxUint32 {
//	//	return 0, fmt.Errorf("bad bits: %v", bits)
//	//}

//	////TODO: works, using createbitmap instead (simpler)
//	//// map the image into a bitmap
//	//bm0 := _Bitmap{
//	//	BmType:       0, // must be zero
//	//	BmWidth:      int32(size.X),
//	//	BmHeight:     int32(size.Y),
//	//	BmWidthBytes: int32(rgba.Stride),
//	//	BmPlanes:     1,
//	//	BmBitsPixel:  4 * 8,
//	//	BmBits:       bits,
//	//}
//	//win.bm = &bm0
//	//// bitmap handle
//	//bm, err := _CreateBitmapIndirect(win.bm)

//	// map the image into a bitmap
//	bm, err := _CreateBitmap(int32(size.X), int32(size.Y), 1, 4*8, bits)

//	// improve error
//	if err != nil {
//		err2 := windows.GetLastError()
//		err = fmt.Errorf("buildbitmap: fail: %v, %v", err, err2)
//	}
//	return bm, err
//}

func (win *Window) buildBitmap(size image.Point) (bmH windows.Handle, bits *byte, _ error) {
	bmi := _BitmapInfo{
		BmiHeader: _BitmapInfoHeader{
			BiSize:        uint32(unsafe.Sizeof(_BitmapInfoHeader{})),
			BiWidth:       int32(size.X),
			BiHeight:      -int32(size.Y), // negative to invert y
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: _BI_RGB,
			BiSizeImage:   uint32(size.X * size.Y * 4),
		},
	}

	bmH, err := _CreateDIBSection(0, &bmi, _DIB_RGB_COLORS, &bits, 0, 0)
	if err != nil {
		return 0, nil, err
	}
	return bmH, bits, nil
}

//----------

func (win *Window) Image() draw.Image {
	//godebug:annotateoff
	return win.img
}

//----------

func (win *Window) ResizeImage(r image.Rectangle) error {
	app := newAppMsg(&appResizeImage{r})
	if err := win.PostEvent(app); err != nil {
		return err
	}
	return appCheckError(<-app.Reply)
}

func (win *Window) ostResizeImage(r image.Rectangle) error {
	//win.img = imageutil.NewBGRA(&r)

	//bmH, err := win.buildBitmap()
	//if err != nil {
	//	return err
	//}
	//if win.bmH != 0 {
	//	_DeleteObject(win.bmH)
	//}
	//win.bmH = bmH

	bmH, bits, err := win.buildBitmap(r.Size())
	if err != nil {
		return err
	}
	if win.bmH != 0 {
		_DeleteObject(win.bmH)
	}
	win.bmH = bmH

	// mask mem into a slice
	nbytes := imageutil.BGRASize(&r)
	h := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(bits)), Len: nbytes, Cap: nbytes}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	win.img = imageutil.NewBGRAFromBuffer(buf, &r)

	return nil
}

//----------

func (win *Window) PutImage(r image.Rectangle) error {
	app := newAppMsg(&appPutImage{r})
	if err := win.PostEvent(app); err != nil {
		return err
	}
	return appCheckError(<-app.Reply)
}

//----------

func (win *Window) SetCursor(c event.Cursor) {
	app := newAppMsg(&appSetCursor{c})
	if err := win.PostEvent(app); err != nil {
		log.Println(err)
		return
	}
	if err := appCheckError(<-app.Reply); err != nil {
		log.Println(err)
	}
}

//----------

func (win *Window) ostSetCursor(c event.Cursor) (err error) {
	sc := func(cId int) {
		err = win.loadAndSetCursor(cId)
	}

	switch c {
	case event.NoneCursor:
		// TODO: parent window cursor
		//sc(0) // TODO: failing
		sc(_IDC_ARROW)
	case event.DefaultCursor:
		sc(_IDC_ARROW)
	case event.NSResizeCursor:
		sc(_IDC_SIZENS)
	case event.WEResizeCursor:
		sc(_IDC_SIZEWE)
	case event.CloseCursor:
		//sc(_IDC_HAND)
		sc(_IDC_CROSS)
	case event.MoveCursor:
		sc(_IDC_SIZEALL)
	case event.PointerCursor:
		//sc(_IDC_HAND)
		sc(_IDC_UPARROW)
	case event.BeamCursor:
		sc(_IDC_IBEAM)
	case event.WaitCursor:
		sc(_IDC_WAIT)
	}
	return
}

func (win *Window) loadAndSetCursor(cursorId int) error {
	cursorHandle, err := win.loadCursor(cursorId)
	if err != nil {
		return err
	}
	_ = _SetCursor(cursorHandle) // returns prevCursorH
	win.cursors.currentId = cursorId
	return nil
}

func (win *Window) loadCursor(cursorId int) (windows.Handle, error) {
	cursorHandle, ok := win.cursors.cache[cursorId]
	if !ok {
		ch, err := win.loadCursor2(cursorId)
		if err != nil {
			return 0, err
		}
		win.cursors.cache[cursorId] = ch
		cursorHandle = ch
	}
	return cursorHandle, nil
}

func (win *Window) loadCursor2(c int) (windows.Handle, error) {
	cursorId := packLowHigh(uint16(c), 0)

	// TODO: failing on windows 10 with instance=0
	//cursor, err := _LoadImageW(
	//	0, // use nil instance not the win.instance (won't find resource)
	//	uintptr(cursorId),
	//	_IMAGE_CURSOR,
	//	0, 0, // w,h: use zeros with _LR_DEFAULTSIZE
	//	_LR_DEFAULTSIZE)

	//return 0, nil

	// Alternative func superseeded by LoadImageW(...)
	//cursor, err := _LoadCursorW(win.instance, cursorId)
	cursor, err := _LoadCursorW(0, cursorId)

	if err != nil {
		return 0, fmt.Errorf("loadimage: %v: %v\n", c, err)
	}
	return cursor, nil
}

//----------

func (win *Window) QueryPointer() (image.Point, error) {
	app := newAppMsg(&appQueryPointer{})
	if err := win.PostEvent(app); err != nil {
		return image.ZP, err
	}
	res := <-app.Reply
	switch t := res.(type) {
	case image.Point:
		return t, nil
	case error:
		return image.ZP, nil
	default:
		panic(res)
	}
}

func (win *Window) ostQueryPointer() (image.Point, error) {
	csp, err := win.cursorScreenPos()
	if err != nil {
		return image.ZP, err
	}
	return win.screenToWindowPoint(csp)
}

//----------

func (win *Window) cursorScreenPos() (image.Point, error) {
	cp := _Point{}
	if !_GetCursorPos(&cp) {
		return image.ZP, fmt.Errorf("getcursorpos: false")
	}
	return cp.ToImagePoint(), nil
}

//----------

func (win *Window) screenToWindowPoint(sp image.Point) (image.Point, error) {
	wsp, err := win.windowScreenPos()
	if err != nil {
		return image.ZP, err
	}
	return sp.Sub(wsp), nil
}

func (win *Window) windowScreenPos() (image.Point, error) {
	// NOTE: returns window area (need client area)
	//wr := _Rect{}
	//if !_GetWindowRect(win.hwnd, &wr) {
	//	return image.ZP, fmt.Errorf("getwindowrect: false")
	//}
	//return wr.ToImageRectangle().Min, nil

	// NOTE: works, but apparently has issues on right-to-left systems...
	//p := _Point{0, 0}
	//if !_ClientToScreen(win.hwnd, &p) {
	//	return image.ZP, fmt.Errorf("clienttoscreen: false")
	//}
	//return p.ToImagePoint(), nil

	p := _Point{0, 0}
	_ = _MapWindowPoints(win.hwnd, 0, &p, 1)
	return p.ToImagePoint(), nil
}

//----------

func (win *Window) WarpPointer(p image.Point) {
	app := newAppMsg(&appWarpPointer{p: p})
	if err := win.PostEvent(app); err != nil {
		log.Println(err)
		return
	}
	if err := appCheckError(<-app.Reply); err != nil {
		log.Println(err)
	}
}

func (win *Window) ostWarpPointer(p image.Point) error {
	wsp, err := win.windowScreenPos()
	if err != nil {
		return err
	}
	p2 := p.Add(wsp)
	if !_SetCursorPos(int32(p2.X), int32(p2.Y)) {
		return fmt.Errorf("setcursorpos: false")
	}
	return nil
}

//----------

func (win *Window) GetCPPaste(idx event.CopyPasteIndex, fn func(string, error)) {
	if idx == event.CPIClipboard {
		app := newAppMsg(&appGetClipboardData{})
		if err := win.PostEvent(app); err != nil {
			fn("", err)
			return
		}
		res := <-app.Reply
		switch t := res.(type) {
		case string:
			fn(t, nil)
		case error:
			fn("", t)
		default:
			panic(res)
		}
	}
}
func (win *Window) ostGetClipboardData() (string, error) {
	if !_OpenClipboard(0) {
		return "", fmt.Errorf("openclipboard: false")
	}
	defer _CloseClipboard()

	h, err := _GetClipboardData(_CF_UNICODETEXT)
	if err != nil {
		return "", fmt.Errorf("getclipboarddata: %v", err)
	}

	ptr, err := _GlobalLock(h)
	if err != nil {
		return "", fmt.Errorf("getclipboarddata: globallock: %v", err)
	}
	defer _GlobalUnlock(h)

	// TODO: improve this, could crash
	// translate ptr to []uint16
	sh := reflect.SliceHeader{Data: ptr, Len: 5000, Cap: 5000}
	buf := *(*[]uint16)(unsafe.Pointer(&sh))
	// find string end (nil terminated)
	for i, v := range buf {
		if v == 0 {
			buf = buf[:i]
			break
		}
	}

	s := windows.UTF16ToString(buf)
	return s, nil
}

//----------

func (win *Window) SetCPCopy(idx event.CopyPasteIndex, s string) error {
	if idx == event.CPIClipboard {
		app := newAppMsg(&appSetClipboardData{s: s})
		if err := win.PostEvent(app); err != nil {
			return err
		}
		return appCheckError(<-app.Reply)
	}
	return nil
}

func (win *Window) ostSetClipboardData(s string) error {
	if !_OpenClipboard(0) {
		return fmt.Errorf("openclipboard: false")
	}
	defer _CloseClipboard()

	// translate string to utf16 (will include nil termination)
	sl, err := windows.UTF16FromString(s)
	if err != nil {
		return err
	}
	// allocate memory for the clipboard
	unit := int(unsafe.Sizeof(uint16(0)))
	size := len(sl) * unit
	h, err := _GlobalAlloc(_GMEM_MOVEABLE, uintptr(size))
	if err != nil {
		return err
	}
	// get handle pointer
	ptr, err := _GlobalLock(h)
	if err != nil {
		return fmt.Errorf("getclipboarddata: globallock: %v", err)
	}
	defer _GlobalUnlock(h)
	// mask pointer to slice
	sh := reflect.SliceHeader{Data: ptr, Len: len(sl), Cap: len(sl)}
	cbBuf := *(*[]uint16)(unsafe.Pointer(&sh))

	// copy data to the allocated memory
	copy(cbBuf, sl)

	if _, err := _SetClipboardData(_CF_UNICODETEXT, h); err != nil {
		return fmt.Errorf("setclipboarddata: %v", err)
	}
	return nil
}

//----------

func (win *Window) SetWindowName(str string) {
	// TODO
}

//----------

// Called from app.
func (win *Window) Close() error {
	app := newAppMsg(&appClose{})
	if err := win.PostEvent(app); err != nil {
		return err
	}
	if err := appCheckError(<-app.Reply); err != nil {
		return err
	}
	<-win.evLoopEnd
	return nil
}

//----------

// Messages sent to the ost loop (operating system thread).

type appMsg struct {
	Event interface{}
	Reply chan interface{}
}

func newAppMsg(ev interface{}) *appMsg {
	return &appMsg{Event: ev, Reply: make(chan interface{}, 1)}
}

func appCheckError(res interface{}) error {
	switch t := res.(type) {
	case nil:
		return nil
	case error:
		return t
	default:
		panic(res)
	}
}

type appClose struct{}
type appPutImage struct{ r image.Rectangle }
type appResizeImage struct{ r image.Rectangle }
type appSetCursor struct{ Cursor event.Cursor }
type appQueryPointer struct{}
type appWarpPointer struct{ p image.Point }
type appGetClipboardData struct{}
type appSetClipboardData struct{ s string }

//----------

func paramToPoint(param uint32) image.Point {
	x, y := unpackLowHigh(param)
	return image.Point{X: x, Y: y}
}

//----------

func vkeyRune(vkey uint32, kstate *[256]byte) (rune, bool) {
	scanCode := _MapVirtualKeyW(vkey, _MAPVK_VK_TO_VSC)
	wFlags := uint32(0) // 2: windows 10 no keyb state?
	var res uint32      // TODO: low/high byte order?
	resPtr := (*uint16)(unsafe.Pointer(&res))
	v := _ToUnicode(vkey, scanCode, kstate, resPtr, 2, wFlags)
	isDeadKey := v == -1
	return rune(res), isDeadKey
}

//----------
