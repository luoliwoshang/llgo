// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build wasm && !go1.26

package runtime

import "unsafe"

func memhash32Fallback(p unsafe.Pointer, seed uintptr) uintptr {
	a, b := mix32(uint32(seed), uint32(4^hashkey[0]))
	t := readUnaligned32(p)
	a ^= t
	b ^= t
	a, b = mix32(a, b)
	a, b = mix32(a, b)
	return uintptr(a ^ b)
}

func memhash64Fallback(p unsafe.Pointer, seed uintptr) uintptr {
	a, b := mix32(uint32(seed), uint32(8^hashkey[0]))
	a ^= readUnaligned32(p)
	b ^= readUnaligned32(add(p, 4))
	a, b = mix32(a, b)
	a, b = mix32(a, b)
	return uintptr(a ^ b)
}

func memhashFallback(p unsafe.Pointer, seed, s uintptr) uintptr {
	a, b := mix32(uint32(seed), uint32(s^hashkey[0]))
	if s == 0 {
		return uintptr(a ^ b)
	}
	for ; s > 8; s -= 8 {
		a ^= readUnaligned32(p)
		b ^= readUnaligned32(add(p, 4))
		a, b = mix32(a, b)
		p = add(p, 8)
	}
	if s >= 4 {
		a ^= readUnaligned32(p)
		b ^= readUnaligned32(add(p, s-4))
	} else {
		t := uint32(*(*byte)(p))
		t |= uint32(*(*byte)(add(p, s>>1))) << 8
		t |= uint32(*(*byte)(add(p, s-1))) << 16
		b ^= t
	}
	a, b = mix32(a, b)
	a, b = mix32(a, b)
	return uintptr(a ^ b)
}

func mix32(a, b uint32) (uint32, uint32) {
	c := uint64(a^uint32(hashkey[1])) * uint64(b^uint32(hashkey[2]))
	return uint32(c), uint32(c >> 32)
}
