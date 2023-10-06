package lsproto

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

//godebug:annotatepackage

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
	// replace if already exists
	for i, lang2 := range man.langs {
		if lang2.Reg.Language == reg.Language {
			man.langs[i] = lang
			return nil
		}
	}
	// append new
	man.langs = append(man.langs, lang)
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
	filename2, err := UrlToAbsFilename(string(loc.Uri))
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
	filename2, err := UrlToAbsFilename(string(loc.Uri))
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
	clist, err := man.TextDocumentCompletion(ctx, filename, rd, offset)
	if err != nil {
		return nil, err
	}
	w := CompletionListToString(clist)
	return w, nil
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
//	cli, _, err := man.langInstanceClient(ctx, filename)
//	if err != nil {
//		return err
//	}
//	return cli.TextDocumentDidSave(ctx, filename, text)
//}

//----------

func (man *Manager) SyncText(ctx context.Context, filename string, rd iorw.ReaderAt) error {
	cli, _, err := man.langInstanceClient(ctx, filename)
	if err != nil {
		return err
	}

	// opening/closing is enough to give the content to the server (using a didsave/didchange would just make it slower since our strategy is to open/close for every change)
	didCloseFn, err := man.didOpen(ctx, cli, filename, rd)
	if err != nil {
		return err
	}
	defer didCloseFn()

	return nil
}

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

func (man *Manager) TextDocumentRenameAndPatch(ctx context.Context, filename string, rd iorw.ReaderAt, offset int, newName string, prePatchFn func([]*WorkspaceEditChange) error) ([]*WorkspaceEditChange, error) {

	we, err := man.TextDocumentRename(ctx, filename, rd, offset, newName)
	if err != nil {
		return nil, err
	}

	wecs, err := we.GetChanges()
	if err != nil {
		return nil, err
	}

	if prePatchFn != nil {
		if err := prePatchFn(wecs); err != nil {
			return nil, err
		}
	}

	// two or more changes to the same file can give trouble (don't using concurrency for this)
	for _, wec := range wecs {
		filename := wec.Filename
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		res, err := PatchTextEdits(b, wec.Edits)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filename, res, 0o644); err != nil {
			return nil, err
		}
		if err := man.SyncText(ctx, filename, rd); err != nil {
			return nil, err
		}
	}

	return wecs, nil
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

//----------

func (man *Manager) TextDocumentReferences(ctx context.Context, filename string, rd iorw.ReaderAt, offset int) ([]*Location, error) {
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

	return cli.TextDocumentReferences(ctx, filename, pos)
}
