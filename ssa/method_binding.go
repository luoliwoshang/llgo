package ssa

import (
	"strings"
	"sync/atomic"
)

const (
	// MethodBindingAttrIFN marks an interface-call target function (IFn).
	MethodBindingAttrIFN = "llgo.method.ifn"
	// MethodBindingAttrTFN marks a direct-method-call target function (TFn).
	MethodBindingAttrTFN = "llgo.method.tfn"
)

const methodBindingSep = "\x1f"
const methodBindingListSep = "\x1e"

type MethodBindingEntry struct {
	TypeSymbol       string
	MethodName       string
	MethodTypeSymbol string
}

var methodLateBinding atomic.Bool

// SetMethodLateBinding toggles experimental late binding for Method.Ifn_/Tfn_.
func SetMethodLateBinding(enabled bool) {
	methodLateBinding.Store(enabled)
}

// MethodLateBindingEnabled reports whether experimental late binding is enabled.
func MethodLateBindingEnabled() bool {
	return methodLateBinding.Load()
}

func encodeMethodBindingPayload(typeSymbol, methodName, methodTypeSymbol string) string {
	return typeSymbol + methodBindingSep + methodName + methodBindingSep + methodTypeSymbol
}

// DecodeMethodBindingPayload decodes method binding metadata payload.
func DecodeMethodBindingPayload(payload string) (typeSymbol, methodName, methodTypeSymbol string, ok bool) {
	parts := strings.Split(payload, methodBindingSep)
	if len(parts) != 3 {
		return "", "", "", false
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func mergeMethodBindingPayload(existing, payload string) string {
	if existing == "" {
		return payload
	}
	if payload == "" {
		return existing
	}
	for _, item := range strings.Split(existing, methodBindingListSep) {
		if item == payload {
			return existing
		}
	}
	return existing + methodBindingListSep + payload
}

// DecodeMethodBindingAttrValue decodes one method binding attribute value.
func DecodeMethodBindingAttrValue(value string) []MethodBindingEntry {
	if value == "" {
		return nil
	}
	items := strings.Split(value, methodBindingListSep)
	out := make([]MethodBindingEntry, 0, len(items))
	for _, item := range items {
		typeSym, methodName, methodTypeSym, ok := DecodeMethodBindingPayload(item)
		if !ok {
			continue
		}
		out = append(out, MethodBindingEntry{
			TypeSymbol:       typeSym,
			MethodName:       methodName,
			MethodTypeSymbol: methodTypeSym,
		})
	}
	return out
}
