package main

import (
	_ "unsafe"

	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/lua"
)

func main() {
	L := lua.Newstate__1()
	defer L.Close()
	L.Openlibs()
	if res := L.Dostring(c.Str("print('hello world')")); res != lua.OK {
		println("error")
	}
}

/* Expected output:
hello world
*/
