package godebug

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"

	"github.com/jmigpin/editor/util/parseutil"
)

//----------

func TestAnnotator1(t *testing.T) {
	inout := []string{
		`f1(1)`,
		`Σ0 := Σ.IV(1)
	        Σ.Line(0, 0, 27, Σ.ICe("f1", Σ0))
	        Σ1 := Σ.IC("f1", nil, Σ0)
	        f1(1)
	        Σ.Line(0, 0, 28, Σ1)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator2(t *testing.T) {
	inout := []string{
		`f1(a,1,nil,"s")`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(nil)
	        Σ3 := Σ.IV("s")
	        Σ.Line(0, 0, 37, Σ.ICe("f1", Σ0, Σ1, Σ2, Σ3))
	        Σ4 := Σ.IC("f1", nil, Σ0, Σ1, Σ2, Σ3)
	        f1(a, 1, nil, "s")
	        Σ.Line(0, 0, 38, Σ4)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator3(t *testing.T) {
	inout := []string{
		`f1(f2(a,f3()))`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 34, Σ.ICe("f3"))
	        Σ1 := f3()
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f3", Σ2)
	        Σ.Line(0, 0, 35, Σ.ICe("f2", Σ0, Σ3))
	        Σ4 := f2(a, Σ1)
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("f2", Σ5, Σ0, Σ3)
	        Σ.Line(0, 0, 36, Σ.ICe("f1", Σ6))
	        Σ7 := Σ.IC("f1", nil, Σ6)
	        f1(Σ4)
	        Σ.Line(0, 0, 37, Σ7)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator4(t *testing.T) {
	inout := []string{
		`f1(1 * 200)`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(200)
	        Σ2 := Σ.IV(1 * 200)
	        Σ3 := Σ.IB(Σ2, 14, Σ0, Σ1)
	        Σ.Line(0, 0, 33, Σ.ICe("f1", Σ3))
	        Σ4 := Σ.IC("f1", nil, Σ3)
	        f1(1 * 200)
	        Σ.Line(0, 0, 34, Σ4)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator5(t *testing.T) {
	inout := []string{
		`f1(1 * 200 * f2())`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(200)
	        Σ2 := Σ.IV(1 * 200)
	        Σ3 := Σ.IB(Σ2, 14, Σ0, Σ1)
	        Σ.Line(0, 0, 39, Σ.ICe("f2"))
	        Σ4 := f2()
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("f2", Σ5)
	        Σ7 := 1 * 200 * Σ4
	        Σ8 := Σ.IV(Σ7)
	        Σ9 := Σ.IB(Σ8, 14, Σ3, Σ6)
	        Σ.Line(0, 0, 40, Σ.ICe("f1", Σ9))
	        Σ10 := Σ.IC("f1", nil, Σ9)
	        f1(Σ7)
	        Σ.Line(0, 0, 41, Σ10)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator6(t *testing.T) {
	inout := []string{
		`f1(f2(&a), f3(&a))`,
		`Σ0 := Σ.IV(a)
	        Σ1 := &a
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 17, Σ0)
	        Σ.Line(0, 0, 31, Σ.ICe("f2", Σ3))
	        Σ4 := f2(Σ1)
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("f2", Σ5, Σ3)
	        Σ7 := Σ.IV(a)
	        Σ8 := &a
	        Σ9 := Σ.IV(Σ8)
	        Σ10 := Σ.IU(Σ9, 17, Σ7)
	        Σ.Line(0, 0, 39, Σ.ICe("f3", Σ10))
	        Σ11 := f3(Σ8)
	        Σ12 := Σ.IV(Σ11)
	        Σ13 := Σ.IC("f3", Σ12, Σ10)
	        Σ.Line(0, 0, 40, Σ.ICe("f1", Σ6, Σ13))
	        Σ14 := Σ.IC("f1", nil, Σ6, Σ13)
	        f1(Σ4, Σ11)
	        Σ.Line(0, 0, 41, Σ14)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator7(t *testing.T) {
	inout := []string{
		`f1(a, func(){f2()})`,
		`Σ0 := Σ.IV(a)
	        Σ1 := func() { Σ.Line(0, 0, 39, Σ.ICe("f2")); Σ2 := Σ.IC("f2", nil); f2(); Σ.Line(0, 0, 40, Σ2) }
	        Σ3 := Σ.IV(Σ1)
	        Σ.Line(0, 1, 41, Σ.ICe("f1", Σ0, Σ3))
	        Σ4 := Σ.IC("f1", nil, Σ0, Σ3)
	        f1(a, Σ1)
	        Σ.Line(0, 1, 42, Σ4)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator8(t *testing.T) {
	inout := []string{
		`a:=1`,
		`Σ0 := Σ.IV(1)
	        a := 1
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator9(t *testing.T) {
	inout := []string{
		`a,b:=1,c`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(c)
	        a, b := 1, c
	        Σ2 := Σ.IV(a)
	        Σ3 := Σ.IV(b)
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ2, Σ3), Σ.IL(Σ0, Σ1)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator10(t *testing.T) {
	inout := []string{
		`a,b,_:=1,c,d`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(c)
	        Σ2 := Σ.IV(d)
	        a, b, _ := 1, c, d
	        Σ3 := Σ.IV(a)
	        Σ4 := Σ.IV(b)
	        Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ3, Σ4, Σ.IAn()), Σ.IL(Σ0, Σ1, Σ2)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator11(t *testing.T) {
	inout := []string{
		`a=1`,
		`Σ0 := Σ.IV(1)
	        a = 1
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator12(t *testing.T) {
	inout := []string{
		`_=1`,
		`Σ0 := Σ.IV(1)
	        _ = 1
	        Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ.IAn()), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator13(t *testing.T) {
	inout := []string{
		`a,_:=1,"s"`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV("s")
	        a, _ := 1, "s"
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ2, Σ.IAn()), Σ.IL(Σ0, Σ1)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator14(t *testing.T) {
	inout := []string{
		`a,_=1,"s"`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV("s")
	        a, _ = 1, "s"
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 32, Σ.IA(Σ.IL(Σ2, Σ.IAn()), Σ.IL(Σ0, Σ1)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator15(t *testing.T) {
	inout := []string{
		`a.b = true`,
		`Σ0 := Σ.IV(true)
	        a.b = true
	        Σ1 := Σ.IV(a.b)
	        Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator16(t *testing.T) {
	inout := []string{
		`i, _ = a.b(c)`,
		`Σ0 := Σ.IV(c)
	        Σ.Line(0, 0, 35, Σ.ICe("b", Σ0))
	        Σ1, Σ2 := a.b(c)
	        Σ3 := Σ.IL(Σ.IV(Σ1), Σ.IV(Σ2))
	        Σ4 := Σ.IC("b", Σ3, Σ0)
	        i, _ = Σ1, Σ2
	        Σ5 := Σ.IV(i)
	        Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ5, Σ.IAn()), Σ.IL(Σ4)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator16a(t *testing.T) {
	inout := []string{
		`i, _ = a().b(c)`,
		`Σ.Line(0, 0, 32, Σ.ICe("a"))
	        Σ0 := a()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("a", Σ1)
	        Σ.Line(0, 0, 35, Σ2)
	        Σ3 := Σ.IV(c)
	        Σ.Line(0, 0, 37, Σ.ICe("b", Σ3))
	        Σ4, Σ5 := Σ0.b(c)
	        Σ6 := Σ.IL(Σ.IV(Σ4), Σ.IV(Σ5))
	        Σ7 := Σ.IC("b", Σ6, Σ3)
	        i, _ = Σ4, Σ5
	        Σ8 := Σ.IV(i)
	        Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ8, Σ.IAn()), Σ.IL(Σ7)))
	        `,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator17(t *testing.T) {
	inout := []string{
		`c:=f1()`,
		`Σ.Line(0, 0, 29, Σ.ICe("f1"))
	        Σ0 := f1()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f1", Σ1)
	        c := Σ0
	        Σ3 := Σ.IV(c)
	        Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator18(t *testing.T) {
	inout := []string{
		`_, b := c.d(e, f())`,
		`Σ0 := Σ.IV(e)
	        Σ.Line(0, 0, 40, Σ.ICe("f"))
	        Σ1 := f()
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f", Σ2)
	        Σ.Line(0, 0, 41, Σ.ICe("d", Σ0, Σ3))
	        Σ4, Σ5 := c.d(e, Σ1)
	        Σ6 := Σ.IL(Σ.IV(Σ4), Σ.IV(Σ5))
	        Σ7 := Σ.IC("d", Σ6, Σ0, Σ3)
	        _, b := Σ4, Σ5
	        Σ8 := Σ.IV(b)
	        Σ.Line(0, 0, 42, Σ.IA(Σ.IL(Σ.IAn(), Σ8), Σ.IL(Σ7)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator19(t *testing.T) {
	inout := []string{
		`a, _ = 1, c`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(c)
	        a, _ = 1, c
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 34, Σ.IA(Σ.IL(Σ2, Σ.IAn()), Σ.IL(Σ0, Σ1)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator20(t *testing.T) {
	inout := []string{
		`a, _ = c.d(1, f(u), 'c', nil)`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(u)
	        Σ.Line(0, 0, 40, Σ.ICe("f", Σ1))
	        Σ2 := f(u)
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IC("f", Σ3, Σ1)
	        Σ5 := Σ.IV('c')
	        Σ6 := Σ.IV(nil)
	        Σ.Line(0, 0, 51, Σ.ICe("d", Σ0, Σ4, Σ5, Σ6))
	        Σ7, Σ8 := c.d(1, Σ2, 'c', nil)
	        Σ9 := Σ.IL(Σ.IV(Σ7), Σ.IV(Σ8))
	        Σ10 := Σ.IC("d", Σ9, Σ0, Σ4, Σ5, Σ6)
	        a, _ = Σ7, Σ8
	        Σ11 := Σ.IV(a)
	        Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ11, Σ.IAn()), Σ.IL(Σ10)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator21(t *testing.T) {
	inout := []string{
		`a, b = f1(c, "s")`,
		`Σ0 := Σ.IV(c)
	        Σ1 := Σ.IV("s")
	        Σ.Line(0, 0, 39, Σ.ICe("f1", Σ0, Σ1))
	        Σ2, Σ3 := f1(c, "s")
	        Σ4 := Σ.IL(Σ.IV(Σ2), Σ.IV(Σ3))
	        Σ5 := Σ.IC("f1", Σ4, Σ0, Σ1)
	        a, b = Σ2, Σ3
	        Σ6 := Σ.IV(a)
	        Σ7 := Σ.IV(b)
	        Σ.Line(0, 0, 40, Σ.IA(Σ.IL(Σ6, Σ7), Σ.IL(Σ5)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator22(t *testing.T) {
	inout := []string{
		`a=f1(f2())`,
		`Σ.Line(0, 0, 31, Σ.ICe("f2"))
	        Σ0 := f2()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f2", Σ1)
	        Σ.Line(0, 0, 32, Σ.ICe("f1", Σ2))
	        Σ3 := f1(Σ0)
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IC("f1", Σ4, Σ2)
	        a = Σ3
	        Σ6 := Σ.IV(a)
	        Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ6), Σ.IL(Σ5)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator23(t *testing.T) {
	inout := []string{
		`a:=path[f1(d)]`,
		`Σ0 := Σ.IV(d)
	        Σ.Line(0, 0, 35, Σ.ICe("f1", Σ0))
	        Σ1 := f1(d)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f1", Σ2, Σ0)
	        Σ4 := path[Σ1]
	        Σ5 := Σ.IV(Σ4)
	        a := Σ4
	        Σ6 := Σ.IV(a)
	        Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ6), Σ.IL(Σ.II(Σ5, nil, Σ3))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator24(t *testing.T) {
	inout := []string{
		`a,b:=c-d, e+f`,
		`Σ0 := Σ.IV(c)
	        Σ1 := Σ.IV(d)
	        Σ2 := Σ.IV(c - d)
	        Σ3 := Σ.IB(Σ2, 13, Σ0, Σ1)
	        Σ4 := Σ.IV(e)
	        Σ5 := Σ.IV(f)
	        Σ6 := Σ.IV(e + f)
	        Σ7 := Σ.IB(Σ6, 12, Σ4, Σ5)
	        a, b := c-d, e+f
	        Σ8 := Σ.IV(a)
	        Σ9 := Σ.IV(b)
	        Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ8, Σ9), Σ.IL(Σ3, Σ7)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator25(t *testing.T) {
	inout := []string{
		`a[i] = b`,
		`Σ0 := Σ.IV(b)
	        a[i] = b
	        Σ1 := Σ.IV(i)
	        Σ2 := Σ.IV(a[i])
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(Σ2, nil, Σ1)), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator26(t *testing.T) {
	inout := []string{
		`a:=b[c]`,
		`Σ0 := Σ.IV(c)
	        Σ1 := b[c]
	        Σ2 := Σ.IV(Σ1)
	        a := Σ1
	        Σ3 := Σ.IV(a)
	        Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ.II(Σ2, nil, Σ0))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator27(t *testing.T) {
	inout := []string{
		`s = s[:i] + "a"`,
		`Σ0 := Σ.IV(i)
	        Σ1 := s[:i]
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IV("a")
	        Σ4 := Σ1 + "a"
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IB(Σ5, 12, Σ.II2(Σ2, nil, nil, Σ0, nil, false), Σ3)
	        s = Σ4
	        Σ7 := Σ.IV(s)
	        Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ7), Σ.IL(Σ6)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator28(t *testing.T) {
	inout := []string{
		`b[1] = u[:2]`,
		`Σ0 := Σ.IV(2)
	        Σ1 := u[:2]
	        Σ2 := Σ.IV(Σ1)
	        b[1] = Σ1
	        Σ3 := Σ.IV(1)
	        Σ4 := Σ.IV(b[1])
	        Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.II(Σ4, nil, Σ3)), Σ.IL(Σ.II2(Σ2, nil, nil, Σ0, nil, false))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator29(t *testing.T) {
	inout := []string{
		`u[f2()] = u[:2]`,
		`Σ0 := Σ.IV(2)
	        Σ1 := u[:2]
	        Σ2 := Σ.IV(Σ1)
	        Σ.Line(0, 0, 28, Σ.ICe("f2"))
	        Σ3 := f2()
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IC("f2", Σ4)
	        u[Σ3] = Σ1
	        Σ6 := Σ.IV(u[Σ3])
	        Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(Σ6, nil, Σ5)), Σ.IL(Σ.II2(Σ2, nil, nil, Σ0, nil, false))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator30(t *testing.T) {
	inout := []string{
		`a:=s[:]`,
		`Σ0 := s[:]
	        Σ1 := Σ.IV(Σ0)
	        a := Σ0
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ2), Σ.IL(Σ.II2(Σ1, nil, nil, nil, nil, false))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator31(t *testing.T) {
	inout := []string{
		`u[1+a] = u[1+b]`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(b)
	        Σ2 := Σ.IV(1 + b)
	        Σ3 := Σ.IB(Σ2, 12, Σ0, Σ1)
	        Σ4 := u[1+b]
	        Σ5 := Σ.IV(Σ4)
	        u[1+a] = Σ4
	        Σ6 := Σ.IV(1)
	        Σ7 := Σ.IV(a)
	        Σ8 := Σ.IV(1 + a)
	        Σ9 := Σ.IB(Σ8, 12, Σ6, Σ7)
	        Σ10 := Σ.IV(u[1+a])
	        Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(Σ10, nil, Σ9)), Σ.IL(Σ.II(Σ5, nil, Σ3))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator32(t *testing.T) {
	inout := []string{
		`p[1+a]=1`,
		`Σ0 := Σ.IV(1)
	        p[1+a] = 1
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(a)
	        Σ3 := Σ.IV(1 + a)
	        Σ4 := Σ.IB(Σ3, 12, Σ1, Σ2)
	        Σ5 := Σ.IV(p[1+a])
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(Σ5, nil, Σ4)), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator33(t *testing.T) {
	inout := []string{
		`a:=&Struct1{A:f1(u), B:2}`,
		`Σ0 := Σ.IV(u)
	        Σ.Line(0, 0, 41, Σ.ICe("f1", Σ0))
	        Σ1 := f1(u)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f1", Σ2, Σ0)
	        Σ4 := Σ.IV(2)
	        Σ5 := &Struct1{A: Σ1, B: 2}
	        Σ6 := Σ.IV(Σ5)
	        Σ7 := Σ.IU(Σ6, 17, Σ.ILit(Σ.IKV(Σ.IVs("A"), Σ3), Σ.IKV(Σ.IVs("B"), Σ4)))
	        a := Σ5
	        Σ8 := Σ.IV(a)
	        Σ.Line(0, 0, 48, Σ.IA(Σ.IL(Σ8), Σ.IL(Σ7)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator34(t *testing.T) {
	inout := []string{
		`a += f3(a + 1)`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(a + 1)
	        Σ3 := Σ.IB(Σ2, 12, Σ0, Σ1)
	        Σ.Line(0, 0, 36, Σ.ICe("f3", Σ3))
	        Σ4 := f3(a + 1)
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("f3", Σ5, Σ3)
	        a += Σ4
	        Σ7 := Σ.IV(a)
	        Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ7), Σ.IL(Σ6)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator35(t *testing.T) {
	inout := []string{
		`a := &c[i]`,
		`Σ0 := Σ.IV(i)
	        Σ1 := Σ.IV(c[i])
	        Σ2 := &c[i]
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IU(Σ3, 17, Σ.II(Σ1, nil, Σ0))
	        a := Σ2
	        Σ5 := Σ.IV(a)
	        Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ5), Σ.IL(Σ4)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator36(t *testing.T) {
	inout := []string{
		`switch x.(type){}`,
		`Σ.Line(0, 0, 38, Σ.IVt(x))
	        switch x.(type) {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator36a(t *testing.T) {
	inout := []string{
		`switch f().(type){}`,
		`Σ.Line(0, 0, 32, Σ.ICe("f"))
	        Σ0 := f()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f", Σ1)
	        Σ.Line(0, 0, 40, Σ.ITA(Σ2, Σ.IVt(Σ0)))
	        switch Σ0.(type) {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator36b(t *testing.T) {
	inout := []string{
		`switch (<-x).(type){}`,
		`Σ0 := Σ.IV(x)
	        Σ.Line(0, 0, 34, Σ.IUe(36, Σ0))
	        Σ1 := <-x
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 36, Σ0)
	        Σ.Line(0, 0, 42, Σ.ITA(Σ.IP(Σ3), Σ.IVt((Σ1))))
	        switch (Σ1).(type) {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator37(t *testing.T) {
	inout := []string{
		`switch b:=x.(type){}`,
		`Σ.Line(0, 0, 41, Σ.IL(Σ.IVt(x)))
	        switch b := x.(type) {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator38(t *testing.T) {
	inout := []string{
		`switch a>b {}`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(b)
	        Σ2 := a > b
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IB(Σ3, 41, Σ0, Σ1)
	        Σ.Line(0, 0, 33, Σ4)
	        switch Σ2 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator39(t *testing.T) {
	inout := []string{
		`switch a {}`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 31, Σ0)
	        switch a {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator40(t *testing.T) {
	inout := []string{
		`b:=1
		switch a:=f1(u); a {}`,
		`Σ0 := Σ.IV(1)
	        b := 1
	        Σ1 := Σ.IV(b)
	        Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        {
	        Σ2 := Σ.IV(u)
	        Σ.Line(0, 1, 42, Σ.ICe("f1", Σ2))
	        Σ3 := f1(u)
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IC("f1", Σ4, Σ2)
	        a := Σ3
	        Σ6 := Σ.IV(a)
	        Σ.Line(0, 1, 43, Σ.IA(Σ.IL(Σ6), Σ.IL(Σ5)))
	        Σ7 := Σ.IV(a)
	        Σ.Line(0, 2, 46, Σ7)
	        switch a {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator40a(t *testing.T) {
	inout := []string{
		`switch f1(u) {}`,
		`Σ0 := Σ.IV(u)
	        Σ.Line(0, 0, 34, Σ.ICe("f1", Σ0))
	        Σ1 := f1(u)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f1", Σ2, Σ0)
	        Σ.Line(0, 0, 35, Σ3)
	        switch Σ1 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator41(t *testing.T) {
	inout := []string{
		`if a {}`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ0)
	        if a {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator42(t *testing.T) {
	inout := []string{
		`if a {b=1}`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ0)
	        if a {
	        Σ1 := Σ.IV(1)
	        b = 1
	        Σ2 := Σ.IV(b)
	        Σ.Line(0, 1, 32, Σ.IA(Σ.IL(Σ2), Σ.IL(Σ1)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator43(t *testing.T) {
	inout := []string{
		`if c:=f1(); c>2{}`,
		` {
	        Σ.Line(0, 0, 32, Σ.ICe("f1"))
	        Σ0 := f1()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f1", Σ1)
	        c := Σ0
	        Σ3 := Σ.IV(c)
	        Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))
	        Σ4 := Σ.IV(c)
	        Σ5 := Σ.IV(2)
	        Σ6 := c > 2
	        Σ7 := Σ.IV(Σ6)
	        Σ8 := Σ.IB(Σ7, 41, Σ4, Σ5)
	        Σ.Line(0, 1, 38, Σ8)
	        if Σ6 {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator44(t *testing.T) {
	inout := []string{
		`if a{}else if b{}`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ0)
	        if a {
	        } else {
	        Σ1 := Σ.IV(b)
	        Σ.Line(0, 1, 38, Σ1)
	        if b {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator45(t *testing.T) {
	inout := []string{
		`if v > f1(f2(v)) {}`,
		`Σ0 := Σ.IV(v)
	        Σ1 := Σ.IV(v)
	        Σ.Line(0, 0, 37, Σ.ICe("f2", Σ1))
	        Σ2 := f2(v)
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IC("f2", Σ3, Σ1)
	        Σ.Line(0, 0, 38, Σ.ICe("f1", Σ4))
	        Σ5 := f1(Σ2)
	        Σ6 := Σ.IV(Σ5)
	        Σ7 := Σ.IC("f1", Σ6, Σ4)
	        Σ8 := v > Σ5
	        Σ9 := Σ.IV(Σ8)
	        Σ10 := Σ.IB(Σ9, 41, Σ0, Σ7)
	        Σ.Line(0, 0, 39, Σ10)
	        if Σ8 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator46_0(t *testing.T) {
	inout := []string{
		`if !a {}`,
		`Σ0 := Σ.IV(a)
	        Σ1 := !a
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 43, Σ0)
	        Σ.Line(0, 0, 28, Σ3)
	        if Σ1 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator46(t *testing.T) {
	inout := []string{
		`if n := f1("s1"); !f2(n, "s2") {}`,
		`{
	        Σ0 := Σ.IV("s1")
	        Σ.Line(0, 0, 38, Σ.ICe("f1", Σ0))
	        Σ1 := f1("s1")
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f1", Σ2, Σ0)
	        n := Σ1
	        Σ4 := Σ.IV(n)
	        Σ.Line(0, 0, 39, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))
	        Σ5 := Σ.IV(n)
	        Σ6 := Σ.IV("s2")
	        Σ.Line(0, 1, 52, Σ.ICe("f2", Σ5, Σ6))
	        Σ7 := f2(n, "s2")
	        Σ8 := Σ.IV(Σ7)
	        Σ9 := Σ.IC("f2", Σ8, Σ5, Σ6)
	        Σ10 := !Σ7
	        Σ11 := Σ.IV(Σ10)
	        Σ12 := Σ.IU(Σ11, 43, Σ9)
	        Σ.Line(0, 1, 53, Σ12)
	        if Σ10 {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator47(t *testing.T) {
	inout := []string{
		`if nil!=nil{}`,
		`Σ0 := Σ.IV(nil)
	        Σ1 := Σ.IV(nil)
	        Σ2 := nil != nil
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IB(Σ3, 44, Σ0, Σ1)
	        Σ.Line(0, 0, 34, Σ4)
	        if Σ2 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator48(t *testing.T) {
	inout := []string{
		`if a!=1 && b!=2 {}`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(1)
	        Σ2 := a != 1
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IB(Σ3, 44, Σ0, Σ1)
	        Σ5 := Σ.IVs("?")
	        Σ6 := Σ2
	        if Σ6 {
	        Σ7 := Σ.IV(b)
	        Σ8 := Σ.IV(2)
	        Σ9 := b != 2
	        Σ10 := Σ.IV(Σ9)
	        Σ11 := Σ.IB(Σ10, 44, Σ7, Σ8)
	        Σ6 = Σ9
	        Σ5 = Σ11
	        }
	        Σ12 := Σ.IB(Σ.IV(Σ6), 34, Σ4, Σ5)
	        Σ.Line(0, 0, 38, Σ12)
	        if Σ6 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator49(t *testing.T) {
	inout := []string{
		`if a || f2() {}`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IVs("?")
	        Σ2 := a
	        if !Σ2 {
	        Σ.Line(0, 0, 34, Σ.ICe("f2"))
	        Σ3 := f2()
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IC("f2", Σ4)
	        Σ2 = Σ3
	        Σ1 = Σ5
	        }
	        Σ6 := Σ.IB(Σ.IV(Σ2), 35, Σ0, Σ1)
	        Σ.Line(0, 0, 35, Σ6)
	        if Σ2 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator50(t *testing.T) {
	inout := []string{
		`for i:=0; ; i++{}`,
		`{
	        Σ0 := Σ.IV(0)
	        i := 0
	        Σ1 := Σ.IV(i)
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        for ; ; i++ {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator51(t *testing.T) {
	inout := []string{
		`for i:=0; i<10; i++{a=1}`,
		`{
	        Σ0 := Σ.IV(0)
	        i := 0
	        Σ1 := Σ.IV(i)
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        for ; ; i++ {
	        {
	        Σ2 := Σ.IV(i)
	        Σ3 := Σ.IV(10)
	        Σ4 := i < 10
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IB(Σ5, 40, Σ2, Σ3)
	        Σ.Line(0, 1, 37, Σ6)
	        if !Σ4 {
	        break
	        }
	        }
	        Σ7 := Σ.IV(1)
	        a = 1
	        Σ8 := Σ.IV(a)
	        Σ.Line(0, 2, 46, Σ.IA(Σ.IL(Σ8), Σ.IL(Σ7)))
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator52(t *testing.T) {
	inout := []string{
		`for a,b:=range f2() {}`,
		`Σ.Line(0, 0, 41, Σ.ICe("f2"))
	        Σ0 := f2()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f2", Σ1)
	        _ = Σ2
	        Σ3 := Σ.IVl(len(Σ0))
	        for a, b := range Σ0 {
	        {
	        Σ4 := Σ.IL(Σ3)
	        Σ5 := Σ.IL(Σ.IV(a), Σ.IV(b))
	        Σ.Line(0, 0, 42, Σ.IA(Σ5, Σ4))
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator53(t *testing.T) {
	inout := []string{
		`for a,_:=range f2() {}`,
		`Σ.Line(0, 0, 41, Σ.ICe("f2"))
	        Σ0 := f2()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f2", Σ1)
	        _ = Σ2
	        Σ3 := Σ.IVl(len(Σ0))
	        for a, _ := range Σ0 {
	        {
	        Σ4 := Σ.IL(Σ3)
	        Σ5 := Σ.IL(Σ.IV(a), Σ.IAn())
	        Σ.Line(0, 0, 42, Σ.IA(Σ5, Σ4))
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator54(t *testing.T) {
	inout := []string{
		`for _,_=range a {}`,
		` Σ0 := Σ.IV(a)
	        _ = Σ0
	        Σ1 := Σ.IVl(len(a))
	        for _, _ = range a {
	        {
	        Σ2 := Σ.IL(Σ1)
	        Σ3 := Σ.IL(Σ.IAn(), Σ.IAn())
	        Σ.Line(0, 0, 38, Σ.IA(Σ3, Σ2))
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator55(t *testing.T) {
	inout := []string{
		`for a,_=range c {}`,
		`Σ0 := Σ.IV(c)
	        _ = Σ0
	        Σ1 := Σ.IVl(len(c))
	        for a, _ = range c {
	        {
	        Σ2 := Σ.IL(Σ1)
	        Σ3 := Σ.IL(Σ.IV(a), Σ.IAn())
	        Σ.Line(0, 0, 38, Σ.IA(Σ3, Σ2))
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator56(t *testing.T) {
	inout := []string{
		`label1:
		a++
		goto label1`,
		`Σ.Line(0, 0, 23, Σ.ILa())
	        Σ0 := Σ.IV(a)
	        label1:
	        a++
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 1, 34, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        Σ.Line(0, 2, 35, Σ.IBr())
	        goto label1`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator56a(t *testing.T) {
	inout := []string{
		`label1:
		for i:=f();i<2;i++{break label1}`,
		`Σ.Line(0, 0, 23, Σ.ILa())
	        {
	        Σ.Line(0, 1, 40, Σ.ICe("f"))
	        Σ0 := f()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f", Σ1)
	        i := Σ0
	        Σ3 := Σ.IV(i)
	        Σ.Line(0, 1, 41, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))
	        Σ.Line(0, 2, 23, Σ.ILa())
	        label1:
	        for ; ; i++ {
	        {
	        Σ4 := Σ.IV(i)
	        Σ5 := Σ.IV(2)
	        Σ6 := i < 2
	        Σ7 := Σ.IV(Σ6)
	        Σ8 := Σ.IB(Σ7, 40, Σ4, Σ5)
	        Σ.Line(0, 3, 45, Σ8)
	        if !Σ6 {
	        break
	        }
	        }
	        Σ.Line(0, 4, 50, Σ.IBr())
	        break label1
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator56b(t *testing.T) {
	inout := []string{
		`label1:
		switch a:=f();a {}`,
		`Σ.Line(0, 0, 23, Σ.ILa())
	        {
	        Σ.Line(0, 1, 43, Σ.ICe("f"))
	        Σ0 := f()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("f", Σ1)
	        a := Σ0
	        Σ3 := Σ.IV(a)
	        Σ.Line(0, 1, 44, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))
	        Σ.Line(0, 2, 23, Σ.ILa())
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 3, 46, Σ4)
	        label1:
	        switch a {
	        }
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator56c(t *testing.T) {
	inout := []string{
		`label1:
		switch x.(type){}`,
		`Σ.Line(0, 0, 23, Σ.ILa())
	        Σ.Line(0, 1, 46, Σ.IVt(x))
	        label1:
	        switch x.(type) {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator57(t *testing.T) {
	inout := []string{
		`a,b:=1, func(a int)int{return 3}`,
		`Σ0 := Σ.IV(1)
	        Σ1 := func(a int) int {
	        {
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 45, Σ.IL(Σ2))
	        }
	        Σ3 := Σ.IV(3)
	        Σ.Line(0, 1, 54, Σ.IL(Σ3))
	        return 3
	        }
	        Σ4 := Σ.IV(Σ1)
	        a, b := 1, Σ1
	        Σ5 := Σ.IV(a)
	        Σ6 := Σ.IV(b)
	        Σ.Line(0, 2, 55, Σ.IA(Σ.IL(Σ5, Σ6), Σ.IL(Σ0, Σ4)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator58(t *testing.T) {
	inout := []string{
		`a:=make(map[string]string)`,
		`Σ0 := Σ.IVs("type")
	        Σ.Line(0, 0, 48, Σ.ICe("make", Σ0))
	        Σ1 := make(map[string]string)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("make", Σ2, Σ0)
	        a := Σ1
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 0, 49, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator59(t *testing.T) {
	inout := []string{
		`a:=map[string]string{"a":"b"}`,
		`Σ0 := Σ.IV("a")
	        Σ1 := Σ.IV("b")
	        a := map[string]string{"a": "b"}
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ2), Σ.IL(Σ.ILit(Σ.IKV(Σ0, Σ1)))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator60(t *testing.T) {
	inout := []string{
		`tbuf := new(bytes.Buffer)`,
		`Σ0 := Σ.IVs("type")
	        Σ.Line(0, 0, 47, Σ.ICe("new", Σ0))
	        Σ1 := new(bytes.Buffer)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("new", Σ2, Σ0)
	        tbuf := Σ1
	        Σ4 := Σ.IV(tbuf)
	        Σ.Line(0, 0, 48, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator61(t *testing.T) {
	inout := []string{
		`defer f1(a,b)`,
		`Σ0, Σ1 := a, b
	        defer func() {
	        Σ2 := Σ.IV(Σ0)
	        Σ3 := Σ.IV(Σ1)
	        Σ.Line(0, 0, 35, Σ.ICe("f1", Σ2, Σ3))
	        Σ4 := Σ.IC("f1", nil, Σ2, Σ3)
	        f1(Σ0, Σ1)
	        Σ.Line(0, 0, 36, Σ4)
	        }()`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator62(t *testing.T) {
	inout := []string{
		`defer func(a int) bool{return true}(3)`,
		`Σ0 := 3
	        defer func() {
	        Σ1 := func(a int) bool {
	        {
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 0, 45, Σ.IL(Σ2))
	        }
	        Σ3 := Σ.IV(true)
	        Σ.Line(0, 1, 57, Σ.IL(Σ3))
	        return true
	        }
	        Σ4 := Σ.IV(Σ1)
	        Σ.Line(0, 2, 58, Σ4)
	        Σ5 := Σ.IV(Σ0)
	        Σ.Line(0, 3, 60, Σ.ICe("f", Σ5))
	        Σ6 := Σ.IC("f", nil, Σ5)
	        Σ1(Σ0)
	        Σ.Line(0, 3, 61, Σ6)
	        }()`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator62_2(t *testing.T) {
	inout := []string{
		`defer f1()`,
		`defer func() {
	        Σ.Line(0, 0, 32, Σ.ICe("f1"))
	        Σ0 := Σ.IC("f1", nil)
	        f1()
	        Σ.Line(0, 0, 33, Σ0)
	        }()`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator62_3(t *testing.T) {
	inout := []string{
		`defer func(){a=1}()`,
		`defer func() {
	        Σ0 := func() { Σ1 := Σ.IV(1); a = 1; Σ2 := Σ.IV(a); Σ.Line(0, 0, 39, Σ.IA(Σ.IL(Σ2), Σ.IL(Σ1))) }
	        Σ3 := Σ.IV(Σ0)
	        Σ.Line(0, 1, 40, Σ3)
	        Σ.Line(0, 2, 41, Σ.ICe("f"))
	        Σ4 := Σ.IC("f", nil)
	        Σ0()
	        Σ.Line(0, 2, 42, Σ4)
	        }()`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator63(t *testing.T) {
	inout := []string{
		`var a,b int = 1, 2`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(2)
	        var a, b int = 1, 2
	        Σ2 := Σ.IV(a)
	        Σ3 := Σ.IV(b)
	        Σ.Line(0, 0, 41, Σ.IA(Σ.IL(Σ2, Σ3), Σ.IL(Σ0, Σ1)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

//----------

func TestAnnotator64(t *testing.T) {
	inout := []string{
		`return`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(b)
	        Σ2 := Σ.IV(c)
	        Σ.Line(0, 0, 57, Σ.IL(Σ0, Σ1, Σ2))
	        return a, b, c`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator65(t *testing.T) {
	inout := []string{
		`return 1,f1(u),1`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(u)
	        Σ.Line(0, 0, 64, Σ.ICe("f1", Σ1))
	        Σ2 := f1(u)
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IC("f1", Σ3, Σ1)
	        Σ5 := Σ.IV(1)
	        Σ.Line(0, 0, 67, Σ.IL(Σ0, Σ4, Σ5))
	        return 1, Σ2, 1`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator66(t *testing.T) {
	inout := []string{
		`return f1(f2(u))`,
		`Σ0 := Σ.IV(u)
	        Σ.Line(0, 0, 65, Σ.ICe("f2", Σ0))
	        Σ1 := f2(u)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f2", Σ2, Σ0)
	        Σ.Line(0, 0, 66, Σ.ICe("f1", Σ3))
	        Σ4, Σ5, Σ6 := f1(Σ1)
	        Σ7 := Σ.IL(Σ.IV(Σ4), Σ.IV(Σ5), Σ.IV(Σ6))
	        Σ8 := Σ.IC("f1", Σ7, Σ3)
	        Σ.Line(0, 0, 67, Σ.IL(Σ8))
	        return Σ4, Σ5, Σ6`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator67(t *testing.T) {
	inout := []string{
		`return f1(f2(u)),3,f2(u)`,
		`Σ0 := Σ.IV(u)
	        Σ.Line(0, 0, 65, Σ.ICe("f2", Σ0))
	        Σ1 := f2(u)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f2", Σ2, Σ0)
	        Σ.Line(0, 0, 66, Σ.ICe("f1", Σ3))
	        Σ4 := f1(Σ1)
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("f1", Σ5, Σ3)
	        Σ7 := Σ.IV(3)
	        Σ8 := Σ.IV(u)
	        Σ.Line(0, 0, 74, Σ.ICe("f2", Σ8))
	        Σ9 := f2(u)
	        Σ10 := Σ.IV(Σ9)
	        Σ11 := Σ.IC("f2", Σ10, Σ8)
	        Σ.Line(0, 0, 75, Σ.IL(Σ6, Σ7, Σ11))
	        return Σ4, 3, Σ9`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator68(t *testing.T) {
	inout := []string{
		`return a.b, c, d`,
		`Σ0 := Σ.IV(a.b)
	        Σ1 := Σ.IV(c)
	        Σ2 := Σ.IV(d)
	        Σ.Line(0, 0, 67, Σ.IL(Σ0, Σ1, Σ2))
	        return a.b, c, d`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator69(t *testing.T) {
	inout := []string{
		`return 1,1,f1(f2(u))`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(u)
	        Σ.Line(0, 0, 69, Σ.ICe("f2", Σ2))
	        Σ3 := f2(u)
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IC("f2", Σ4, Σ2)
	        Σ.Line(0, 0, 70, Σ.ICe("f1", Σ5))
	        Σ6 := f1(Σ3)
	        Σ7 := Σ.IV(Σ6)
	        Σ8 := Σ.IC("f1", Σ7, Σ5)
	        Σ.Line(0, 0, 71, Σ.IL(Σ0, Σ1, Σ8))
	        return 1, 1, Σ6`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator70(t *testing.T) {
	inout := []string{
		`return 1, 1, &Struct1{a,f1(a+1)}`,
		`Σ0 := Σ.IV(1)
        	Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(a)
	        Σ3 := Σ.IV(a)
	        Σ4 := Σ.IV(1)
	        Σ5 := Σ.IV(a + 1)
	        Σ6 := Σ.IB(Σ5, 12, Σ3, Σ4)
	        Σ.Line(0, 0, 81, Σ.ICe("f1", Σ6))
	        Σ7 := f1(a + 1)
	        Σ8 := Σ.IV(Σ7)
	        Σ9 := Σ.IC("f1", Σ8, Σ6)
	        Σ10 := &Struct1{a, Σ7}
	        Σ11 := Σ.IV(Σ10)
	        Σ12 := Σ.IU(Σ11, 17, Σ.ILit(Σ2, Σ9))
	        Σ.Line(0, 0, 83, Σ.IL(Σ0, Σ1, Σ12))
	        return 1, 1, Σ10`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator71(t *testing.T) {
	inout := []string{
		`return 1, 1, &Struct1{a,uint16((1<<16) / 360)}`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(a)
	        Σ3 := Σ.IV(1)
	        Σ4 := Σ.IV(16)
	        Σ5 := Σ.IV(1 << 16)
	        Σ6 := Σ.IB(Σ5, 20, Σ3, Σ4)
	        Σ7 := Σ.IV(360)
	        Σ8 := Σ.IV((1 << 16) / 360)
	        Σ9 := Σ.IB(Σ8, 15, Σ.IP(Σ6), Σ7)
	        Σ.Line(0, 0, 95, Σ.ICe("uint16", Σ9))
	        Σ10 := uint16((1 << 16) / 360)
	        Σ11 := Σ.IV(Σ10)
	        Σ12 := Σ.IC("uint16", Σ11, Σ9)
	        Σ13 := &Struct1{a, Σ10}
	        Σ14 := Σ.IV(Σ13)
	        Σ15 := Σ.IU(Σ14, 17, Σ.ILit(Σ2, Σ12))
	        Σ.Line(0, 0, 97, Σ.IL(Σ0, Σ1, Σ15))
	        return 1, 1, Σ13`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}
func TestAnnotator72(t *testing.T) {
	inout := []string{
		`return 1, f1(u)+f1(u), nil`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(u)
	        Σ.Line(0, 0, 65, Σ.ICe("f1", Σ1))
	        Σ2 := f1(u)
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IC("f1", Σ3, Σ1)
	        Σ5 := Σ.IV(u)
	        Σ.Line(0, 0, 71, Σ.ICe("f1", Σ5))
	        Σ6 := f1(u)
	        Σ7 := Σ.IV(Σ6)
	        Σ8 := Σ.IC("f1", Σ7, Σ5)
	        Σ9 := Σ2 + Σ6
	        Σ10 := Σ.IV(Σ9)
	        Σ11 := Σ.IB(Σ10, 12, Σ4, Σ8)
	        Σ12 := Σ.IV(nil)
	        Σ.Line(0, 0, 77, Σ.IL(Σ0, Σ11, Σ12))
	        return 1, Σ9, nil`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}

func TestAnnotator73(t *testing.T) {
	inout := []string{
		`return path[len(d):], 1, 1`,
		`Σ0 := Σ.IV(d)
	        Σ.Line(0, 0, 68, Σ.ICe("len", Σ0))
	        Σ1 := len(d)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("len", Σ2, Σ0)
	        Σ4 := path[Σ1:]
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IV(1)
	        Σ7 := Σ.IV(1)
	        Σ.Line(0, 0, 77, Σ.IL(Σ.II2(Σ5, nil, Σ3, nil, nil, false), Σ6, Σ7))
	        return Σ4, 1, 1`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc2)
}

func TestAnnotator74(t *testing.T) {
	inout := []string{
		``, // empty, to test func args
		`{
	        Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV(b)
	        Σ2 := Σ.IV(c)
	        Σ.Line(0, 0, 36, Σ.IL(Σ0, Σ1, Σ2))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc3)
}

func TestAnnotator75(t *testing.T) {
	inout := []string{
		`a++`,
		`Σ0 := Σ.IV(a)
	        a++
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator76(t *testing.T) {
	inout := []string{
		`switch a {
		case 1:
			b=2
		}`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 31, Σ0)
	        switch a {
	        case 1:
	        Σ.Line(0, 1, 40, Σ.ISt())
	        Σ1 := Σ.IV(2)
	        b = 2
	        Σ2 := Σ.IV(b)
	        Σ.Line(0, 2, 45, Σ.IA(Σ.IL(Σ2), Σ.IL(Σ1)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator76a(t *testing.T) {
	inout := []string{
		`switch {
		case a==1:
			return
		default:
		}`,
		`switch {
	        case a == 1:
	        Σ.Line(0, 0, 41, Σ.ISt())
	        Σ.Line(0, 1, 49, Σ.ISt())
	        return
	        default:
	        Σ.Line(0, 2, 57, Σ.ISt())
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator77(t *testing.T) {
	inout := []string{
		`go f1()`,
		`go func() {
	        Σ.Line(0, 0, 29, Σ.ICe("f1"))
	        Σ0 := Σ.IC("f1", nil)
	        f1()
	        Σ.Line(0, 0, 30, Σ0)
	        }()`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator78(t *testing.T) {
	inout := []string{
		`*a=1`,
		`Σ0 := Σ.IV(1)
	        *a = 1
	        Σ1 := Σ.IV(a)
	        Σ2 := Σ.IV(*a)
	        Σ3 := Σ.IU(Σ2, 14, Σ1)
	        Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator79(t *testing.T) {
	inout := []string{
		`a:=W{a:1,b:2,c:3}`,
		`Σ0 := Σ.IV(1)
	        Σ1 := Σ.IV(2)
	        Σ2 := Σ.IV(3)
	        a := W{a: 1, b: 2, c: 3}
	        Σ3 := Σ.IV(a)
	        Σ.Line(0, 0, 40, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ.ILit(Σ.IKV(Σ.IVs("a"), Σ0), Σ.IKV(Σ.IVs("b"), Σ1), Σ.IKV(Σ.IVs("c"), Σ2)))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator80(t *testing.T) {
	inout := []string{
		`type A struct {a int}`,
		`type A struct{ a int }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator81(t *testing.T) {
	inout := []string{
		`var a = f1(1)`,
		`Σ0 := Σ.IV(1)
	        Σ.Line(0, 0, 35, Σ.ICe("f1", Σ0))
	        Σ1 := f1(1)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f1", Σ2, Σ0)
	        var a = Σ1
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator82(t *testing.T) {
	inout := []string{
		`var a = S{1}`,
		` Σ0 := Σ.IV(1)
	        var a = S{1}
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ.ILit(Σ0))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator84(t *testing.T) {
	inout := []string{
		`select {
		case a,ok:=<-c:
			_=a
		}`,
		`Σ.Line(0, 0, 23, Σ.ISt())
	        select {
	        case a, ok := <-c:
	        Σ.Line(0, 1, 46, Σ.ISt())
	        Σ0 := Σ.IV(a)
	        _ = a
	        Σ.Line(0, 2, 51, Σ.IA(Σ.IL(Σ.IAn()), Σ.IL(Σ0)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator84a(t *testing.T) {
	inout := []string{
		`select {
		case a:=<-c:
		case <-b:
			break
		case <-c:
			return
		}`,
		`Σ.Line(0, 0, 23, Σ.ISt())
	        select {
	        case a := <-c:
	        Σ.Line(0, 1, 43, Σ.ISt())
	        case <-b:
	        Σ.Line(0, 2, 53, Σ.ISt())
	        Σ.Line(0, 3, 55, Σ.IBr())
	        break
	        case <-c:
	        Σ.Line(0, 4, 69, Σ.ISt())
	        Σ.Line(0, 5, 77, Σ.ISt())
	        return
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator85(t *testing.T) {
	inout := []string{
		`a, ok := b.c.d[e]`, // map access
		`Σ0 := Σ.IV(e)
	        Σ1, Σ2 := b.c.d[e]
	        Σ3 := Σ.IL(Σ.IV(Σ1), Σ.IV(Σ2))
	        a, ok := Σ1, Σ2
	        Σ4 := Σ.IV(a)
	        Σ5 := Σ.IV(ok)
	        Σ.Line(0, 0, 40, Σ.IA(Σ.IL(Σ4, Σ5), Σ.IL(Σ.II(Σ3, nil, Σ0))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator86(t *testing.T) {
	inout := []string{
		`panic(a)`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 30, Σ.ICe("panic", Σ0))
	        panic(a)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator87(t *testing.T) {
	inout := []string{
		`<-c`,
		`Σ0 := Σ.IV(c)
	        Σ.Line(0, 0, 26, Σ.IUe(36, Σ0))
	        <-c
	        Σ1 := Σ.IU(nil, 36, Σ0)
	        Σ.Line(0, 0, 26, Σ1)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator87a(t *testing.T) {
	inout := []string{
		`a <- <-c`,
		`Σ0 := Σ.IV(c)
	        Σ.Line(0, 0, 31, Σ.IUe(36, Σ0))
	        Σ1 := <-c
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 36, Σ0)
	        a <- Σ1
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 0, 31, Σ.IS(Σ4, Σ3))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator87b(t *testing.T) {
	inout := []string{
		`c:=(<-a).(*b1)`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 30, Σ.IUe(36, Σ0))
	        Σ1 := <-a
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 36, Σ0)
	        c := (Σ1).(*b1)
	        Σ4 := Σ.IV(c)
	        Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ.IP(Σ3))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator87c(t *testing.T) {
	inout := []string{
		`c,ok:=(<-a).(b)`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 33, Σ.IUe(36, Σ0))
	        Σ1 := <-a
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 36, Σ0)
	        c, ok := (Σ1).(b)
	        Σ4 := Σ.IV(c)
	        Σ5 := Σ.IV(ok)
	        Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ4, Σ5), Σ.IL(Σ.IP(Σ3))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator87d(t *testing.T) {
	inout := []string{
		`(<-a).f()`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ.IUe(36, Σ0))
	        Σ1 := <-a
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IU(Σ2, 36, Σ0)
	        Σ.Line(0, 0, 30, Σ.IP(Σ3))
	        Σ.Line(0, 0, 31, Σ.ICe("f"))
	        Σ4 := Σ.IC("f", nil)
	        (Σ1).f()
	        Σ.Line(0, 0, 32, Σ4)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator87e(t *testing.T) {
	inout := []string{
		`(<-a)`,
		`Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ.IUe(36, Σ0))
	        (<-a)
	        Σ1 := Σ.IU(nil, 36, Σ0)
	        Σ.Line(0, 0, 28, Σ.IP(Σ1))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator88(t *testing.T) {
	inout := []string{
		`{a}`,
		`{
	        a
	        Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 25, Σ0)
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator89(t *testing.T) {
	inout := []string{
		`a.b["s"] = 1`,
		`Σ0 := Σ.IV(1)
	        a.b["s"] = 1
	        Σ1 := Σ.IV("s")
	        Σ2 := Σ.IV(a.b["s"])
	        Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.II(Σ2, nil, Σ1)), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator90(t *testing.T) {
	inout := []string{
		`a[i], a[j] = a[j], a[i]`,
		`Σ0 := Σ.IV(j)
	        Σ1 := a[j]
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IV(i)
	        Σ4 := a[i]
	        Σ5 := Σ.IV(Σ4)
	        a[i], a[j] = Σ1, Σ4
	        Σ6 := Σ.IV(i)
	        Σ7 := Σ.IV(a[i])
	        Σ8 := Σ.IV(j)
	        Σ9 := Σ.IV(a[j])
	        Σ.Line(0, 0, 46, Σ.IA(Σ.IL(Σ.II(Σ7, nil, Σ6), Σ.II(Σ9, nil, Σ8)), Σ.IL(Σ.II(Σ2, nil, Σ0), Σ.II(Σ5, nil, Σ3))))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator91(t *testing.T) {
	inout := []string{
		`a:=[]byte{}`,
		`a := []byte{}
	        Σ0 := Σ.IV(a)
	        Σ.Line(0, 0, 34, Σ.IA(Σ.IL(Σ0), Σ.IL(Σ.ILit())))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator92(t *testing.T) {
	inout := []string{
		`a:=func(a...int)[]int{return a}`,
		`Σ0 := func(a ...int) []int {
	        {
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 44, Σ.IL(Σ1))
	        }
	        Σ2 := Σ.IV(a)
	        Σ.Line(0, 1, 53, Σ.IL(Σ2))
	        return a
	        }
	        Σ3 := Σ.IV(Σ0)
	        a := Σ0
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 2, 54, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator93(t *testing.T) {
	inout := []string{
		`a:=[]byte(b)`,
		`Σ0 := Σ.IV(b)
	        Σ.Line(0, 0, 34, Σ.ICe("f", Σ0))
	        Σ1 := []byte(b)
	        Σ2 := Σ.IV(Σ1)
	        Σ3 := Σ.IC("f", Σ2, Σ0)
	        a := Σ1
	        Σ4 := Σ.IV(a)
	        Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator94(t *testing.T) {
	inout := []string{
		// a could be a const (result int), so can't be replaced
		`var evMask uint32 = 0 | a`,
		`Σ0 := Σ.IV(0)
	        Σ1 := Σ.IV(a)
	        Σ2 := Σ.IV(0 | a)
	        Σ3 := Σ.IB(Σ2, 18, Σ0, Σ1)
	        var evMask uint32 = 0 | a
	        Σ4 := Σ.IV(evMask)
	        Σ.Line(0, 0, 48, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator95(t *testing.T) {
	inout := []string{
		`a := v < -1`,
		`Σ0 := Σ.IV(v)
	        Σ1 := Σ.IV(1)
	        Σ2 := Σ.IV(-1)
	        Σ3 := Σ.IU(Σ2, 13, Σ1)
	        Σ4 := v < -1
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IB(Σ5, 40, Σ0, Σ3)
	        a := Σ4
	        Σ7 := Σ.IV(a)
	        Σ.Line(0, 0, 34, Σ.IA(Σ.IL(Σ7), Σ.IL(Σ6)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}
func TestAnnotator96(t *testing.T) {
	inout := []string{
		`a[b]=true
		fn(func(){a[b]=true})`,
		`Σ0 := Σ.IV(true)
	        a[b] = true
	        Σ1 := Σ.IV(b)
	        Σ2 := Σ.IV(a[b])
	        Σ.Line(0, 0, 32, Σ.IA(Σ.IL(Σ.II(Σ2, nil, Σ1)), Σ.IL(Σ0)))
	        Σ3 := func() {
	        Σ4 := Σ.IV(true)
	        a[b] = true
	        Σ5 := Σ.IV(b)
	        Σ6 := Σ.IV(a[b])
	        Σ.Line(0, 1, 52, Σ.IA(Σ.IL(Σ.II(Σ6, nil, Σ5)), Σ.IL(Σ4)))
	        }
	        Σ7 := Σ.IV(Σ3)
	        Σ.Line(0, 2, 53, Σ.ICe("fn", Σ7))
	        Σ8 := Σ.IC("fn", nil, Σ7)
	        fn(Σ3)
	        Σ.Line(0, 2, 54, Σ8)`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator97(t *testing.T) {
	inout := []string{
		`p = 'a' - 'A'`, // mismatched types byte and rune
		`Σ0 := Σ.IV('a')
	        Σ1 := Σ.IV('A')
	        Σ2 := Σ.IV('a' - 'A')
	        Σ3 := Σ.IB(Σ2, 13, Σ0, Σ1)
	        p = 'a' - 'A'
	        Σ4 := Σ.IV(p)
	        Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ4), Σ.IL(Σ3)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator98(t *testing.T) {
	inout := []string{
		`//godebug:annotateoff
		a = 1+1`,
		`a = 1 + 1`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator99(t *testing.T) {
	inout := []string{
		`a:=1
		//godebug:annotateoff
		for {
			a:=2
			//godebug:annotateblock
			a=3
		}
		`,
		`Σ0 := Σ.IV(1)
	        a := 1
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        for {
	        a := 2
	        Σ2 := Σ.IV(3)
	        a = 3
	        Σ3 := Σ.IV(a)
	        Σ.Line(0, 1, 88, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator100(t *testing.T) {
	inout := []string{
		`a = b
		/*aaa*/`,
		`Σ0 := Σ.IV(b)
	        a = b
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 28, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator101(t *testing.T) {
	inout := []string{
		`_=func(int)int{return 1}`,
		`Σ0 := func(Σ1 int) int {
	        {
	        Σ2 := Σ.IV(Σ1)
	        Σ.Line(0, 0, 37, Σ.IL(Σ2))
	        }
	        Σ3 := Σ.IV(1)
	        Σ.Line(0, 1, 46, Σ.IL(Σ3))
	        return 1
	        }
	        Σ4 := Σ.IV(Σ0)
	        _ = Σ0
	        Σ.Line(0, 2, 47, Σ.IA(Σ.IL(Σ.IAn()), Σ.IL(Σ4)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator102(t *testing.T) {
	inout := []string{
		`d=a.b.c()`, // should have msgs on same debug index (selector expr)
		`Σ.Line(0, 0, 31, Σ.ICe("c"))
	        Σ0 := a.b.c()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("c", Σ1)
	        d = Σ0
	        Σ3 := Σ.IV(d)
	        Σ.Line(0, 0, 32, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator103(t *testing.T) {
	inout := []string{
		`e=a.b().c.d()`,
		`Σ.Line(0, 0, 29, Σ.ICe("b"))
	        Σ0 := a.b()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("b", Σ1)
	        Σ3 := Σ.ISel(Σ2, Σ.IV(Σ0.c))
	        Σ.Line(0, 0, 34, Σ3)
	        Σ.Line(0, 0, 35, Σ.ICe("d"))
	        Σ4 := Σ0.c.d()
	        Σ5 := Σ.IV(Σ4)
	        Σ6 := Σ.IC("d", Σ5)
	        e = Σ4
	        Σ7 := Σ.IV(e)
	        Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ7), Σ.IL(Σ6)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator104(t *testing.T) {
	inout := []string{
		`//godebug:annotateoff
		if ok := f(u); ok {}`,
		`if ok := f(u); ok {
        	}`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator105(t *testing.T) {
	inout := []string{
		`
		//godebug:annotateoff
		if ok := a; ok {
		}else if ok2:=b;ok2{
		}else if ok3:=c;ok3{
			//godebug:annotateblock
			c=1
		}`,
		`if ok := a; ok {
	        } else if ok2 := b; ok2 {
	        } else if ok3 := c; ok3 {
	        Σ0 := Σ.IV(1)
	        c = 1
	        Σ1 := Σ.IV(c)
	        Σ.Line(0, 0, 131, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator106(t *testing.T) {
	inout := []string{
		`return`,
		`Σ.Line(0, 0, 29, Σ.ISt())
		return`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator107(t *testing.T) {
	// type A string
	// a:=A("a")
	// if a == "a" {} // ok
	// b:= "a"
	// if a == b {} // fails to compile, can't replace string literal by variable
	inout := []string{
		`if a == "a"{}`,
		`Σ0 := Σ.IV(a)
	        Σ1 := Σ.IV("a")
	        Σ2 := a == "a"
	        Σ3 := Σ.IV(Σ2)
	        Σ4 := Σ.IB(Σ3, 39, Σ0, Σ1)
	        Σ.Line(0, 0, 34, Σ4)
	        if Σ2 {
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator108(t *testing.T) {
	inout := []string{
		`var(
			a=1
			b=a
		)`,
		`Σ0 := Σ.IV(1)
	        var a = 1
	        Σ1 := Σ.IV(a)
	        Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ1), Σ.IL(Σ0)))
	        Σ2 := Σ.IV(a)
	        var b = a
	        Σ3 := Σ.IV(b)
	        Σ.Line(0, 1, 35, Σ.IA(Σ.IL(Σ3), Σ.IL(Σ2)))`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator109(t *testing.T) {
	inout := []string{
		`return <-fn()`,
		`Σ.Line(0, 0, 39, Σ.ICe("fn"))
	        Σ0 := fn()
	        Σ1 := Σ.IV(Σ0)
	        Σ2 := Σ.IC("fn", Σ1)
	        Σ.Line(0, 0, 40, Σ.IUe(36, Σ2))
	        Σ3 := <-Σ0
	        Σ4 := Σ.IV(Σ3)
	        Σ5 := Σ.IU(Σ4, 36, Σ2)
	        Σ.Line(0, 0, 40, Σ.IL(Σ5))
	        return Σ3`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc4)
}

func TestAnnotator110(t *testing.T) {
	inout := []string{
		`//godebug:annotateoff
		for i:=0; i<2;i++{
			_=i+3
		}`,
		` for i := 0; i < 2; i++ {
	        _ = i + 3
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator111(t *testing.T) {
	inout := []string{
		`//godebug:annotateoff
		for i:=0; i<2;i++{
			//godebug:annotateblock
			_=i+3
		}`,
		`for i := 0; i < 2; i++ {
	        Σ0 := Σ.IV(i)
	        Σ1 := Σ.IV(3)
	        Σ2 := Σ.IV(i + 3)
	        Σ3 := Σ.IB(Σ2, 12, Σ0, Σ1)
	        _ = i + 3
	        Σ.Line(0, 0, 93, Σ.IA(Σ.IL(Σ.IAn()), Σ.IL(Σ3)))
	        }`,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

func TestAnnotator_(t *testing.T) {
	inout := []string{
		``,
		``,
	}
	testAnnotator1(t, inout[0], inout[1], srcFunc1)
}

//----------

func TestAnnConfigContent(t *testing.T) {
	src1 := `
		package pkg1
		import "fmt"
		func main(){a:=1}
	`
	src2 := `
		package pkg1
		import "fmt"
		func main2(){b:=1}
	`

	srcs := []string{src1, src2}
	files, names := newFilesFromSrcs(t, srcs...)
	annset := NewAnnotatorSet()

	for i := 0; i < len(srcs); i++ {
		astFile, err := files.fullAstFile(names[i])
		if err != nil {
			t.Fatal(err)
		}
		err = annset.AnnotateAstFile(astFile, names[i], files)
		if err != nil {
			t.Fatal(err)
		}
	}

	// annotate config
	src := annset.ConfigContent("test_network", "test_addr")
	t.Logf("%v", src) // TODO: test output
}

//----------
//----------
//----------

func testAnnotator1(t *testing.T, in0, out0 string, fn func(s string) string) {
	t.Helper()

	in := parseutil.TrimLineSpaces(fn(in0))
	out := parseutil.TrimLineSpaces(fn(out0))
	typ := AnnotationTypeFile

	files, names := newFilesFromSrcs(t, in)
	astFile, err := files.fullAstFile(names[0])
	if err != nil {
		t.Fatal(err)
	}

	ann := NewAnnotator(files.fset, files.NodeAnnType)
	ann.debugPkgName = "Σ"   // expected by tests
	ann.debugVarPrefix = "Σ" // expected by tests
	ann.AnnotateAstFile(astFile, typ)

	var buf bytes.Buffer
	ann.PrintSimple(&buf, astFile)
	res := parseutil.TrimLineSpaces(buf.String())

	if res != out {
		u := fmt.Sprintf("\n*in:\n%s\n*expecting:\n%s\n*got:\n%s", in, out, res)
		t.Fatalf(u)
	}
}

//----------

func srcFunc1(s string) string {
	return `package p1
		func f0() {
			` + s + `
		}`
}

func srcFunc2(s string) string {
	return `package p1
		func f0() (a int, b *int, c *Struct1) {
			` + s + `
		}`
}

func srcFunc3(s string) string {
	return `package p1
		func f0(a, b int, c bool) {
			` + s + `
		}`
}

func srcFunc4(s string) string {
	return `package p1
		func f0() int {
			` + s + `
		}`
}

//----------

func parseAndStringify(t *testing.T, src string) string {
	mode := parser.ParseComments // to support cgo directives on imports
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "a.go", src, mode)
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(nil)
	cfg := &printer.Config{Mode: printer.RawFormat}
	if err := cfg.Fprint(buf, fset, astFile); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

//----------
