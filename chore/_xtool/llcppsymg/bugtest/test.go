package main

import "errors"

func main() {
	syms := []string{"sym1"}
	m := make(map[string]bool)

	err := errors.New("failed to read symbol table")

	// 正常运行
	// if err != nil {
	// 	panic(err)
	// }
	// 错误运行（取消注释以模拟错误）
	check(err)

	for i := range syms {
		if _, ok := m[syms[i]]; ok {
			println(syms[i])
		}
	}

	defer println("bye")

	for _, sym := range syms {
		_ = sym
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
