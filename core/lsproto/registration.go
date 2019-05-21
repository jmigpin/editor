package lsproto

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
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

	man *Manager

	cs struct { //client/server
		sync.Mutex
		cli        *Client
		sw         *ServerWrap
		connCancel context.CancelFunc
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

func (reg *Registration) onConnErrAsync(err error) {
	me := iout.MultiError{}
	me.Add(err)
	if err := reg.CloseCSLocked(); err != nil {
		me.Add(err)
	}
	err = reg.WrapError(me.Result())
	reg.man.errorAsync(err)
}

func (reg *Registration) WrapError(err error) error {
	return fmt.Errorf("lsproto(%s): %v", reg.Language, err)
}

//----------

func (reg *Registration) CloseCSLocked() error {
	reg.cs.Lock()
	defer reg.cs.Unlock()
	return reg.CloseCSUnlocked()
}

func (reg *Registration) CloseCSUnlocked() error {
	me := &iout.MultiError{}
	if reg.cs.cli != nil {
		if err := reg.cs.cli.Close(); err != nil {
			me.Add(fmt.Errorf("client: %v", err))
		}
	}

	if reg.cs.sw != nil {
		if err := reg.cs.sw.Close(); err != nil {
			me.Add(fmt.Errorf("serverwrap: %v", err))
		}
	}
	err := me.Result()
	if err != nil {
		err = fmt.Errorf("closeclient/server(%v): %v", reg.Language, err)
	}

	reg.cs.cli = nil
	reg.cs.sw = nil

	// Clears cancel resources
	// The client might need to send a close msg to the server. Canceling before doing that will give misleading errors. It's ok to cancel at close time after calling client/server close since the main objective of connCancel is to be able to cancel at connection time.
	if reg.cs.connCancel != nil {
		reg.cs.connCancel()
	}

	return err
}

//----------

func (reg *Registration) connClientServer(ctx context.Context) (*Client, error) {
	reg.cs.Lock()
	defer reg.cs.Unlock()

	if reg.cs.cli != nil {
		return reg.cs.cli, nil
	}

	// independent context for client/server
	ctx0 := context.Background()
	ctx2, cancel := context.WithCancel(ctx0)
	reg.cs.connCancel = cancel

	// new client/server
	err := reg.connClientServer2(ctx2)
	if err != nil {
		return nil, err
	}

	// initialize
	if err := reg.cs.cli.Initialize(ctx, "/"); err != nil {
		return nil, err
	}

	return reg.cs.cli, nil
}

func (reg *Registration) connClientServer2(ctx context.Context) error {
	switch reg.Network {
	case "tcp":
		return reg.connClientServerTCP(ctx)
	case "tcpclient":
		return reg.connClientTCP(ctx, reg.Cmd)
	case "stdio":
		return reg.connClientServerStdio(ctx)
	default:
		return fmt.Errorf("unexpected network: %v", reg.Network)
	}
}

//----------

func (reg *Registration) connClientServerTCP(ctx context.Context) error {
	// server wrap
	sw, addr, err := NewServerWrapTCP(ctx, reg.Cmd, reg)
	if err != nil {
		return err
	}
	reg.cs.sw = sw
	// client
	return reg.connClientTCP(ctx, addr)
}

//----------

func (reg *Registration) connClientTCP(ctx context.Context, addr string) error {
	// client connect with retries
	var cli *Client
	fn := func() error {
		cli0, err := NewClientTCP(ctx, addr, reg)
		if err != nil {
			return err
		}
		cli = cli0
		return nil
	}
	lateFn := func(err error) {
		err2 := reg.CloseCSLocked()
		reg.man.errorAsync(iout.MultiErrors(err, err2))
	}
	sleep := 250 * time.Millisecond
	err := ctxutil.Retry(ctx, sleep, "clienttcp", fn, lateFn)
	if err != nil {
		err2 := reg.CloseCSUnlocked()
		return iout.MultiErrors(err, err2)
	}
	reg.cs.cli = cli
	return nil
}

//----------

func (reg *Registration) connClientServerStdio(ctx context.Context) error {
	var stderr io.Writer
	if reg.HasOptional("stderr") {
		stderr = os.Stderr
	}

	// server wrap
	sw, rwc, err := NewServerWrapIO(ctx, reg.Cmd, stderr, reg)
	if err != nil {
		return err
	}
	reg.cs.sw = sw

	// client
	cli := NewClientIO(rwc, reg)
	reg.cs.cli = cli

	return nil
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
		"c,.c,tcpclient,127.0.0.1:9000",
	}
	return strings.Join(u, "\n")
}

// golang.org/x/tools/cmd/gopls
// golang.org/x/tools/internal/lsp
// golang.org/x/tools/internal/jsonrpc2
// https://github.com/golang/tools/tree/master/internal/lsp
// https://github.com/golang/tools/tree/master/internal/jsonrpc2
var GoplsRegistrationStr = GoplsRegistration(false)

func GoplsRegistration(trace bool) string {
	cmdStr := ""
	if trace {
		cmdStr = " -v -rpc.trace"
	}
	c := osutil.ExecName("gopls") + cmdStr + " serve -listen={{.Addr}}"
	return fmt.Sprintf("go,.go,tcp,%q", c)
}

var CLangRegistrationStr = func() string {
	c := osutil.ExecName("clangd")
	e := ".c .h .cpp .hpp"
	return fmt.Sprintf("c++,%q,stdio,%s", e, c) //+ ",stderr"
}()
