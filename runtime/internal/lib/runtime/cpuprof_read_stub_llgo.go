//go:build baremetal || wasm || (!darwin && !linux) || (!amd64 && !arm64)

package runtime

import "unsafe"

var (
	cpuProfilePeriodRecord = [3]uint64{3, 0, 100}
	cpuProfilePeriodTags   [1]unsafe.Pointer
)

//go:linkname runtime_pprof_readProfile runtime/pprof.readProfile
func runtime_pprof_readProfile() (data []uint64, tags []unsafe.Pointer, eof bool) {
	return cpuProfilePeriodRecord[:], cpuProfilePeriodTags[:], true
}
