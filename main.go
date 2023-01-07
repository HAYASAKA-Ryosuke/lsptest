package main

func main() {
	path := "/home/hayasaka/go/src/lsptest"
	lsp := NewLsp("/home/hayasaka/go/bin/gopls")
	lsp.Init(path)
}
