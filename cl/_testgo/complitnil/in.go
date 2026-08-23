// LITTEST
package main

type pair struct {
	x int
	y int
}

var calls int

// Both calls must run before assigning through the nil pointer panics.
// CHECK-LABEL: define void @main.assignNil(){{.*}} {
// CHECK: call i64 @main.count()
// CHECK: call i64 @main.count()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"
func assignNil() {
	defer func() {
		if recover() == nil {
			panic("nil composite assignment did not panic")
		}
		if calls != 2 {
			panic("nil composite assignment panicked before evaluating its right-hand side")
		}
	}()

	var dst *pair
	*dst = pair{count(), count()}
}

func count() int {
	calls++
	return calls
}

func main() {
	assignNil()
	println("ok")
}
