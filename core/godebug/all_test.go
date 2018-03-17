package godebug

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/godebug/debug"
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
		`Σ.Line(0, 0, 28, Σ.IC(nil, Σ.IV(1)))
		f1(1)`,

		"f1(a,1,nil,\"s\")",
		`Σ0 := "s"
		Σ.Line(0, 0, 38, Σ.IC(nil, Σ.IV(a), Σ.IV(1), Σ.IV(nil), Σ.IV(Σ0)))
		f1(a, 1, nil, Σ0)`,

		"f1(f2(a,f3()))",
		`Σ0 := f3()
		Σ1 := f2(a, Σ0)
		Σ.Line(0, 0, 37, Σ.IC(nil, Σ.IC(Σ.IV(Σ1), Σ.IV(a), Σ.IC(Σ.IV(Σ0)))))
		f1(Σ1)`,

		"f1(1 * 200)",
		`Σ0 := 1 * 200
		Σ.Line(0, 0, 34, Σ.IC(nil, Σ.IB(Σ.IV(Σ0), 14, Σ.IV(1), Σ.IV(200))))
		f1(1 * 200)`,

		"f1(1 * 200 * f2())",
		`Σ0 := 1 * 200
		Σ1 := f2()
		Σ2 := 1 * 200 * Σ1
		Σ.Line(0, 0, 41, Σ.IC(nil, Σ.IB(Σ.IV(Σ2), 14, Σ.IB(Σ.IV(Σ0), 14, Σ.IV(1), Σ.IV(200)), Σ.IC(Σ.IV(Σ1)))))
		f1(Σ2)`,

		"f1(f2(&a), f3(&a))",
		`Σ0 := f2(&a)
		Σ1 := f3(&a)
		Σ.Line(0, 0, 41, Σ.IC(nil, Σ.IC(Σ.IV(Σ0), Σ.IU(17, Σ.IV(a))), Σ.IC(Σ.IV(Σ1), Σ.IU(17, Σ.IV(a)))))
		f1(Σ0, Σ1)`,

		`f1(a, func(){f2()})`,
		`Σ.Line(0, 1, 42, Σ.IC(nil, Σ.IV(a), Σ.ILit()))
		f1(a, func() { Σ.Line(0, 0, 40, Σ.IC(nil)); f2() })`,

		// assign

		"a:=1",
		`a := 1
		Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IV(1))))`,

		"a,b:=1,c",
		`a, b := 1, c
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IL(Σ.IV(1), Σ.IV(c))))`,

		"a,b,_:=1,c,d",
		`a, b, Σ0 := 1, c, d
		Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b), Σ.IV(Σ0)), Σ.IL(Σ.IV(1), Σ.IV(c), Σ.IV(d))))`,

		"a=1",
		`a = 1
		Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IV(1))))`,

		"_=1",
		`Σ0 := 1
		Σ.Line(0, 0, 26, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(1))))`,

		`a,_:=1,"s"`,
		`a, Σ0 := 1, "s"
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(Σ0)), Σ.IL(Σ.IV(1), Σ.IV(Σ0))))`,

		`a,_=1,"s"`,
		`Σ0 := "s"
		a = 1
		Σ.Line(0, 0, 32, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(Σ0)), Σ.IL(Σ.IV(1), Σ.IV(Σ0))))`,

		"a.b = true",
		`a.b = true
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a.b)), Σ.IL(Σ.IV(true))))`,

		"i, _ = a.b(c)",
		`Σ0, Σ1 := a.b(c)
		i = Σ0
		Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ.IV(i), Σ.IV(Σ1)), Σ.IL(Σ.IC(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1)), Σ.IV(c)))))`,

		"c:=f1()",
		`c := f1()
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(c)), Σ.IL(Σ.IC(Σ.IV(c)))))`,

		"_, b := c.d(e, f())",
		`Σ1 := f()
		Σ0, b := c.d(e, Σ1)
		Σ.Line(0, 0, 42, Σ.IA(Σ.IL(Σ.IV(Σ0), Σ.IV(b)), Σ.IL(Σ.IC(Σ.IL(Σ.IV(Σ0), Σ.IV(b)), Σ.IV(e), Σ.IC(Σ.IV(Σ1))))))`,

		"a, _ = 1, c",
		`Σ0 := c
		a = 1
		Σ.Line(0, 0, 34, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(Σ0)), Σ.IL(Σ.IV(1), Σ.IV(c))))`,

		"a, _ = c.d(1, f(u), 'c', nil)",
		`Σ2 := f(u)
		Σ0, Σ1 := c.d(1, Σ2, 'c', nil)
		a = Σ0
		Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(Σ1)), Σ.IL(Σ.IC(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1)), Σ.IV(1), Σ.IC(Σ.IV(Σ2), Σ.IV(u)), Σ.IV('c'), Σ.IV(nil)))))`,

		`a, b = f1(c, "s")`,
		`Σ0 := "s"
		a, b = f1(c, Σ0)
		Σ.Line(0, 0, 40, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IL(Σ.IC(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IV(c), Σ.IV(Σ0)))))`,

		`a=f1(f2())`,
		`Σ0 := f2()
		a = f1(Σ0)
		Σ.Line(0, 0, 33, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IC(Σ.IV(a), Σ.IC(Σ.IV(Σ0))))))`,

		`a:=path[f1(d)]`,
		`Σ0 := f1(d)
		a := path[Σ0]
		Σ.Line(0, 0, 37, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.II(Σ.IV(a), nil, Σ.IC(Σ.IV(Σ0), Σ.IV(d))))))`,

		`a,b:=c-d, e+f`,
		`a, b := c-d, e+f
		Σ.Line(0, 0, 36, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IL(Σ.IB(Σ.IV(a), 13, Σ.IV(c), Σ.IV(d)), Σ.IB(Σ.IV(b), 12, Σ.IV(e), Σ.IV(f)))))`,

		"a[i] = b",
		`a[i] = b
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IV(i))), Σ.IL(Σ.IV(b))))`,

		"a:=b[c]",
		`a := b[c]
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.II(Σ.IV(a), nil, Σ.IV(c)))))`,

		`s = s[:i] + "a"`,
		`Σ0 := s[:i]
		Σ1 := "a"
		s = Σ0 + Σ1
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.IV(s)), Σ.IL(Σ.IB(Σ.IV(s), 12, Σ.II2(Σ.IV(Σ0), nil, nil, Σ.IV(i), nil), Σ.IV(Σ1)))))`,

		`b[1] = u[:2]`,
		`b[1] = u[:2]
		Σ.Line(0, 0, 35, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IV(1))), Σ.IL(Σ.II2(Σ.IV(b[1]), nil, nil, Σ.IV(2), nil))))`,

		`u[f2()] = u[:2]`,
		`Σ0 := f2()
		u[Σ0] = u[:2]
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IC(Σ.IV(Σ0)))), Σ.IL(Σ.II2(Σ.IV(u[Σ0]), nil, nil, Σ.IV(2), nil))))`,

		`a:=s[:]`,
		`a := s[:]
		Σ.Line(0, 0, 30, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.II2(Σ.IV(a), nil, nil, nil, nil))))`,

		`u[1+a] = u[1+b]`,
		`Σ0 := 1 + a
		Σ1 := 1 + b
		u[Σ0] = u[Σ1]
		Σ.Line(0, 0, 38, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IB(Σ.IV(Σ0), 12, Σ.IV(1), Σ.IV(a)))), Σ.IL(Σ.II(Σ.IV(u[Σ0]), nil, Σ.IB(Σ.IV(Σ1), 12, Σ.IV(1), Σ.IV(b))))))`,

		`p[1+a]=1`,
		`Σ0 := 1 + a
		p[Σ0] = 1
		Σ.Line(0, 0, 31, Σ.IA(Σ.IL(Σ.II(nil, nil, Σ.IB(Σ.IV(Σ0), 12, Σ.IV(1), Σ.IV(a)))), Σ.IL(Σ.IV(1))))`,

		`a:=&Struct1{A:f1(u), B:2}`,
		`Σ0 := f1(u)
		a := &Struct1{A: Σ0, B: 2}
		Σ.Line(0, 0, 48, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IU(17, Σ.ILit(Σ.IC(Σ.IV(Σ0), Σ.IV(u)), Σ.IV(2))))))`,

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
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IB(Σ.IV(Σ0), 41, Σ.IV(a), Σ.IV(b)))))
		switch Σ0 {
		}`,

		"switch a {}",
		`Σ0 := a
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(a))))
		switch Σ0 {
		}`,

		`b:=1
		switch a:=true; a {}`,
		`b := 1
		Σ.Line(0, 0, 27, Σ.IA(Σ.IL(Σ.IV(b)), Σ.IL(Σ.IV(1))))
		{
		a := true
		Σ0 := a
		Σ.Line(0, 1, 28, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IV(true))), Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(a)))))
		switch Σ0 {
		}
		}`,

		// if stmt

		"if a {}",
		`Σ0 := a
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(a))))
		if Σ0 {
		}`,

		"if a {b=1}",
		`Σ0 := a
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(a))))
		if Σ0 {
		b = 1
		Σ.Line(0, 1, 32, Σ.IA(Σ.IL(Σ.IV(b)), Σ.IL(Σ.IV(1))))
		}`,

		"if c:=f1(); c>2{}",
		`c := f1()
		Σ0 := c > 2
		Σ.Line(0, 0, 23, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(c)), Σ.IL(Σ.IC(Σ.IV(c)))), Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IB(Σ.IV(Σ0), 41, Σ.IV(c), Σ.IV(2))))))
		if Σ0 {
		}`,

		"if a{}else if b{}",
		`Σ0 := a
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(a))))
		if Σ0 {
		} else {
		Σ1 := b
		Σ.Line(0, 1, 34, Σ.IA(Σ.IL(Σ.IV(Σ1)), Σ.IL(Σ.IV(b))))
		if Σ1 {
		}
		}`,

		`if v > f1(f2()) {}`,
		`Σ1 := f2()
		Σ2 := f1(Σ1)
		Σ0 := v > Σ2
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IB(Σ.IV(Σ0), 41, Σ.IV(v), Σ.IC(Σ.IV(Σ2), Σ.IC(Σ.IV(Σ1)))))))
		if Σ0 {
		}`,

		`if n := f1("s1"); !f2(n, "s2") {}`,
		`Σ0 := "s1"
		n := f1(Σ0)
		Σ2 := "s2"
		Σ3 := f2(n, Σ2)
		Σ1 := !Σ3
		Σ.Line(0, 0, 23, Σ.IL2(Σ.IA(Σ.IL(Σ.IV(n)), Σ.IL(Σ.IC(Σ.IV(n), Σ.IV(Σ0)))), Σ.IA(Σ.IL(Σ.IV(Σ1)), Σ.IL(Σ.IU(43, Σ.IC(Σ.IV(Σ3), Σ.IV(n), Σ.IV(Σ2)))))))
		if Σ1 {
		}`,

		`if nil!=nil{}`,
		`Σ0 := nil != nil
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IB(Σ.IV(Σ0), 44, Σ.IV(nil), Σ.IV(nil)))))
		if Σ0 {
		}`,

		// "a.b!=c" should not be tested since "a" must not be nil
		//`if a!=nil && a.b!=c {} else{ d=e }`,
		//``,

		// loops

		"for i:=0; ; i++{}",
		`for i := 0; ; i++ {
		}`,

		"for i:=0; i<10; i++ {}",
		`for i := 0; ; i++ {
		Σ0 := i < 10
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IB(Σ.IV(Σ0), 40, Σ.IV(i), Σ.IV(10)))))
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

		"for a,_=range a {}",
		`Σ0 := a
		for a, _ = range Σ0 {
		Σ.Line(0, 0, 23, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(_)), Σ.IL(Σ.IV(len(Σ0)))))
		}`,

		// labeled stmt

		`label1:
		label2:
		_=1`,
		`label1:
		;
		label2:
		;
		Σ0 := 1
		Σ.Line(0, 0, 42, Σ.IA(Σ.IL(Σ.IV(Σ0)), Σ.IL(Σ.IV(1))))`,

		`a,b:=1, func(a int)int{return 3}`,
		`a, b := 1, func(a int) int { Σ.Line(0, 0, 31, Σ.IV(a)); Σ.Line(0, 1, 46, Σ.IV(3)); return 3 }
		Σ.Line(0, 2, 55, Σ.IA(Σ.IL(Σ.IV(a), Σ.IV(b)), Σ.IL(Σ.IV(1), Σ.ILit())))`,

		// types

		`a:=make(map[string]string)`,
		`a := make(map[string]string)
		Σ.Line(0, 0, 49, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.IC(Σ.IV(a)))))`,

		`a:=map[string]string{"a":"b"}`,
		`Σ0 := "b"
		a := map[string]string{"a": Σ0}
		Σ.Line(0, 0, 52, Σ.IA(Σ.IL(Σ.IV(a)), Σ.IL(Σ.ILit(Σ.IV(Σ0)))))`,

		// defer

		`defer f1(a,b)`,
		`Σ.Line(0, 0, 36, Σ.IC(nil, Σ.IV(a), Σ.IV(b)))
		defer f1(a, b)`,

		`defer func(a int) bool{return true}(3)`,
		`Σ.Line(0, 2, 61, Σ.IC(nil, Σ.ILit(), Σ.IV(3)))
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
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IC(Σ.IV(Σ0), Σ.IV(u)), Σ.IV(1)))
		return 1, Σ0, 1`,

		"return f1(f2(u))",
		`Σ3 := f2(u)
		Σ0, Σ1, Σ2 := f1(Σ3)
		Σ.Line(0, 0, 51, Σ.IA(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1), Σ.IV(Σ2)), Σ.IL(Σ.IC(Σ.IL(Σ.IV(Σ0), Σ.IV(Σ1), Σ.IV(Σ2)), Σ.IC(Σ.IV(Σ3), Σ.IV(u))))))
		return Σ0, Σ1, Σ2`,

		"return f1(f2(u)),3,f2(u)",
		`Σ0 := f2(u)
		Σ1 := f1(Σ0)
		Σ2 := f2(u)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IC(Σ.IV(Σ1), Σ.IC(Σ.IV(Σ0), Σ.IV(u))), Σ.IV(3), Σ.IC(Σ.IV(Σ2), Σ.IV(u))))
		return Σ1, 3, Σ2`,

		"return a.b, c, d",
		`Σ.Line(0, 0, 51, Σ.IL(Σ.IV(a.b), Σ.IV(c), Σ.IV(d)))
		return a.b, c, d`,

		"return 1,1,f1(f2(u))",
		`Σ0 := f2(u)
		Σ1 := f1(Σ0)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ.IC(Σ.IV(Σ1), Σ.IC(Σ.IV(Σ0), Σ.IV(u)))))
		return 1, 1, Σ1`,

		`return 1, 1, &Struct1{
			a,
			f1(a+1),
		}`,
		`Σ0 := a + 1
		Σ1 := f1(Σ0)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ.IU(17, Σ.ILit(Σ.IV(a), Σ.IC(Σ.IV(Σ1), Σ.IB(Σ.IV(Σ0), 12, Σ.IV(a), Σ.IV(1)))))))
		return 1, 1, &Struct1{
		a, Σ1,
		}`,

		`return 1, 1, &Struct1{a,uint16((a<<16) / 360)}`,
		`Σ0 := a << 16
		Σ1 := Σ0 / 360
		Σ2 := uint16(Σ1)
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IV(1), Σ.IU(17, Σ.ILit(Σ.IV(a), Σ.IC(Σ.IV(Σ2), Σ.IB(Σ.IV(Σ1), 15, Σ.IP(Σ.IB(Σ.IV(Σ0), 20, Σ.IV(a), Σ.IV(16))), Σ.IV(360)))))))
		return 1, 1, &Struct1{a, Σ2}`,

		"return 1, f1(u)+f1(u), nil",
		`Σ0 := f1(u)
		Σ1 := f1(u)
		Σ2 := Σ0 + Σ1
		Σ.Line(0, 0, 51, Σ.IL(Σ.IV(1), Σ.IB(Σ.IV(Σ2), 12, Σ.IC(Σ.IV(Σ0), Σ.IV(u)), Σ.IC(Σ.IV(Σ1), Σ.IV(u))), Σ.IV(nil)))
		return 1, Σ2, nil`,

		`return path[len(d):], 1, 1`,
		`Σ0 := len(d)
		Σ1 := path[Σ0:]
		Σ.Line(0, 0, 51, Σ.IL(Σ.II2(Σ.IV(Σ1), nil, Σ.IC(Σ.IV(Σ0), Σ.IV(d)), nil, nil), Σ.IV(1), Σ.IV(1)))
		return Σ1, 1, 1`,
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

//------------

func TestCmdStart1(t *testing.T) {
	src := `
		package main
		import "fmt"
		import "time"
		func main(){
			a:=1
			b:=a
			c:="testing"
			go func(){
				u:=a+b
				c+=fmt.Sprintf("%v", u)
			}()
			c+=fmt.Sprintf("%v", a+b)			
			time.Sleep(10*time.Millisecond)
		}
	`
	filename := "test/src.go"

	cmd := NewCmd([]string{"run", filename}, src)
	defer cmd.Cleanup()

	ctx := context.Background()
	if err := cmd.Start(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		for msg := range cmd.Client.Messages {
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
			default:
				fmt.Printf("recv msg: %v\n", msg)
				//spew.Dump(msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestCmdStart2(t *testing.T) {
	src := `
		package main
		import "fmt"
		func f1() int{
			_=7
			return 1
		}
		func f2() string{
			_=5
			u := []int{9,1,2,3}
			_=5
			if 1 >= f1() && 1 <= f1() {
				b := 10
				u = u[:1-f1()]
				a := 10 + b
				return fmt.Sprintf("%v %v", a, u)
			}
			_=8
			return "aa"
		}
		func main(){
			_=f2()
		}
	`
	filename := "test/src.go"

	args := []string{"run", filename}
	cmd := NewCmd(args, src)
	defer cmd.Cleanup()

	ctx := context.Background()
	if err := cmd.Start(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		for msg := range cmd.Client.Messages {
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
				//spew.Dump(msg)
			default:
				fmt.Printf("recv msg: %v\n", msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestCmdStart3(t *testing.T) {
	proj := "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor"
	filename := proj + "/editor.go"
	args := []string{
		"run",
		"-dirs=" +
			proj +
			"," + proj + "/core" +
			"," + proj + "/ui",
		filename,
	}

	cmd := NewCmd(args, nil)
	defer cmd.Cleanup()

	ctx := context.Background()
	if err := cmd.Start(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	nMsgs := 0
	go func() {
		for msg := range cmd.Client.Messages {
			nMsgs++
			fmt.Printf("recv msg: %v\n", msg)
			//spew.Dump(msg)
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	if nMsgs == 0 {
		t.Fatalf("nmsgs=%v", nMsgs)
	}
}

func TestCmdRun4(t *testing.T) {
	proj := "/home/jorge/bin"
	filename := filepath.Join(proj, "status.go")
	args := []string{
		"run",
		filename,
	}

	cmd := NewCmd(args, nil)
	defer cmd.Cleanup()

	ctx := context.Background()
	if err := cmd.Start(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	nMsgs := 0
	go func() {
		for msg := range cmd.Client.Messages {
			nMsgs++
			fmt.Printf("recv msg: %v\n", msg)
			//spew.Dump(msg)
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	if nMsgs == 0 {
		t.Fatalf("nmsgs=%v", nMsgs)
	}
}

func TestCmdTest1(t *testing.T) {
	proj := "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor/util/imageutil"
	args := []string{
		"test", "-run", "HSV1",
	}

	cmd := NewCmd(args, nil)
	defer cmd.Cleanup()

	cmd.Dir = proj
	ctx := context.Background()
	if err := cmd.Start(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RequestStart(); err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		for msg := range cmd.Client.Messages {
			//fmt.Printf("recv msg: %v\n", msg)
			switch t := msg.(type) {
			case *debug.LineMsg:
				fmt.Printf("%v\n", StringifyItem(t.Item))
			default:
				fmt.Printf("recv msg: %v\n", msg)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}
