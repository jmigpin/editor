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
	rcli      *rpc.Client
	rwc       io.ReadWriteCloser
	fversions map[string]int
	reg       *Registration
}

func NewClientTCP(ctx context.Context, addr string, reg *Registration) (*Client, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
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
	cc.OnIOReadError = cli.reg.onIOReadError
	cc.OnNotificationMessage = cli.onNotificationMessage

	cli.rcli = rpc.NewClientWithCodec(cc)

	reg.cs.mc.Add(cli) // multiclose
	return cli
}

//----------

func (cli *Client) Close() error {
	me := iout.MultiError{}

	// best effort, ignore errors
	_ = cli.ShutdownRequest()
	_ = cli.ExitNotification()
	//me.Add(cli.ShutdownRequest())
	//me.Add(cli.ExitNotification())

	if cli.rcli != nil {
		// possibly also calls codec.close (which in turn calls rwc.close)
		me.Add(cli.rcli.Close())
	}
	if cli.rwc != nil {
		me.Add(cli.rwc.Close())
	}

	me.Add(cli.reg.cs.mc.CloseRest(cli)) // multiclose

	return me.Result()
}

//----------

func (cli *Client) Call(ctx context.Context, method string, args, reply interface{}) error {

	// TEMPORARY: all calls must complete under X time
	// TODO: ensure timeout (at least for now while developing)
	//ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	//go func() {
	//	<-ctx2.Done()
	//	if ctx2.Err() != nil {
	//		err := cli.Close()
	//		if err != nil {
	//			err2 := errors.Wrap(err, "call timeout")
	//			err3 := cli.reg.WrapError(err2)
	//			cli.reg.man.errorAsync(err3)
	//		}
	//	}
	//}()
	//ctx = ctx2

	lspResp := &Response{}
	fn := func() error {
		return cli.rcli.Call(method, args, lspResp)
	}
	lateFn := func(err error) {
		//defer cancel() // clear resources

		if err != nil {
			err2 := errors.Wrap(err, "call late")
			err3 := cli.reg.WrapError(err2)
			cli.reg.man.errorAsync(err3)
		}
	}
	err := ctxutil.Call(ctx, method, fn, lateFn)
	if err != nil {
		err2 := errors.Wrap(err, "call")
		err3 := cli.reg.WrapError(err2)
		return err3
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
	//logJson("notification <--: ", msg)
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
	//logJson("initialize <--: ", capabilities)
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
		{
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
		// Commented: don't increase version, just update. Possibly prevents the server from keeping multiple versions, wasting memory.
		//v++
	}
	cli.fversions[filename] = v

	// TODO: clangd doesn't work well with didchange (works with sending always a didopen)

	//if v == 1 {
	err := cli.TextDocumentDidOpen(ctx, filename, string(b), v)
	if err != nil {
		return err
	}
	//} else {
	//	err := cli.TextDocumentDidChange(ctx, filename, string(b), v)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}
