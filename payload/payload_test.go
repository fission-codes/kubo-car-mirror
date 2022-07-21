package payload

import (
	"reflect"
	"testing"

	"gotest.tools/assert"
)

func TestEncodeDecode(t *testing.T) {
	d := PullRequestor{RS: []string{"a", "b", "c"}, BK: 8, BB: []byte("somebytes")}
	de, err := Encode(d)
	if err != nil {
		t.Error(err)
	}

	var dd PullRequestor
	if err := Decode(de, &dd); err != nil {
		t.Error(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(d.RS, dd.RS), "must be equal")
		assert.Assert(t, reflect.DeepEqual(d.BK, dd.BK), "must be equal")
		assert.Assert(t, reflect.DeepEqual(d.BB, dd.BB), "must be equal")
	}
}
