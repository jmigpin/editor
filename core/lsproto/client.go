package lsproto

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"strings"
	"time"

	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/pkg/errors"
)

type Client struct {
	rcli       *rpc.Client
	rwc        io.ReadWriteCloser
	fversions  map[string]int
	reg        *Registration
	hasConnErr bool
}

func NewClientTCP(ctx context.Context, addr string, reg *Registration) (*Client, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := NewClientIO(conn, reg)
	return cli, nil
}

func NewClientIO(rwc io.ReadWriteCloser, reg *Registration) *Client {
	cli := &Client{reg: reg, fversions: map[string]int{}}
	cli.rwc = rwc
	cc := NewJsonCodec(rwc)
	cc.OnNotificationMessage = cli.onNotificationMessage
	cli.rcli = rpc.NewClientWithCodec(cc)
	return cli
}

//----------

func (cli *Client) Close() error {
	me := iout.MultiError{}
	if !cli.hasConnErr {
		me.Add(cli.ShutdownRequest())
	}
	if !cli.hasConnErr {
		me.Add(cli.ExitNotification())
	}
	if cli.rwc != nil {
		me.Add(cli.rcli.Close()) // will also close rwc
		me.Add(cli.rwc.Close())
	}
	return me.Result()
}

//----------

func (cli *Client) Call(ctx context.Context, method string, args, reply interface{}) error {
	lspResp := &Response{}
	fn := func() error {
		return cli.rcli.Call(method, args, lspResp)
	}
	lateFn := func(err error) {
		err = errors.Wrap(err, "call late")
		cli.reg.onConnErrAsync(err)
	}
	err := ctxutil.Call(ctx, method, fn, lateFn)
	if err != nil {
		// hard error (conn or parse error)
		cli.hasConnErr = true

		// improve msg
		err := errors.Wrap(err, "call")

		go cli.reg.onConnErrAsync(err)
		return err
	}

	// not expecting a reply
	if _, ok := noreplyMethod(method); ok {
		return nil
	}

	// soft error (rpc data with error content)
	if lspResp.Error != nil {
		return cli.reg.WrapError(lspResp.Error)
	}

	// decode result
	return decodeJsonRaw(lspResp.Result, reply)
}

//----------

func (cli *Client) onNotificationMessage(msg *NotificationMessage) {
	logJson("notification <--: ", msg)
}

//----------

func (cli *Client) Initialize(ctx context.Context, dir string) error {
	opt := &InitializeParams{}
	opt.RootUri = addFileScheme(dir)
	//opt.Capabilities.TextDocument.PublishDiagnostics = &PublishDiagnostics{
	//	RelatedInformation: false,
	//}

	var capabilities interface{}
	err := cli.Call(ctx, "initialize", &opt, &capabilities)
	logJson("initialize <--: ", capabilities)
	return err
}

//----------

func (cli *Client) ShutdownRequest() error {
	// https://microsoft.github.io/language-server-protocol/specification#shutdown

	// TODO: shutdown request should expect a reply
	// * clangd is sending a reply (ok)
	// * gopls is not sending a reply (NOT OK)

	// best effort, impose timeout
	ctx := context.Background()
	ctx2, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	ctx = ctx2

	err := cli.Call(ctx, "shutdown", nil, nil)
	return err
}

func (cli *Client) ExitNotification() error {
	// https://microsoft.github.io/language-server-protocol/specification#exit

	// best effort, impose timeout
	ctx := context.Background()
	ctx2, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	ctx = ctx2

	err := cli.Call(ctx, "noreply:exit", nil, nil)
	return err
}

//----------

func (cli *Client) TextDocumentDidOpen(ctx context.Context, filename, text string, version int) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didOpen

	opt := &DidOpenTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.TextDocument.LanguageId = cli.reg.Language
	opt.TextDocument.Version = version
	opt.TextDocument.Text = text
	err := cli.Call(ctx, "noreply:textDocument/didOpen", &opt, nil)
	return err
}

func (cli *Client) TextDocumentDidClose(ctx context.Context, filename string) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didClose

	opt := &DidCloseTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	err := cli.Call(ctx, "noreply:textDocument/didClose", &opt, nil)
	return err
}

func (cli *Client) TextDocumentDidChange(ctx context.Context, filename, text string, version int) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didChange

	opt := &DidChangeTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.TextDocument.Version = version

	// text end line/column
	rd := iorw.NewStringReader(text)
	pos, err := OffsetToPosition(rd, len(text))
	if err != nil {
		return err
	}

	// changes
	opt.ContentChanges = []*TextDocumentContentChangeEvent{
		&TextDocumentContentChangeEvent{
			Range: Range{
				Start: Position{0, 0},
				End:   pos,
			},
			//RangeLength: len(text), // TODO: not working?
			Text: text,
		},
	}
	return cli.Call(ctx, "noreply:textDocument/didChange", &opt, nil)
}

func (cli *Client) TextDocumentDidSave(ctx context.Context, filename string, text []byte) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didSave

	opt := &DidSaveTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.Text = string(text) // NOTE: has omitempty

	return cli.Call(ctx, "noreply:textDocument/didSave", &opt, nil)
}

//----------

func (cli *Client) TextDocumentDefinition(ctx context.Context, filename string, pos Position) (*Location, error) {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_definition

	opt := &TextDocumentPositionParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.Position = pos

	result := []*Location{}
	err := cli.Call(ctx, "textDocument/definition", &opt, &result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no results")
	}
	return result[0], nil // first result only
}

//----------

func (cli *Client) TextDocumentCompletion(ctx context.Context, filename string, pos Position) ([]string, error) {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_completion

	opt := &CompletionParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.Context.TriggerKind = 1 // invoked
	opt.Position = pos

	result := CompletionList{}
	err := cli.Call(ctx, "textDocument/completion", &opt, &result)
	if err != nil {
		return nil, err
	}
	//logJson(result)

	res := []string{}
	for _, ci := range result.Items {
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
	return res, nil
}

//----------

func (cli *Client) SyncText(ctx context.Context, filename string, b []byte) error {
	v, ok := cli.fversions[filename]
	if !ok {
		v = 1
	} else {
		// TODO
		v++ // comment to use always same version
	}
	cli.fversions[filename] = v

	if v == 1 {
		err := cli.TextDocumentDidOpen(ctx, filename, string(b), v)
		if err != nil {
			return err
		}
	} else {
		err := cli.TextDocumentDidChange(ctx, filename, string(b), v)
		if err != nil {
			return err
		}
	}
	return nil
}
