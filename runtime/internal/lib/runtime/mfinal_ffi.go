//go:build !nogc

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/ffi"
)

var finalizerFFITypeClosure = ffi.StructOf(ffi.TypePointer, ffi.TypePointer)

// Keep the Go ABI conversion local to finalizers. The low-level ffi package
// should not depend on runtime/abi.
func finalizerFFIType(typ *abi.Type) *ffi.Type {
	switch kind := typ.Kind(); kind {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64, abi.Complex64, abi.Complex128:
		return ffi.Typ[kind]
	case abi.Array:
		at := typ.ArrayType()
		return ffi.ArrayOf(finalizerFFIType(at.Elem), int(at.Len))
	case abi.Chan, abi.Map, abi.Pointer, abi.UnsafePointer:
		return ffi.TypePointer
	case abi.Func:
		return finalizerFFITypeClosure
	case abi.Interface:
		return ffi.TypeInterface
	case abi.Slice:
		return ffi.TypeSlice
	case abi.String:
		return ffi.TypeString
	case abi.Struct:
		if typ.IsClosure() {
			return finalizerFFITypeClosure
		}
		return finalizerFFIStructType(typ)
	}
	panic("runtime.SetFinalizer: unsupported result type " + typ.String())
}

func finalizerFFIStructType(typ *abi.Type) *ffi.Type {
	st := typ.StructType()
	fields := make([]*ffi.Type, 0, len(st.Fields))
	var off uintptr
	for _, field := range st.Fields {
		if field.Offset > off {
			fields, off = appendFinalizerFFIPadding(fields, off, field.Offset-off)
		}
		if field.Typ.Size_ == 0 {
			continue
		}
		fields = append(fields, finalizerFFIType(field.Typ))
		off = field.Offset + field.Typ.Size_
	}
	// Do not pad to typ.Size_: trailing zero-sized fields can enlarge the
	// Go-visible size without consuming registers in llgo's callable ABI.
	return ffi.StructOf(fields...)
}

func appendFinalizerFFIPadding(fields []*ffi.Type, off, size uintptr) ([]*ffi.Type, uintptr) {
	for size > 0 {
		switch {
		case off%8 == 0 && size >= 8:
			fields = append(fields, ffi.TypeUint64)
			off += 8
			size -= 8
		case off%4 == 0 && size >= 4:
			fields = append(fields, ffi.TypeUint32)
			off += 4
			size -= 4
		case off%2 == 0 && size >= 2:
			fields = append(fields, ffi.TypeUint16)
			off += 2
			size -= 2
		default:
			fields = append(fields, ffi.TypeUint8)
			off++
			size--
		}
	}
	return fields, off
}

func finalizerFFIReturnType(results []*abi.Type) *ffi.Type {
	switch len(results) {
	case 0:
		return ffi.TypeVoid
	case 1:
		return finalizerFFIType(results[0])
	default:
		fields := make([]*ffi.Type, len(results))
		for i, result := range results {
			fields[i] = finalizerFFIType(result)
		}
		return ffi.StructOf(fields...)
	}
}

func newFinalizerFFISignature(ft *abi.FuncType, explicitEnv bool) (*ffi.Signature, []*ffi.Type, uintptr) {
	paramTypes := make([]*ffi.Type, 0, 2)
	if explicitEnv {
		paramTypes = append(paramTypes, ffi.TypePointer)
	}
	// SetFinalizer currently requires the finalizer argument to have the
	// object's pointer type exactly.
	paramTypes = append(paramTypes, ffi.TypePointer)
	sig, err := ffi.NewSignature(finalizerFFIReturnType(ft.Out), paramTypes...)
	if err != nil {
		panic(err)
	}
	if sig.RType == ffi.TypeVoid {
		return sig, paramTypes, 0
	}
	retSize := sig.RType.Size
	if argSize := unsafe.Sizeof(uintptr(0)); retSize < argSize {
		// Conservatively reserve one word for every non-void result. In
		// particular, libffi writes sub-register scalar results as ffi_arg.
		retSize = argSize
	}
	return sig, paramTypes, retSize
}
