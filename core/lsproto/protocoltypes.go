package lsproto

import (
	"encoding/json"
	"fmt"
	"strings"
)

//----------

type RequestMessage struct {
	JsonRpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type Response struct {
	ResponseMessage
	NotificationMessage
}
type ResponseMessage struct {
	Id     int             `json:"id,omitempty"` // id can be zero on first msg
	Error  *ResponseError  `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}
type NotificationMessage struct {
	Method string      `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

func (nm *NotificationMessage) isServerPush() bool {
	return nm.Method != ""
}

//----------

type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (e *ResponseError) Error() string {
	// extra strings
	v := []string{}
	if e.Code != 0 {
		v = append(v, fmt.Sprintf("code=%v", e.Code))
	}
	if e.Data != nil {
		v = append(v, fmt.Sprintf("data=%v", e.Data))
	}
	vs := ""
	if len(v) > 0 {
		vs = fmt.Sprintf("(%v)", strings.Join(v, ", "))
	}

	return fmt.Sprintf("%v%v", e.Message, vs)
}

//----------

type InitializeParams struct {
	RootUri          string              `json:"rootUri,omitempty"`
	Capabilities     *ClientCapabilities `json:"capabilities,omitempty"`
	WorkspaceFolders []*WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

type InitializedParams struct {
	None bool `json:"none"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

type TextDocumentClientCapabilities struct {
	PublishDiagnostics *PublishDiagnostics `json:"publishDiagnostics,omitempty"`
}
type PublishDiagnostics struct {
	RelatedInformation bool `json:"relatedInformation"`
}

type WorkspaceClientCapabilities struct {
	WorkspaceFolders bool `json:"workspaceFolders"`
}

type WorkspaceFolder struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}
type TextDocumentIdentifier struct {
	Uri string `json:"uri"`
}
type Location struct {
	Uri   string `json:"uri,omitempty"`
	Range *Range `json:"range,omitempty"`
}
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}
type CompletionParams struct {
	TextDocumentPositionParams
	Context CompletionContext `json:"context"`
}
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"` // 1=invoked, 2=char, 3=re-trigger
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Deprecated    bool   `json:"deprecated,omitempty"`
}
type CompletionList struct {
	IsIncomplete bool              `json:"isIncomplete"`
	Items        []*CompletionItem `json:"items"`
}
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
type TextDocumentItem struct {
	Uri        string `json:"uri"`
	LanguageId string `json:"languageId,omitempty"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier   `json:"textDocument,omitempty"`
	ContentChanges []*TextDocumentContentChangeEvent `json:"contentChanges,omitempty"`
}
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}
type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text,omitempty"`
}

//----------

type Position struct {
	Line      int `json:"line"`      // zero based
	Character int `json:"character"` // zero based
}

//----------
