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
	Network  string // {stdio,tcpclient,tcp{templatevals:.Addr}}
	Cmd      string
	Optional []string // {stderr,nogotoimpl}
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

	// handle backwards compatibility (keep old defaults)
	if s == "nogotoimpl" {
		// - these won't use gotoimplementation, and there is no way to enable it (it would just be slower)
		// - other languages (ex:c/c++) will use gotoimplementation
		languagesToBypass := "go python javascript"
		if strings.Contains(languagesToBypass, strings.ToLower(reg.Language)) {
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
	if len(fields) > 4 {
		reg.Optional = strings.Split(fields[4], " ")
	}

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
	if len(reg.Optional) >= 1 {
		h := strings.Join(reg.Optional, " ")
		if len(reg.Optional) >= 2 {
			h = fmt.Sprintf("%q", h)
		}
		u = append(u, h)
	}
	return strings.Join(u, ",")
}

//----------

func RegistrationExamples() []string {
	return []string{
		GoplsRegistration(false, false, false),
		GoplsRegistration(false, true, false),
		cLangRegistration(false),
		"python,.py,stdio,pylsp",
		"python,.py,tcpclient,127.0.0.1:9000",
		"python,.py,stdio,pylsp,\"stderr nogotoimpl\"",
	}
}

//----------

func GoplsRegistration(stderr bool, tcp bool, trace bool) string {
	cmd := osutil.ExecName("gopls")
	if trace {
		cmd += " -v"
	}
	cmd += " serve"
	if trace {
		cmd += " -rpc.trace"
	}
	net := "stdio"
	if tcp {
		net = "tcp"
		cmd += " -listen={{.Addr}}"
	}

	errOut := ""
	if net == "stdio" {
		if stderr {
			//errOut = ",stderr"
			// DEBUG
			//errOut = ",stderrmanmsg"
		}
	}

	return fmt.Sprintf("go,.go,%v,%q%s", net, cmd, errOut)
}

func cLangRegistration(stderr bool) string {
	ext := ".c .h .cpp .hpp .cc"
	cmd := osutil.ExecName("clangd")
	errOut := ""
	if stderr {
		//errOut = ",stderr"
	}
	return fmt.Sprintf("cpp,%q,stdio,%q%s", ext, cmd, errOut)
}

func pylspRegistration(stderr bool, tcp bool) string {
	cmd := osutil.ExecName("pylsp")
	net := "stdio"
	if tcp {
		net = "tcp"
		cmd += " --tcp"
		cmd += " --host={{.Host}}"
		cmd += " --port={{.Port}}"
	}
	errOut := ""
	if stderr {
		errOut = ",stderr"
	}
	return fmt.Sprintf("python,.py,%s,%q%s", net, cmd, errOut)
}
