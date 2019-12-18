package lsproto

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

// Notes:
// - Manager manages LangManagers
// - LangManger has a Registration and handles a LangInstance
// - Client handles client connection to the lsp server
// - ServerWrap, if used, runs the lsp server process
type Manager struct {
	langs []*LangManager
	errFn func(error)
}

func NewManager(errFn func(error)) *Manager {
	return &Manager{errFn: errFn}
}

//----------

func (man *Manager) errorAsync(err error) {
	if man.errFn != nil && err != nil {
		man.errFn(err)
	}
}

//----------

func (man *Manager) Register(reg *Registration) error {
	lang := NewLangManager(man, reg)
	man.langs = append(man.langs, lang)
	// TODO: file extentions conflict, will use first added lang that matches
	return nil
}

//----------

func (man *Manager) LangManager(filename string) (*LangManager, error) {
	ext := filepath.Ext(filename)
	for _, lang := range man.langs {
		for _, ext2 := range lang.Reg.Exts {
			if ext2 == ext {
				return lang, nil
			}
		}
	}
	return nil, fmt.Errorf("no lsproto registration for file ext: %q", ext)
}

func (man *Manager) langInstanceClient(ctx context.Context, filename string) (*Client, *LangInstance, error) {
	lang, err := man.LangManager(filename)
	if err != nil {
		return nil, nil, err
	}
	li := lang.instance()
	cli, err := li.client(ctx, filename)
	if err != nil {
		err = lang.WrapError(err)
	}
	return cli, li, err
}

//----------

func (man *Manager) TextDocumentDefinition(ctx context.Context, filename string, rd iorw.Reader, offset int) (string, *Range, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return "", nil, err
	}

	if err := man.didOpenVersion(ctx, cli, filename, rd); err != nil {
		return "", nil, err
	}
	defer man.didClose(ctx, cli, filename)

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return "", nil, err
	}

	loc, err := cli.TextDocumentDefinition(ctx, filename, pos)
	if err != nil {
		return "", nil, err
	}

	// target filename
	filename2 := trimFileScheme(loc.Uri)
	if u, err := url.PathUnescape(filename2); err == nil {
		filename2 = u
	}

	return filename2, loc.Range, nil
}

//----------

func (man *Manager) TextDocumentCompletion(ctx context.Context, filename string, rd iorw.Reader, offset int) ([]string, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return nil, err
	}

	if err := man.didOpenVersion(ctx, cli, filename, rd); err != nil {
		return nil, err
	}
	defer man.didClose(ctx, cli, filename)

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	comp, err := cli.TextDocumentCompletion(ctx, filename, pos)
	return comp, err
}

//----------

func (man *Manager) didOpenVersion(ctx context.Context, cli *Client, filename string, rd iorw.Reader) error {
	b, err := iorw.ReadFullSlice(rd)
	if err != nil {
		return err
	}
	return cli.TextDocumentDidOpenVersion(ctx, filename, b)
}

func (man *Manager) didClose(ctx context.Context, cli *Client, filename string) error {
	return cli.TextDocumentDidClose(ctx, filename)
}

//----------

//func (man *Manager) DidSave(ctx context.Context, filename string, text []byte) error {
//	// no error if there is no lang registered
//	_, err := man.lang(filename)
//	if err != nil {
//		return nil
//	}
//	return man.TextDocumentDidSave(ctx, filename, text)
//}

//func (man *Manager) TextDocumentDidSave(ctx context.Context, filename string, text []byte) error {
//	cli, _, err := man.langClient(ctx, filename)
//	if err != nil {
//		return err
//	}
//	return cli.TextDocumentDidSave(ctx, filename, text)
//}

//----------

//func (man *Manager) syncText(ctx context.Context, filename string, rd iorw.Reader) error {
//	cli, _, err := man.autoStart(ctx, filename)
//	if err != nil {
//		return err
//	}
//	b, err := iorw.ReadFullSlice(rd)
//	if err != nil {
//		return err
//	}
//	return cli.SyncText(ctx, filename, b)
//}

//----------

func (man *Manager) Close() error {
	me := &iout.MultiError{}
	for _, lang := range man.langs {
		me.Add(lang.Close())
	}
	return me.Result()
}
