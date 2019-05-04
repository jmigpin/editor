package lsproto

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type Manager struct {
	Regs        []*Registration
	asyncErrors chan<- error
}

func NewManager(asyncErrors chan<- error) *Manager {
	if asyncErrors == nil {
		panic("asyncerrors is nil")
	}
	return &Manager{asyncErrors: asyncErrors}
}

//----------

func (man *Manager) Register(reg *Registration) {
	reg.asyncErrors = man.asyncErrors // setup for all registrations
	man.Regs = append(man.Regs, reg)
}

func (man *Manager) RegisterStr(s string) error {
	reg, err := ParseRegistration(s)
	if err != nil {
		return err
	}
	man.Register(reg)
	return nil
}

func (man *Manager) FileRegistration(filename string) (*Registration, error) {
	ext := filepath.Ext(filename)
	for _, reg := range man.Regs {
		for _, ext2 := range reg.Exts {
			if ext2 == ext {
				return reg, nil
			}
		}
	}
	return nil, fmt.Errorf("no lsproto registration for file ext: %q", ext)
}

//----------

func (man *Manager) Close() error {
	me := &iout.MultiError{}
	for _, reg := range man.Regs {
		me.Add(reg.CloseInstanceLocked())
	}
	return me.Result()
}

//----------

func (man *Manager) autoStart(ctx context.Context, filename string) (*Client, *Registration, error) {
	reg, err := man.FileRegistration(filename)
	if err != nil {
		return nil, nil, err
	}
	cli, err := man.autoStartReg(ctx, reg)
	return cli, reg, err
}

func (man *Manager) autoStartReg(ctx context.Context, reg *Registration) (*Client, error) {
	reg.ri.Lock()
	defer reg.ri.Unlock()

	if reg.ri.ri != nil {
		return reg.ri.ri.cli, nil
	}

	// new client/server
	cli, err := man.autoStartClientServer(ctx, reg)
	if err != nil {
		// ensure instance is closed if can't get a client
		err2 := reg.CloseInstanceUnlocked()
		return nil, iout.MultiErrors(err, err2)
	}

	// initialize
	if err := cli.Initialize(ctx, "/"); err != nil {
		return nil, err
	}

	return cli, nil
}

func (man *Manager) autoStartClientServer(ctx context.Context, reg *Registration) (*Client, error) {
	switch reg.Network {
	case "tcp":
		return man.autoStartClientServerTCP(ctx, reg)
	case "tcpclient":
		return man.autoStartClientTCP(ctx, reg)
	case "stdio":
		return man.autoStartClientServerStdio(ctx, reg)
	default:
		return nil, fmt.Errorf("unexpected network: %v", reg.Network)
	}
}

func (man *Manager) autoStartClientServerTCP(ctx context.Context, reg *Registration) (*Client, error) {

	// server wrap
	sw, addr, err := NewServerWrapTCP(ctx, reg.Cmd, reg)
	if err != nil {
		return nil, err
	}

	// keep instance in the registration
	reg.ri.ri = &RegistrationInstance{sw: sw}

	// client (with connect retries)
	retry := 3 * time.Second
	sleep := 250 * time.Millisecond
	err = chanutil.RetryTimeout(ctx, retry, sleep, "clientservertcp", func() error {
		cli0, err := NewClientTCP(ctx, addr, reg)
		if err != nil {
			return err
		}
		reg.ri.ri.cli = cli0
		return nil
	})

	// client connect error
	if err != nil {
		err2 := reg.CloseInstanceUnlocked()
		return nil, iout.MultiErrors(err, err2)
	}

	return reg.ri.ri.cli, nil
}

func (man *Manager) autoStartClientTCP(ctx context.Context, reg *Registration) (*Client, error) {
	addr := reg.Cmd

	// client (with connect retries)
	retry := 5 * time.Second
	sleep := 200 * time.Millisecond
	var cli *Client
	err := chanutil.RetryTimeout(ctx, retry, sleep, "clienttcp", func() error {
		cli0, err := NewClientTCP(ctx, addr, reg)
		if err != nil {
			return err
		}
		cli = cli0
		return nil
	})

	// client connect error
	if err != nil {
		return nil, err
	}

	// keep instance in register
	reg.ri.ri = &RegistrationInstance{cli: cli}

	return cli, err
}

func (man *Manager) autoStartClientServerStdio(ctx context.Context, reg *Registration) (*Client, error) {
	var stderr io.Writer
	if reg.HasOptional("stderr") {
		stderr = os.Stderr
	}

	// server wrap
	sw, rwc, err := NewServerWrapIO(ctx, reg.Cmd, stderr, reg)
	if err != nil {
		return nil, err
	}

	// client
	cli := NewClientIO(rwc, reg)

	// keep instance in register
	reg.ri.ri = &RegistrationInstance{sw: sw, cli: cli}

	return cli, nil
}

//----------

//func (man *Manager) SyncText(filename string, rd iorw.Reader) error {
//	_, reg, err := man.autoStart(filename)
//	if err != nil {
//		return err
//	}
//	b, err := iorw.ReadFullSlice(rd)
//	if err != nil {
//		return err
//	}
//	return man.regSyncText(reg, filename, b)
//}

func (man *Manager) regSyncText(ctx context.Context, reg *Registration, filename string, b []byte) error {
	err := reg.ri.ri.cli.SyncText(ctx, filename, b)
	return err
}

//----------

func (man *Manager) TextDocumentDefinition(ctx context.Context, filename string, rd iorw.Reader, offset int) (string, *Range, error) {
	cli, reg, err := man.autoStart(ctx, filename)
	if err != nil {
		return "", nil, err
	}

	b, err := iorw.ReadFullSlice(rd)
	if err != nil {
		return "", nil, err
	}

	if err := man.regSyncText(ctx, reg, filename, b); err != nil {
		return "", nil, err
	}

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return "", nil, err
	}

	loc, err := cli.TextDocumentDefinition(ctx, filename, pos)
	if err != nil {
		return "", nil, err
	}

	// filename
	filename2 := trimFileScheme(loc.Uri)
	if u, err := url.PathUnescape(filename2); err == nil {
		filename2 = u
	}

	return filename2, loc.Range, nil
}

//----------

func (man *Manager) TextDocumentCompletion(ctx context.Context, filename string, rd iorw.Reader, offset int) ([]string, error) {
	cli, reg, err := man.autoStart(ctx, filename)
	if err != nil {
		return nil, err
	}

	b, err := iorw.ReadFullSlice(rd)
	if err != nil {
		return nil, err
	}

	if err := man.regSyncText(ctx, reg, filename, b); err != nil {
		return nil, err
	}

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	comp, err := cli.TextDocumentCompletion(ctx, filename, pos)
	return comp, err
}
