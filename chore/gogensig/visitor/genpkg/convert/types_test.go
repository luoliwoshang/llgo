package convert_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/convert"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

func TestBasicLitOK(t *testing.T) {
	t.Parallel()
	type CaseType[T any] struct {
		name string
		expr ast.Expr
		want T
	}
	type CaseTypeSlice[T any] []CaseType[T]
	testCases := CaseTypeSlice[any]{
		{
			name: "ToInt",
			expr: &ast.BasicLit{Kind: ast.IntLit, Value: "123"},
			want: 123,
		},
		{
			name: "ToFloat",
			expr: &ast.BasicLit{Kind: ast.FloatLit, Value: "123.123"},
			want: 123.123,
		},
		{
			name: "ToString",
			expr: &ast.BasicLit{Kind: ast.StringLit, Value: "abcd"},
			want: "abcd",
		},
		/* todo
		{
			name: "ToChar",
			expr: &ast.BasicLit{Kind: ast.CharLit, Value: "98"},
			want: 98,
		},*/
	}

	for _, tc := range testCases {
		t.Run("convert "+tc.name, func(t *testing.T) {
			if tc.name == "ToInt" {
				result, err := convert.NewExpr(tc.expr).ToInt()
				CheckResult(t, result, err, tc.want)
			} else if tc.name == "ToFloat" {
				result, err := convert.NewExpr(tc.expr).ToFloat(64)
				CheckResult(t, result, err, tc.want)
			} else if tc.name == "ToChar" {
				result, err := convert.NewExpr(tc.expr).ToChar()
				CheckResult(t, result, err, tc.want)
			} else if tc.name == "ToString" {
				result, err := convert.NewExpr(tc.expr).ToString()
				CheckResult(t, result, err, tc.want)
			}
		})
	}
}

func CheckResult(t *testing.T, result any, err error, want any) {
	t.Helper()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(result, want) {
		t.Error(cmp.Diff(result, want))
	}
}
