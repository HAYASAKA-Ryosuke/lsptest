package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	p "go.lsp.dev/protocol"
)

type Request struct {
	JsonRpcVersion string `json:"jsonrpc"`
	ID             int    `json:"id"`

	Method string `json:"method"`

	Params interface{} `json:"params,omitempty"`
}

type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type Lsp struct {
	Command *exec.Cmd
	Writer  io.WriteCloser
	Reader  io.ReadCloser
}

func NewLsp(lspServerPath string) *Lsp {
	command := exec.Command(lspServerPath)
	stdin, _ := command.StdinPipe()
	stdout, _ := command.StdoutPipe()
	return &Lsp{Command: command, Writer: stdin, Reader: stdout}
}

func (l *Lsp) Init(rootPath string) {
	initializedParams := p.InitializeParams{
		ProcessID: 1,
		RootPath:  rootPath,
		//RootURI:   "file://" + rootPath,
		Capabilities: p.ClientCapabilities{
			TextDocument: &p.TextDocumentClientCapabilities{
				Completion: &p.CompletionTextDocumentClientCapabilities{
					CompletionItem: &p.CompletionTextDocumentClientCapabilitiesItem{},
				},
			},
			Window: &p.WindowClientCapabilities{},
			Workspace: &p.WorkspaceClientCapabilities{
				CodeLens: &p.CodeLensWorkspaceClientCapabilities{
					RefreshSupport: true,
				},
				SemanticTokens: &p.SemanticTokensWorkspaceClientCapabilities{
					RefreshSupport: true,
				},
				WorkspaceFolders: true,
				WorkspaceEdit: &p.WorkspaceClientCapabilitiesWorkspaceEdit{
					DocumentChanges: true,
				},
			},
		},
	}

	l.Command.Start()
	l.sendCommand(1, p.MethodInitialize, initializedParams)
}

func (l *Lsp) Close() {
	l.Reader.Close()
	l.Writer.Close()
}

func (l *Lsp) sendCommand(id int, method string, params interface{}) *Response {
	request := Request{
		JsonRpcVersion: "2.0",
		ID:             id,
		Method:         method,
		Params:         params,
	}

	b, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("error marshalling request: %v\n", err)
		return nil
	}

	l.Writer.Write([]byte(fmt.Sprintf("Content-Length: %v\r\n\r\n%s", len(b), b)))

	buff := make([]byte, 1024)
	n, err := l.Reader.Read(buff)

	contentLength, err := strconv.Atoi(strings.Split(strings.Split(string(buff[:n]), "Content-Length: ")[1], "\r\n")[0])
	if err != nil {
		fmt.Printf("missing Content-Length: %v\n", err)
		return nil
	}
	rawBody := make([]byte, contentLength)

	bodyIndex := copy(rawBody, strings.Split(string(buff[:n]), "\r\n\r\n")[1])

	n, err = l.Reader.Read(rawBody[bodyIndex:])

	if err != nil {
		fmt.Println("failed read body")
		return nil
	}
	var response Response
	if err := json.Unmarshal(rawBody, &response); err != nil {
		fmt.Printf("error Unmarshalling response: %v\n", err)
		return nil
	}

	fmt.Printf("response: %v\n", response)
	return &response
}
