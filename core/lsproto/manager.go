package lsproto

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

// Notes:
// - Manager manages LangManagers
// - LangManager has a Registration and handles a LangInstance
// - Client handles client connection to the lsp server
// - ServerWrap, if used, runs the lsp server process
type Manager struct {
	langs []*LangManager
	msgFn func(string)

	serverWrapW io.Writer // test purposes only
}

func NewManager(msgFn func(string)) *Manager {
	return &Manager{msgFn: msgFn}
}

//----------

func (man *Manager) Error(err error) {
	man.Message(fmt.Sprintf("error: %v", err))
}

func (man *Manager) Message(s string) {
	if man.msgFn != nil {
		man.msgFn(s)
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
	return nil, fmt.Errorf("no lsproto for file ext: %q", ext)
}

func (man *Manager) langInstanceClient(ctx context.Context, filename string) (*Client, *LangInstance, error) {
	lang, err := man.LangManager(filename)
	if err != nil {
		return nil, nil, err
	}
	li, err := lang.instance(ctx)
	if err != nil {
		return nil, nil, err
	}
	return li.cli, li, nil
}

//----------

func (man *Manager) Close() error {
	count := 0
	me := &iout.MultiError{}
	for _, lang := range man.langs {
		err, ok := lang.Close()
		if ok {
			count++
			if err != nil {
				me.Add(err)
			} else {
				man.Message(lang.WrapMsg("closed"))
			}
		}
	}
	if count == 0 {
		return fmt.Errorf("no instances are running")
	}
	return me.Result()
}

//----------

func (man *Manager) TextDocumentImplementation(ctx context.Context, filename string, rd iorw.ReaderAt, offset int) (string, *Range, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return "", nil, err
	}

	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return "", nil, err
	}
	defer didCloseFn()

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return "", nil, err
	}

	loc, err := cli.TextDocumentImplementation(ctx, filename, pos)
	if err != nil {
		return "", nil, err
	}

	// target filename
	filename2, err := urlToAbsFilename(string(loc.Uri))
	if err != nil {
		return "", nil, err
	}

	return filename2, loc.Range, nil
}

//----------

func (man *Manager) TextDocumentDefinition(ctx context.Context, filename string, rd iorw.ReaderAt, offset int) (string, *Range, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return "", nil, err
	}

	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return "", nil, err
	}
	defer didCloseFn()

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return "", nil, err
	}

	loc, err := cli.TextDocumentDefinition(ctx, filename, pos)
	if err != nil {
		return "", nil, err
	}

	// target filename
	filename2, err := urlToAbsFilename(string(loc.Uri))
	if err != nil {
		return "", nil, err
	}

	return filename2, loc.Range, nil
}

//----------

func (man *Manager) TextDocumentCompletion(ctx context.Context, filename string, rd iorw.ReaderAt, offset int) (*CompletionList, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return nil, err
	}

	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return nil, err
	}
	defer didCloseFn()

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	return cli.TextDocumentCompletion(ctx, filename, pos)
}

func (man *Manager) TextDocumentCompletionDetailStrings(ctx context.Context, filename string, rd iorw.ReaderAt, offset int) ([]string, error) {
	compList, err := man.TextDocumentCompletion(ctx, filename, rd, offset)
	if err != nil {
		return nil, err
	}

	res := []string{}
	for _, ci := range compList.Items {
		u := []string{}
		if ci.Deprecated {
			u = append(u, "*deprecated*")
		}
		u = append(u, ci.Label)
		if ci.Detail != "" {
			u = append(u, ci.Detail)
		}
		res = append(res, strings.Join(u, " "))
	}

	//// add documentation if there is only 1 result
	//if len(compList.Items) == 1 {
	//	doc := compList.Items[0].Documentation
	//	if doc != "" {
	//		res[0] += "\n\n" + doc
	//	}
	//}

	return res, nil
}

//----------

func (man *Manager) didOpen(ctx context.Context, cli *Client, filename string, rd iorw.ReaderAt) (func(), error) {
	b, err := iorw.ReadFastFull(rd)
	if err != nil {
		return nil, err
	}
	if err := cli.TextDocumentDidOpenVersion(ctx, filename, b); err != nil {
		return nil, err
	}

	// ISSUE: file1 src is sent to the server (didopen). Assume now that the request that follows (ex: lsprotoCallers) takes too long such that the ctx expires. The usual "defer didclose" will fail since the context is no longer valid. And so the server stays with the version that might have compile errors. The user corrects the errors without asking anything else from the lspserver. Later on, on another file2, asks for the lspserver to assist with something. This could fail since the lspserver still has the file1 cached with errors.
	// solution: if the didopen was successful, return a func to always run the didClose with defer even if the ctx is no longer valid.
	didCloseFn := func() {
		ctx2 := context.Background()                 // don't use a possible canceled ctx
		_ = cli.TextDocumentDidClose(ctx2, filename) // best effort, ignore error
	}
	return didCloseFn, nil
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

func (man *Manager) TextDocumentRename(ctx context.Context, filename string, rd iorw.ReaderAt, offset int, newName string) (*WorkspaceEdit, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return nil, err
	}

	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return nil, err
	}
	defer didCloseFn()

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	return cli.TextDocumentRename(ctx, filename, pos, newName)
}

//----------

func (man *Manager) CallHierarchyCalls(ctx context.Context, filename string, rd iorw.ReaderAt, offset int, typ CallHierarchyCallType) ([]*ManagerCallHierarchyCalls, error) {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return nil, err
	}

	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return nil, err
	}
	defer didCloseFn()

	pos, err := OffsetToPosition(rd, offset)
	if err != nil {
		return nil, err
	}

	items, err := cli.TextDocumentPrepareCallHierarchy(ctx, filename, pos)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("preparecallhierarchy returned no items")
	}

	res := []*ManagerCallHierarchyCalls{}
	for _, item := range items {
		calls, err := cli.CallHierarchyCalls(ctx, typ, item)
		if err != nil {
			return nil, err
		}
		u := &ManagerCallHierarchyCalls{item, calls}
		res = append(res, u)
	}

	return res, nil
}
