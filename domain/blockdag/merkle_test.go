// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"github.com/kaspanet/kaspad/util/daghash"
	"testing"

	"github.com/kaspanet/kaspad/util"
)

// TestMerkle tests the BuildHashMerkleTreeStore API.
func TestMerkle(t *testing.T) {
	block := util.NewBlock(&Block100000)

	hashMerkleTree := BuildHashMerkleTreeStore(block.Transactions())
	calculatedHashMerkleRoot := hashMerkleTree.Root()
	wantHashMerkleRoot := Block100000.Header.HashMerkleRoot
	if !wantHashMerkleRoot.IsEqual(calculatedHashMerkleRoot) {
		t.Errorf("BuildHashMerkleTreeStore: hash merkle root mismatch - "+
			"got %v, want %v", calculatedHashMerkleRoot, wantHashMerkleRoot)
	}

	idMerkleTree := BuildIDMerkleTreeStore(block.Transactions())
	calculatedIDMerkleRoot := idMerkleTree.Root()
	wantIDMerkleRoot, err := daghash.NewHashFromStr("3f69feb7edf5d0d67930afc990c8ec931e3428d7c7a65d7af6b81079319eb110")
	if err != nil {
		t.Errorf("BuildIDMerkleTreeStore: unexpected error: %s", err)
	}
	if !calculatedIDMerkleRoot.IsEqual(wantIDMerkleRoot) {
		t.Errorf("BuildIDMerkleTreeStore: ID merkle root mismatch - "+
			"got %v, want %v", calculatedIDMerkleRoot, wantIDMerkleRoot)
	}
}
