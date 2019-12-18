package lsproto

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/pkg/errors"
)

type Client struct {
	rcli *rpc.Client
	rwc  io.ReadWriteCloser
	li   *LangInstance

	fversions map[string]int
	folders   []*WorkspaceFolder

	supportsWorkspaceUpdate bool
}

//----------

func NewClientTCP(ctx context.Context, addr string, li *LangInstance) (*Client, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := NewClientIO(conn, li)
	return cli, nil
}

//----------

func NewClientIO(rwc io.ReadWriteCloser, li *LangInstance) *Client {
	cli := &Client{li: li, fversions: map[string]int{}}

	cli.rwc = rwc

	cc := NewJsonCodec(rwc)
	cc.OnIOReadError = cli.onIOReadError
	cc.OnNotificationMessage = cli.onNotificationMessage
	cc.OnUnexpectedServerReply = cli.onUnexpectedServerReply
	go cc.ReadLoop()

	cli.rcli = rpc.NewClientWithCodec(cc)

	return cli
}

//----------

func (cli *Client) onIOReadError(err error) {
	_ = cli.li.lang.Close()
	cli.li.lang.ErrorAsync(err)
}

//----------

func (cli *Client) closeFromLangInstance() error {
	if cli == nil {
		return nil
	}

	me := iout.MultiError{}

	// best effort, ignore errors
	_ = cli.ShutdownRequest()
	_ = cli.ExitNotification()
	//me.Add(cli.ShutdownRequest())
	//me.Add(cli.ExitNotification())

	// possibly calls codec.close (which in turn calls rwc.close)
	me.Add(cli.rcli.Close())
	//if cli.rwc != nil {
	//	me.Add(cli.rwc.Close())
	//}

	return me.Result()
}

//----------

func (cli *Client) Call(ctx context.Context, method string, args, reply interface{}) error {
	lspResp := &Response{}
	fn := func() error {
		return cli.rcli.Call(method, args, lspResp)
	}
	lateFn := func(err error) {
		if err != nil {
			err = errors.Wrap(err, "call late")
			cli.li.lang.ErrorAsync(err)
		}
	}
	err := ctxutil.Call(ctx, method, fn, lateFn)
	if err != nil {
		err = errors.Wrap(err, "call")
		return cli.li.lang.WrapError(err)
	}

	// not expecting a reply
	if _, ok := noreplyMethod(method); ok {
		return nil
	}

	// soft error (rpc data with error content)
	if lspResp.Error != nil {
		return cli.li.lang.WrapError(lspResp.Error)
	}

	// decode result
	return decodeJsonRaw(lspResp.Result, reply)
}

//----------

func (cli *Client) onNotificationMessage(msg *NotificationMessage) {
	// Msgs like:
	// - a notification was sent to the srv, not expecting a reply, but it receives one because it was an error (has id)
	// {"error":{"code":-32601,"message":"method not found"},"id":2,"jsonrpc":"2.0"}

	//logJson("notification <--: ", msg)
}

func (cli *Client) onUnexpectedServerReply(resp *Response) {
	if resp.Error != nil {
		// json-rpc error codes: https://www.jsonrpc.org/specification
		report := false
		switch resp.Error.Code {
		case -32601: // method not found
			report = true
		case -32602: // invalid params
			report = true
			//case -32603: // internal error
			//report = true
		}
		if report {
			err := fmt.Errorf("id=%v, code=%v, msg=%q", resp.Id, resp.Error.Code, resp.Error.Message)
			cli.li.lang.ErrorAsync(err)
		}
	}
}

//----------

// Filename is the file that triggered the server to be started.
func (cli *Client) Initialize(ctx context.Context, filename string) error {
	//rootDir := "/" // (gopls: slow)
	//rootDir := "" // (gopls: fails with "rootUri=null")
	//rootDir := osutil.HomeEnvVar() // (gopls: slow)
	//rootDir := filepath.Dir(filename)
	// Use a non-existent dir and send an updateworkspacefolder on each request later. Attempt to prevent the lsp server to start looking at the user disk.
	rootDir := filepath.Join(os.TempDir(), "some-non-existent-dir---")

	opt := &InitializeParams{RootUri: nil}
	if rootDir != "" {
		s := addFileScheme(rootDir)
		opt.RootUri = &s
	}
	opt.Capabilities = &ClientCapabilities{
		//Workspace: &WorkspaceClientCapabilities{
		//	WorkspaceFolders: true,
		//},
		//TextDocument: &TextDocumentClientCapabilities{
		//	PublishDiagnostics: &PublishDiagnostics{
		//		RelatedInformation: false,
		//	},
		//},
	}

	logJson("opt -->: ", opt)
	var serverCapabilities interface{}
	err := cli.Call(ctx, "initialize", &opt, &serverCapabilities)
	if err != nil {
		return err
	}
	logJson("initialize <--: ", serverCapabilities)

	// keep track of some capabilities
	path := "capabilities.workspace.workspaceFolders.supported"
	v, err := JsonGetPath(serverCapabilities, path)
	if err == nil {
		if b, ok := v.(bool); ok && b == true {
			cli.supportsWorkspaceUpdate = true
		}
	}

	// send "initialized" (gopls: "no views" error without this)
	opt2 := &InitializedParams{}
	err2 := cli.Call(ctx, "noreply:initialized", &opt2, nil)
	return err2
}

//----------

func (cli *Client) ShutdownRequest() error {
	// https://microsoft.github.io/language-server-protocol/specification#shutdown

	// TODO: shutdown request should expect a reply
	// * clangd is sending a reply (ok)
	// * gopls is not sending a reply (NOT OK)

	// best effort, impose timeout
	ctx := context.Background()
	ctx2, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	ctx = ctx2

	err := cli.Call(ctx, "shutdown", nil, nil)
	return err
}

func (cli *Client) ExitNotification() error {
	// https://microsoft.github.io/language-server-protocol/specification#exit

	ctx := context.Background()
	err := cli.Call(ctx, "noreply:exit", nil, nil)
	return err
}

//----------

func (cli *Client) TextDocumentDidOpen(ctx context.Context, filename, text string, version int) error {
	// https://microsoft.github.io/language-server-protocol/specification#textDocument_didOpen

	opt := &DidOpenTextDocumentParams{}
	opt.TextDocument.Uri = addFileScheme(filename)
	opt.TextDocument.LanguageId = cli.li.lang.Reg.Language
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

func (cli *Client) TextDocumentDidOpenVersion(ctx context.Context, filename string, b []byte) error {
	v, ok := cli.fversions[filename]
	if !ok {
		v = 1
	} else {
		v++
	}
	cli.fversions[filename] = v
	return cli.TextDocumentDidOpen(ctx, filename, string(b), v)
}

//----------

func (cli *Client) WorkspaceDidChangeWorkspaceFolders(ctx context.Context, added, removed []*WorkspaceFolder) error {
	opt := &DidChangeWorkspaceFoldersParams{}
	opt.Event = &WorkspaceFoldersChangeEvent{}
	opt.Event.Added = added
	opt.Event.Removed = removed
	err := cli.Call(ctx, "noreply:workspace/didChangeWorkspaceFolders", &opt, nil)
	return err
}

func (cli *Client) UpdateWorkspaceFolder(ctx context.Context, dir string) error {
	if !cli.supportsWorkspaceUpdate {
		return nil
	}

	removed := cli.folders
	cli.folders = []*WorkspaceFolder{{Uri: addFileScheme(dir)}}
	return cli.WorkspaceDidChangeWorkspaceFolders(ctx, cli.folders, removed)
}

//----------

//func (cli *Client) SyncText(ctx context.Context, filename string, b []byte) error {
//	v, ok := cli.fversions[filename]
//	if !ok {
//		v = 1
//	} else {
//		v++
//	}
//	cli.fversions[filename] = v

//	// close before opening. Keeps open/close balanced since not using "didchange", while needing to update the src.
//	if v > 1 {
//		err := cli.TextDocumentDidClose(ctx, filename)
//		if err != nil {
//			return err
//		}
//	}
//	// send latest version of the document
//	err := cli.TextDocumentDidOpen(ctx, filename, string(b), v)
//	if err != nil {
//		return err
//	}

//	// TODO: clangd doesn't work well with didchange (works with sending always a didopen)
//	//} else {
//	//	err := cli.TextDocumentDidChange(ctx, filename, string(b), v)
//	//	if err != nil {
//	//		return err
//	//	}
//	//}
//	return nil
//}

//----------

func JsonGetPath(v interface{}, path string) (interface{}, error) {
	args := strings.Split(path, ".")
	return jsonGetPath2(v, args)
}

// TODO: incomplete
func jsonGetPath2(v interface{}, args []string) (interface{}, error) {
	switch t := v.(type) {
	case map[string]interface{}:
		if len(args) > 0 {
			a := args[0]
			if v, ok := t[a]; ok {
				return jsonGetPath2(v, args[1:])
			}
		}
	case bool, int, float32, float64:
		return t, nil
	}
	return nil, fmt.Errorf("not found: %v", args[0])
}
