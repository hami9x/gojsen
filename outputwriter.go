package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

type outputWriter struct {
	w        io.Writer
	finished chan bool
	codeChan chan string
}

func newOutputWriter(w io.Writer) outputWriter {
	return outputWriter{
		w:        w,
		finished: make(chan bool),
		codeChan: make(chan string),
	}
}

func (ow outputWriter) Start() {
	go func() {
		for {
			s := <-ow.codeChan
			if s == "" {
				break
			}

			ow.w.Write([]byte(s))
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
