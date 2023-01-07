package main

func main() {
	path := "/home/hayasaka/go/src/lsptest"
	lsp := NewLsp()
	lsp.Init(path)
}
