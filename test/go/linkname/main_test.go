package main

import (
	"testing"
	_ "unsafe"
)

func TestLinknameFromSameTestPackage(t *testing.T) {
	if got := demo(); got != 42 {
		t.Fatalf("same-package linkname = %d, want 42", got)
	}
}

func demo3() int {
	return 42
}

//go:linkname demo4 github.com/xgo-dev/llgo/test/go/linkname.demo2
func demo4() int

//go:linkname demo5 github.com/xgo-dev/llgo/test/go/linkname.demo3
func demo5() int

func TestLinknameToMainPackage(t *testing.T) {
	if got := demo4(); got != 43 {
		t.Fatalf("test-to-main linkname = %d, want 43", got)
	}
	if got := demo5(); got != 42 {
		t.Fatalf("test-to-main linkname = %d, want 42", got)
	}
}
