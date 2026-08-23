// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

// Compiler is the name of the compiler toolchain that built the
// running binary. Known toolchains are:
//
//   - gc      Also known as cmd/compile.
//   - gccgo   The gccgo front end, part of the GCC compiler suite.
//   - llgo    Our project
const Compiler = "llgo"
