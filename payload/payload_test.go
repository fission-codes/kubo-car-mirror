package payload

import (
	"reflect"
	"testing"

	"github.com/fission-suite/car-mirror/bloom"
	"gotest.tools/assert"
)

func TestEncodeDecode(t *testing.T) {

	f := bloom.New(100, 6)
	f.Add([]byte("one"))
	f.Add([]byte("two"))
	f.Add([]byte("three"))
	f.Add([]byte("four"))
	bb := f.Bytes()
	// assert.NilError(t, err, "Failed to marshal binary from bitset")

	d := PullRequestor{RS: []string{"a", "b", "c"}, BK: 8, BB: bb}
	de, err := Encode(d)
	assert.NilError(t, err)

	var dd PullRequestor
	if err := Decode(de, &dd); err != nil {
		t.Error(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(d.RS, dd.RS), "must be equal")
		assert.Assert(t, reflect.DeepEqual(d.BK, dd.BK), "must be equal")
		assert.Assert(t, reflect.DeepEqual(d.BB, dd.BB), "must be equal")
	}

	// bbEncoded, _ := Encode(bb)
	// err = os.WriteFile("/tmp/bloom1.cbor", bbEncoded, 0644)
	// if err != nil {
	// 	fmt.Printf("Failed to write file.  err: %v\n", err)
	// }

	// encodedStr := hex.EncodeToString(bb)

	// fmt.Printf("Binary BB = %b\n", bb)
	// fmt.Printf("Hex BB    = %s\n", encodedStr)
	// fmt.Printf("CBOR BB   = %b\n", bbEncoded)
}
