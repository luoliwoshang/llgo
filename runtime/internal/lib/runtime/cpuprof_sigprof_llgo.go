//go:build !baremetal && !wasm && (darwin || linux) && (amd64 || arm64)

package runtime

import (
	latomic "sync/atomic"
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
	csyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

const (
	maxCPUProfileStack        = 64
	maxCPUProfileDrainRecords = 256
	maxCPUProfileDrainData    = 4 + maxCPUProfileDrainRecords*(3+maxCPUProfileStack)
	maxCPUProfileDrainTags    = 1 + maxCPUProfileDrainRecords
	cpuProfilePollUsec        = 10000
	cpuProfileSignal          = uint32(csyscall.SIGPROF)
)

//go:linkname c_cpuProfileStart C.llgo_cpu_profile_start
func c_cpuProfileStart(hz int32) int32

//go:linkname c_cpuProfileStop C.llgo_cpu_profile_stop
func c_cpuProfileStop()

//go:linkname c_cpuProfileDrain C.llgo_cpu_profile_drain
func c_cpuProfileDrain(pcs, lengths unsafe.Pointer, maxRecords, maxStack int32, lost *uint64, empty *int32) int32

//go:linkname c_cpuProfileRefreshSignal C.llgo_cpu_profile_refresh_signal
func c_cpuProfileRefreshSignal() int32

var (
	// cpuProfileStateMu is the Go control-plane lock. It serializes native
	// sampler start/stop with libuv SIGPROF watcher changes, but is never
	// acquired by the asynchronous signal handler.
	cpuProfileStateOnce psync.Once
	cpuProfileStateMu   psync.Mutex

	cpuProfileRate          int32
	cpuProfileOpen          uint32
	cpuProfilePeriodPending uint32
	cpuProfilePeriodRecord  = [3]uint64{3, 0, 100}
	cpuProfilePeriodTags    [1]unsafe.Pointer

	// The native sampler fills these fixed scratch buffers in one call. The
	// runtime/pprof contract permits only one reader and consumes the returned
	// slices before calling readProfile again.
	cpuProfileDrainData    [maxCPUProfileDrainData]uint64
	cpuProfileDrainTags    [maxCPUProfileDrainTags]unsafe.Pointer
	cpuProfileDrainPCs     [maxCPUProfileDrainRecords * maxCPUProfileStack]uintptr
	cpuProfileDrainLengths [maxCPUProfileDrainRecords]uint32
)

func ensureCPUProfileState() {
	cpuProfileStateOnce.Do(func() {
		cpuProfileStateMu.Init(nil)
	})
}

func cpuProfileSignalLock(sig uint32) bool {
	if sig != cpuProfileSignal {
		return false
	}
	ensureCPUProfileState()
	cpuProfileStateMu.Lock()
	return true
}

func cpuProfileSignalUnlock(locked bool) {
	if !locked {
		return
	}
	if c_cpuProfileRefreshSignal() != 0 {
		print("runtime: failed to restore CPU profiling SIGPROF handler.\n")
	}
	cpuProfileStateMu.Unlock()
}

// SetCPUProfileRate starts or stops process CPU-time sampling. The signal
// handler and its ring buffer live in profile.c so the sampling path neither
// allocates Go memory nor enters the Go runtime.
func SetCPUProfileRate(hz int) {
	if hz < 0 {
		hz = 0
	}
	if hz > 1000000 {
		hz = 1000000
	}
	ensureCPUProfileState()
	cpuProfileStateMu.Lock()
	defer cpuProfileStateMu.Unlock()

	if hz == 0 {
		if latomic.LoadInt32(&cpuProfileRate) != 0 {
			c_cpuProfileStop()
			latomic.StoreInt32(&cpuProfileRate, 0)
		}
		return
	}
	if !latomic.CompareAndSwapUint32(&cpuProfileOpen, 0, 1) {
		print("runtime: cannot set cpu profile rate until previous profile has finished.\n")
		return
	}

	switch c_cpuProfileStart(int32(hz)) {
	case 1:
		cpuProfilePeriodRecord[2] = uint64(hz)
		latomic.StoreUint32(&cpuProfilePeriodPending, 1)
		latomic.StoreInt32(&cpuProfileRate, int32(hz))
	case 0:
		// Samples from a stopped profile have not been drained yet.
		latomic.StoreUint32(&cpuProfileOpen, 0)
		print("runtime: cannot set cpu profile rate until previous profile has finished.\n")
	default:
		// Preserve runtime/pprof's stream contract even if the OS rejected
		// SIGPROF setup: the caller gets a valid empty profile, not a
		// malformed stream or a blocked writer.
		cpuProfilePeriodRecord[2] = uint64(hz)
		latomic.StoreUint32(&cpuProfilePeriodPending, 1)
	}
}

//go:linkname runtime_pprof_readProfile runtime/pprof.readProfile
func runtime_pprof_readProfile() (data []uint64, tags []unsafe.Pointer, eof bool) {
	if latomic.CompareAndSwapUint32(&cpuProfilePeriodPending, 1, 0) {
		return cpuProfilePeriodRecord[:], cpuProfilePeriodTags[:], false
	}

	for {
		var lost uint64
		var empty int32
		records := int(c_cpuProfileDrain(
			unsafe.Pointer(&cpuProfileDrainPCs[0]),
			unsafe.Pointer(&cpuProfileDrainLengths[0]),
			maxCPUProfileDrainRecords,
			maxCPUProfileStack,
			&lost,
			&empty,
		))
		if lost != 0 || records != 0 {
			// runtime/pprof waits 100 ms between reads on Darwin. Drain a
			// chunk, rather than one sample, so a 100 Hz producer cannot
			// outrun the writer.
			// The writer consumes these package-level scratch slices before
			// its next call, as required by readProfile's contract.
			data = cpuProfileDrainData[:0]
			tags = cpuProfileDrainTags[:0]
			if lost != 0 {
				data = append(data, 4, 0, 0, lost)
				tags = append(tags, nil)
			}
			for record := 0; record < records; record++ {
				n := int(cpuProfileDrainLengths[record])
				data = append(data, uint64(3+n), 0, 1)
				base := record * maxCPUProfileStack
				for i := 0; i < n; i++ {
					data = append(data, uint64(cpuProfileDrainPCs[base+i]))
				}
				tags = append(tags, nil)
			}
			return data, tags, false
		}
		if latomic.LoadInt32(&cpuProfileRate) == 0 && empty != 0 {
			latomic.StoreUint32(&cpuProfileOpen, 0)
			return nil, nil, true
		}

		// Linux's profile writer expects readProfile to block. Keep the wait
		// out of the signal path and poll every 10 ms (the default 100 Hz
		// period); Darwin already sleeps between non-blocking reads.
		c.Usleep(cpuProfilePollUsec)
	}
}
