package main

import _ "unsafe"

func main() {
	beforeAlias()
	after()
	crossFile()
}

func before() {
	println("before")
}

//go:linkname beforeAlias main.before
func beforeAlias()

func after()

//go:linkname after main.afterImpl

//go:linkname crossFile main.crossFileImpl
