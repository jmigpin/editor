package lsproto

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"strings"
	"time"

	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type Client struct {
	rcli      *rpc.Client
	conn      io.ReadWriteCloser
	fversions map[string]int
	reg       *Registration
}

func NewClientTCP(addr string, reg *Registration) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := NewClientIO(conn, reg)
	return cli, nil
}

func NewClientIO(conn io.ReadWriteCloser, reg *Registration) *Client {
	cli := &Client{reg: reg, fversions: map[string]int{}}
	cli.conn = conn
	cc := NewJsonCodec(conn)
	cc.OnNotificationMessage = cli.onNotificationMessage
	cli.rcli = rpc.NewClientWithCodec(cc)
	return cli
}

//----------

func (cli *Client) Close() error {
	me := iout.MultiError{}
	me.Add(cli.ShutdownRequest())
	me.Add(cli.ExitNotification())
	me.Add(cli.conn.Close())
	return me.Result()
}

//----------

// Ensures server callback or a timeout error will surface.
func (cli *Client) Call(method string, args, reply interface{}) error {
	lspResp := &Response{}
	fn := func() error {
		return cli.rcli.Call(method, args, lspResp)
	}
	err := chanutil.CallTimeout(context.Background(), 8*time.Second, method, cli.reg.asyncErrors, fn)
	if err != nil {
		go cli.reg.onConnErrAsync(err)
		return err
	}

	// not expecting a reply
	if _, ok := noreplyMethod(method); ok {
		return nil
	}

	// func error (soft error)
	if lspResp.Error != nil {
		return lspResp.Error
	}

	// decode result
	return decodeJsonRaw(lspResp.Result, reply)
}

//----------

func (cli *Client) onNotificationMessage(msg *NotificationMessage) {
	logJson("notification <--: ", msg)
}

//----------

func (cli *Client) Initialize(dir string) error {
	opt := &InitializeParams{}
	opt.RootUri = addFileScheme(dir)
	//opt.Capabilities.TextDocument.PublishDiagnostics = &PublishDiagnostics{
	//	RelatedInformation: false,
	//}

	var capabilities interface{}
	err := cli.Call("initialize", &opt, &capabilities)
	logJson("initialize <--: ", capabilities)
	return err
}

//----------

func (cli *Client) ShutdownRequest() error {
	// https://microsoft.github.io/language-server-protocol/specification#shutdown

	err := cli.Call("shutdown", nil, nil)
	return err
}

func (cli *Client) ExitNotification() error {
	// https://microsoft.github.io/language-server-protocol/specification#exit

	err := cli.Call("noreply:exit", nil, nil)
	return err
}

//----------

func (cli *Client) TextDocumentDidOpen(filename, text string, version int) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didOpen

	opt := &DidOpenTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.TextDocument.LanguageId = cli.reg.Language
	opt.TextDocument.Version = version
	opt.TextDocument.Text = text
	err := cli.Call("noreply:textDocument/didOpen", &opt, nil)
	return err
}

func (cli *Client) TextDocumentDidClose(filename string) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didClose

	opt := &DidCloseTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	err := cli.Call("noreply:textDocument/didClose", &opt, nil)
	return err
}

func (cli *Client) TextDocumentDidChange(filename, text string, version int) error {
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
	return cli.Call("noreply:textDocument/didChange", &opt, nil)
}

//----------

func (cli *Client) TextDocumentDefinition(filename string, pos Position) (*Location, error) {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_definition

	opt := &TextDocumentPositionParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.Position = pos

	result := []*Location{}
	err := cli.Call("textDocument/definition", &opt, &result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no results")
	}
	return result[0], nil // first result only
}

//----------

func (cli *Client) TextDocumentCompletion(filename string, pos Position) ([]string, error) {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_completion

	opt := &CompletionParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.Context.TriggerKind = 1 // invoked
	opt.Position = pos

	result := CompletionList{}
	err := cli.Call("textDocument/completion", &opt, &result)
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

func (cli *Client) SyncText(filename string, b []byte) error {
	v, ok := cli.fversions[filename]
	if !ok {
		v = 1
	} else {
		v++
	}
	cli.fversions[filename] = v

	//if v == 1 {
	err := cli.TextDocumentDidOpen(filename, string(b), v)
	if err != nil {
		return err
	}
	//} else {
	//	err := cli.TextDocumentDidChange(filename, string(b), v)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}
