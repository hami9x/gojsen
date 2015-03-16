package main

var v = f()
var b bool

func f() int {
	return 1
}

func main() {
	b = false
	a := 0
	println(a + v)
}
