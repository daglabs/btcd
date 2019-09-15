package blockdag_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/daglabs/btcd/util/subnetworkid"

	"github.com/daglabs/btcd/util/daghash"
	"github.com/daglabs/btcd/util/testtools"

	"github.com/daglabs/btcd/blockdag"
	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/mining"
	"github.com/daglabs/btcd/txscript"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/wire"
)

// TestFinality checks that the finality mechanism works as expected.
// This is how the flow goes:
// 1) We build a chain of blockdag.FinalityInterval blocks and call its tip altChainTip.
// 2) We build another chain (let's call it mainChain) of 2 * blockdag.FinalityInterval
// blocks, which points to genesis, and then we check that the block in that
// chain with height of blockdag.FinalityInterval is marked as finality point (This is
// very predictable, because the blue score of each new block in a chain is the
// parents plus one).
// 3) We make a new child to block with height (2 * blockdag.FinalityInterval - 1)
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
	dag, teardownFunc, err := blockdag.DAGSetup("TestFinality", blockdag.Config{
		DAGParams: &params,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()
	buildNodeToDag := func(parentHashes []*daghash.Hash) (*util.Block, error) {
		msgBlock, err := mining.PrepareBlockForTest(dag, &params, parentHashes, nil, false)
		if err != nil {
			return nil, err
		}
		block := util.NewBlock(msgBlock)

		isOrphan, delay, err := dag.ProcessBlock(block, blockdag.BFNoPoWCheck)
		if err != nil {
			return nil, err
		}
		if delay != 0 {
			return nil, fmt.Errorf("ProcessBlock: block " +
				"is too far in the future")
		}
		if isOrphan {
			return nil, fmt.Errorf("ProcessBlock: unexpected returned orphan block")
		}

		return block, nil
	}

	genesis := util.NewBlock(params.GenesisBlock)
	currentNode := genesis

	// First we build a chain of blockdag.FinalityInterval blocks for future use
	for i := 0; i < blockdag.FinalityInterval; i++ {
		currentNode, err = buildNodeToDag([]*daghash.Hash{currentNode.Hash()})
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	altChainTip := currentNode

	// Now we build a new chain of 2 * blockdag.FinalityInterval blocks, pointed to genesis, and
	// we expect the block with height 1 * blockdag.FinalityInterval to be the last finality point
	currentNode = genesis
	for i := 0; i < blockdag.FinalityInterval; i++ {
		currentNode, err = buildNodeToDag([]*daghash.Hash{currentNode.Hash()})
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	expectedFinalityPoint := currentNode

	for i := 0; i < blockdag.FinalityInterval; i++ {
		currentNode, err = buildNodeToDag([]*daghash.Hash{currentNode.Hash()})
		if err != nil {
			t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
		}
	}

	if !dag.LastFinalityPointHash().IsEqual(expectedFinalityPoint.Hash()) {
		t.Errorf("TestFinality: dag.lastFinalityPoint expected to be %v but got %v", expectedFinalityPoint, dag.LastFinalityPointHash())
	}

	// Here we check that even if we create a parallel tip (a new tip with
	// the same parents as the current one) with the same blue score as the
	// current tip, it still won't affect the last finality point.
	_, err = buildNodeToDag(currentNode.MsgBlock().Header.ParentHashes)
	if err != nil {
		t.Fatalf("TestFinality: buildNodeToDag unexpectedly returned an error: %v", err)
	}
	if !dag.LastFinalityPointHash().IsEqual(expectedFinalityPoint.Hash()) {
		t.Errorf("TestFinality: dag.lastFinalityPoint was unexpectly changed")
	}

	// Here we check that a block with lower blue score than the last finality
	// point will get rejected
	fakeCoinbaseTx, err := dag.NextBlockCoinbaseTransaction(nil, nil)
	if err != nil {
		t.Errorf("NextBlockCoinbaseTransaction: %s", err)
	}
	merkleRoot := blockdag.BuildHashMerkleTreeStore([]*util.Tx{fakeCoinbaseTx}).Root()
	beforeFinalityBlock := wire.NewMsgBlock(&wire.BlockHeader{
		Version:              0x10000000,
		ParentHashes:         []*daghash.Hash{genesis.Hash()},
		HashMerkleRoot:       merkleRoot,
		AcceptedIDMerkleRoot: &daghash.ZeroHash,
		UTXOCommitment:       &daghash.ZeroHash,
		Timestamp:            dag.SelectedTipHeader().Timestamp,
		Bits:                 genesis.MsgBlock().Header.Bits,
	})
	beforeFinalityBlock.AddTransaction(fakeCoinbaseTx.MsgTx())
	_, _, err = dag.ProcessBlock(util.NewBlock(beforeFinalityBlock), blockdag.BFNoPoWCheck)
	if err == nil {
		t.Errorf("TestFinality: buildNodeToDag expected an error but got <nil>")
	}
	rErr, ok := err.(blockdag.RuleError)
	if ok {
		if rErr.ErrorCode != blockdag.ErrFinality {
			t.Errorf("TestFinality: buildNodeToDag expected an error with code %v but instead got %v", blockdag.ErrFinality, rErr.ErrorCode)
		}
	} else {
		t.Errorf("TestFinality: buildNodeToDag got unexpected error: %v", err)
	}

	// Here we check that a block that doesn't have the last finality point in
	// its selected parent chain will get rejected
	_, err = buildNodeToDag([]*daghash.Hash{altChainTip.Hash()})
	if err == nil {
		t.Errorf("TestFinality: buildNodeToDag expected an error but got <nil>")
	}
	rErr, ok = err.(blockdag.RuleError)
	if ok {
		if rErr.ErrorCode != blockdag.ErrFinality {
			t.Errorf("TestFinality: buildNodeToDag expected an error with code %v but instead got %v", blockdag.ErrFinality, rErr.ErrorCode)
		}
	} else {
		t.Errorf("TestFinality: buildNodeToDag got unexpected error: %v", rErr)
	}
}

// TestFinalityInterval tests that the finality interval is
// smaller then wire.MaxInvPerMsg, so when a peer receives
// a getblocks message it should always be able to send
// all the necessary invs.
func TestFinalityInterval(t *testing.T) {
	if blockdag.FinalityInterval > wire.MaxInvPerMsg {
		t.Errorf("blockdag.FinalityInterval should be lower or equal to wire.MaxInvPerMsg")
	}
}

// TestSubnetworkRegistry tests the full subnetwork registry flow
func TestSubnetworkRegistry(t *testing.T) {
	params := dagconfig.SimNetParams
	params.K = 1
	params.BlockCoinbaseMaturity = 0
	dag, teardownFunc, err := blockdag.DAGSetup("TestSubnetworkRegistry", blockdag.Config{
		DAGParams: &params,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()

	gasLimit := uint64(12345)
	subnetworkID, err := testtools.RegisterSubnetworkForTest(dag, &params, gasLimit)
	if err != nil {
		t.Fatalf("could not register network: %s", err)
	}
	limit, err := dag.SubnetworkStore.GasLimit(subnetworkID)
	if err != nil {
		t.Fatalf("could not retrieve gas limit: %s", err)
	}
	if limit != gasLimit {
		t.Fatalf("unexpected gas limit. want: %d, got: %d", gasLimit, limit)
	}
}

func TestChainedTransactions(t *testing.T) {
	params := dagconfig.SimNetParams
	params.BlockCoinbaseMaturity = 0
	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := blockdag.DAGSetup("TestChainedTransactions", blockdag.Config{
		DAGParams: &params,
	})
	if err != nil {
		t.Fatalf("Failed to setup dag instance: %v", err)
	}
	defer teardownFunc()

	block1, err := mining.PrepareBlockForTest(dag, &params, []*daghash.Hash{params.GenesisHash}, nil, false)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}
	isOrphan, delay, err := dag.ProcessBlock(util.NewBlock(block1), blockdag.BFNoPoWCheck)
	if err != nil {
		t.Fatalf("ProcessBlock: %v", err)
	}
	if delay != 0 {
		t.Fatalf("ProcessBlock: block1 " +
			"is too far in the future")
	}
	if isOrphan {
		t.Fatalf("ProcessBlock: block1 got unexpectedly orphaned")
	}
	cbTx := block1.Transactions[0]

	signatureScript, err := txscript.PayToScriptHashSignatureScript(blockdag.OpTrueScript, nil)
	if err != nil {
		t.Fatalf("Failed to build signature script: %s", err)
	}
	txIn := &wire.TxIn{
		PreviousOutpoint: wire.Outpoint{TxID: *cbTx.TxID(), Index: 0},
		SignatureScript:  signatureScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	txOut := &wire.TxOut{
		ScriptPubKey: blockdag.OpTrueScript,
		Value:        uint64(1),
	}
	tx := wire.NewNativeMsgTx(wire.TxVersion, []*wire.TxIn{txIn}, []*wire.TxOut{txOut})

	chainedTxIn := &wire.TxIn{
		PreviousOutpoint: wire.Outpoint{TxID: *tx.TxID(), Index: 0},
		SignatureScript:  signatureScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}

	scriptPubKey, err := txscript.PayToScriptHashScript(blockdag.OpTrueScript)
	if err != nil {
		t.Fatalf("Failed to build public key script: %s", err)
	}
	chainedTxOut := &wire.TxOut{
		ScriptPubKey: scriptPubKey,
		Value:        uint64(1),
	}
	chainedTx := wire.NewNativeMsgTx(wire.TxVersion, []*wire.TxIn{chainedTxIn}, []*wire.TxOut{chainedTxOut})

	block2, err := mining.PrepareBlockForTest(dag, &params, []*daghash.Hash{block1.BlockHash()}, []*wire.MsgTx{tx, chainedTx}, true)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}

	//Checks that dag.ProcessBlock fails because we don't allow a transaction to spend another transaction from the same block
	isOrphan, delay, err = dag.ProcessBlock(util.NewBlock(block2), blockdag.BFNoPoWCheck)
	if err == nil {
		t.Errorf("ProcessBlock expected an error")
	} else if rErr, ok := err.(blockdag.RuleError); ok {
		if rErr.ErrorCode != blockdag.ErrMissingTxOut {
			t.Errorf("ProcessBlock expected an %v error code but got %v", blockdag.ErrMissingTxOut, rErr.ErrorCode)
		}
	} else {
		t.Errorf("ProcessBlock expected a blockdag.RuleError but got %v", err)
	}
	if delay != 0 {
		t.Fatalf("ProcessBlock: block2 " +
			"is too far in the future")
	}
	if isOrphan {
		t.Errorf("ProcessBlock: block2 got unexpectedly orphaned")
	}

	nonChainedTxIn := &wire.TxIn{
		PreviousOutpoint: wire.Outpoint{TxID: *cbTx.TxID(), Index: 0},
		SignatureScript:  signatureScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	nonChainedTxOut := &wire.TxOut{
		ScriptPubKey: scriptPubKey,
		Value:        uint64(1),
	}
	nonChainedTx := wire.NewNativeMsgTx(wire.TxVersion, []*wire.TxIn{nonChainedTxIn}, []*wire.TxOut{nonChainedTxOut})

	block3, err := mining.PrepareBlockForTest(dag, &params, []*daghash.Hash{block1.BlockHash()}, []*wire.MsgTx{nonChainedTx}, false)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}

	//Checks that dag.ProcessBlock doesn't fail because all of its transaction are dependant on transactions from previous blocks
	isOrphan, delay, err = dag.ProcessBlock(util.NewBlock(block3), blockdag.BFNoPoWCheck)
	if err != nil {
		t.Errorf("ProcessBlock: %v", err)
	}
	if delay != 0 {
		t.Fatalf("ProcessBlock: block3 " +
			"is too far in the future")
	}
	if isOrphan {
		t.Errorf("ProcessBlock: block3 got unexpectedly orphaned")
	}
}

// TestGasLimit tests the gas limit rules
func TestGasLimit(t *testing.T) {
	params := dagconfig.SimNetParams
	params.K = 1
	params.BlockCoinbaseMaturity = 0
	dag, teardownFunc, err := blockdag.DAGSetup("TestSubnetworkRegistry", blockdag.Config{
		DAGParams: &params,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()

	// First we prepare a subnetwork and a block with coinbase outputs to fund our tests
	gasLimit := uint64(12345)
	subnetworkID, err := testtools.RegisterSubnetworkForTest(dag, &params, gasLimit)
	if err != nil {
		t.Fatalf("could not register network: %s", err)
	}

	cbTxs := []*wire.MsgTx{}
	for i := 0; i < 4; i++ {
		fundsBlock, err := mining.PrepareBlockForTest(dag, &params, dag.TipHashes(), nil, false)
		if err != nil {
			t.Fatalf("PrepareBlockForTest: %v", err)
		}
		isOrphan, delay, err := dag.ProcessBlock(util.NewBlock(fundsBlock), blockdag.BFNoPoWCheck)
		if err != nil {
			t.Fatalf("ProcessBlock: %v", err)
		}
		if delay != 0 {
			t.Fatalf("ProcessBlock: the funds block " +
				"is too far in the future")
		}
		if isOrphan {
			t.Fatalf("ProcessBlock: fundsBlock got unexpectedly orphan")
		}

		cbTxs = append(cbTxs, fundsBlock.Transactions[util.CoinbaseTransactionIndex])
	}

	signatureScript, err := txscript.PayToScriptHashSignatureScript(blockdag.OpTrueScript, nil)
	if err != nil {
		t.Fatalf("Failed to build signature script: %s", err)
	}

	scriptPubKey, err := txscript.PayToScriptHashScript(blockdag.OpTrueScript)
	if err != nil {
		t.Fatalf("Failed to build public key script: %s", err)
	}

	tx1In := &wire.TxIn{
		PreviousOutpoint: *wire.NewOutpoint(cbTxs[0].TxID(), 0),
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  signatureScript,
	}
	tx1Out := &wire.TxOut{
		Value:        cbTxs[0].TxOut[0].Value,
		ScriptPubKey: scriptPubKey,
	}
	tx1 := wire.NewSubnetworkMsgTx(wire.TxVersion, []*wire.TxIn{tx1In}, []*wire.TxOut{tx1Out}, subnetworkID, 10000, []byte{})

	tx2In := &wire.TxIn{
		PreviousOutpoint: *wire.NewOutpoint(cbTxs[1].TxID(), 0),
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  signatureScript,
	}
	tx2Out := &wire.TxOut{
		Value:        cbTxs[1].TxOut[0].Value,
		ScriptPubKey: scriptPubKey,
	}
	tx2 := wire.NewSubnetworkMsgTx(wire.TxVersion, []*wire.TxIn{tx2In}, []*wire.TxOut{tx2Out}, subnetworkID, 10000, []byte{})

	// Here we check that we can't process a block that has transactions that exceed the gas limit
	overLimitBlock, err := mining.PrepareBlockForTest(dag, &params, dag.TipHashes(), []*wire.MsgTx{tx1, tx2}, true)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}
	isOrphan, delay, err := dag.ProcessBlock(util.NewBlock(overLimitBlock), blockdag.BFNoPoWCheck)
	if err == nil {
		t.Fatalf("ProcessBlock expected to have an error in block that exceeds gas limit")
	}
	rErr, ok := err.(blockdag.RuleError)
	if !ok {
		t.Fatalf("ProcessBlock expected a RuleError, but got %v", err)
	} else if rErr.ErrorCode != blockdag.ErrInvalidGas {
		t.Fatalf("ProcessBlock expected error code %s but got %s", blockdag.ErrInvalidGas, rErr.ErrorCode)
	}
	if delay != 0 {
		t.Fatalf("ProcessBlock: overLimitBlock " +
			"is too far in the future")
	}
	if isOrphan {
		t.Fatalf("ProcessBlock: overLimitBlock got unexpectedly orphan")
	}

	overflowGasTxIn := &wire.TxIn{
		PreviousOutpoint: *wire.NewOutpoint(cbTxs[2].TxID(), 0),
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  signatureScript,
	}
	overflowGasTxOut := &wire.TxOut{
		Value:        cbTxs[2].TxOut[0].Value,
		ScriptPubKey: scriptPubKey,
	}
	overflowGasTx := wire.NewSubnetworkMsgTx(wire.TxVersion, []*wire.TxIn{overflowGasTxIn}, []*wire.TxOut{overflowGasTxOut},
		subnetworkID, math.MaxUint64, []byte{})

	// Here we check that we can't process a block that its transactions' gas overflows uint64
	overflowGasBlock, err := mining.PrepareBlockForTest(dag, &params, dag.TipHashes(), []*wire.MsgTx{tx1, overflowGasTx}, true)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}
	isOrphan, delay, err = dag.ProcessBlock(util.NewBlock(overflowGasBlock), blockdag.BFNoPoWCheck)
	if err == nil {
		t.Fatalf("ProcessBlock expected to have an error")
	}
	rErr, ok = err.(blockdag.RuleError)
	if !ok {
		t.Fatalf("ProcessBlock expected a RuleError, but got %v", err)
	} else if rErr.ErrorCode != blockdag.ErrInvalidGas {
		t.Fatalf("ProcessBlock expected error code %s but got %s", blockdag.ErrInvalidGas, rErr.ErrorCode)
	}
	if isOrphan {
		t.Fatalf("ProcessBlock: overLimitBlock got unexpectedly orphan")
	}

	nonExistentSubnetwork := &subnetworkid.SubnetworkID{123}
	nonExistentSubnetworkTxIn := &wire.TxIn{
		PreviousOutpoint: *wire.NewOutpoint(cbTxs[3].TxID(), 0),
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  signatureScript,
	}
	nonExistentSubnetworkTxOut := &wire.TxOut{
		Value:        cbTxs[3].TxOut[0].Value,
		ScriptPubKey: scriptPubKey,
	}
	nonExistentSubnetworkTx := wire.NewSubnetworkMsgTx(wire.TxVersion, []*wire.TxIn{nonExistentSubnetworkTxIn},
		[]*wire.TxOut{nonExistentSubnetworkTxOut}, nonExistentSubnetwork, 1, []byte{})

	nonExistentSubnetworkBlock, err := mining.PrepareBlockForTest(dag, &params, dag.TipHashes(), []*wire.MsgTx{nonExistentSubnetworkTx, overflowGasTx}, true)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}

	// Here we check that we can't process a block with a transaction from a non-existent subnetwork
	isOrphan, delay, err = dag.ProcessBlock(util.NewBlock(nonExistentSubnetworkBlock), blockdag.BFNoPoWCheck)
	expectedErrStr := fmt.Sprintf("Error getting gas limit for subnetworkID '%s': subnetwork '%s' not found",
		nonExistentSubnetwork, nonExistentSubnetwork)
	if err.Error() != expectedErrStr {
		t.Fatalf("ProcessBlock expected error \"%v\" but got \"%v\"", expectedErrStr, err)
	}

	// Here we check that we can process a block with a transaction that doesn't exceed the gas limit
	validBlock, err := mining.PrepareBlockForTest(dag, &params, dag.TipHashes(), []*wire.MsgTx{tx1}, true)
	if err != nil {
		t.Fatalf("PrepareBlockForTest: %v", err)
	}
	isOrphan, delay, err = dag.ProcessBlock(util.NewBlock(validBlock), blockdag.BFNoPoWCheck)
	if err != nil {
		t.Fatalf("ProcessBlock: %v", err)
	}
	if delay != 0 {
		t.Fatalf("ProcessBlock: overLimitBlock " +
			"is too far in the future")
	}
	if isOrphan {
		t.Fatalf("ProcessBlock: overLimitBlock got unexpectedly orphan")
	}
}
