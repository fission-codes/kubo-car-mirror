package bitset

import (
	"bytes"
	"testing"
)

func TestSize(t *testing.T) {
	b1 := New(7)
	if b1.BitsCount() != 7 {
		t.Errorf("should have %v bits, got %v", 7, b1.BitsCount())
	}
	if b1.BytesCount() != 1 {
		t.Errorf("should have %v bytes, got %v", 1, b1.BytesCount())
	}

	b2 := New(8)
	if b2.BitsCount() != 8 {
		t.Errorf("should have %v bits, got %v", 8, b2.BitsCount())
	}
	if b2.BytesCount() != 1 {
		t.Errorf("should have %v bytes, got %v", 1, b2.BytesCount())
	}

	b3 := New(9)
	if b3.BitsCount() != 9 {
		t.Errorf("should have %v bits, got %v", 9, b3.BitsCount())
	}
	if b3.BytesCount() != 2 {
		t.Errorf("should have %v bytes, got %v", 2, b3.BytesCount())
	}
}

func TestNewFromBytes(t *testing.T) {
	b1 := New(1000)
	b1.Set(1)
	b1.Set(100)
	b1.Set(354)
	b2 := NewFromBytes(1000, b1.Bytes())
	if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
		t.Errorf("bytes should be identical")
	}
}

func TestSetAndTest(t *testing.T) {
	b1 := New(1000)
	if b1.BytesCount() != 125 {
		t.Errorf("should have %v bytes, got %v", 125, b1.BytesCount())
	}
	if b1.OnesCount() != 0 {
		t.Errorf("should have %v ones, got %v", 0, b1.OnesCount())
	}

	b1.Set(7)
	if b1.OnesCount() != 1 {
		t.Errorf("should have %v ones, got %v", 1, b1.OnesCount())
	}
	if !b1.Test(7) {
		t.Error("should be true, got false")
	}

	b1.Set(27)
	if b1.OnesCount() != 2 {
		t.Errorf("should have %v ones, got %v", 2, b1.OnesCount())
	}
	if !b1.Test(27) {
		t.Error("should be true, got false")
	}
	if b1.Test(230) {
		t.Error("should be false, got true")
	}
}

func TestHexEncode(t *testing.T) {
	b := New(1)
	b.Set(0)
	if b.HexEncode() != "01" {
		t.Errorf("expected 01, got %v", b.HexEncode())
	}
	b.Set(7)
	if b.HexEncode() != "81" {
		t.Errorf("expected 01, got %v", b.HexEncode())
	}
}
