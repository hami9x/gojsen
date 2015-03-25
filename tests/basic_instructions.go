package main

import "github.com/phaikawl/gojsen/test/null"
import n "github.com/phaikawl/gojsen/test/null/null"

var t0 int

func f(a int) (int, int) {
	return a, 2
}

func main() {
	t0 = 5
	z, a := f(t0)
	for t0 == 9 {
		a = 6
	}
	println(a + t0 + z)
	null.Test()
	n.Test()
}
