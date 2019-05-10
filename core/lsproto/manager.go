package lsproto

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type Manager struct {
	Regs  []*Registration
	errFn func(error)
}

func NewManager(errFn func(error)) *Manager {
	return &Manager{errFn: errFn}
}

//----------

func (man *Manager) errorAsync(err error) {
	if man.errFn != nil {
		man.errFn(err)
	}
}

//----------

func (man *Manager) Register(reg *Registration) {
	// setup for all registrations
	reg.man = man

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
		me.Add(reg.CloseCSLocked())
	}
	return me.Result()
}

//----------

func (man *Manager) autoStart(ctx context.Context, filename string) (*Client, *Registration, error) {
	reg, err := man.FileRegistration(filename)
	if err != nil {
		return nil, nil, err
	}
	cli, err := reg.connClientServer(ctx)
	return cli, reg, err
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
	err := reg.cs.cli.SyncText(ctx, filename, b)
	return err
}

//----------

func (man *Manager) TextDocumentDefinition(ctx context.Context, filename string, rd iorw.Reader, offset int) (string, *Range, error) {
	cli, reg, err := man.autoStart(ctx, filename)
	if err != nil {
		return "", nil, err
	}
	_ = reg

	//	b, err := iorw.ReadFullSlice(rd)
	//	if err != nil {
	//		return "", nil, err
	//	}

	//	if err := man.regSyncText(ctx, reg, filename, b); err != nil {
	//		return "", nil, err
	//	}

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
	_ = reg

	//	b, err := iorw.ReadFullSlice(rd)
	//	if err != nil {
	//		return nil, err
	//	}

	//	if err := man.regSyncText(ctx, reg, filename, b); err != nil {
	//		return nil, err
	//	}

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	comp, err := cli.TextDocumentCompletion(ctx, filename, pos)
	return comp, err
}
