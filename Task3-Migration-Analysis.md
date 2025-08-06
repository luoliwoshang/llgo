# Task 3: Generic Machine Library - TinyGo Migration Analysis

## Current Status Analysis

**goplus/lib/emb** (commit bc42bc75) has completed basic migration from TinyGo:

1. **Machine package migrated** - Contains 200+ hardware-related files
2. **Runtime/Volatile package migrated** - Provides hardware register access
3. **Device packages missing** - Critical dependency packages not yet migrated

## Key Migration Challenges

### 1. Package Import Path Issues

```go
// Current TinyGo style imports (not available in LLGO)
import (
    "device/esp"      // Missing
    "device/riscv"    // Missing 
    "device/arm"      // Missing
    "device/stm32"    // Missing
    "device/avr"      // Missing
    // ... 15+ device subpackages
)
```

### 2. Assembly Code Dependencies

```go
// machine_esp32c3.go:673, machine_k210.go
for bus.GetID_REG_UPDATE() > 0 {
    riscv.Asm("nop")  // device/riscv package missing
}
```

## Migration Requirements Statistics

| Dependency Type | File Count | Usage Frequency | Migration Priority |
|----------------|------------|-----------------|-------------------|
| `device/esp` | 6 | High | High |
| `device/stm32` | 47 | Very High | Critical |
| `device/rp` | 24 | High | High |
| `device/nrf` | 21 | High | High |
| `device/avr` | 15 | Medium | Medium |
| `device/sam` | 12 | Medium | Medium |
| `device/riscv` | 2 | Assembly | High |
| Other device/* | 10+ | Low-Medium | Low |

## Step 1: TinyGo Asset Migration Implementation Plan

### Phase 1: Device Package Migration Architecture

```
goplus/lib/emb/
├── machine/          # Completed
├── runtime/volatile/ # Completed  
└── device/          # New
    ├── arm/         # ARM architecture common
    ├── esp/         # ESP32/ESP8266 series
    ├── stm32/       # STM32 all series
    ├── rp/          # Raspberry Pi RP2040/2350
    ├── nrf/         # Nordic nRF series
    ├── avr/         # AVR microcontrollers
    ├── sam/         # Microchip SAM series
    ├── riscv/       # RISC-V architecture common
    └── ...
```

### Phase 2: Package Import Path Adaptation

```go
// Change to goplus/lib path style
import (
    "github.com/goplus/lib/emb/device/esp"
    "github.com/goplus/lib/emb/device/riscv"
    "github.com/goplus/lib/emb/device/arm"
    "github.com/goplus/lib/emb/runtime/volatile"
)
```

### Phase 3: Assembly Code Adaptation Solutions

```go
// Option 1: LLGO style inline assembly
//go:linkname nop runtime.nop
func nop()

// Option 2: C wrapper
//go:linkname riscv_nop C.riscv_nop
func riscv_nop()

// Option 3: Direct LLVM IR generation
// Generate corresponding instructions at LLGO compiler level
```

### Phase 4: Staged Migration Priorities

#### First Priority - Core Platforms
1. **device/esp** - ESP32/ESP32-C3 ecosystem
2. **device/stm32** - Largest ARM Cortex-M ecosystem
3. **device/rp** - Raspberry Pi Pico series
4. **device/riscv** - Resolve assembly dependencies

#### Second Priority - Mainstream Platforms
5. **device/nrf** - Nordic low-power Bluetooth
6. **device/avr** - Arduino classic platform
7. **device/sam** - Microchip ARM series

#### Third Priority - Specialized Platforms
8. Other specialized hardware platforms (kendryte, sifive, tkey, etc.)

### Phase 5: Automation Tool Support

```bash
# Suggested migration assistance tool
go install github.com/goplus/llgo/chore/device-migrate

# Automatically convert TinyGo device packages to goplus/lib format
device-migrate --input tinygo-src/ --output goplus-lib/emb/device/
```