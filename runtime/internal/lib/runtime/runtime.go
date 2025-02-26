/*
 * Copyright (c) 2024 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

/*
#include <unistd.h>

int llgo_maxprocs() {
	#ifdef _SC_NPROCESSORS_ONLN
		return (int)sysconf(_SC_NPROCESSORS_ONLN);
	#else
		return 1;
	#endif
}
*/
import "C"
import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/runtime"
)

// llgo:skipall
type _runtime struct{}

// GOROOT returns the root of the Go tree. It uses the
// GOROOT environment variable, if set at process start,
// or else the root used during the Go build.
func GOROOT() string {
	return ""
}

//go:linkname c_maxprocs C.llgo_maxprocs
func c_maxprocs() int32

func GOMAXPROCS(n int) int {
	return int(c_maxprocs())
}

func Goexit() {
	runtime.Goexit()
}

func KeepAlive(x any) {
}

func write(fd uintptr, p unsafe.Pointer, n int32) int32 {
	return int32(C.write(C.int(fd), p, C.size_t(n)))
}
