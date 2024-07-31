package main

import "errors"

func main() {
	syms := []int{}
	m := make(map[int]bool) // 添加 map

	err := errors.New("failed to read symbol table")

	check(err)

	// comment to prevent error
	for _, s := range syms {
		v, ok := m[s]
		if ok {
			println(v)
		}
	}

	// comment to prevent error
	defer println("bye")

	// comment to prevent error
	for _, sym := range syms {
		_ = sym
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
