// LITTEST darwin/arm64 linux/amd64
package main

// CHECK-LABEL: define linkonce { ptr, ptr } @"{{.*}}Y[{{.*}}int,int]"(
// DARWIN-ARM64: call { ptr, ptr } %{{.*}}(ptr swiftself %{{.*}}, [[INT_INTERNAL:%"[^"]+"]] %{{.*}})
// LINUX-AMD64: call { ptr, ptr } %{{.*}}(ptr nest %{{.*}}, [[INT_INTERNAL:%"[^"]+"]] %{{.*}})
// DARWIN-ARM64: define linkonce { ptr, ptr } @"{{.*}}Y$1[{{.*}}int,int]"(ptr swiftself %{{.*}}, [[INT_INTERNAL]] %{{.*}})
// LINUX-AMD64: define linkonce { ptr, ptr } @"{{.*}}Y$1[{{.*}}int,int]"(ptr nest %{{.*}}, [[INT_INTERNAL]] %{{.*}})
// CHECK-LABEL: define linkonce { ptr, ptr } @"{{.*}}Y[{{.*}}string,string]"(
// DARWIN-ARM64: call { ptr, ptr } %{{.*}}(ptr swiftself %{{.*}}, [[STRING_INTERNAL:%"[^"]+"]] %{{.*}})
// LINUX-AMD64: call { ptr, ptr } %{{.*}}(ptr nest %{{.*}}, [[STRING_INTERNAL:%"[^"]+"]] %{{.*}})
// DARWIN-ARM64: define linkonce { ptr, ptr } @"{{.*}}Y$1[{{.*}}string,string]"(ptr swiftself %{{.*}}, [[STRING_INTERNAL]] %{{.*}})
// LINUX-AMD64: define linkonce { ptr, ptr } @"{{.*}}Y$1[{{.*}}string,string]"(ptr nest %{{.*}}, [[STRING_INTERNAL]] %{{.*}})

func Y[Endo ~func(RecFct) RecFct, RecFct ~func(T) R, T, R any](f Endo) RecFct {
	type internal[RecFct ~func(T) R, T, R any] func(internal[RecFct, T, R]) RecFct

	g := func(h internal[RecFct, T, R]) RecFct {
		return func(t T) R {
			return f(h(h))(t)
		}
	}
	return g(g)
}

func main() {
	factorial := Y(func(recur func(int) int) func(int) int {
		return func(n int) int {
			if n == 0 {
				return 1
			}
			return n * recur(n-1)
		}
	})
	repeat := Y(func(recur func(string) string) func(string) string {
		return func(s string) string {
			if len(s) == 3 {
				return s
			}
			return recur(s + "x")
		}
	})
	println(factorial(10))
	println(repeat(""))
}
