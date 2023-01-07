package main

import (
	"encoding/json"
	"fmt"
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

func main() {
	command := exec.Command("/home/hayasaka/go/bin/gopls")
	stdin, _ := command.StdinPipe()
	defer stdin.Close()
	stdout, _ := command.StdoutPipe()
	defer stdout.Close()
	command.Start()
	initializedParams := p.InitializeParams{
		ProcessID: 1,
		RootPath:  "/home/hayasaka/go/src/goplstest",
		RootURI:   "file:///home/hayasaka/go/src/goplstest",
		//Capabilities: p.ClientCapabilities{TextDocument: &p.TextDocumentClientCapabilities{Completion: &p.CompletionTextDocumentClientCapabilities{
		//      CompletionItem: &p.CompletionTextDocumentClientCapabilitiesItem{},
		//}}},
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

	req := Request{
		JsonRpcVersion: "2.0",
		ID:             1,
		Method:         p.MethodInitialize,
		Params:         initializedParams,
	}

	b, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("error marshalling request: %v\n", err)
		return
	}

	stdin.Write([]byte(fmt.Sprintf("Content-Length: %v\r\n\r\n%s", len(b), b)))

	buff := make([]byte, 1024)
	n, err := stdout.Read(buff)

	contentLength, err := strconv.Atoi(strings.Split(strings.Split(string(buff[:n]), "Content-Length: ")[1], "\r\n")[0])
	if err != nil {
		fmt.Printf("missing Content-Length: %v\n", err)
		return
	}
	rawBody := make([]byte, contentLength)

	bodyIndex := copy(rawBody, strings.Split(string(buff[:n]), "\r\n\r\n")[1])

	n, err = stdout.Read(rawBody[bodyIndex:])

	if err != nil {
		fmt.Println("failed read body")
		return
	}
	var response Response
	if err := json.Unmarshal(rawBody, &response); err != nil {
		fmt.Printf("error Unmarshalling response: %v\n", err)
		return
	}

	fmt.Printf("response: %v\n", response)
}
