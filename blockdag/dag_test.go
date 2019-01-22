// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/daglabs/btcd/database"

	"math/rand"

	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/wire"
)

func TestBlockCount(t *testing.T) {
	// Load up blocks such that there is a fork in the DAG.
	// (genesis block) -> 1 -> 2 -> 3 -> 4
	//                          \-> 3b
	testFiles := []string{
		"blk_0_to_4.dat",
		"blk_3B.dat",
	}

	var blocks []*util.Block
	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Fatalf("Error loading file: %v\n", err)
		}
		blocks = append(blocks, blockTmp...)
	}

	// Create a new database and DAG instance to run tests against.
	dag, teardownFunc, err := DAGSetup("haveblock", Config{
		DAGParams:    &dagconfig.SimNetParams,
		SubnetworkID: &wire.SubnetworkIDSupportsAll,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()

	// Since we're not dealing with the real block DAG, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	for i := 1; i < len(blocks); i++ {
		isOrphan, err := dag.ProcessBlock(blocks[i], BFNone)
		if err != nil {
			t.Fatalf("ProcessBlock fail on block %v: %v\n", i, err)
		}
		if isOrphan {
			t.Fatalf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
		}
	}

	expectedBlockCount := uint64(6)
	if dag.BlockCount() != expectedBlockCount {
		t.Errorf("TestBlockCount: BlockCount expected to return %v but got %v", expectedBlockCount, dag.BlockCount())
	}
}

// TestHaveBlock tests the HaveBlock API to ensure proper functionality.
func TestHaveBlock(t *testing.T) {
	// Load up blocks such that there is a fork in the DAG.
	// (genesis block) -> 1 -> 2 -> 3 -> 4
	//                          \-> 3b
	testFiles := []string{
		"blk_0_to_4.dat",
		"blk_3B.dat",
	}

	var blocks []*util.Block
	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Fatalf("Error loading file: %v\n", err)
		}
		blocks = append(blocks, blockTmp...)
	}

	// Create a new database and chain instance to run tests against.
	dag, teardownFunc, err := DAGSetup("haveblock", Config{
		DAGParams:    &dagconfig.SimNetParams,
		SubnetworkID: &wire.SubnetworkIDSupportsAll,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()

	// Since we're not dealing with the real block chain, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	for i := 1; i < len(blocks); i++ {
		isOrphan, err := dag.ProcessBlock(blocks[i], BFNone)
		if err != nil {
			t.Fatalf("ProcessBlock fail on block %v: %v\n", i, err)
		}
		if isOrphan {
			t.Fatalf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
		}
	}

	// Test a block with related parents
	testFiles = []string{
		"blk_3C.dat",
	}

	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Fatalf("Error loading file: %v\n", err)
		}
		blocks = append(blocks, blockTmp...)
	}
	isOrphan, err := dag.ProcessBlock(blocks[6], BFNone)

	// Block 3C should fail to connect since its parents are related. (It points to 1 and 2, and 1 is the parent of 2)
	if err == nil {
		t.Fatalf("ProcessBlock for block 3C has no error when expected to have an error\n")
	}
	if isOrphan {
		t.Fatalf("ProcessBlock incorrectly returned block 3C " +
			"is an orphan\n")
	}

	// Test a block with the same input twice
	testFiles = []string{
		"blk_3D.dat",
	}

	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Fatalf("Error loading file: %v\n", err)
		}
		blocks = append(blocks, blockTmp...)
	}
	isOrphan, err = dag.ProcessBlock(blocks[7], BFNone)

	// Block 3D should fail to connect since it has a transaction with the same input twice
	if err == nil {
		t.Fatalf("ProcessBlock for block 3D has no error when expected to have an error\n")
	}
	rErr, ok := err.(RuleError)
	if !ok {
		t.Fatalf("ProcessBlock for block 3D expected a RuleError, but got something else\n")
	}
	if !ok || rErr.ErrorCode != ErrDuplicateTxInputs {
		t.Fatalf("ProcessBlock for block 3D expected error code %s but got %s\n", ErrDuplicateTxInputs, rErr.ErrorCode)
	}
	if isOrphan {
		t.Fatalf("ProcessBlock incorrectly returned block 3D " +
			"is an orphan\n")
	}

	// Insert an orphan block.
	isOrphan, err = dag.ProcessBlock(util.NewBlock(&Block100000),
		BFNone)
	if err != nil {
		t.Fatalf("Unable to process block: %v", err)
	}
	if !isOrphan {
		t.Fatalf("ProcessBlock indicated block is an not orphan when " +
			"it should be\n")
	}

	tests := []struct {
		hash string
		want bool
	}{
		// Genesis block should be present.
		{hash: dagconfig.SimNetParams.GenesisHash.String(), want: true},

		// Block 3b should be present (as a second child of Block 2).
		{hash: "2664223a8b2abba475ed5760433e8204806c17b60f12d826b876cccbf5f74be6", want: true},

		// Block 100000 should be present (as an orphan).
		{hash: "66965d8ebcdccae2b3791f652326ef1063fa0a7e506c66f68e0c7bbb59104711", want: true},

		// Random hashes should not be available.
		{hash: "123", want: false},
	}

	for i, test := range tests {
		hash, err := daghash.NewHashFromStr(test.hash)
		if err != nil {
			t.Fatalf("NewHashFromStr: %v", err)
		}

		result, err := dag.HaveBlock(hash)
		if err != nil {
			t.Fatalf("HaveBlock #%d unexpected error: %v", i, err)
		}
		if result != test.want {
			t.Fatalf("HaveBlock #%d got %v want %v", i, result,
				test.want)
		}
	}
}

// TestCalcSequenceLock tests the LockTimeToSequence function, and the
// CalcSequenceLock method of a Chain instance. The tests exercise several
// combinations of inputs to the CalcSequenceLock function in order to ensure
// the returned SequenceLocks are correct for each test instance.
func TestCalcSequenceLock(t *testing.T) {
	netParams := &dagconfig.SimNetParams

	blockVersion := int32(0x10000000)

	// Generate enough synthetic blocks for the rest of the test
	dag := newTestDAG(netParams)
	node := dag.selectedTip()
	blockTime := node.Header().Timestamp
	numBlocksToGenerate := uint32(5)
	for i := uint32(0); i < numBlocksToGenerate; i++ {
		blockTime = blockTime.Add(time.Second)
		node = newTestNode(setFromSlice(node), blockVersion, 0, blockTime, netParams.K)
		dag.index.AddNode(node)
		dag.virtual.SetTips(setFromSlice(node))
	}

	// Create a utxo view with a fake utxo for the inputs used in the
	// transactions created below.  This utxo is added such that it has an
	// age of 4 blocks.
	targetTx := util.NewTx(&wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    10,
		}},
	})
	utxoSet := NewFullUTXOSet()
	utxoSet.AddTx(targetTx.MsgTx(), int32(numBlocksToGenerate)-4)

	// Create a utxo that spends the fake utxo created above for use in the
	// transactions created in the tests.  It has an age of 4 blocks.  Note
	// that the sequence lock heights are always calculated from the same
	// point of view that they were originally calculated from for a given
	// utxo.  That is to say, the height prior to it.
	utxo := wire.OutPoint{
		Hash:  *targetTx.Hash(),
		Index: 0,
	}
	prevUtxoHeight := int32(numBlocksToGenerate) - 4

	// Obtain the median time past from the PoV of the input created above.
	// The MTP for the input is the MTP from the PoV of the block *prior*
	// to the one that included it.
	medianTime := node.RelativeAncestor(5).CalcPastMedianTime().Unix()

	// The median time calculated from the PoV of the best block in the
	// test chain.  For unconfirmed inputs, this value will be used since
	// the MTP will be calculated from the PoV of the yet-to-be-mined
	// block.
	nextMedianTime := node.CalcPastMedianTime().Unix()
	nextBlockHeight := int32(numBlocksToGenerate) + 1

	// Add an additional transaction which will serve as our unconfirmed
	// output.
	unConfTx := &wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    5,
		}},
	}
	unConfUtxo := wire.OutPoint{
		Hash:  unConfTx.TxHash(),
		Index: 0,
	}

	// Adding a utxo with a height of 0x7fffffff indicates that the output
	// is currently unmined.
	utxoSet.AddTx(unConfTx, 0x7fffffff)

	tests := []struct {
		tx      *wire.MsgTx
		utxoSet UTXOSet
		mempool bool
		want    *SequenceLock
	}{
		// A transaction with a single input with max sequence number.
		// This sequence number has the high bit set, so sequence locks
		// should be disabled.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         wire.MaxTxInSequenceNum,
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: -1,
			},
		},
		// A transaction with a single input whose lock time is
		// expressed in seconds.  However, the specified lock time is
		// below the required floor for time based lock times since
		// they have time granularity of 512 seconds.  As a result, the
		// seconds lock-time should be just before the median time of
		// the targeted block.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     medianTime - 1,
				BlockHeight: -1,
			},
		},
		// A transaction with a single input whose lock time is
		// expressed in seconds.  The number of seconds should be 1023
		// seconds after the median past time of the last block in the
		// chain.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 1024),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     medianTime + 1023,
				BlockHeight: -1,
			},
		},
		// A transaction with multiple inputs.  The first input has a
		// lock time expressed in seconds.  The second input has a
		// sequence lock in blocks with a value of 4.  The last input
		// has a sequence number with a value of 5, but has the disable
		// bit set.  So the first lock should be selected as it's the
		// latest lock that isn't disabled.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 4),
				}, {
					PreviousOutPoint: utxo,
					Sequence: LockTimeToSequence(false, 5) |
						wire.SequenceLockTimeDisabled,
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     medianTime + (5 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: prevUtxoHeight + 3,
			},
		},
		// Transaction with a single input.  The input's sequence number
		// encodes a relative lock-time in blocks (3 blocks).  The
		// sequence lock should  have a value of -1 for seconds, but a
		// height of 2 meaning it can be included at height 3.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 3),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: prevUtxoHeight + 2,
			},
		},
		// A transaction with two inputs with lock times expressed in
		// seconds.  The selected sequence lock value for seconds should
		// be the time further in the future.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 5120),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     medianTime + (10 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: -1,
			},
		},
		// A transaction with two inputs with lock times expressed in
		// blocks.  The selected sequence lock value for blocks should
		// be the height further in the future, so a height of 10
		// indicating it can be included at height 11.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 1),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 11),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: prevUtxoHeight + 10,
			},
		},
		// A transaction with multiple inputs.  Two inputs are time
		// based, and the other two are block based. The lock lying
		// further into the future for both inputs should be chosen.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 6656),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 3),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 9),
				}},
			},
			utxoSet: utxoSet,
			want: &SequenceLock{
				Seconds:     medianTime + (13 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: prevUtxoHeight + 8,
			},
		},
		// A transaction with a single unconfirmed input.  As the input
		// is confirmed, the height of the input should be interpreted
		// as the height of the *next* block.  So, a 2 block relative
		// lock means the sequence lock should be for 1 block after the
		// *next* block height, indicating it can be included 2 blocks
		// after that.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: unConfUtxo,
					Sequence:         LockTimeToSequence(false, 2),
				}},
			},
			utxoSet: utxoSet,
			mempool: true,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: nextBlockHeight + 1,
			},
		},
		// A transaction with a single unconfirmed input.  The input has
		// a time based lock, so the lock time should be based off the
		// MTP of the *next* block.
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: unConfUtxo,
					Sequence:         LockTimeToSequence(true, 1024),
				}},
			},
			utxoSet: utxoSet,
			mempool: true,
			want: &SequenceLock{
				Seconds:     nextMedianTime + 1023,
				BlockHeight: -1,
			},
		},
	}

	t.Logf("Running %v SequenceLock tests", len(tests))
	for i, test := range tests {
		utilTx := util.NewTx(test.tx)
		seqLock, err := dag.CalcSequenceLock(utilTx, utxoSet, test.mempool)
		if err != nil {
			t.Fatalf("test #%d, unable to calc sequence lock: %v", i, err)
		}

		if seqLock.Seconds != test.want.Seconds {
			t.Fatalf("test #%d got %v seconds want %v seconds",
				i, seqLock.Seconds, test.want.Seconds)
		}
		if seqLock.BlockHeight != test.want.BlockHeight {
			t.Fatalf("test #%d got height of %v want height of %v ",
				i, seqLock.BlockHeight, test.want.BlockHeight)
		}
	}
}

func TestCalcPastMedianTime(t *testing.T) {
	netParams := &dagconfig.SimNetParams

	blockVersion := int32(0x10000000)

	dag := newTestDAG(netParams)
	numBlocks := uint32(60)
	nodes := make([]*blockNode, numBlocks)
	nodes[0] = dag.genesis
	blockTime := dag.genesis.Header().Timestamp
	for i := uint32(1); i < numBlocks; i++ {
		blockTime = blockTime.Add(time.Second)
		nodes[i] = newTestNode(setFromSlice(nodes[i-1]), blockVersion, 0, blockTime, netParams.K)
		dag.index.AddNode(nodes[i])
	}

	tests := []struct {
		blockNumber                 uint32
		expectedSecondsSinceGenesis int64
	}{
		{
			blockNumber:                 50,
			expectedSecondsSinceGenesis: 25,
		},
		{
			blockNumber:                 59,
			expectedSecondsSinceGenesis: 34,
		},
		{
			blockNumber:                 40,
			expectedSecondsSinceGenesis: 15,
		},
		{
			blockNumber:                 5,
			expectedSecondsSinceGenesis: 0,
		},
	}

	for _, test := range tests {
		secondsSinceGenesis := nodes[test.blockNumber].CalcPastMedianTime().Unix() - dag.genesis.Header().Timestamp.Unix()
		if secondsSinceGenesis != test.expectedSecondsSinceGenesis {
			t.Errorf("TestCalcPastMedianTime: expected past median time of block %v to be %v seconds from genesis but got %v", test.blockNumber, test.expectedSecondsSinceGenesis, secondsSinceGenesis)
		}
	}

}

// nodeHashes is a convenience function that returns the hashes for all of the
// passed indexes of the provided nodes.  It is used to construct expected hash
// slices in the tests.
func nodeHashes(nodes []*blockNode, indexes ...int) []daghash.Hash {
	hashes := make([]daghash.Hash, 0, len(indexes))
	for _, idx := range indexes {
		hashes = append(hashes, nodes[idx].hash)
	}
	return hashes
}

// testNoncePrng provides a deterministic prng for the nonce in generated fake
// nodes.  The ensures that the node have unique hashes.
var testNoncePrng = rand.New(rand.NewSource(0))

// chainedNodes returns the specified number of nodes constructed such that each
// subsequent node points to the previous one to create a chain.  The first node
// will point to the passed parent which can be nil if desired.
func chainedNodes(parents blockSet, numNodes int) []*blockNode {
	nodes := make([]*blockNode, numNodes)
	tips := parents
	for i := 0; i < numNodes; i++ {
		// This is invalid, but all that is needed is enough to get the
		// synthetic tests to work.
		header := wire.BlockHeader{Nonce: testNoncePrng.Uint64()}
		header.ParentHashes = tips.hashes()
		nodes[i] = newBlockNode(&header, tips, dagconfig.SimNetParams.K)
		tips = setFromSlice(nodes[i])
	}
	return nodes
}

// testTip is a convenience function to grab the tip of a chain of block nodes
// created via chainedNodes.
func testTip(nodes []*blockNode) *blockNode {
	return nodes[len(nodes)-1]
}

// TestHeightToHashRange ensures that fetching a range of block hashes by start
// height and end hash works as expected.
func TestHeightToHashRange(t *testing.T) {
	// Construct a synthetic block chain with a block index consisting of
	// the following structure.
	// 	genesis -> 1 -> 2 -> ... -> 15 -> 16  -> 17  -> 18
	// 	                              \-> 16a -> 17a -> 18a (unvalidated)
	tip := testTip
	blockDAG := newTestDAG(&dagconfig.SimNetParams)
	branch0Nodes := chainedNodes(setFromSlice(blockDAG.genesis), 18)
	branch1Nodes := chainedNodes(setFromSlice(branch0Nodes[14]), 3)
	for _, node := range branch0Nodes {
		blockDAG.index.SetStatusFlags(node, statusValid)
		blockDAG.index.AddNode(node)
	}
	for _, node := range branch1Nodes {
		if node.height < 18 {
			blockDAG.index.SetStatusFlags(node, statusValid)
		}
		blockDAG.index.AddNode(node)
	}
	blockDAG.virtual.SetTips(setFromSlice(tip(branch0Nodes)))

	tests := []struct {
		name        string
		startHeight int32          // locator for requested inventory
		endHash     daghash.Hash   // stop hash for locator
		maxResults  int            // max to locate, 0 = wire const
		hashes      []daghash.Hash // expected located hashes
		expectError bool
	}{
		{
			name:        "blocks below tip",
			startHeight: 11,
			endHash:     branch0Nodes[14].hash,
			maxResults:  10,
			hashes:      nodeHashes(branch0Nodes, 10, 11, 12, 13, 14),
		},
		{
			name:        "blocks on main chain",
			startHeight: 15,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			hashes:      nodeHashes(branch0Nodes, 14, 15, 16, 17),
		},
		{
			name:        "blocks on stale chain",
			startHeight: 15,
			endHash:     branch1Nodes[1].hash,
			maxResults:  10,
			hashes: append(nodeHashes(branch0Nodes, 14),
				nodeHashes(branch1Nodes, 0, 1)...),
		},
		{
			name:        "invalid start height",
			startHeight: 19,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			expectError: true,
		},
		{
			name:        "too many results",
			startHeight: 1,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			expectError: true,
		},
		{
			name:        "unvalidated block",
			startHeight: 15,
			endHash:     branch1Nodes[2].hash,
			maxResults:  10,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := blockDAG.HeightToHashRange(test.startHeight, &test.endHash,
			test.maxResults)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}

// TestIntervalBlockHashes ensures that fetching block hashes at specified
// intervals by end hash works as expected.
func TestIntervalBlockHashes(t *testing.T) {
	// Construct a synthetic block chain with a block index consisting of
	// the following structure.
	// 	genesis -> 1 -> 2 -> ... -> 15 -> 16  -> 17  -> 18
	// 	                              \-> 16a -> 17a -> 18a (unvalidated)
	tip := testTip
	chain := newTestDAG(&dagconfig.SimNetParams)
	branch0Nodes := chainedNodes(setFromSlice(chain.genesis), 18)
	branch1Nodes := chainedNodes(setFromSlice(branch0Nodes[14]), 3)
	for _, node := range branch0Nodes {
		chain.index.SetStatusFlags(node, statusValid)
		chain.index.AddNode(node)
	}
	for _, node := range branch1Nodes {
		if node.height < 18 {
			chain.index.SetStatusFlags(node, statusValid)
		}
		chain.index.AddNode(node)
	}
	chain.virtual.SetTips(setFromSlice(tip(branch0Nodes)))

	tests := []struct {
		name        string
		endHash     daghash.Hash
		interval    int
		hashes      []daghash.Hash
		expectError bool
	}{
		{
			name:     "blocks on main chain",
			endHash:  branch0Nodes[17].hash,
			interval: 8,
			hashes:   nodeHashes(branch0Nodes, 7, 15),
		},
		{
			name:     "blocks on stale chain",
			endHash:  branch1Nodes[1].hash,
			interval: 8,
			hashes: append(nodeHashes(branch0Nodes, 7),
				nodeHashes(branch1Nodes, 0)...),
		},
		{
			name:     "no results",
			endHash:  branch0Nodes[17].hash,
			interval: 20,
			hashes:   []daghash.Hash{},
		},
		{
			name:        "unvalidated block",
			endHash:     branch1Nodes[2].hash,
			interval:    8,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := chain.IntervalBlockHashes(&test.endHash, test.interval)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}

// TestPastUTXOErrors tests all error-cases in restoreUTXO.
// The non-error-cases are tested in the more general tests.
func TestPastUTXOErrors(t *testing.T) {
	targetErrorMessage := "dbFetchBlockByNode error"
	defer func() {
		if recover() == nil {
			t.Errorf("Got no panic on past UTXO error, while expected panic")
		}
	}()
	testErrorThroughPatching(
		t,
		targetErrorMessage,
		dbFetchBlockByNode,
		func(dbTx database.Tx, node *blockNode) (*util.Block, error) {
			return nil, errors.New(targetErrorMessage)
		},
	)
}

// TestRestoreUTXOErrors tests all error-cases in restoreUTXO.
// The non-error-cases are tested in the more general tests.
func TestRestoreUTXOErrors(t *testing.T) {
	targetErrorMessage := "WithDiff error"
	testErrorThroughPatching(
		t,
		targetErrorMessage,
		(*FullUTXOSet).WithDiff,
		func(fus *FullUTXOSet, other *UTXODiff) (UTXOSet, error) {
			return nil, errors.New(targetErrorMessage)
		},
	)
}

func testErrorThroughPatching(t *testing.T, expectedErrorMessage string, targetFunction interface{}, replacementFunction interface{}) {
	// Load up blocks such that there is a fork in the DAG.
	// (genesis block) -> 1 -> 2 -> 3 -> 4
	//                          \-> 3b
	testFiles := []string{
		"blk_0_to_4.dat",
		"blk_3B.dat",
	}

	var blocks []*util.Block
	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Fatalf("Error loading file: %v\n", err)
		}
		blocks = append(blocks, blockTmp...)
	}

	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := DAGSetup("testErrorThroughPatching", Config{
		DAGParams:    &dagconfig.SimNetParams,
		SubnetworkID: &wire.SubnetworkIDSupportsAll,
	})
	if err != nil {
		t.Fatalf("Failed to setup dag instance: %v", err)
	}
	defer teardownFunc()

	// Since we're not dealing with the real block chain, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	guard := monkey.Patch(targetFunction, replacementFunction)
	defer guard.Unpatch()

	err = nil
	for i := 1; i < len(blocks); i++ {
		var isOrphan bool
		isOrphan, err = dag.ProcessBlock(blocks[i], BFNone)
		if isOrphan {
			t.Fatalf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		t.Errorf("ProcessBlock unexpectedly succeeded. "+
			"Expected: %s", expectedErrorMessage)
	}
	if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("ProcessBlock returned wrong error. "+
			"Want: %s, got: %s", expectedErrorMessage, err)
	}
}

// TestFinality checks that the finality mechanism works as expected.
// This is how the flow goes:
// 1) We build a chain of finalityInterval blocks and call its tip altChainTip.
// 2) We build another chain (let's call it mainChain) of 2 * finalityInterval
// blocks, which points to genesis, and then we check that the block in that
// chain with height of finalityInterval is marked as finality point (This is
// very predictable, because the blue score of each new block in a chain is the
// parents plus one).
// 3) We make a new child to block with height (2 * finalityInterval - 1)
// in mainChain, and we check that connecting it to the DAG
// doesn't affect the last finality point.
// 4) We make a block that points to genesis, and check that it
// gets rejected because its blue score is lower then the last finality
// point.
// 5) We make a block that points to altChainTip, and check that it
// gets rejected because it doesn't have the last finality point in
// its selected parent chain.
func TestFinality(t *testing.T) {
	params := dagconfig.SimNetParams
	params.K = 1
	dag, teardownFunc, err := DAGSetup("TestFinality", Config{
		DAGParams:    &params,
		SubnetworkID: &wire.SubnetworkIDSupportsAll,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()
	blockTime := time.Unix(dag.genesis.timestamp, 0)
	extraNonce := int64(0)
	buildNodeToDag := func(parents blockSet) (*blockNode, error) {
		// We need to change the blockTime to keep all block hashes unique
		blockTime = blockTime.Add(time.Second)

		// We need to change the extraNonce to keep coinbase hashes unique
		extraNonce++

		bh := &wire.BlockHeader{
			Version:      1,
			Bits:         dag.genesis.bits,
			ParentHashes: parents.hashes(),
			Timestamp:    blockTime,
		}
		msgBlock := wire.NewMsgBlock(bh)
		blockHeight := parents.maxHeight() + 1
		coinbaseTx, err := createCoinbaseTxForTest(blockHeight, 1, extraNonce, dag.dagParams)
		if err != nil {
			return nil, err
		}
		msgBlock.AddTransaction(coinbaseTx)
		block := util.NewBlock(msgBlock)

		dag.dagLock.Lock()
		defer dag.dagLock.Unlock()

		err = dag.maybeAcceptBlock(block, BFNone)
		if err != nil {
			return nil, err
		}

		return dag.index.LookupNode(block.Hash()), nil
	}

	currentNode := dag.genesis

	// First we build a chain of finalityInterval blocks for future use
	for currentNode.blueScore < finalityInterval {
		currentNode, err = buildNodeToDag(setFromSlice(currentNode))
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	altChainTip := currentNode

	// Now we build a new chain of 2 * finalityInterval blocks, pointed to genesis, and
	// we expect the block with height 1 * finalityInterval to be the last finality point
	currentNode = dag.genesis
	for currentNode.blueScore < finalityInterval {
		currentNode, err = buildNodeToDag(setFromSlice(currentNode))
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	expectedFinalityPoint := currentNode

	for currentNode.blueScore < 2*finalityInterval {
		currentNode, err = buildNodeToDag(setFromSlice(currentNode))
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	if dag.lastFinalityPoint != expectedFinalityPoint {
		t.Errorf("TestFinality: dag.lastFinalityPoint expected to be %v but got %v", expectedFinalityPoint, dag.lastFinalityPoint)
	}

	// Here we check that even if we create a parallel tip (a new tip with
	// the same parents as the current one) with the same blue score as the
	// current tip, it still won't affect the last finality point.
	_, err = buildNodeToDag(setFromSlice(currentNode.selectedParent))
	if err != nil {
		t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
	}
	if dag.lastFinalityPoint != expectedFinalityPoint {
		t.Errorf("TestFinality: dag.lastFinalityPoint was unexpectly changed")
	}

	// Here we check that a block with lower blue score than the last finality
	// point will get rejected
	_, err = buildNodeToDag(setFromSlice(dag.genesis))
	if err == nil {
		t.Errorf("TestFinality: buildNodeToDag expected an error but got <nil>")
	}
	rErr, ok := err.(RuleError)
	if ok {
		if rErr.ErrorCode != ErrFinality {
			t.Errorf("TestFinality: buildNodeToDag expected an error with code %v but instead got %v", ErrFinality, rErr.ErrorCode)
		}
	} else {
		t.Errorf("TestFinality: buildNodeToDag got unexpected error: %v", rErr)
	}

	// Here we check that a block that doesn't have the last finality point in
	// its selected parent chain will get rejected
	_, err = buildNodeToDag(setFromSlice(altChainTip))
	if err == nil {
		t.Errorf("TestFinality: buildNodeToDag expected an error but got <nil>")
	}
	rErr, ok = err.(RuleError)
	if ok {
		if rErr.ErrorCode != ErrFinality {
			t.Errorf("TestFinality: buildNodeToDag expected an error with code %v but instead got %v", ErrFinality, rErr.ErrorCode)
		}
	} else {
		t.Errorf("TestFinality: buildNodeToDag got unexpected error: %v", rErr)
	}
}
