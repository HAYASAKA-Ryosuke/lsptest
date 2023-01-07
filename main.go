package main

func main() {
	path := "/home/hayasaka/go/src/lsptest"
	lsp := NewLsp("/home/hayasaka/go/bin/gopls")
	lsp.Init(path)
	lsp.DidOpen("/home/hayasaka/go/src/lsptest/test.go")
	lsp.Completion("/home/hayasaka/go/src/lsptest/test.go", 5, 2)
}
