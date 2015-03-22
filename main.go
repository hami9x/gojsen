package main

import (
	"os"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

const TestFile = "test/test.go"

func main() {
	file := TestFile
	if len(os.Args) > 1 {
		file = os.Args[1]
	}
	outFile := file + ".js"

	outw, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}

	var conf loader.Config
	af, err := conf.ParseFile(file, nil)
	if err != nil {
		panic(err)
	}

	conf.CreateFromFiles("main", af)
	lprog, err := conf.Load()
	if err != nil {
		panic(err)
	}

	prog := ssa.Create(lprog, 0)
	prog.BuildAll()

	ow := newOutputWriter(outw)
	ow.Start()

	codeEmitter := newCodeEmitter(ow.codeChan)
	compiler := NewCompiler(codeEmitter)
	compiler.Compile(prog)

	ow.Close(outFile, outw)
}
