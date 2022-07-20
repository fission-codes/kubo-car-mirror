package bloom

import (
	"encoding/binary"
	"testing"
)

func TestBasic(t *testing.T) {
	f := New(1000, 4)
	n1 := []byte("Bess")
	n2 := []byte("Jane")
	n3 := []byte("Emma")
	f.Add(n1)
	n3a := f.Has(n3)
	f.Add(n3)
	n1b := f.Has(n1)
	n2b := f.Has(n2)
	n3b := f.Has(n3)
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
	n3a := f.Has(n3)
	f.Add(n3)
	n1b := f.Has(n1)
	n2b := f.Has(n2)
	n3b := f.Has(n3)
	n5a := f.Has(n5)
	f.Add(n5)
	n5b := f.Has(n5)
	f.Has(n4)
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
	if f.K() != 1 {
		t.Errorf("%v should be 1", f.K())
	}
	if f.M() != 1 {
		t.Errorf("%v should be 1", f.M())
	}
}

func TestK(t *testing.T) {
	f := New(1000, 4)
	if f.K() != f.k {
		t.Error("not accessing K() correctly")
	}
}

func TestM(t *testing.T) {
	f := New(1000, 4)
	if f.M() != f.m {
		t.Error("not accessing M() correctly")
	}
}

func TestB(t *testing.T) {
	f := New(1000, 4)
	if f.B() != f.b {
		t.Error("not accessing B() correctly")
	}
}

// TestEncodeDecodeCbor

// TestEqual

func TestApproximatedSize(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)
	f.Add([]byte("Love"))
	f.Add([]byte("is"))
	f.Add([]byte("in"))
	f.Add([]byte("bloom"))
	size := f.ApproximatedSize()
	if size != 4 {
		t.Errorf("%d should equal 4.", size)
	}
}

func TestFrom(t *testing.T) {
	var (
		k    = uint64(5)
		data = make([]uint64, 10)
		test = []byte("test")
	)

	bf := From(data, k)
	if bf.K() != k {
		t.Errorf("Constant k does not match the expected value")
	}

	if bf.M() != uint64(len(data)*64) {
		t.Errorf("Capacity does not match the expected value")
	}

	if bf.Has(test) {
		t.Errorf("Bloom filter should not contain the value")
	}

	bf.Add(test)
	if !bf.Has(test) {
		t.Errorf("Bloom filter should contain the value")
	}

	// create a new Bloom filter from an existing (populated) data slice.
	bf = From(data, k)
	if !bf.Has(test) {
		t.Errorf("Bloom filter should contain the value")
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
		if f.Has(b) {
			count += 1
		}
	}
	if f.FPP(1000) > 0.001 {
		t.Errorf("Excessive FPP()! n=%v, m=%v, k=%v, fpp=%v", 1000, f.M(), f.K(), f.FPP(1000))
	}
}
