package dag

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
)

// Converts a list of CID strings into a list of `cid.Cid`s.
// Returns `nil, error` if any CID strings fail to parse.
func ParseCids(cids []string) ([]cid.Cid, error) {
	var rootCids []cid.Cid

	// Create a CAR file containing all CIDs
	for _, rootCid := range cids {
		parsedRootCid, err := ParseCid(rootCid)
		if err != nil {
			return nil, err
		}
		rootCids = append(rootCids, *parsedRootCid)
	}

	return rootCids, nil
}

func ParseCid(cidStr string) (*cid.Cid, error) {
	parsedCid, err := cid.Decode(cidStr)
	if err != nil {
		return nil, fmt.Errorf("CID %q cannot be parsed: %v", cidStr, err)
	}

	return &parsedCid, nil
}

// // TODO: Rename
// type Tracker struct {
// 	Context context.Context
// 	Filter  *bloom.Filter
// 	api     coreiface.CoreAPI
// }

// func NewTracker(filter *bloom.Filter) *Tracker {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	t := &Tracker{Filter: filter, Context: ctx}

// 	return t
// }

// func (t *Tracker) AddToBloom(cid string) error {
// 	p := path.New(cid)
// 	// TODO: Don't use Unixfs resolver?
// 	rp, err := t.api.ResolvePath(t.Context, p)
// 	if err != nil {
// 		return err
// 	}

// 	nodeGetter := mdag.NewSession(t.Context, t.api.Dag())
// 	// consider setting a deadline in the context
// 	obj, err := nodeGetter.Get(t.Context, rp.Cid())
// 	if err != nil {
// 		return err
// 	}

// 	err = traverse.Traverse(obj, traverse.Options{
// 		DAG: nodeGetter,
// 		// Pull is pre-order traversal
// 		Order: traverse.DFSPre,
// 		Func: func(current traverse.State) error {
// 			// Question: do we have to add the raw data, not just he CID, due to lack of trust?
// 			t.Filter.Add(current.Node.Cid().Bytes())

// 			return nil
// 		},
// 		ErrFunc:        nil,
// 		SkipDuplicates: true,
// 	})

// 	if err != nil {
// 		return fmt.Errorf("error traversing DAG: %w", err)
// 	}

// 	return nil
// }
