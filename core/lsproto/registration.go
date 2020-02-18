package lsproto

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/scanutil"
)

//----------

type Registration struct {
	Language string
	Exts     []string
	Cmd      string
	Network  string   // {stdio, tcp(runs text/template on cmd)}
	Optional []string // optional extra fields
}

func NewRegistration(s string) (*Registration, error) {
	reg, err := parseRegistration(s)
	if err != nil {
		return nil, err
	}
	return reg, nil
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

func parseRegistration(s string) (*Registration, error) {
	rd := iorw.NewStringReader(s)
	sc := scanutil.NewScanner(rd)

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

//----------

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
		goplsRegistration(false, false, false),
		goplsRegistration(false, false, true),
		cLangRegistration(false),
		"python,.py,tcpclient,127.0.0.1:9000",
	}
	return strings.Join(u, "\n")
}

//----------

func goplsRegistration(trace bool, stderr bool, tcp bool) string {
	cmdStr := ""
	if trace {
		cmdStr = " -v -rpc.trace"
	}
	errOut := ""
	if stderr {
		errOut = ",stderr"
	}
	cmd := osutil.ExecName("gopls") + cmdStr + " serve"
	net := "stdio"
	if tcp {
		net = "tcp"
		cmd += " -listen={{.Addr}}"
	}
	return fmt.Sprintf("go,.go,%v,%q%s", net, cmd, errOut)
}

func cLangRegistration(stderr bool) string {
	ext := ".c .h .cpp .hpp .cc"
	cmd := osutil.ExecName("clangd")
	errOut := ""
	if stderr {
		errOut = ",stderr"
	}
	return fmt.Sprintf("cpp,%q,stdio,%s%s", ext, cmd, errOut)
}
