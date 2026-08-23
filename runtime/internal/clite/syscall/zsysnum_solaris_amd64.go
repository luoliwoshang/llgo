// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build amd64 && solaris

package syscall

// TODO(aram): remove these before Go 1.3.
const (
	SYS_EXECVE = 59
	SYS_FCNTL  = 62
)
