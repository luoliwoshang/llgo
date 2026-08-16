//go:build 386 || amd64 || arm || arm64 || loong64 || mips64le || mipsle || ppc64le || riscv64 || wasm
// +build 386 amd64 arm arm64 loong64 mips64le mipsle ppc64le riscv64 wasm

package goarch

const (
	BigEndian    = false
	LittleEndian = true
)
