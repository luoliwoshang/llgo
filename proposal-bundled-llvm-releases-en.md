# Proposal: Bundle LLVM Toolchain in Release Packages

## Background Issues

Current LLGO release approach has the following problems:

### User Dependency Complexity
- **Current situation**: Users need to pre-install LLVM 19 (`brew install llvm@19`) for regular target compilation
- **Pain point**: Embedded targets require additional ESP Clang toolchain download (~500MB), creating dual dependencies
- **Limitation**: Version matching issues between two toolchains, complex environment configuration

### Current Complex Architecture
```
Regular target compilation:
User system: /opt/homebrew/opt/llvm@19/lib/libLLVM.dylib (82MB)  
LLGO binary: Dynamically depends on @rpath/libLLVM.dylib

Embedded target compilation:  
Cache directory: ~/Library/Caches/llgo/crosscompile/esp-clang-*/  (~500MB)
LLGO: Detects embedded target → Downloads ESP Clang (first-time use)

Result: Dual dependencies + complex toolchain management
```

## Proposal Goals

**Unified toolchain architecture**, replacing all dependencies with ESP Clang:

### Core Pain Points
```bash
# Current user installation
brew install llvm@19          # Install system LLVM (82MB)
brew install llgo             # Install LLGO

# First embedded compilation
llgo build -target esp32c3 .
# Downloading ESP Clang toolchain... (~500MB)
# Result: Two LLVM installations in system (82MB + 500MB)
```

### Target Experience
```bash
# New user experience
curl -L https://github.com/goplus/llgo/releases/download/v0.12.14/llgo-v0.12.14.darwin-arm64.tar.gz | tar -xz
export LLGO_ROOT=/path/to/llgo-v0.12.14

llgo build .                  # Regular targets: Use bundled ESP Clang
llgo build -target esp32c3 .  # Embedded targets: Use bundled ESP Clang
# One toolchain, supports all targets
```

**Core Value**: **Zero-dependency** unified toolchain, simplifying from "dual dependencies" to "ready-to-use".

### Unified Distribution Architecture 
```
Current release (problematic):
llgo-v0.12.14.darwin-arm64.tar.gz
├── bin/llgo                    # Dynamically depends on external LLVM
├── runtime/                    # Regular target compilation works
└── README.md                   # Embedded target compilation requires toolchain download

New unified release (solution):
llgo-v0.12.14.darwin-arm64.tar.gz          # Keep same filename
├── bin/llgo                                # Smart toolchain selection
├── crosscompile/clang/                     # ESP Clang toolchain (pre-bundled)
│   ├── bin/
│   │   ├── clang++                        # Clang with Xtensa support
│   │   └── llvm-*                         # LLVM toolset
│   ├── lib/
│   │   └── libLLVM.dylib                  # Supports Xtensa targets
├── runtime/                                # Go runtime
└── README.md

Unified experience:
export LLGO_ROOT=/path/to/llgo-v0.12.14
llgo build .                     # Regular targets: Use bundled ESP Clang
llgo build -target esp32 .      # Embedded targets: Use bundled ESP Clang  
llgo build -target rp2040 .     # All targets: Unified pre-bundled toolchain
```

## Release Strategy

### Unified Release Strategy
- **Single version**: `llgo-v0.12.14.*` (with bundled ESP Clang toolchain)
- **Backward compatible**: Keep same download paths and filenames
- **Progressive enhancement**: Existing functionality unchanged, adding instant embedded compilation capability

### User Value
- **Zero learning cost**: Download commands and usage patterns remain completely unchanged
- **Enhanced functionality**: Upgrade from "regular compiler" to "full-featured embedded compiler"  
- **Ready-to-use**: Embedded development requires no additional configuration or downloads

## Next Actions

1. [ ] Modify existing `.goreleaser.yaml` configuration to integrate ESP Clang download
2. [ ] Update build scripts to pre-download toolchain to `crosscompile/clang/` during release