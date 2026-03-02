package ssa

import "testing"

func TestDecodeMethodBindingPayload(t *testing.T) {
	payload := encodeMethodBindingPayload("type.sym", "M", "func.type")
	typeSym, methodName, methodTypeSym, ok := DecodeMethodBindingPayload(payload)
	if !ok {
		t.Fatalf("DecodeMethodBindingPayload returned !ok")
	}
	if typeSym != "type.sym" || methodName != "M" || methodTypeSym != "func.type" {
		t.Fatalf("unexpected decode result: (%q,%q,%q)", typeSym, methodName, methodTypeSym)
	}
}

func TestDecodeMethodBindingAttrValue(t *testing.T) {
	p1 := encodeMethodBindingPayload("type.A", "M1", "func.A")
	p2 := encodeMethodBindingPayload("type.B", "M2", "func.B")
	joined := p1 + methodBindingListSep + p2

	items := DecodeMethodBindingAttrValue(joined)
	if len(items) != 2 {
		t.Fatalf("DecodeMethodBindingAttrValue len=%d, want 2", len(items))
	}
	if items[0].TypeSymbol != "type.A" || items[0].MethodName != "M1" || items[0].MethodTypeSymbol != "func.A" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].TypeSymbol != "type.B" || items[1].MethodName != "M2" || items[1].MethodTypeSymbol != "func.B" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}

func TestMergeMethodBindingPayload(t *testing.T) {
	p1 := encodeMethodBindingPayload("type.A", "M1", "func.A")
	p2 := encodeMethodBindingPayload("type.B", "M2", "func.B")
	merged := mergeMethodBindingPayload(p1, p2)
	merged = mergeMethodBindingPayload(merged, p1)

	items := DecodeMethodBindingAttrValue(merged)
	if len(items) != 2 {
		t.Fatalf("merged items len=%d, want 2", len(items))
	}
}
