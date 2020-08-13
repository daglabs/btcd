// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaspanet/kaspad/domain/txscript"
)

// TestCheckBlockScripts ensures that validating the all of the scripts in a
// known-good block doesn't return an error.
func TestCheckBlockScripts(t *testing.T) {
	t.Skip() // TODO: Reactivate this test once we have blocks from testnet.
	runtime.GOMAXPROCS(runtime.NumCPU())

	testBlockNum := 277647
	blockDataFile := fmt.Sprintf("%d.dat", testBlockNum)
	blocks, err := LoadBlocks(filepath.Join("testdata/", blockDataFile))
	if err != nil {
		t.Errorf("Error loading file: %v\n", err)
		return
	}
	if len(blocks) > 1 {
		t.Errorf("The test block file must only have one block in it")
		return
	}
	if len(blocks) == 0 {
		t.Errorf("The test block file may not be empty")
		return
	}

	storeDataFile := fmt.Sprintf("%d.utxostore", testBlockNum)
	utxoSet, err := loadUTXOSet(storeDataFile)
	if err != nil {
		t.Errorf("Error loading txstore: %v\n", err)
		return
	}

	node := &blockNode{
		hash: blocks[0].Hash(),
	}

	scriptFlags := txscript.ScriptNoFlags
	err = checkBlockScripts(node, utxoSet, blocks[0].Transactions(), scriptFlags, nil)
	if err != nil {
		t.Errorf("Transaction script validation failed: %v\n", err)
		return
	}
}
