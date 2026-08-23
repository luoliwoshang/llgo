// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build wasm

package atomic

import "unsafe"

// StorepNoWB performs *ptr = val without a write barrier.
func StorepNoWB(ptr unsafe.Pointer, val unsafe.Pointer) {
	*(*uintptr)(ptr) = uintptr(val)
}
