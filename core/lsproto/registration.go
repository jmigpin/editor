package lsproto

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
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

func (reg *Registration) String() string {
	return stringifyRegistration(reg)
}

//----------

func parseRegistration(s string) (*Registration, error) {
	fields, err := parseutil.ParseFields(s, ',')
	if err != nil {
		return nil, err
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

func stringifyRegistration(reg *Registration) string {
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

func RegistrationExamples() []string {
	return []string{
		goplsRegistration(false, false, false),
		goplsRegistration(false, false, true),
		cLangRegistration(false),
		"python,.py,tcpclient,127.0.0.1:9000",
	}
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
