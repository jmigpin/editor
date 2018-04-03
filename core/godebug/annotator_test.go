package godebug

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func testSourceResult(t *testing.T, src, res string, srcFn func(string) string, index int) {
	t.Helper()

	src2 := filterSrc(srcFn(src))
	res2 := filterSrc(srcFn(res))

	ann := NewAnnotator()
	ann.improveAssign = false
	ann.debugPkgName = string(rune(931))
	ann.debugVarPrefix = ann.debugPkgName
	ann.simpleOut = true
	astFile, err := ann.ParseAnnotate("test/src.go", src2)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	ann.PrintSimple(&buf, astFile)
	res3 := buf.String()

	in := src2
	exp := res2
	got := filterSrc(res3)
	if exp != got {
		u := fmt.Sprintf("\n*test index: %v\n*input:\n%v*expecting:\n%v*got:\n%v",
			index, in, exp, got)
		t.Fatalf(u)
	}
}

func testSourceResults(t *testing.T, src, res []string, srcFn func(string) string) {
	t.Helper()
	for i, _ := range src {
		testSourceResult(t, src[i], res[i], srcFn, i)
	}
}

func splitSrcRes(srcRes []string) (src, res []string) {
	for i := 0; i < len(srcRes); i += 2 {
		//if i/2 == 0 {
		//if i/2 == 3 {
		src = append(src, srcRes[i])
		res = append(res, srcRes[i+1])
		//}
	}
	return
}

//------------

func filterSrc(s string) string {
	rex := regexp.MustCompile("^[ ]+")
	s = rex.ReplaceAllString(s, "")

	s = strings.Replace(s, "\t", "", -1)
	s = strings.Replace(s, "\n\n", "\n", -1)
	return s
}

//------------

func TestCmdAnnotate1(t *testing.T) {

	srcFunc := func(s string) string {
		return `package p1
		func f0() {
			` + s + `
		}
		`
	}

	// TODO: a=f(a), must keep var a to debug before/after - use a flag
	// TODO: just test: a--
	// TODO: just test: goto aaa

	srcRes := []string{
		// DEBUG

		// call expr

		"f1(1)",
		`Σ0 := Σ.IC(nil, Σ.IV(1))
		Σ.Line(0, 0, 28, Σ0)
		f1(1)`,

		"f1(a,1,nil,\"s\")",
		`Σ0 := "s"
		Σ1 := Σ.IC(nil, Σ.IV(a), Σ.IV(1), Σ.IV(nil), Σ.IV(Σ0))
		Σ.Line(0, 0, 38, Σ1)
		f1(a, 1, nil, Σ0)`,

		"f1(f2(a,f3()))",
		`Σ0 := f3()
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2 := f2(a, Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ.IV(a), Σ1)
		Σ4 := Σ.IC(nil, Σ3)
		Σ.Line(0, 0, 37, Σ4)
		f1(Σ2)`,

		"f1(1 * 200)",
		`Σ0 := 1 * 200
		Σ1 := Σ.IB(Σ.IV(Σ0), 14, Σ.IV(1), Σ.IV(200))
		Σ2 := Σ.IC(nil, Σ1)
		Σ.Line(0, 0, 34, Σ2)
		f1(1 * 200)`,

		"f1(1 * 200 * f2())",
		`Σ0 := 1 * 200
		Σ1 := Σ.IB(Σ.IV(Σ0), 14, Σ.IV(1), Σ.IV(200))
		Σ2 := f2()
		Σ3 := Σ.IC(Σ.IV(Σ2))
		Σ4 := 1 * 200 * Σ2
		Σ5 := Σ.IB(Σ.IV(Σ4), 14, Σ1, Σ3)
		Σ6 := Σ.IC(nil, Σ5)
		Σ.Line(0, 0, 41, Σ6)
		f1(Σ4)`,

		"f1(f2(&a), f3(&a))",
		`Σ0 := &a
		Σ1 := Σ.IU(Σ.IV(Σ0), 17, Σ.IV(a))
		Σ2 := f2(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := &a
		Σ5 := Σ.IU(Σ.IV(Σ4), 17, Σ.IV(a))
		Σ6 := f3(Σ4)
		Σ7 := Σ.IC(Σ.IV(Σ6), Σ5)
		Σ8 := Σ.IC(nil, Σ3, Σ7)
		Σ.Line(0, 0, 41, Σ8)
		f1(Σ2, Σ6)`,

		`f1(a, func(){f2()})`,
		`Σ1 := Σ.IC(nil, Σ.IV(a), Σ.ILit())
		Σ.Line(0, 1, 42, Σ1)
		f1(a, func() { Σ0 := Σ.IC(nil); Σ.Line(0, 0, 40, Σ0); f2() })`,

		// assign

		"a:=1",
		`Σ0 := Σ.IL(Σ.IV(1))
		a := 1
		Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ.IV(a)), Σ0))`,

		"a,b:=1,c",
		`Σ0 := Σ.IL(Σ.IV(1), Σ.IV(c))
		a, b := 1, c
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ0))`,

		"a,b,_:=1,c,d",
		`Σ0 := Σ.IL(Σ.IV(1), Σ.IV(c), Σ.IV(d))
		a, b, _ := 1, c, d
		Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b), Σ.IAn()), Σ0))`,

		"a=1",
		`Σ0 := Σ.IL(Σ.IV(1))
		a = 1
		Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ.IV(a)), Σ0))`,

		"_=1",
		`Σ0 := Σ.IL(Σ.IV(1))
		_ = 1
		Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ.IAn()), Σ0))`,

		`a,_:=1,"s"`,
		`Σ0 := "s"
		Σ1 := Σ.IL(Σ.IV(1), Σ.IV(Σ0))
		a, _ := 1, Σ0
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a), Σ.IAn()), Σ1))`,

		`a,_=1,"s"`,
		`Σ0 := "s"
		Σ1 := Σ.IL(Σ.IV(1), Σ.IV(Σ0))
		a, _ = 1, Σ0
		Σ.Line(0, 0, 32, Σ.IA(Σ.IL(Σ.IV(a), Σ.IAn()), Σ1))`,

		"a.b = true",
		`Σ0 := Σ.IL(Σ.IV(true))
		a.b = true
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a.b)), Σ0))`,

		"i, _ = a.b(c)",
		`Σ0, Σ1 := a.b(c)
		Σ2 := Σ.IC(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1)), Σ.IV(c))
		Σ3 := Σ.IL(Σ2)
		i, _ = Σ0, Σ1
		Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ.IV(i), Σ.IAn()), Σ3))`,

		"c:=f1()",
		`Σ0 := f1()
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2 := Σ.IL(Σ1)
		c := Σ0
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(c)), Σ2))`,

		"_, b := c.d(e, f())",
		`Σ0 := f()
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2, Σ3 := c.d(e, Σ0)
		Σ4 := Σ.IC(Σ.IL(Σ.IV(Σ2), Σ.IV(Σ3)), Σ.IV(e), Σ1)
		Σ5 := Σ.IL(Σ4)
		_, b := Σ2, Σ3
		Σ.Line(0, 0, 42, Σ.IA(Σ.IL(Σ.IAn(), Σ.IV(b)), Σ5))`,

		"a, _ = 1, c",
		`Σ0 := Σ.IL(Σ.IV(1), Σ.IV(c))
		a, _ = 1, c
		Σ.Line(0, 0, 34, Σ.IA(Σ.IL(Σ.IV(a), Σ.IAn()), Σ0))`,

		"a, _ = c.d(1, f(u), 'c', nil)",
		`Σ0 := f(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ2, Σ3 := c.d(1, Σ0, 'c', nil)
		Σ4 := Σ.IC(Σ.IL(Σ.IV(Σ2), Σ.IV(Σ3)), Σ.IV(1), Σ1, Σ.IV('c'), Σ.IV(nil))
		Σ5 := Σ.IL(Σ4)
		a, _ = Σ2, Σ3
		Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ.IV(a), Σ.IAn()), Σ5))`,

		`a, b = f1(c, "s")`,
		`Σ0 := "s"
		Σ1, Σ2 := f1(c, Σ0)
		Σ3 := Σ.IC(Σ.IL(Σ.IV(Σ1), Σ.IV(Σ2)), Σ.IV(c), Σ.IV(Σ0))
		Σ4 := Σ.IL(Σ3)
		a, b = Σ1, Σ2
		Σ.Line(0, 0, 40, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ4))`,

		`a=f1(f2())`,
		`Σ0 := f2()
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2 := f1(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := Σ.IL(Σ3)
		a = Σ2
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a)), Σ4))`,

		`a:=path[f1(d)]`,
		`Σ0 := f1(d)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(d))
		Σ2 := path[Σ0]
		Σ3 := Σ.IL(Σ.II(Σ.IV(Σ2), nil, Σ1))
		a := Σ2
		Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ.IV(a)), Σ3))`,

		`a,b:=c-d, e+f`,
		`Σ0 := c - d
		Σ1 := Σ.IB(Σ.IV(Σ0), 13, Σ.IV(c), Σ.IV(d))
		Σ2 := e + f
		Σ3 := Σ.IB(Σ.IV(Σ2), 12, Σ.IV(e), Σ.IV(f))
		Σ4 := Σ.IL(Σ1, Σ3)
		a, b := Σ0, Σ2
		Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ4))`,

		"a[i] = b",
		`Σ0 := Σ.IL(Σ.IV(b))
		a[i] = b
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IV(i))), Σ0))`,

		"a:=b[c]",
		`Σ0 := b[c]
		Σ1 := Σ.IL(Σ.II(Σ.IV(Σ0), nil, Σ.IV(c)))
		a := Σ0
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(a)), Σ1))`,

		`s = s[:i] + "a"`,
		`Σ0 := s[:i]
		Σ1 := "a"
		Σ2 := Σ0 + Σ1
		Σ3 := Σ.IB(Σ.IV(Σ2), 12, Σ.II2(Σ.IV(Σ0), nil, nil, Σ.IV(i), nil), Σ.IV(Σ1))
		Σ4 := Σ.IL(Σ3)
		s = Σ2
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.IV(s)), Σ4))`,

		`b[1] = u[:2]`,
		`Σ0 := u[:2]
		Σ1 := Σ.IL(Σ.II2(Σ.IV(Σ0), nil, nil, Σ.IV(2), nil))
		b[1] = Σ0
		Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IV(1))), Σ1))`,

		`u[f2()] = u[:2]`,
		`Σ0 := u[:2]
		Σ1 := Σ.IL(Σ.II2(Σ.IV(Σ0), nil, nil, Σ.IV(2), nil))
		Σ2 := f2()
		Σ3 := Σ.IC(Σ.IV(Σ2))
		u[Σ2] = Σ0
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ3)), Σ1))`,

		`a:=s[:]`,
		`Σ0 := s[:]
		Σ1 := Σ.IL(Σ.II2(Σ.IV(Σ0), nil, nil, nil, nil))
		a := Σ0
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(a)), Σ1))`,

		`u[1+a] = u[1+b]`,
		`Σ0 := 1 + b
		Σ1 := Σ.IB(Σ.IV(Σ0), 12, Σ.IV(1), Σ.IV(b))
		Σ2 := u[Σ0]
		Σ3 := Σ.IL(Σ.II(Σ.IV(Σ2), nil, Σ1))
		Σ4 := 1 + a
		Σ5 := Σ.IB(Σ.IV(Σ4), 12, Σ.IV(1), Σ.IV(a))
		u[Σ4] = Σ2
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ5)), Σ3))`,

		`p[1+a]=1`,
		`Σ0 := Σ.IL(Σ.IV(1))
		Σ1 := 1 + a
		Σ2 := Σ.IB(Σ.IV(Σ1), 12, Σ.IV(1), Σ.IV(a))
		p[Σ1] = 1
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ2)), Σ0))`,

		`a:=&Struct1{A:f1(u), B:2}`,
		`Σ0 := f1(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ2 := &Struct1{A: Σ0, B: 2}
		Σ3 := Σ.IU(Σ.IV(Σ2), 17, Σ.ILit(Σ1, Σ.IV(2)))
		Σ4 := Σ.IL(Σ3)
		a := Σ2
		Σ.Line(0, 0, 48, Σ.IA(Σ.IL(Σ.IV(a)), Σ4))`,

		`a += f3(a + 1)`,
		`Σ0 := a + 1
		Σ1 := Σ.IB(Σ.IV(Σ0), 12, Σ.IV(a), Σ.IV(1))
		Σ2 := f3(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := Σ.IL(Σ3)
		a += Σ2
		Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ.IV(a)), Σ4))`,

		`a := &c[i]`,
		`Σ0 := &c[i]
		Σ1 := Σ.IU(Σ.IV(Σ0), 17, Σ.II(nil, nil, Σ.IV(i)))
		Σ2 := Σ.IL(Σ1)
		a := Σ0
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a)), Σ2))`,

		// type switch

		"switch x.(type){}",
		`Σ.Line(0, 0, 23, Σ.IVt(x))
		switch x.(type) {
		}`,

		"switch b:=x.(type){}",
		`Σ.Line(0, 0, 23, Σ.IVt(x))
		switch b := x.(type) {
		}`,

		// switch

		"switch a>b {}",
		`Σ0 := a > b
		Σ1 := Σ.IB(Σ.IV(Σ0), 41, Σ.IV(a), Σ.IV(b))
		Σ.Line(0, 0, 23, Σ1)
		switch Σ0 {
		}`,

		"switch a {}",
		`Σ.Line(0, 0, 23, Σ.IV(a))
		switch a {
		}`,

		`b:=1
		switch a:=f1(u); a {}`,
		`Σ0 := Σ.IL(Σ.IV(1))
		b := 1
		Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ.IV(b)), Σ0))
		{
		Σ1 := f1(u)
		Σ2 := Σ.IC(Σ.IV(Σ1), Σ.IV(u))
		Σ3 := Σ.IL(Σ2)
		a := Σ1
		Σ.Line(0, 1, 28, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(a)), Σ3), Σ.IV(a)))
		switch a {
		}
		}`,

		// if stmt

		"if a {}",
		`Σ.Line(0, 0, 23, Σ.IV(a))
		if a {
		}`,

		"if a {b=1}",
		`Σ.Line(0, 0, 23, Σ.IV(a))
		if a {
		Σ0 := Σ.IL(Σ.IV(1))
		b = 1
		Σ.Line(0, 1, 32, Σ.IA(Σ.IL(Σ.IV(b)), Σ0))
		}`,

		"if c:=f1(); c>2{}",
		`Σ0 := f1()
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2 := Σ.IL(Σ1)
		c := Σ0
		Σ3 := c > 2
		Σ4 := Σ.IB(Σ.IV(Σ3), 41, Σ.IV(c), Σ.IV(2))
		Σ.Line(0, 0, 23, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(c)), Σ2), Σ4))
		if Σ3 {
		}`,

		"if a{}else if b{}",
		`Σ.Line(0, 0, 23, Σ.IV(a))
		if a {
		} else {
		Σ.Line(0, 1, 34, Σ.IV(b))
		if b {
		}
		}`,

		`if v > f1(f2(v)) {}`,
		`Σ0 := f2(v)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(v))
		Σ2 := f1(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := v > Σ2
		Σ5 := Σ.IB(Σ.IV(Σ4), 41, Σ.IV(v), Σ3)
		Σ.Line(0, 0, 23, Σ5)
		if Σ4 {
		}`,

		`if n := f1("s1"); !f2(n, "s2") {}`,
		`Σ0 := "s1"
		Σ1 := f1(Σ0)
		Σ2 := Σ.IC(Σ.IV(Σ1), Σ.IV(Σ0))
		Σ3 := Σ.IL(Σ2)
		n := Σ1
		Σ4 := "s2"
		Σ5 := f2(n, Σ4)
		Σ6 := Σ.IC(Σ.IV(Σ5), Σ.IV(n), Σ.IV(Σ4))
		Σ7 := !Σ5
		Σ8 := Σ.IU(Σ.IV(Σ7), 43, Σ6)
		Σ.Line(0, 0, 23, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(n)), Σ3), Σ8))
		if Σ7 {
		}`,

		`if nil!=nil{}`,
		`Σ0 := nil != nil
		Σ1 := Σ.IB(Σ.IV(Σ0), 44, Σ.IV(nil), Σ.IV(nil))
		Σ.Line(0, 0, 23, Σ1)
		if Σ0 {
		}`,

		`if a!=nil && a.b!=c {} else {}`,
		`Σ0 := a != nil
		Σ1 := Σ.IB(Σ.IV(Σ0), 44, Σ.IV(a), Σ.IV(nil))
		Σ2 := Σ0
		Σ3 := Σ.IVs("?")
		if Σ0 {
		Σ4 := a.b != c
		Σ5 := Σ.IB(Σ.IV(Σ4), 44, Σ.IV(a.b), Σ.IV(c))
		Σ3 = Σ5
		Σ2 = Σ4
		}
		Σ.Line(0, 0, 23, Σ.IB(Σ.IV(Σ2), 34, Σ1, Σ3))
		if Σ2 {
		} else {
		}`,

		`if a || f2() {}`,
		`Σ0 := a
		Σ1 := Σ.IVs("?")
		if !a {
		Σ2 := f2()
		Σ3 := Σ.IC(Σ.IV(Σ2))
		Σ1 = Σ3
		Σ0 = Σ2
		}
		Σ.Line(0, 0, 23, Σ.IB(Σ.IV(Σ0), 35, Σ.IV(a), Σ1))
		if Σ0 {
		}`,

		// loops

		"for i:=0; ; i++{}",
		`for i := 0; ; i++ {
		}`,

		"for i:=0; i<10; i++ {}",
		`for i := 0; ; i++ {
		Σ0 := i < 10
		Σ1 := Σ.IB(Σ.IV(Σ0), 40, Σ.IV(i), Σ.IV(10))
		Σ.Line(0, 0, 23, Σ1)
		if !Σ0 {
		break
		}
		}`,

		"for a,b:=range f2() {}",
		`Σ0 := f2()
		for a, b := range Σ0 {
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IL(Σ.IV(len(Σ0)))))
		}`,

		"for a,_:=range f2() {}",
		`Σ0 := f2()
		for a, Σ1 := range Σ0 {
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(Σ1)), Σ.IL(Σ.IV(len(Σ0)))))
		}`,

		"for _,_=range a {}",
		`Σ0 := a
		for Σ1, Σ2 := range Σ0 {
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ1), Σ.IV(Σ2)), Σ.IL(Σ.IV(len(Σ0)))))
		}`,

		"for a,_=range c {}",
		`Σ0 := c
		for a, _ = range Σ0 {
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(a), Σ.IAn()), Σ.IL(Σ.IV(len(Σ0)))))
		}`,

		// labeled stmt

		`label1:
		label2:
		_=1`,
		`label1:
		;
		label2:
		;
		Σ0 := Σ.IL(Σ.IV(1))
		_ = 1
		Σ.Line(0, 0, 42, Σ.IA(Σ.IL(Σ.IAn()), Σ0))`,

		`a,b:=1, func(a int)int{return 3}`,
		`Σ0 := Σ.IL(Σ.IV(1), Σ.ILit())
		a, b := 1, func(a int) int { Σ.Line(0, 0, 31, Σ.IV(a)); Σ.Line(0, 1, 46, Σ.IV(3)); return 3 }
		Σ.Line(0, 2, 55, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ0))`,

		// types

		`a:=make(map[string]string)`,
		`Σ0 := make(map[string]string)
		Σ1 := Σ.IC(Σ.IV(Σ0))
		Σ2 := Σ.IL(Σ1)
		a := Σ0
		Σ.Line(0, 0, 49, Σ.IA(Σ.IL(Σ.IV(a)), Σ2))`,

		`a:=map[string]string{"a":"b"}`,
		`Σ0 := "b"
		Σ1 := Σ.IL(Σ.ILit(Σ.IV(Σ0)))
		a := map[string]string{"a": Σ0}
		Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ.IV(a)), Σ1))`,

		// defer

		`defer f1(a,b)`,
		`Σ0 := Σ.IC(nil, Σ.IV(a), Σ.IV(b))
		Σ.Line(0, 0, 36, Σ0)
		defer f1(a, b)`,

		`defer func(a int) bool{return true}(3)`,
		`Σ0 := Σ.IC(nil, Σ.ILit(), Σ.IV(3))
		Σ.Line(0, 2, 61, Σ0)
		defer func(a int) bool { Σ.Line(0, 0, 29, Σ.IV(a)); Σ.Line(0, 1, 46, Σ.IV(true)); return true }(3)`,
	}

	src, res := splitSrcRes(srcRes)
	testSourceResults(t, src, res, srcFunc)
}

func TestCmdAnnotate2(t *testing.T) {

	srcFunc := func(s string) string {
		return `package p1
		func f0() (a int, b *int, c *Struct1) {
			` + s + `
		}
		`
	}

	srcRes := []string{
		"return",
		`Σ.Line(0, 0, 51, Σ.IL(Σ.IV(a), Σ.IV(b), Σ.IV(c)))
		return`,

		"return 1,f1(u),1",
		`Σ0 := f1(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ1, Σ.IV(1)))
		return 1, Σ0, 1`,

		"return f1(f2(u))",
		`Σ3 := f2(u)
		Σ4 := Σ.IC(Σ.IV(Σ3), Σ.IV(u))
		Σ5, Σ6, Σ7 := f1(Σ3)
		Σ8 := Σ.IC(Σ.IL(Σ.IV(Σ5), Σ.IV(Σ6), Σ.IV(Σ7)), Σ4)
		Σ9 := Σ.IL(Σ8)
		Σ0, Σ1, Σ2 := Σ5, Σ6, Σ7
		Σ.Line(0, 0, 51, Σ.IA(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1), Σ.IV(Σ2)), Σ9))
		return Σ0, Σ1, Σ2`,

		"return f1(f2(u)),3,f2(u)",
		`Σ0 := f2(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ2 := f1(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := f2(u)
		Σ5 := Σ.IC(Σ.IV(Σ4), Σ.IV(u))
		Σ.Line(0, 0, 51, Σ.IL(Σ3, Σ.IV(3), Σ5))
		return Σ2, 3, Σ4`,

		"return a.b, c, d",
		`Σ.Line(0, 0, 51, Σ.IL(Σ.IV(a.b), Σ.IV(c), Σ.IV(d)))
		return a.b, c, d`,

		"return 1,1,f1(f2(u))",
		`Σ0 := f2(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ2 := f1(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ3))
		return 1, 1, Σ2`,

		`return 1, 1, &Struct1{
			a,
			f1(a+1),
		}`,
		`Σ0 := a + 1
		Σ1 := Σ.IB(Σ.IV(Σ0), 12, Σ.IV(a), Σ.IV(1))
		Σ2 := f1(Σ0)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ1)
		Σ4 := &Struct1{
		a, Σ2,
		}
		Σ5 := Σ.IU(Σ.IV(Σ4), 17, Σ.ILit(Σ.IV(a), Σ3))
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ5))
		return 1, 1, Σ4`,

		`return 1, 1, &Struct1{a,uint16((1<<16) / 360)}`,
		`Σ0 := 1 << 16
		Σ1 := Σ.IB(Σ.IV(Σ0), 20, Σ.IV(1), Σ.IV(16))
		Σ2 := (1 << 16) / 360
		Σ3 := Σ.IB(Σ.IV(Σ2), 15, Σ.IP(Σ1), Σ.IV(360))
		Σ4 := uint16((1 << 16) / 360)
		Σ5 := Σ.IC(Σ.IV(Σ4), Σ3)
		Σ6 := &Struct1{a, Σ4}
		Σ7 := Σ.IU(Σ.IV(Σ6), 17, Σ.ILit(Σ.IV(a), Σ5))
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ7))
		return 1, 1, Σ6`,

		"return 1, f1(u)+f1(u), nil",
		`Σ0 := f1(u)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(u))
		Σ2 := f1(u)
		Σ3 := Σ.IC(Σ.IV(Σ2), Σ.IV(u))
		Σ4 := Σ0 + Σ2
		Σ5 := Σ.IB(Σ.IV(Σ4), 12, Σ1, Σ3)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ5, Σ.IV(nil)))
		return 1, Σ4, nil`,

		`return path[len(d):], 1, 1`,
		`Σ0 := len(d)
		Σ1 := Σ.IC(Σ.IV(Σ0), Σ.IV(d))
		Σ2 := path[Σ0:]
		Σ.Line(0, 0, 51, Σ.IL(Σ.II2(Σ.IV(Σ2), nil, Σ1, nil, nil), Σ.IV(1), Σ.IV(1)))
		return Σ2, 1, 1`,
	}
	src, res := splitSrcRes(srcRes)
	testSourceResults(t, src, res, srcFunc)
}

func TestCmdAnnotate3(t *testing.T) {

	srcFunc := func(s string) string {
		return `package p1
		func f0(a, b int, c bool) {
			` + s + `
		}
		`
	}

	srcRes := []string{
		``,
		`Σ.Line(0, 0, 11, Σ.IL(Σ.IV(a), Σ.IV(b), Σ.IV(c)))`,
	}
	src, res := splitSrcRes(srcRes)
	testSourceResults(t, src, res, srcFunc)
}

//------------

func TestCmdConfig1(t *testing.T) {
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
	ann := NewAnnotator()

	// annotate srcs
	_, _ = ann.ParseAnnotate("test/src1.go", src1)
	_, _ = ann.ParseAnnotate("test/src2.go", src2)

	// annotate config
	src, _ := ann.ConfigSource()
	fmt.Printf("%v", src)
}
