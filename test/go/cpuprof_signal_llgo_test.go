//go:build llgo && !baremetal && !wasm && (darwin || linux) && (amd64 || arm64)

package gotest

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

//go:linkname readCPUProfileRaw runtime/pprof.readProfile
func readCPUProfileRaw() (data []uint64, tags []unsafe.Pointer, eof bool)

//go:linkname raiseCPUProfileSignal C.raise
func raiseCPUProfileSignal(sig int32) int32

//go:linkname testCPUProfileFaultRecovery C.llgo_cpu_profile_test_fault_recovery
func testCPUProfileFaultRecovery() int32

//go:noinline
func cpuProfileSignalHotLoop(d time.Duration) uint64 {
	deadline := time.Now().Add(d)
	x := uint64(1)
	for time.Now().Before(deadline) {
		for i := 0; i < 10000; i++ {
			x = x*1664525 + 1013904223
		}
	}
	return x
}

func requireCPUProfileSignalFunction(t *testing.T, data []byte) {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("CPU profile is not valid gzip: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read CPU profile: %v", err)
	}
	if err := zr.Close(); err != nil {
		t.Fatalf("close CPU profile reader: %v", err)
	}
	if !bytes.Contains(raw, []byte("cpuProfileSignalHotLoop")) {
		t.Fatalf("CPU profile does not contain the hot function (compressed=%d bytes)", len(data))
	}
}

func sendSIGPROF(t *testing.T, phase string) {
	t.Helper()
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("%s: FindProcess: %v", phase, err)
	}
	if err := proc.Signal(syscall.SIGPROF); err != nil {
		t.Fatalf("%s: send SIGPROF: %v", phase, err)
	}
}

func waitForSIGPROF(t *testing.T, c <-chan os.Signal, phase string) {
	t.Helper()
	select {
	case got := <-c:
		if got != syscall.SIGPROF {
			t.Fatalf("%s: got signal %v, want SIGPROF", phase, got)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s: timeout waiting for SIGPROF", phase)
	}
}

func requireNoSIGPROF(t *testing.T, c <-chan os.Signal, phase string) {
	t.Helper()
	select {
	case got := <-c:
		t.Fatalf("%s: SIGPROF unexpectedly reached os/signal as %v", phase, got)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestCPUProfileFaultRecovery(t *testing.T) {
	if got := testCPUProfileFaultRecovery(); got != 1 {
		t.Fatalf("guarded frame walk returned %d frames, want interrupted PC only", got)
	}
}

func TestCPUProfileSIGPROFNotifyBeforeStart(t *testing.T) {
	c := make(chan os.Signal, 8)
	signal.Notify(c, syscall.SIGPROF)
	t.Cleanup(func() { signal.Stop(c) })

	var profile bytes.Buffer
	if err := pprof.StartCPUProfile(&profile); err != nil {
		t.Fatalf("StartCPUProfile: %v", err)
	}
	t.Cleanup(pprof.StopCPUProfile)
	_ = cpuProfileSignalHotLoop(300 * time.Millisecond)
	requireNoSIGPROF(t, c, "watcher started before profiling")
	sendSIGPROF(t, "while profiling")
	requireNoSIGPROF(t, c, "user SIGPROF while profiling")
	pprof.StopCPUProfile()
	requireCPUProfileSignalFunction(t, profile.Bytes())

	// Stopping profiling must restore the libuv watcher that was active when
	// profiling began.
	sendSIGPROF(t, "after profiling")
	waitForSIGPROF(t, c, "after profiling")
}

func TestCPUProfileSIGPROFNotifyDuringProfile(t *testing.T) {
	var profile bytes.Buffer
	if err := pprof.StartCPUProfile(&profile); err != nil {
		t.Fatalf("StartCPUProfile: %v", err)
	}
	t.Cleanup(pprof.StopCPUProfile)

	c := make(chan os.Signal, 8)
	signal.Notify(c, syscall.SIGPROF)
	t.Cleanup(func() { signal.Stop(c) })
	_ = cpuProfileSignalHotLoop(300 * time.Millisecond)
	requireNoSIGPROF(t, c, "watcher started during profiling")

	// Removing the watcher must not remove the profiler's handler.
	signal.Stop(c)
	_ = cpuProfileSignalHotLoop(300 * time.Millisecond)
	pprof.StopCPUProfile()
	requireCPUProfileSignalFunction(t, profile.Bytes())
}

func TestCPUProfileSIGPROFIgnoreReset(t *testing.T) {
	var profile bytes.Buffer
	if err := pprof.StartCPUProfile(&profile); err != nil {
		t.Fatalf("StartCPUProfile: %v", err)
	}
	t.Cleanup(pprof.StopCPUProfile)

	signal.Ignore(syscall.SIGPROF)
	t.Cleanup(func() { signal.Reset(syscall.SIGPROF) })
	if !signal.Ignored(syscall.SIGPROF) {
		t.Fatal("SIGPROF is not ignored after signal.Ignore")
	}
	_ = cpuProfileSignalHotLoop(300 * time.Millisecond)

	// Reset changes the libuv watcher while profiling is still active. It
	// must restore the logical signal state without removing the profiler.
	signal.Reset(syscall.SIGPROF)
	if signal.Ignored(syscall.SIGPROF) {
		t.Fatal("SIGPROF is still ignored after signal.Reset")
	}
	_ = cpuProfileSignalHotLoop(300 * time.Millisecond)

	pprof.StopCPUProfile()
	requireCPUProfileSignalFunction(t, profile.Bytes())
}

func TestCPUProfileSIGPROFRepeatedLifecycle(t *testing.T) {
	c := make(chan os.Signal, 8)
	signal.Notify(c, syscall.SIGPROF)
	t.Cleanup(func() { signal.Stop(c) })

	for round := 0; round < 3; round++ {
		var profile bytes.Buffer
		if err := pprof.StartCPUProfile(&profile); err != nil {
			t.Fatalf("round %d: StartCPUProfile: %v", round, err)
		}
		_ = cpuProfileSignalHotLoop(200 * time.Millisecond)
		pprof.StopCPUProfile()
		requireCPUProfileSignalFunction(t, profile.Bytes())

		// Every Stop must hand SIGPROF ownership back to the same watcher,
		// and the next Start must be able to take ownership again.
		phase := "after repeated profile"
		sendSIGPROF(t, phase)
		waitForSIGPROF(t, c, phase)
	}
}

func TestCPUProfileLostSamples(t *testing.T) {
	// Install the real profiling handler, then synchronously deliver more
	// signals than the 2048-entry ring can hold while no writer drains it.
	// Using raise instead of a high-rate timer keeps this independent of the
	// process CPU time available on a busy CI runner.
	runtime.SetCPUProfileRate(1)
	defer runtime.SetCPUProfileRate(0)
	for i := 0; i < 4096; i++ {
		if ret := raiseCPUProfileSignal(int32(syscall.SIGPROF)); ret != 0 {
			t.Fatalf("raise SIGPROF %d: return value %d", i, ret)
		}
	}
	runtime.SetCPUProfileRate(0)

	var lost uint64
	for {
		data, _, eof := readCPUProfileRaw()
		for len(data) != 0 {
			n := int(data[0])
			if n <= 0 || n > len(data) {
				t.Fatalf("malformed raw CPU profile record length %d in %d words", n, len(data))
			}
			if n == 4 && data[1] == 0 && data[2] == 0 {
				lost += data[3]
			}
			data = data[n:]
		}
		if eof {
			break
		}
	}
	if lost == 0 {
		t.Fatal("high-rate CPU profile did not report lost samples")
	}
}
