package main

import _ "unsafe"

func main() {
	main_demo()
}

func demo() int {
	return 42
}

func demo2() int {
	return 43
}

//go:linkname main_demo main.demo
func main_demo() int
