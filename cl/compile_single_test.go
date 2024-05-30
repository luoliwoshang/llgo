package cl_test

import (
	"testing"

	"github.com/goplus/llgo/cl/cltest"
)

// 测试普通包下的函数定义，包级别的初始化
// func TestApkg(t *testing.T) {
// 	cltest.FromFolder(t, "apkg", "./_testdata/apkg", false)
// }

// 测试Main包下的函数定义，调用，初始化（main包的处理）
func TestFnCall(t *testing.T) {
	cltest.FromFolder(t, "fncall", "./_testdata/fncall", false)
}
