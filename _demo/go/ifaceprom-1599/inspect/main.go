package main

import (
	"fmt"
	"unsafe"

	"github.com/goplus/llgo/_demo/go/ifaceprom-1599/foo"
)

// Game1 embeds foo.Game to reproduce the ifaceprom case.
type Game1 struct{ *foo.Game }

type nameOff = int32
type typeOff = int32
type textOff = int32

type method struct {
	name nameOff
	mtyp typeOff
	ifn  textOff
	tfn  textOff
}

type uncommonType struct {
	pkgPath nameOff
	mcount  uint16
	xcount  uint16
	moff    uint32
	_       uint32
}

type rtype struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      uint8
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      unsafe.Pointer
	gcdata     unsafe.Pointer
	str        nameOff
	ptrToThis  typeOff
}

type emptyInterface struct {
	typ  *rtype
	word unsafe.Pointer
}

// dumpPointerMethods decodes methods from a pointer type (*T) metadata.
// It assumes method table is sorted by name, matching runtime/iface.go expectations.
func dumpPointerMethods[T any](label string) {
	var zero *T
	var iface any = zero
	rt := (*emptyInterface)(unsafe.Pointer(&iface)).typ // rtype for *T

	// Pointer kind: uncommon data sits after rtype + Elem pointer.
	off := unsafe.Sizeof(rtype{})
	off += unsafe.Sizeof(uintptr(0)) // PtrType.Elem
	u := (*uncommonType)(unsafe.Add(unsafe.Pointer(rt), off))

	if u.mcount == 0 {
		fmt.Printf("%s: mcount=0 (no methods)\n", label)
		return
	}
	fmt.Printf("%s: mcount=%d moff=%d\n", label, u.mcount, u.moff)
	base := unsafe.Add(unsafe.Pointer(u), uintptr(u.moff))
	for i := 0; i < int(u.mcount); i++ {
		m := (*method)(unsafe.Add(base, uintptr(i)*unsafe.Sizeof(method{})))
		fmt.Printf("  #%d: ifn=%d tfn=%d (nameOff=%d mtyp=%d)\n", i, m.ifn, m.tfn, m.name, m.mtyp)
	}
}

func main() {
	// Force interface assertion and call to keep exported method reachable.
	var any1 any = &Game1{&foo.Game{}}
	v1 := any1.(foo.Gamer)
	v1.Load()

	var any2 any = &foo.Game{}
	v2 := any2.(foo.Gamer)
	v2.Load()

	dumpPointerMethods[Game1]("*Game1 pointer type")
	dumpPointerMethods[foo.Game]("*foo.Game pointer type")
}
