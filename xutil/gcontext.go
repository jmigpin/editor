package xutil

import "github.com/BurntSushi/xgb/xproto"

type GContext struct {
	xu     *XUtil
	mask   uint32
	values []uint32
	Ctx    xproto.Gcontext
}

// Options should be set before calling this (Ex: foreground, background)
func (ctx *GContext) Create() error {
	ctx0, err := xproto.NewGcontextId(ctx.xu.Conn)
	if err != nil {
		return err
	}
	ctx.Ctx = ctx0
	_ = xproto.CreateGC(ctx.xu.Conn, ctx.Ctx, ctx.xu.Drawable, ctx.mask, ctx.values)
	return nil
}
func (ctx *GContext) Change() {
	_ = xproto.ChangeGC(ctx.xu.Conn, ctx.Ctx, ctx.mask, ctx.values)
}
func (ctx *GContext) PutImageData(x, y, w, h int, data []uint8) {
	_ = xproto.PutImage(
		ctx.xu.Conn,
		xproto.ImageFormatZPixmap,
		ctx.xu.Drawable,
		ctx.Ctx,
		uint16(w), uint16(h), int16(x), int16(y),
		0, // left pad
		ctx.xu.Screen.RootDepth,
		data)
}

//func (ctx *GContext) Foreground(r, g, b uint16) error {
//cookie := ctx.xu.AllocColor(r, g, b)
//rep, err := cookie.Reply()
//if err != nil {
//return err
//}
//ctx.mask |= xproto.GcForeground
//ctx.values = append(ctx.values, rep.Pixel)
//return nil
//}

//func (ctx *GContext) FillRectangles(rects []xproto.Rectangle) {
//cookie := xproto.PolyFillRectangle(ctx.xu.Conn, ctx.xu.Drawable, ctx.ctx, rects)
//_ = cookie // default is unchecked - ignore error
//}
//func (ctx *GContext) Points(points []xproto.Point) {
//cookie := xproto.PolyPoint(ctx.xu.Conn, xproto.CoordModeOrigin,
//ctx.xu.Drawable, ctx.ctx, points)
//_ = cookie // default is unchecked - ignore error
//}

// Closing the returned channel cancels the remaining draw tiles.
//func (ctx *GContext) PutImageTiles(x, y int, img *image.RGBA) chan struct{} {
//cancel := make(chan struct{})
//ctx.putImageTiles2(x, y, img, cancel)
//return cancel
//}
//func (ctx *GContext) putImageTiles2(x, y int, img *image.RGBA, cancel chan struct{}) {
//chunk := 64
//bo := img.Bounds()
//for yi := bo.Min.Y; yi < bo.Max.Y; yi += chunk {
//my := yi + chunk
//for xi := bo.Min.X; xi < bo.Max.X; xi += chunk {
//mx := xi + chunk
//r := image.Rect(xi, yi, mx, my)
//img2 := RGBASubImage(img, &r) // contains intersection
//ctx.putImageTile(x+xi-bo.Min.X, y+yi-bo.Min.Y, img2, cancel)
//}
//}
//}
//func (ctx *GContext) putImageTile(x, y int, img *image.RGBA, cancel chan struct{}) {
//bo := img.Bounds()
//width := bo.Dx()
//height := bo.Dy()
//if chanIsClosed(cancel) {
////println("canceled tile 1")
//return
//}
//data := RGBADataForX(img)
//if chanIsClosed(cancel) {
////println("canceled tile 2")
//return
//}
//ctx.PutImageData(x, y, width, height, data)
//}
//func chanIsClosed(ch chan struct{}) bool {
//select {
//case <-ch:
//return true
//default:
//return false
//}
//}

//func (ctx *GContext) PutImageTiles(x, y int, img image.Image, r*image.Rectangle) {
//chunk := 64
//for yi := r.Min.Y; yi < r.Max.Y; yi += chunk {
//my := yi + chunk
//for xi := r.Min.X; xi < r.Max.X; xi += chunk {
//mx := xi + chunk
//r2 := image.Rect(xi, yi, mx, my).Intersect(*r)
//data := RGBADataForX2(img, &r2)
//w, h := r2.Dx(), r2.Dy()
//ctx.PutImageData(x, y, w, h, data)
//}
//}
//}

//func (ctx *GContext) PutImageTiles_(x, y int, img *image.RGBA, cancel chan struct{}) {
//var wg sync.WaitGroup
//chunk := 64
//bo := img.Bounds()
//for yi := bo.Min.Y; yi < bo.Max.Y; yi += chunk {
//my := yi + chunk
//for xi := bo.Min.X; xi < bo.Max.X; xi += chunk {
//mx := xi + chunk
//r := image.Rect(xi, yi, mx, my).Intersect(bo)
//wg.Add(1)
//go ctx.putImageTile(x+xi-bo.Min.X, y+yi-bo.Min.Y, img, &r, cancel, &wg)
//}
//}
//wg.Wait()
//}
//func (ctx *GContext) putImageTile(x, y int, img *image.RGBA, rect *image.Rectangle, cancel chan struct{}, wg *sync.WaitGroup) {
//defer wg.Done()

//data := RGBADataForX(img, rect)

//select {
//case <-cancel:
//return
//default:
//}

//w, h := rect.Dx(), rect.Dy()
//ctx.PutImageData(x, y, w, h, data)
//}

// put any image
//func (ctx *GContext) PutImage(x, y int, img *image.RGBA) {
//bo := img.Bounds()
//width := bo.Dx()
//height := bo.Dy()
//data := RGBADataForX(img)
//ctx.PutImageData(x, y, width, height, data)
//}
