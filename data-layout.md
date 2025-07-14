# LLVM Data Layout in LLGO

This document explains how LLGO handles LLVM data layout configuration and target-specific memory layout settings.

## Overview

LLGO uses LLVM's automatic data layout inference based on target triples rather than manually specifying data layout strings. This approach ensures compatibility across different platforms while leveraging LLVM's built-in knowledge of target-specific memory layouts.

## Current Implementation

### Target Data Generation

The data layout is automatically determined in `ssa/target.go:33-44`:

```go
func (p *Target) targetData() llvm.TargetData {
    spec := p.Spec()
    if spec.Triple == "" {
        spec.Triple = llvm.DefaultTargetTriple()
    }
    t, err := llvm.GetTargetFromTriple(spec.Triple)
    if err != nil {
        panic(err)
    }
    machine := t.CreateTargetMachine(spec.Triple, spec.CPU, spec.Features, 
        llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
    return machine.CreateTargetData()
}
```

### Target Triple Configuration

LLGO generates target triples based on GOOS/GOARCH environment variables:

- **Architecture mapping**: `386` → `i386`, `amd64` → `x86_64`, `arm64` → `aarch64`
- **OS mapping**: `darwin` → `macosx`, `windows` → `windows-gnu`
- **ABI suffixes**: ARM targets get `-gnueabihf`, Windows gets `-gnu`

Examples:
- `x86_64-apple-macosx` (macOS/amd64)
- `aarch64-unknown-linux-gnueabihf` (Linux/arm64)
- `i386-unknown-windows-gnu` (Windows/386)

## Platform-Specific Considerations

### Word Size and Alignment

The data layout automatically handles platform differences:

- **32-bit platforms** (386, arm, wasm): 4-byte pointers and alignment
- **64-bit platforms** (amd64, arm64): 8-byte pointers and alignment

This is referenced in code generation, for example in `internal/build/build.go:695-698`:

```go
declSizeT := "%size_t = type i64"
if is32Bits(ctx.buildConf.Goarch) {
    declSizeT = "%size_t = type i32"
}
```

### Special Targets

#### WebAssembly
- Uses `wasm32-unknown-wasip1` triple
- 32-bit address space with specific memory model
- Handled in `ssa/target.go:156-159`

#### Darwin/ARM64
- Maps to `arm64-apple-macosx` (not `aarch64`)
- Apple-specific calling conventions and ABI

## Benefits of Automatic Data Layout

1. **Consistency**: LLVM ensures data layout matches the target's actual ABI
2. **Maintenance**: No need to manually maintain data layout strings for each platform
3. **Compatibility**: Automatic updates when LLVM adds new target support
4. **Correctness**: Reduces risk of mismatched data layouts causing runtime issues

## Integration Points

### SSA Package Creation
Target data is used when creating SSA packages to ensure correct type sizes and alignments.

### Cross-Compilation
The `crosscompile` package works with the target configuration to enable building for different platforms.

### Runtime Integration
The generated data layout affects:
- Go type sizes and field offsets
- Function calling conventions
- Memory allocation alignment
- Garbage collector metadata

## Related Documentation

- 📖 **[LLVM Target Triple Configuration](target-triple.md)** - Complete guide to target triple generation
- 📖 **[Export Directive Guide](export-directive.md)** - C interoperability documentation

## Advanced Usage

For debugging or analysis, you can inspect the actual data layout used by LLGO:

```bash
# Generate LLVM IR to see the data layout
./llgen your_program.go
head -n 5 your_program.ll  # Look for target datalayout line
```

The generated LLVM IR will include a line like:
```llvm
target datalayout = "e-m:o-i64:64-i128:128-n32:64-S128"
```

This string encodes endianness, mangling, pointer sizes, and alignment requirements specific to your target platform.