package main

import (
	"unsafe"

	"github.com/goplus/llgo/c"
)

func main() {
	var stack *int
	var length c.Uint
	locations := unsafe.Slice(stack, length)
	lens := len(locations)
	c.Printf(c.Str("len: %d\n"), lens)
	c.Printf(c.Str("len > 0: %d\n"), lens > 0)
}
