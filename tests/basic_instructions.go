package main

import "github.com/phaikawl/gojsen/test/null"
import n "github.com/phaikawl/gojsen/test/null/null"

var v int

func f(a int) (int, int) {
	return a, 2
}

func main() {
	v = 5
	z, a := f(v)
	for v == 9 {
		a = 6
	}
	println(a + v + z)
	null.Test()
	n.Test()
}
