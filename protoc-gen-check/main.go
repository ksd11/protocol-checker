package main

import (
	"bytes"
	"fmt"
	"os"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/spf13/afero"
)

func main() {
	// pgs.Init(
	// 	pgs.DebugEnv("DEBUG"),
	// ).RegisterModule(
	// 	ASTPrinter(),
	// 	// JSONify(),
	// ).RegisterPostProcessor(
	// 	pgsgo.GoFmt(),
	// ).Render()
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Usage: protoc-gen-check [pb_bin]")
		return

	}
	req, err := os.Open(args[1])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	fs := afero.NewMemMapFs()
	res := &bytes.Buffer{}

	pgs.Init(
		pgs.ProtocInput(req),  // use the pre-generated request
		pgs.ProtocOutput(res), // capture CodeGeneratorResponse
		pgs.FileSystem(fs),    // capture any custom files written directly to disk
	).RegisterModule(ASTPrinter()).Render()
}
