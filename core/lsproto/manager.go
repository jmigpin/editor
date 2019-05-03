package lsproto

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type Manager struct {
	Regs []*Registration
}

func NewManager() *Manager {
	return &Manager{}
}

//----------

func (man *Manager) Register(reg *Registration) {
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
	errs := []string{}
	for _, reg := range man.Regs {
		err := man.closeRegistration(reg)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("man close (%d err): %v", len(errs), strings.Join(errs, ", "))
}

func (man *Manager) closeRegistration(reg *Registration) error {
	reg.ri.Lock()
	defer reg.ri.Unlock()
	return man.closeRegistration2(reg)
}
func (man *Manager) closeRegistration2(reg *Registration) error {
	if reg.ri.ri != nil {
		ri := reg.ri.ri
		reg.ri.ri = nil
		return ri.Close()
	}
	return nil
}

//----------

func (man *Manager) autoStart(filename string) (*Client, *Registration, error) {
	reg, err := man.FileRegistration(filename)
	if err != nil {
		return nil, nil, err
	}
	cli, err := man.autoStartReg(reg)
	return cli, reg, err
}

func (man *Manager) autoStartReg(reg *Registration) (*Client, error) {
	reg.ri.Lock()
	defer reg.ri.Unlock()

	if reg.ri.ri != nil {
		if reg.ri.ri.cli.hasReadErr {
			// client instance has read error
			if err := man.closeRegistration2(reg); err != nil {
				log.Printf("%v", err)
			}
		} else {
			// still good
			return reg.ri.ri.cli, nil
		}
	}

	// new client/server
	cli, err := man.autoStartClientServer(reg)
	if err != nil {
		return nil, err
	}

	// set language param
	cli.Language = reg.Language

	// initialize
	if err := cli.Initialize("/"); err != nil {
		if err := man.closeRegistration2(reg); err != nil {
			log.Printf("register close err (early close due to initialize err): %v", err)
		}
		return nil, err
	}

	return cli, nil
}

func (man *Manager) autoStartClientServer(reg *Registration) (*Client, error) {
	switch reg.Network {
	case "tcp":
		return man.autoStartClientServerTCP(reg)
	case "tcpclient":
		return man.autoStartClientTCP(reg)
	case "stdio":
		return man.autoStartClientServerStdio(reg)
	default:
		return nil, fmt.Errorf("unexpected network: %v", reg.Network)
	}
}

func (man *Manager) autoStartClientServerTCP(reg *Registration) (*Client, error) {
	// server wrap
	sw, addr, err := NewServerWrapTCP(reg.Cmd)
	if err != nil {
		return nil, err
	}

	// client (with connect retries)
	retry := 5 * time.Second
	sleep := 200 * time.Millisecond
	ctx := context.Background()
	var cli *Client
	err = chanutil.RetryTimeout(ctx, retry, sleep, "client server tcp", func() error {
		cli0, err := NewClientTCP(addr)
		if err != nil {
			return err
		}
		cli = cli0
		return nil
	})

	// client connect error
	if err != nil {
		// close the already started sw
		if err := sw.CloseWait(); err != nil {
			log.Printf("sw close err (early close due to cli err): %v", err)
		}

		return nil, err
	}

	// keep instance in register
	reg.ri.ri = &RegistrationInstance{sw: sw, cli: cli}

	return cli, err
}

func (man *Manager) autoStartClientTCP(reg *Registration) (*Client, error) {
	addr := reg.Cmd

	// client (with connect retries)
	retry := 5 * time.Second
	sleep := 200 * time.Millisecond
	ctx := context.Background()
	var cli *Client
	err := chanutil.RetryTimeout(ctx, retry, sleep, "client tcp", func() error {
		cli0, err := NewClientTCP(addr)
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

func (man *Manager) autoStartClientServerStdio(reg *Registration) (*Client, error) {
	var stderr io.Writer
	if reg.HasOptional("stderr") {
		stderr = os.Stderr
	}

	// server wrap
	sw, rwc, err := NewServerWrapIO(reg.Cmd, stderr)
	if err != nil {
		return nil, err
	}

	// client
	cli := NewClientIO(rwc)

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

func (man *Manager) regSyncText(reg *Registration, filename string, b []byte) error {
	err := reg.ri.ri.cli.SyncText(filename, b)
	return err
}

//----------

func (man *Manager) TextDocumentDefinition(filename string, rd iorw.Reader, offset int) (string, *Range, error) {
	cli, reg, err := man.autoStart(filename)
	if err != nil {
		return "", nil, err
	}

	b, err := iorw.ReadFullSlice(rd)
	if err != nil {
		return "", nil, err
	}

	if err := man.regSyncText(reg, filename, b); err != nil {
		return "", nil, err
	}

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return "", nil, err
	}

	loc, err := cli.TextDocumentDefinition(filename, pos)
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

func (man *Manager) TextDocumentCompletion(filename string, rd iorw.Reader, offset int) ([]string, error) {
	cli, reg, err := man.autoStart(filename)
	if err != nil {
		return nil, err
	}

	b, err := iorw.ReadFullSlice(rd)
	if err != nil {
		return nil, err
	}

	if err := man.regSyncText(reg, filename, b); err != nil {
		return nil, err
	}

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	comp, err := cli.TextDocumentCompletion(filename, pos)
	return comp, err
}
