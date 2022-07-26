package bloom

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestBasic(t *testing.T) {
	f := New(1000, 4)

	if f.bitSet.BitsCount() != 1000 {
		t.Errorf("should be sized to %v, got %v.", 1000, f.bitSet.BitsCount())
	}

	expectedBytes := int(math.Ceil(1000 / 8))
	if len(f.Bytes()) != expectedBytes {
		// This is off because .Bytes() is not a list of bytes.  It's a list of uint64s.
		t.Errorf("should be sized to %v, got %v.", expectedBytes, len(f.Bytes()))
	}

	n1 := []byte("one")
	n2 := []byte("two")
	n3 := []byte("three")
	f.Add(n1)
	n3a := f.Test(n3)
	f.Add(n3)
	n1b := f.Test(n1)
	n2b := f.Test(n2)
	n3b := f.Test(n3)
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
}

func TestBasicUint32(t *testing.T) {
	f := New(1000, 4)
	n1 := make([]byte, 4)
	n2 := make([]byte, 4)
	n3 := make([]byte, 4)
	n4 := make([]byte, 4)
	n5 := make([]byte, 4)
	binary.BigEndian.PutUint32(n1, 100)
	binary.BigEndian.PutUint32(n2, 101)
	binary.BigEndian.PutUint32(n3, 102)
	binary.BigEndian.PutUint32(n4, 103)
	binary.BigEndian.PutUint32(n5, 104)
	f.Add(n1)
	n3a := f.Test(n3)
	f.Add(n3)
	n1b := f.Test(n1)
	n2b := f.Test(n2)
	n3b := f.Test(n3)
	n5a := f.Test(n5)
	f.Add(n5)
	n5b := f.Test(n5)
	f.Test(n4)
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
	if n5a {
		t.Errorf("%v should not be in the first time we look.", n5)
	}
	if !n5b {
		t.Errorf("%v should be in the second time we look.", n5)
	}
}

func TestNewWithLowNumbers(t *testing.T) {
	f := New(0, 0)
	if f.HashCount() != 1 {
		t.Errorf("%v should be 1", f.HashCount())
	}
	if f.BitCount() != 1 {
		t.Errorf("%v should be 1", f.BitCount())
	}
}

func TestK(t *testing.T) {
	f := New(1000, 4)
	if f.HashCount() != f.hashCount {
		t.Error("not accessing K() correctly")
	}
}

func TestM(t *testing.T) {
	f := New(1000, 4)
	if f.BitCount() != f.bitCount {
		t.Error("not accessing M() correctly")
	}
}

func TestB(t *testing.T) {
	b := make([]byte, 8)
	u := uint64(1)
	binary.BigEndian.PutUint64(b, u)

	f := New(8, 1)
	expected := []byte{byte(0)}
	if !bytes.Equal(f.Bytes(), expected) {
		t.Errorf("expected B() to be %v, got %v", expected, f.Bytes())
	}
}

func TestFPP(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)

	for i := uint32(0); i < 1000; i++ {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, i)
		f.Add(b)
	}
	count := 0

	for i := uint32(0); i < 1000; i++ {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, i+1000)
		if f.Test(b) {
			count += 1
		}
	}
	if f.FPP(1000) > 0.001 {
		t.Errorf("Excessive FPP()! n=%v, m=%v, k=%v, fpp=%v", 1000, f.BitCount(), f.HashCount(), f.FPP(1000))
	}
}
