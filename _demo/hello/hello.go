package main

import (
	"github.com/goplus/llgo/c"
)

func main() {
	// fmt.Print("dsadas")
	c.Printf(c.Str("Hello world\n"))
}

/* Expected output:
Hello World
*/
