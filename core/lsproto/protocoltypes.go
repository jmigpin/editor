package lsproto

import (
	"encoding/json"
	"fmt"
	"strings"
)

// lsp protocol: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/

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
	Method string `json:"method,omitempty"`

	//Params any                        `json:"params,omitempty"`
	Params _notificationMessageParams `json:"params,omitempty"`
}

//----------

// The reason to have this struct is that there was an error when using godebug that was only getting reported through a logmessageparams notification. This resulted in a different result when not using godebug, causing confusion. The error is connected to the build. Because of the error, the completionitem action would not fail, but report zero completions. While with the regular compiler it would fail showing an issue with a json field that was not declared in one of the structs.
type _notificationMessageParams struct {
	lmp *LogMessageParams
	any any
}

func (nmp *_notificationMessageParams) MarshalJSON() ([]byte, error) {
	if nmp.lmp != nil {
		json.Marshal(nmp.lmp)
	}
	return json.Marshal(nmp.any)
}
func (nmp *_notificationMessageParams) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &nmp.lmp); err == nil {
		return nil
	}
	nmp.any = string(b)
	return nil
}

//----------

type LogMessageParams struct {
	Type    MessageType `json:"type,omitempty"`
	Message string      `json:"message,omitempty"`
}
type MessageType int

//----------

type Response struct {
	*ResponseMessage
	*NotificationMessage
}

func (res *Response) IsNotification() bool {
	return res.NotificationMessage != nil
}

//----------

type ResponseMessage struct {
	Message
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

//----------

type CompletionParams struct {
	TextDocumentPositionParams
	Context CompletionContext `json:"context"`
}
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"` // 1=invoked, 2=char, 3=re-trigger
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}
type CompletionList struct {
	IsIncomplete bool              `json:"isIncomplete"`
	Items        []*CompletionItem `json:"items"`
}
type CompletionItem struct {
	Label         string                       `json:"label"`
	Kind          CompletionItemKind           `json:"kind,omitempty"`
	Detail        string                       `json:"detail,omitempty"`
	Documentation _completionItemDocumentation `json:"documentation,omitempty"`
	Deprecated    bool                         `json:"deprecated,omitempty"` // deprecated in favor of "tags"
	Tags          []CompletionItemTag          `json:"tags,omitempty"`
}
type CompletionItemKind int
type CompletionItemTag int

//----------

type _completionItemDocumentation struct {
	mc  *MarkupContent
	str *string
}

func (u *_completionItemDocumentation) MarshalJSON(b []byte) ([]byte, error) {
	if u.mc != nil {
		return json.Marshal(u.mc)
	}
	return json.Marshal(u.str)
}
func (u *_completionItemDocumentation) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &u.mc); err == nil {
		return nil
	}
	return json.Unmarshal(b, &u.str)
}

//----------

type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}
type MarkupKind string // ex: plaintext, markup

//----------

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

//----------

type WorkspaceEdit struct {
	Changes         map[DocumentUri][]*TextEdit `json:"changes,omitempty"`
	DocumentChanges []*TextDocumentEdit         `json:"documentChanges,omitempty"`
}

func (we *WorkspaceEdit) GetChanges() ([]*WorkspaceEditChange, error) {
	w := []*WorkspaceEditChange{}
	if len(w) == 0 && len(we.Changes) != 0 {
		for url, edits := range we.Changes {
			filename, err := UrlToAbsFilename(string(url))
			if err != nil {
				return nil, err
			}
			wec := &WorkspaceEditChange{filename, edits}
			w = append(w, wec)
		}
	}
	if len(w) == 0 && len(we.DocumentChanges) != 0 {
		for _, tde := range we.DocumentChanges {
			filename, err := UrlToAbsFilename(string(tde.TextDocument.Uri))
			if err != nil {
				return nil, err
			}
			wec := &WorkspaceEditChange{filename, tde.Edits}
			w = append(w, wec)
		}
	}
	return w, nil
}

//----------

type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []*TextEdit                     `json:"edits"`
}
type TextEdit struct {
	Range   *Range `json:"range"`
	NewText string `json:"newText"`
}

//----------

type CallHierarchyPrepareParams struct {
	TextDocumentPositionParams
}

// Commented: here for doc only; using the unified/simplified version below

//type CallHierarchyIncomingCallsParams struct { // request
//	Item *CallHierarchyItem `json:"item"`
//}
//type CallHierarchyIncomingCall struct { // response
//	From       *CallHierarchyItem `json:"from"`
//	FromRanges []*Range           `json:"fromRanges"`
//}

//type CallHierarchyOutgoingCallsParams struct { // request
//	Item *CallHierarchyItem `json:"item"`
//}
//type CallHierarchyOutgoingCall struct { // response
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
	Data           interface{}  `json:"data,omitempty"` // optional (related to prepare calls)
}

//----------

type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type Position struct {
	Line      int `json:"line"`      // zero based
	Character int `json:"character"` // zero based
}

func (pos *Position) OneBased() (int, int) {
	return pos.Line + 1, pos.Character + 1
}

type DocumentUri string
type SymbolKind int
type SymbolTag int

//----------
//----------
//----------

// Not part of the protocol, used to unify/simplify
type CallHierarchyCallType int

const (
	IncomingChct CallHierarchyCallType = iota
	OutgoingChct
)

//----------

// Not part of the protocol, used to unify/simplify
type WorkspaceEditChange struct {
	Filename string
	Edits    []*TextEdit
}
