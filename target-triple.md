# LLVM Target Triple Configuration in LLGO

LLGO uses LLVM target triples to configure cross-compilation and platform-specific code generation. This document describes how target triples are generated, configured, and used throughout the LLGO compilation pipeline.

## Overview

A target triple is a string that describes the target platform for compilation in the format:
```
architecture-vendor-os(-environment)
```

For example: `x86_64-apple-macosx`, `aarch64-unknown-linux-gnu`, `wasm32-unknown-wasi`

## Core Components

### 1. Target Triple Generation

**Location:** `internal/xtool/llvm/llvm.go:5-55`

The `GetTargetTriple(goos, goarch string) string` function is the primary entry point for converting Go's GOOS/GOARCH values to LLVM target triples.

**Architecture Mapping:**
- `"amd64"` → `"x86_64"`
- `"arm64"` → `"aarch64"`
- `"386"` → `"i386"`
- `"wasm"` → `"wasm32"`

**Platform-Specific Handling:**
- **Darwin (macOS):** Uses `"apple"` vendor and `"macosx"` OS
- **Linux:** Uses `"unknown"` vendor and `"linux-gnu"` environment
- **Windows:** Uses `"pc"` vendor and `"windows-msvc"` environment
- **WASI:** Uses `"unknown"` vendor and `"wasi"` OS

### 2. Target Configuration

**Location:** `ssa/target.go:27-162`

The target configuration system creates LLVM target machines and data structures:

**Key Methods:**
- `targetData()` (lines 33-44): Creates LLVM target data from target triple
- `Spec()` (lines 73-161): Generates detailed target specifications

**LLVM Integration:**
```go
// Uses llvm.DefaultTargetTriple() as fallback
triple := llvm.DefaultTargetTriple()
if p.target.Triple != "" {
    triple = p.target.Triple
}

// Creates LLVM target machine
target, err := llvm.GetTargetFromTriple(triple)
targetMachine := target.CreateTargetMachine(triple, "", "", ...)
```

### 3. Cross-Compilation Support

**Location:** `internal/crosscompile/crosscompile.go`

Target triples are used extensively in cross-compilation:

**Usage Examples:**
```go
// Generate target triple for cross-compilation
targetTriple := llvm.GetTargetTriple(goos, goarch)

// Use in compiler flags
flags = append(flags, "-target", targetTriple)
```

**Special Cases:**
- **WASI Threads:** Modifies target triple for thread support
- **Platform-specific flags:** Different compiler flags based on target

### 4. Build System Integration

**Location:** `internal/build/build.go:180-191`

The build system creates target configurations and passes them through the compilation pipeline:

```go
target := &ssa.Target{
    GOOS:   cfg.GOOS,
    GOARCH: cfg.GOARCH,
}

prog := ssa.NewProgram(target, ...)
```

## Configuration Examples

### Common Target Triples

| Platform | GOOS | GOARCH | Target Triple |
|----------|------|---------|---------------|
| macOS Intel | darwin | amd64 | x86_64-apple-macosx |
| macOS Apple Silicon | darwin | arm64 | aarch64-apple-macosx |
| Linux x86_64 | linux | amd64 | x86_64-unknown-linux-gnu |
| Linux ARM64 | linux | arm64 | aarch64-unknown-linux-gnu |
| Windows x86_64 | windows | amd64 | x86_64-pc-windows-msvc |
| WebAssembly | js | wasm | wasm32-unknown-wasi |

### Cross-Compilation Example

```bash
# Compile for Linux ARM64 from macOS
GOOS=linux GOARCH=arm64 llgo build .

# This internally generates target triple: aarch64-unknown-linux-gnu
```

## Advanced Features

### Target-Specific Optimizations

**Location:** `ssa/target.go:132-159`

The target system supports CPU-specific optimizations:

```go
// CPU feature detection
if target.GOARCH == "amd64" {
    spec.Features = "+sse2,+cx16"
} else if target.GOARCH == "arm64" {
    spec.Features = "+neon"
}
```

### Module Target Setting

**Location:** `ssa/package.go:402-404`

Module-level target setting is currently disabled:

```go
// Currently commented out to avoid snapshot test issues
// if p.target.GOARCH != runtime.GOARCH && p.target.GOOS != runtime.GOOS {
//     mod.SetTarget(p.target.Spec().Triple)
// }
```

### WASI Thread Support

**Location:** `internal/crosscompile/crosscompile.go:87-90`

Special handling for WebAssembly with threads:

```go
if goos == "js" && goarch == "wasm" {
    // Modify target triple for WASI threads
    targetTriple = "wasm32-wasi-threads"
}
```

## Testing

**Location:** `internal/xtool/llvm/llvm_test.go:12-150`

Comprehensive tests validate target triple generation for various platform combinations:

```go
func TestGetTargetTriple(t *testing.T) {
    tests := []struct {
        goos, goarch, expected string
    }{
        {"darwin", "amd64", "x86_64-apple-macosx"},
        {"linux", "arm64", "aarch64-unknown-linux-gnu"},
        // ... more test cases
    }
}
```

## Debugging Target Issues

### Common Problems

1. **Unsupported Target:** Check if the GOOS/GOARCH combination is supported
2. **Cross-compilation Failures:** Verify target triple matches available toolchain
3. **WASM Issues:** Ensure proper WASI target configuration

### Debugging Commands

```bash
# Check current target
llgo version

# Verbose compilation to see target triple
llgo build -v .

# Compare with Clang target
clang -print-target-triple
```

## Implementation Notes

- Target triples are generated at compilation time based on environment variables
- The system supports both native and cross-compilation scenarios
- LLVM target machines are created once per compilation unit
- Target configuration affects code generation, optimization, and linking

This target triple system enables LLGO to generate platform-specific code while maintaining Go's cross-compilation capabilities.