// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import (
	_ "unsafe"
)

type covCounterBlob struct {
	Counters *uint32
	Len      uint64
}

//go:linkname coverage_getCovCounterList internal/coverage/cfile.getCovCounterList
func coverage_getCovCounterList() []covCounterBlob {
	return nil
}
