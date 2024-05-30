package cl_test

import (
	"testing"

	"github.com/goplus/llgo/cl/cltest"
)

func TestApkg(t *testing.T) {
	cltest.FromFolder(t, "apkg", "./_testdata/apkg", false)
}
