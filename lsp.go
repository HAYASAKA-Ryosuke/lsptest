package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os/exec"
	"path/filepath"
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
	Id      int
	Command *exec.Cmd
	Writer  io.WriteCloser
	Reader  io.ReadCloser
}

func NewLsp(lspServerPath string) *Lsp {
	fmt.Println(lspServerPath)
	command := exec.Command("/home/hayasaka/go/bin/gopls", "-rpc.trace")
	stdin, _ := command.StdinPipe()
	stdout, _ := command.StdoutPipe()
	command.Start()
	return &Lsp{Id: 1321, Command: command, Writer: stdin, Reader: stdout}
}

func (l *Lsp) Close() {
	l.Reader.Close()
	l.Writer.Close()
}

func (l *Lsp) Init(rootPath string) {
	initializedParams := p.InitializeParams{
		ProcessID: int32(l.Id),
		RootPath:  rootPath,
		RootURI:   getURI(rootPath),
		Capabilities: p.ClientCapabilities{
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

	l.sendCommand(l.Id, p.MethodInitialize, initializedParams)

	l.sendCommand(l.Id, p.MethodInitialized, map[string]interface{}{})
}

func (l *Lsp) DidOpen(filePath string) *Response {

	uri := getURI(filePath)

	text, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	didOpenParams := p.DidOpenTextDocumentParams{
		TextDocument: p.TextDocumentItem{
			Text:       string(text),
			Version:    1,
			LanguageID: "go",
			URI:        uri,
		},
	}
	result := l.sendCommand(
		l.Id,
		p.MethodTextDocumentDidOpen,
		didOpenParams,
	)
	return result
}

func (l *Lsp) DocumentSymbol(filePath string) *Response {
	params := p.DocumentSymbolParams{TextDocument: p.TextDocumentIdentifier{URI: getURI(filePath)}}
	result := l.sendCommand(l.Id, p.MethodTextDocumentDocumentSymbol, params)
	return result
}

func (l *Lsp) Completion(filePath string, row uint32, col uint32) *p.CompletionList {
	params := p.CompletionParams{
		TextDocumentPositionParams: p.TextDocumentPositionParams{
			Position:     p.Position{Line: row, Character: col},
			TextDocument: p.TextDocumentIdentifier{URI: getURI(filePath)},
		},
		Context: &p.CompletionContext{
			TriggerKind: 1,
		},
	}
	response := l.sendCommand(l.Id, p.MethodTextDocumentCompletion, params)
	b, err := json.Marshal(response.Result)
	if err != nil {
		return nil
	}
	var result p.CompletionList
	err = json.Unmarshal(b, &result)
	if err != nil {
		fmt.Println(err)
	}
	return &result
}

func (l *Lsp) Shutdown() error {
	response := l.sendCommand(l.Id, p.MethodShutdown, map[string]interface{}{})
	if response.Error != nil {
		return errors.New("shutdown error")
	}
	return nil
}

func getURI(filePath string) p.DocumentURI {
	path := filepath.Clean(filePath)
	path = filepath.ToSlash(path)
	volume := filepath.VolumeName(path)
	if strings.HasSuffix(volume, ":") {
		path = "/" + path
	}

	u := &url.URL{
		Scheme: "file",
		Path:   path,
	}
	uri := p.DocumentURI(u.String())
	return uri
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

	for {
		buff := make([]byte, 64)
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

		// 異なるIDの通信は無視
		if response.ID != l.Id {
			continue
		}
		return &response
	}
}
