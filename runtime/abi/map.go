// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package abi

// Map constants common to several packages
// runtime/runtime-gdb.py:MapTypePrinter contains its own copy
const (
	MapBucketCountBits = 3 // log2 of number of elements in a bucket.
	MapBucketCount     = 1 << MapBucketCountBits
	MapMaxKeyBytes     = 128 // Must fit in a uint8.
	MapMaxElemBytes    = 128 // Must fit in a uint8.
)
