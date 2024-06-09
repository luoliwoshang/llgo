package cl_test

import (
	"testing"

	"github.com/goplus/llgo/cl/cltest"
)

// 测试初始化全局变量
func TestVarInit(t *testing.T) {
	cltest.FromFolder(t, "varinit", "./_testdata/varinit", false)
}

// 测试Untyped变量
func TestUntyped(t *testing.T) {
	cltest.FromFolder(t, "untyped", "./_testdata/varinit", false)
}

// 测试普通包下的函数定义，包级别的初始化
func TestApkg(t *testing.T) {
	cltest.FromFolder(t, "apkg", "./_testdata/apkg", false)
}

// 测试Main包下的函数定义，调用，初始化（main包的处理）
func TestFnCall(t *testing.T) {
	cltest.FromFolder(t, "fncall", "./_testdata/fncall", false)
}

// 测试导入包，并使用导入的包的函数
func TestImportPkg(t *testing.T) {
	cltest.FromFolder(t, "importpkg", "./_testdata/importpkg", false)
}

// 测试方法(method)的定义，调用
func TestMethod(t *testing.T) {
	cltest.FromFolder(t, "method", "./_testdata/method", false)
}

// TODO: 了解这里的print是否可以使用 测试runtime c.printf
func TestPrintf(t *testing.T) {
	cltest.FromFolder(t, "printf", "./_testdata/printf", false)
}

// 测试utf8，内置的runtime println
func TestUtf8(t *testing.T) {
	cltest.FromFolder(t, "utf8", "./_testdata/utf8", false)
}
