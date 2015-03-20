package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

type cnType string

const (
	Normal     cnType = "Normal"
	BlockOpen         = "BlockOpen"
	BlockClose        = "BlockClose"
)

const (
	TabStr       = "    "
	EndStatement = ";\n"
)

type codeNode struct {
	code string
	typ  cnType
}

type outputWriter struct {
	w        io.Writer
	finished chan bool
	codeChan chan *codeNode
}

func newOutputWriter(w io.Writer) outputWriter {
	return outputWriter{
		w:        w,
		finished: make(chan bool),
		codeChan: make(chan *codeNode, 200),
	}
}

func (ow outputWriter) Start() {
	go func() {
		tabs := []rune{}
		spaceRune := []rune("      ")
		indent := 0
		for {
			cn := <-ow.codeChan
			if cn == nil {
				break
			}

			if cn.typ == BlockClose {
				indent--
			}

			indentation := string(tabs[0 : indent*len(TabStr)])

			ow.w.Write([]byte(indentation + cn.code))
			ow.w.Write([]byte("\n"))
			if cn.typ == BlockClose {
				ow.w.Write([]byte("\n"))
			}

			if cn.typ == BlockOpen {
				indent++
				if indent*len(TabStr) > len(tabs) {
					tabs = append(tabs, spaceRune...)
				}
			}
		}

		ow.finished <- true
	}()
}

func (ow outputWriter) Close(testFile string, foutw io.Closer) {
	close(ow.codeChan)
	<-ow.finished
	foutw.Close()

	cmd := exec.Command("uglifyjs", "-b", "-nm", "-ns", "--no-dead-code", testFile)
	b, err := cmd.CombinedOutput()
	fmt.Printf("%v\n", string(b))
	if err != nil {
		log.Fatal(err)
	}
}
