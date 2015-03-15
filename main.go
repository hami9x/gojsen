package main

import (
	"os"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

const testFile = "test/test.go"
const testOut = "test/test.js"

func main() {
	outw, err := os.Create(testOut)
	if err != nil {
		panic(err)
	}

	defer outw.Close()

	var conf loader.Config
	af, err := conf.ParseFile(testFile, nil)
	if err != nil {
		panic(err)
	}

	conf.CreateFromFiles("main", af)
	lprog, err := conf.Load()
	if err != nil {
		panic(err)
	}

	prog := ssa.Create(lprog, ssa.NaiveForm)
	prog.BuildAll()

	ow := newOutputWriter(outw)
	ow.Start()

	codeWriter := &codeWriter{ow.codeChan}
	compiler := &Compiler{codeWriter}
	compiler.Compile(prog)

	ow.Close()
}
