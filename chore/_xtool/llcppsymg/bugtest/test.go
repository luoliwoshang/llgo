package main

import "errors"

func main() {
	syms := []int{}
	m := make(map[int]bool)
	// m[0] = true // also with error

	// Uncommenting the following will prevent llgo run . from crashing
	// for i := range syms {
	// 	_ = i
	// }

	err := errors.New("expect error")

	// Normal execution
	// if err != nil {
	// 	panic(err)
	// }

	// Erroneous execution (uncomment to simulate error)
	check(err)

	// comment to prevent error
	for range syms {
		_, ok := m[0]

		// comment to prevent error
		if ok {
			println("ok")
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
