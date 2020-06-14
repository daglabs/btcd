package blockdag

import (
	"testing"

	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/util/daghash"
)

// TestBlockHeap tests pushing, popping, and determining the length of the heap.
func TestBlockHeap(t *testing.T) {
	// Create a new database and DAG instance to run tests against.
	dag, teardownFunc, err := DAGSetup("TestBlockHeap", true, Config{
		DAGParams: &dagconfig.SimnetParams,
	})
	if err != nil {
		t.Fatalf("TestBlockHeap: Failed to setup DAG instance: %s", err)
	}
	defer teardownFunc()

	block0Header := dagconfig.SimnetParams.GenesisBlock.Header
	block0, _ := dag.newBlockNode(&block0Header, newBlockSet())

	block100000Header := Block100000.Header
	block100000, _ := dag.newBlockNode(&block100000Header, blockSetFromSlice(block0))

	block0smallHash, _ := dag.newBlockNode(&block0Header, newBlockSet())
	block0smallHash.hash = &daghash.Hash{}

	tests := []struct {
		name            string
		toPush          []*blockNode
		expectedLength  int
		expectedPopUp   *blockNode
		expectedPopDown *blockNode
	}{
		{
			name:            "empty heap must have length 0",
			toPush:          []*blockNode{},
			expectedLength:  0,
			expectedPopDown: nil,
			expectedPopUp:   nil,
		},
		{
			name:            "heap with one push must have length 1",
			toPush:          []*blockNode{block0},
			expectedLength:  1,
			expectedPopDown: nil,
			expectedPopUp:   nil,
		},
		{
			name:            "heap with one push and one pop",
			toPush:          []*blockNode{block0},
			expectedLength:  0,
			expectedPopDown: block0,
			expectedPopUp:   block0,
		},
		{
			name: "push two blocks with different heights, heap shouldn't have to rebalance " +
				"for down direction, but will have to rebalance for up direction",
			toPush:          []*blockNode{block100000, block0},
			expectedLength:  1,
			expectedPopDown: block100000,
			expectedPopUp:   block0,
		},
		{
			name: "push two blocks with different heights, heap shouldn't have to rebalance " +
				"for up direction, but will have to rebalance for down direction",
			toPush:          []*blockNode{block0, block100000},
			expectedLength:  1,
			expectedPopDown: block100000,
			expectedPopUp:   block0,
		},
		{
			name: "push two blocks with equal heights but different hashes, heap shouldn't have to rebalance " +
				"for down direction, but will have to rebalance for up direction",
			toPush:          []*blockNode{block0, block0smallHash},
			expectedLength:  1,
			expectedPopDown: block0,
			expectedPopUp:   block0smallHash,
		},
		{
			name: "push two blocks with equal heights but different hashes, heap shouldn't have to rebalance " +
				"for up direction, but will have to rebalance for down direction",
			toPush:          []*blockNode{block0smallHash, block0},
			expectedLength:  1,
			expectedPopDown: block0,
			expectedPopUp:   block0smallHash,
		},
	}

	for _, test := range tests {
		dHeap := newDownHeap()
		for _, block := range test.toPush {
			dHeap.Push(block)
		}

		var poppedBlock *blockNode
		if test.expectedPopDown != nil {
			poppedBlock = dHeap.pop()
		}
		if dHeap.Len() != test.expectedLength {
			t.Errorf("unexpected down heap length in test \"%s\". "+
				"Expected: %v, got: %v", test.name, test.expectedLength, dHeap.Len())
		}
		if poppedBlock != test.expectedPopDown {
			t.Errorf("unexpected popped block for down heap in test \"%s\". "+
				"Expected: %v, got: %v", test.name, test.expectedPopDown, poppedBlock)
		}

		uHeap := newUpHeap()
		for _, block := range test.toPush {
			uHeap.Push(block)
		}

		poppedBlock = nil
		if test.expectedPopUp != nil {
			poppedBlock = uHeap.pop()
		}
		if uHeap.Len() != test.expectedLength {
			t.Errorf("unexpected up heap length in test \"%s\". "+
				"Expected: %v, got: %v", test.name, test.expectedLength, uHeap.Len())
		}
		if poppedBlock != test.expectedPopUp {
			t.Errorf("unexpected popped block for up heap in test \"%s\". "+
				"Expected: %v, got: %v", test.name, test.expectedPopDown, poppedBlock)
		}
	}
}
