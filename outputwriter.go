package main

import "io"

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

func (ow outputWriter) Close() {
	close(ow.codeChan)
	<-ow.finished
}
