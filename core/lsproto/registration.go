package lsproto

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/statemach"
)

//----------

type Registration struct {
	Language string
	Exts     []string
	Cmd      string
	Network  string   // {stdio, tcp(runs text/template on cmd)}
	Optional []string // optional extra fields

	ri struct {
		sync.Mutex
		ri *RegistrationInstance
	}
}

func (reg *Registration) HasOptional(s string) bool {
	for _, v := range reg.Optional {
		if v == s {
			return true
		}
	}
	return false
}

//----------

type RegistrationInstance struct {
	cli *Client
	sw  *ServerWrap
}

func (inst *RegistrationInstance) Close() error {
	errs := []string{}
	if inst.cli != nil {
		if err := inst.cli.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("client: %v", err))
		}
	}
	if inst.sw != nil {
		if err := inst.sw.CloseWait(); err != nil {
			errs = append(errs, fmt.Sprintf("serverwrap: %v", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("register close (%d err): %v", len(errs), strings.Join(errs, "; "))
}

//----------

func ParseRegistration(s string) (*Registration, error) {
	rd := iorw.NewStringReader(s)
	sc := statemach.NewScanner(rd)

	fields := []string{}
	for i := 0; ; i++ {
		if sc.Match.End() {
			break
		}

		// field separator
		if i > 0 && !sc.Match.Rune(',') {
			return nil, sc.Errorf("comma")
		}
		sc.Advance()

		// field (can be empty)
		for {
			if sc.Match.Quoted("\"'", '\\', true, 5000) {
				continue
			}
			if sc.Match.Except(",") {
				continue
			}
			break
		}
		f := sc.Value()

		// unquote field
		f2, err := strconv.Unquote(f)
		if err == nil {
			f = f2
		}

		// add field
		fields = append(fields, f)
		sc.Advance()
	}

	minFields := 4
	if len(fields) < minFields {
		return nil, fmt.Errorf("expecting at least %v fields: %v", minFields, len(fields))
	}

	reg := &Registration{}
	reg.Language = fields[0]
	if reg.Language == "" {
		return nil, fmt.Errorf("empty language")
	}
	reg.Exts = strings.Split(fields[1], " ")
	reg.Network = fields[2]
	reg.Cmd = fields[3]
	reg.Optional = fields[4:]

	return reg, nil
}

func RegistrationString(reg *Registration) string {
	exts := strings.Join(reg.Exts, " ")
	if len(reg.Exts) >= 2 {
		exts = fmt.Sprintf("%q", exts)
	}

	cmd := reg.Cmd
	cmd2 := parseutil.AddEscapes(cmd, '\\', " ,")
	if cmd != cmd2 {
		cmd = fmt.Sprintf("%q", cmd)
	}

	u := []string{
		reg.Language,
		exts,
		reg.Network,
		cmd,
	}
	u = append(u, reg.Optional...)
	return strings.Join(u, ",")
}

//----------

func RegistrationExamples() string {
	u := []string{
		GoplsRegistrationStr,
		CLangRegistrationStr,
	}
	return strings.Join(u, "\n")
}

// golang.org/x/tools/cmd/gopls
// golang.org/x/tools/internal/lsp
// golang.org/x/tools/internal/jsonrpc2
// https://github.com/golang/tools/tree/master/internal/lsp
// https://github.com/golang/tools/tree/master/internal/jsonrpc2
var GoplsRegistrationStr = func() string {
	c := osutil.ExecName("gopls") + " serve -listen={{.Addr}}"
	return fmt.Sprintf("go,.go,tcp,%q", c)
}()

var CLangRegistrationStr = func() string {
	c := osutil.ExecName("clangd")
	e := ".c .h .cpp .hpp"
	return fmt.Sprintf("c++,%q,stdio,%s", e, c) //+ ",stderr"
}()
