package main

var v int

func f(a int) (int, int) {
	return a, 2
}

func main() {
	v = 5
	z, a := f(v)
	if v == 9 {
		a = 6
	}
	println(a + v + z)
}
