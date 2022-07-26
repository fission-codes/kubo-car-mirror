package payload

import (
	"reflect"
	"testing"

	"github.com/fission-suite/car-mirror/bloom"
	"gotest.tools/assert"
)

func TestEncodeDecode(t *testing.T) {

	f := bloom.New(128, 6)
	f.Add([]byte("one"))
	f.Add([]byte("two"))
	f.Add([]byte("three"))
	f.Add([]byte("four"))
	bb := f.Bytes()

	// Pull requestor
	pullRequest := PullRequestor{RS: []string{"a", "b", "c"}, BK: 6, BB: bb}
	pullRequestEncoded, err := CborEncode(pullRequest)
	assert.NilError(t, err)

	var pullRequestDecoded PullRequestor
	if err := CborDecode(pullRequestEncoded, &pullRequestDecoded); err != nil {
		t.Error(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(pullRequest.RS, pullRequestDecoded.RS), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pullRequest.BK, pullRequestDecoded.BK), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pullRequest.BB, pullRequestDecoded.BB), "must be equal")
	}

	// Pull provider is just CARv1

	// Push Requestor
	pushRequest := PushRequestor{BK: 6, BB: bb, PL: nil}
	pushRequestEncoded, err := CborEncode(pushRequest)
	assert.NilError(t, err)

	var pushRequestDecoded PushRequestor
	if err := CborDecode(pushRequestEncoded, &pushRequestDecoded); err != nil {
		t.Error(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(pushRequest.BK, pushRequestDecoded.BK), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pushRequest.BB, pushRequestDecoded.BB), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pushRequest.PL, pushRequestDecoded.PL), "must be equal")
	}

	// Push Provider
	pushProvide := PushProvider{SR: []string{"a", "b", "c"}, BK: 6, BB: bb}
	pushProvideEncoded, err := CborEncode(pushProvide)
	assert.NilError(t, err)

	var pushProvideDecoded PushProvider
	if err := CborDecode(pushProvideEncoded, &pushProvideDecoded); err != nil {
		t.Error(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(pushProvide.SR, pushProvideDecoded.SR), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pushProvide.BK, pushProvideDecoded.BK), "must be equal")
		assert.Assert(t, reflect.DeepEqual(pushProvide.BB, pushProvideDecoded.BB), "must be equal")
	}
}
