package cl_test

import (
	"testing"

	"github.com/goplus/llgo/cl/cltest"
)

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
	cltest.FromFolder(t, "importpkg", "./_testdata/method", false)
}
