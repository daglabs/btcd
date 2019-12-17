// Copyright (c) 2016 The Decred developers
// Copyright (c) 2016-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag_test

import (
	"bytes"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaspanet/kaspad/blockdag"
	"github.com/kaspanet/kaspad/blockdag/fullblocktests"
	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/database"
	_ "github.com/kaspanet/kaspad/database/ffldb"
	"github.com/kaspanet/kaspad/txscript"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/wire"
)

const (
	// testDbType is the database backend type to use for the tests.
	testDbType = "ffldb"

	// testDbRoot is the root directory used to create all test databases.
	testDbRoot = "testdbs"

	// blockDataNet is the expected network in the test block data.
	blockDataNet = wire.MainNet
)

// filesExists returns whether or not the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// isSupportedDbType returns whether or not the passed database type is
// currently supported.
func isSupportedDbType(dbType string) bool {
	supportedDrivers := database.SupportedDrivers()
	for _, driver := range supportedDrivers {
		if dbType == driver {
			return true
		}
	}

	return false
}

// DAGSetup is used to create a new db and DAG instance with the genesis
// block already inserted. In addition to the new DAG instance, it returns
// a teardown function the caller should invoke when done testing to clean up.
func DAGSetup(dbName string, params *dagconfig.Params) (*blockdag.BlockDAG, func(), error) {
	if !isSupportedDbType(testDbType) {
		return nil, nil, errors.Errorf("unsupported db type %v", testDbType)
	}

	// Handle memory database specially since it doesn't need the disk
	// specific handling.
	var db database.DB
	var teardown func()
	if testDbType == "memdb" {
		ndb, err := database.Create(testDbType)
		if err != nil {
			return nil, nil, errors.Errorf("error creating db: %v", err)
		}
		db = ndb

		// Setup a teardown function for cleaning up. This function is
		// returned to the caller to be invoked when it is done testing.
		teardown = func() {
			db.Close()
		}
	} else {
		// Create the root directory for test databases.
		if !fileExists(testDbRoot) {
			if err := os.MkdirAll(testDbRoot, 0700); err != nil {
				err := errors.Errorf("unable to create test db "+
					"root: %v", err)
				return nil, nil, err
			}
		}

		// Create a new database to store the accepted blocks into.
		dbPath := filepath.Join(testDbRoot, dbName)
		_ = os.RemoveAll(dbPath)
		ndb, err := database.Create(testDbType, dbPath, blockDataNet)
		if err != nil {
			return nil, nil, errors.Errorf("error creating db: %v", err)
		}
		db = ndb

		// Setup a teardown function for cleaning up. This function is
		// returned to the caller to be invoked when it is done testing.
		teardown = func() {
			db.Close()
			os.RemoveAll(dbPath)
			os.RemoveAll(testDbRoot)
		}
	}

	// Copy the DAG params to ensure any modifications the tests do to
	// the DAG parameters do not affect the global instance.
	paramsCopy := *params

	// Create the DAG instance.
	dag, err := blockdag.New(&blockdag.Config{
		DB:         db,
		DAGParams:  &paramsCopy,
		TimeSource: blockdag.NewMedianTime(),
		SigCache:   txscript.NewSigCache(1000),
	})
	if err != nil {
		teardown()
		err := errors.Errorf("failed to create DAG instance: %v", err)
		return nil, nil, err
	}
	return dag, teardown, nil
}

// TestFullBlocks ensures all tests generated by the fullblocktests package
// have the expected result when processed via ProcessBlock.
func TestFullBlocks(t *testing.T) {
	// TODO: (Stas) This test was disabled for until we have implemented Phantom
	// Ticket: https://daglabs.atlassian.net/browse/DEV-60
	t.SkipNow()

	tests, err := fullblocktests.Generate(false)
	if err != nil {
		t.Fatalf("failed to generate tests: %v", err)
	}

	// Create a new database and DAG instance to run tests against.
	dag, teardownFunc, err := DAGSetup("fullblocktest",
		&dagconfig.RegressionNetParams)
	if err != nil {
		t.Errorf("Failed to setup chain instance: %v", err)
		return
	}
	defer teardownFunc()

	// testAcceptedBlock attempts to process the block in the provided test
	// instance and ensures that it was accepted according to the flags
	// specified in the test.
	testAcceptedBlock := func(item fullblocktests.AcceptedBlock) {
		blockHeight := item.Height
		block := util.NewBlock(item.Block)
		block.SetChainHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		isOrphan, delay, err := dag.ProcessBlock(block,
			blockdag.BFNone)
		if err != nil {
			t.Fatalf("block %q (hash %s, height %d) should "+
				"have been accepted: %v", item.Name,
				block.Hash(), blockHeight, err)
		}

		if delay != item.Delay {
			t.Fatalf("block %q (hash %s, height %d) unexpected "+
				"delay -- got %v, want %v", item.Name,
				block.Hash(), blockHeight, delay,
				item.Delay)
		}

		if isOrphan != item.IsOrphan {
			t.Fatalf("block %q (hash %s, height %d) unexpected "+
				"orphan flag -- got %v, want %v", item.Name,
				block.Hash(), blockHeight, isOrphan,
				item.IsOrphan)
		}
	}

	// testRejectedBlock attempts to process the block in the provided test
	// instance and ensures that it was rejected with the reject code
	// specified in the test.
	testRejectedBlock := func(item fullblocktests.RejectedBlock) {
		blockHeight := item.Height
		block := util.NewBlock(item.Block)
		block.SetChainHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		_, _, err := dag.ProcessBlock(block, blockdag.BFNone)
		if err == nil {
			t.Fatalf("block %q (hash %s, height %d) should not "+
				"have been accepted", item.Name, block.Hash(),
				blockHeight)
		}

		// Ensure the error code is of the expected type and the reject
		// code matches the value specified in the test instance.
		rerr, ok := err.(blockdag.RuleError)
		if !ok {
			t.Fatalf("block %q (hash %s, height %d) returned "+
				"unexpected error type -- got %T, want "+
				"blockchain.RuleError", item.Name, block.Hash(),
				blockHeight, err)
		}
		if rerr.ErrorCode != item.RejectCode {
			t.Fatalf("block %q (hash %s, height %d) does not have "+
				"expected reject code -- got %v, want %v",
				item.Name, block.Hash(), blockHeight,
				rerr.ErrorCode, item.RejectCode)
		}
	}

	// testRejectedNonCanonicalBlock attempts to decode the block in the
	// provided test instance and ensures that it failed to decode with a
	// message error.
	testRejectedNonCanonicalBlock := func(item fullblocktests.RejectedNonCanonicalBlock) {
		headerLen := len(item.RawBlock)
		if headerLen > 80 {
			headerLen = 80
		}
		blockHash := daghash.DoubleHashH(item.RawBlock[0:headerLen])
		blockHeight := item.Height
		t.Logf("Testing block %s (hash %s, height %d)", item.Name,
			blockHash, blockHeight)

		// Ensure there is an error due to deserializing the block.
		var msgBlock wire.MsgBlock
		err := msgBlock.KaspaDecode(bytes.NewReader(item.RawBlock), 0)
		if _, ok := err.(*wire.MessageError); !ok {
			t.Fatalf("block %q (hash %s, height %d) should have "+
				"failed to decode", item.Name, blockHash,
				blockHeight)
		}
	}

	// testOrphanOrRejectedBlock attempts to process the block in the
	// provided test instance and ensures that it was either accepted as an
	// orphan or rejected with a rule violation.
	testOrphanOrRejectedBlock := func(item fullblocktests.OrphanOrRejectedBlock) {
		blockHeight := item.Height
		block := util.NewBlock(item.Block)
		block.SetChainHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		isOrphan, delay, err := dag.ProcessBlock(block, blockdag.BFNone)
		if err != nil {
			// Ensure the error code is of the expected type.
			if _, ok := err.(blockdag.RuleError); !ok {
				t.Fatalf("block %q (hash %s, height %d) "+
					"returned unexpected error type -- "+
					"got %T, want blockdag.RuleError",
					item.Name, block.Hash(), blockHeight,
					err)
			}
		}

		if delay != 0 {
			t.Fatalf("block %q (hash %s, height %d) "+
				"is too far in the future",
				item.Name, block.Hash(), blockHeight)
		}

		if !isOrphan {
			t.Fatalf("block %q (hash %s, height %d) was accepted, "+
				"but is not considered an orphan", item.Name,
				block.Hash(), blockHeight)
		}
	}

	// testExpectedTip ensures the current tip of the blockDAG is the
	// block specified in the provided test instance.
	testExpectedTip := func(item fullblocktests.ExpectedTip) {
		blockHeight := item.Height
		block := util.NewBlock(item.Block)
		block.SetChainHeight(blockHeight)
		t.Logf("Testing tip for block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		// Ensure hash and height match.
		if dag.SelectedTipHash() != item.Block.BlockHash() ||
			dag.ChainHeight() != blockHeight { //TODO: (Ori) the use of dag.ChainHeight() and virtualBlock.HighestTipHash() is wrong, and was done only for compilation

			t.Fatalf("block %q (hash %s, height %d) should be "+
				"the current tip -- got (hash %s, height %d)",
				item.Name, block.Hash(), blockHeight, dag.SelectedTipHash(),
				dag.ChainHeight()) //TODO: (Ori) the use of dag.ChainHeight() and virtualBlock.HighestTipHash() is wrong, and was done only for compilation
		}
	}

	for testNum, test := range tests {
		for itemNum, item := range test {
			switch item := item.(type) {
			case fullblocktests.AcceptedBlock:
				testAcceptedBlock(item)
			case fullblocktests.RejectedBlock:
				testRejectedBlock(item)
			case fullblocktests.RejectedNonCanonicalBlock:
				testRejectedNonCanonicalBlock(item)
			case fullblocktests.OrphanOrRejectedBlock:
				testOrphanOrRejectedBlock(item)
			case fullblocktests.ExpectedTip:
				testExpectedTip(item)
			default:
				t.Fatalf("test #%d, item #%d is not one of "+
					"the supported test instance types -- "+
					"got type: %T", testNum, itemNum, item)
			}
		}
	}
}
