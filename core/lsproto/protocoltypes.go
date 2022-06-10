package lsproto

import (
	"encoding/json"
	"fmt"
	"strings"
)

//----------

type Message struct {
	JsonRpc string `json:"jsonrpc"`
}

func MakeMessage() Message {
	return Message{JsonRpc: "2.0"}
}

//----------

type RequestMessage struct {
	Message
	Id     int         `json:"id"`
	Method string      `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

//----------

// Used as request and response (sent/received).
type NotificationMessage struct {
	Message
	Method string      `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

//----------

type Response struct {
	ResponseMessage
	NotificationMessage
}

func (res *Response) IsNotification() bool {
	return res.NotificationMessage.Method != ""
}

type ResponseMessage struct {
	//Message // commented: not used and avoid clash with definition at notificationmessage (works if defined though)
	Id     int             `json:"id,omitempty"` // id can be zero on first msg
	Error  *ResponseError  `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
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

type WorkspaceFolder struct {
	Uri  DocumentUri `json:"uri"`
	Name string      `json:"name"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}
type TextDocumentIdentifier struct {
	Uri DocumentUri `json:"uri"`
}
type Location struct {
	Uri   DocumentUri `json:"uri,omitempty"`
	Range *Range      `json:"range,omitempty"`
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
	Uri        DocumentUri `json:"uri"`
	LanguageId string      `json:"languageId,omitempty"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
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
	Version *int `json:"version"`
}
type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text,omitempty"`
}

type DidChangeWorkspaceFoldersParams struct {
	Event *WorkspaceFoldersChangeEvent `json:"event,omitempty"`
}
type WorkspaceFoldersChangeEvent struct {
	Added   []*WorkspaceFolder `json:"added"`
	Removed []*WorkspaceFolder `json:"removed"`
}

type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

type WorkspaceEdit struct {
	Changes         map[DocumentUri][]*TextEdit `json:"changes,omitempty"`
	DocumentChanges []*TextDocumentEdit         `json:"documentChanges,omitempty"`
}

type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []*TextEdit                     `json:"edits"`
}
type TextEdit struct {
	Range   *Range `json:"range"`
	NewText string `json:"newText"`
}

type CallHierarchyPrepareParams struct {
	TextDocumentPositionParams
}

// Commented: here for doc only; using the unified/simplified version below
//type CallHierarchyIncomingCallsParams struct {
//	Item *CallHierarchyItem `json:"item"`
//}
//type CallHierarchyOutgoingCallsParams struct {
//	Item *CallHierarchyItem `json:"item"`
//}
//type CallHierarchyIncomingCall struct {
//	From       *CallHierarchyItem `json:"from"`
//	FromRanges []*Range           `json:"fromRanges"`
//}
//type CallHierarchyOutgoingCall struct {
//	To         *CallHierarchyItem `json:"to"`
//	FromRanges []*Range           `json:"fromRanges"`
//}

type CallHierarchyCallsParams struct { // used in Incoming/Outgoing
	Item *CallHierarchyItem `json:"item"`
}
type CallHierarchyCall struct { // used in Incoming/Outgoing
	From       *CallHierarchyItem `json:"from,omitempty"` // incoming
	To         *CallHierarchyItem `json:"to,omitempty"`   // outgoing
	FromRanges []*Range           `json:"fromRanges"`
}

func (chc *CallHierarchyCall) Item() *CallHierarchyItem {
	if chc.From != nil {
		return chc.From
	}
	return chc.To
}

type CallHierarchyItem struct {
	Name           string       `json:"name"`
	Kind           SymbolKind   `json:"kind"`
	Tags           []*SymbolTag `json:"tags,omitempty"` // optional
	Detail         string       `json:"detail"`         // optional
	Uri            DocumentUri  `json:"uri"`
	Range          *Range       `json:"range"`
	SelectionRange *Range       `json:"selectionRange"`
	data           interface{}  `json:"data,omitempty"` // optional (related to prepare calls)
}

type Position struct {
	Line      int `json:"line"`      // zero based
	Character int `json:"character"` // zero based
}

type DocumentUri string
type SymbolKind int
type SymbolTag int

//----------

// Not part of the protocol, used to unify/simplify
type CallHierarchyCallType int

const (
	IncomingChct CallHierarchyCallType = iota
	OutgoingChct
)
