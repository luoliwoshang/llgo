package main_test

import (
	"testing"
	_ "unsafe"
)

//go:linkname demo github.com/xgo-dev/llgo/test/go/linkname.demo
func demo() int

func xdemo1() int {
	return 44
}

//go:linkname xdemo2 github.com/xgo-dev/llgo/test/go/linkname_test.xdemo1
func xdemo2() int

func TestLinknameFromExternalTestPackage(t *testing.T) {
	if got := demo(); got != 42 {
		t.Fatalf("external-test-to-main linkname = %d, want 42", got)
	}
	if got := xdemo2(); got != 44 {
		t.Fatalf("external-test self linkname = %d, want 44", got)
	}
}
