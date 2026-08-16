//go:build llgo && wasip1

package main

// The build-cache WASM suite runs under iwasm, which does not provide the
// setjmp/longjmp imports needed by LLGo's defer implementation. Native cache
// runs exercise the cross-package recover path instead.
func verifyRecoverCache() {}
