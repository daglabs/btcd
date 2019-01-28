// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"container/heap"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/daglabs/btcd/util/subnetworkid"

	"bou.ke/monkey"
	"github.com/daglabs/btcd/blockdag"
	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/txscript"
	"github.com/daglabs/btcd/wire"

	"github.com/daglabs/btcd/util"
)

// TestTxFeePrioHeap ensures the priority queue for transaction fees and
// priorities works as expected.
func TestTxFeePrioHeap(t *testing.T) {
	// Create some fake priority items that exercise the expected sort
	// edge conditions.
	testItems := []*txPrioItem{
		{feePerKB: 5678, priority: 3},
		{feePerKB: 5678, priority: 1},
		{feePerKB: 5678, priority: 1}, // Duplicate fee and prio
		{feePerKB: 5678, priority: 5},
		{feePerKB: 5678, priority: 2},
		{feePerKB: 1234, priority: 3},
		{feePerKB: 1234, priority: 1},
		{feePerKB: 1234, priority: 5},
		{feePerKB: 1234, priority: 5}, // Duplicate fee and prio
		{feePerKB: 1234, priority: 2},
		{feePerKB: 10000, priority: 0}, // Higher fee, smaller prio
		{feePerKB: 0, priority: 10000}, // Higher prio, lower fee
	}

	// Add random data in addition to the edge conditions already manually
	// specified.
	randSeed := rand.Int63()
	defer func() {
		if t.Failed() {
			t.Logf("Random numbers using seed: %v", randSeed)
		}
	}()
	prng := rand.New(rand.NewSource(randSeed))
	for i := 0; i < 1000; i++ {
		testItems = append(testItems, &txPrioItem{
			feePerKB: uint64(prng.Float64() * util.SatoshiPerBitcoin),
			priority: prng.Float64() * 100,
		})
	}

	// Test sorting by fee per KB then priority.
	var highest *txPrioItem
	priorityQueue := newTxPriorityQueue(len(testItems), true)
	for i := 0; i < len(testItems); i++ {
		prioItem := testItems[i]
		if highest == nil {
			highest = prioItem
		}
		if prioItem.feePerKB >= highest.feePerKB &&
			prioItem.priority > highest.priority {

			highest = prioItem
		}
		heap.Push(priorityQueue, prioItem)
	}

	for i := 0; i < len(testItems); i++ {
		prioItem := heap.Pop(priorityQueue).(*txPrioItem)
		if prioItem.feePerKB >= highest.feePerKB &&
			prioItem.priority > highest.priority {

			t.Fatalf("fee sort: item (fee per KB: %v, "+
				"priority: %v) higher than than prev "+
				"(fee per KB: %v, priority %v)",
				prioItem.feePerKB, prioItem.priority,
				highest.feePerKB, highest.priority)
		}
		highest = prioItem
	}

	// Test sorting by priority then fee per KB.
	highest = nil
	priorityQueue = newTxPriorityQueue(len(testItems), false)
	for i := 0; i < len(testItems); i++ {
		prioItem := testItems[i]
		if highest == nil {
			highest = prioItem
		}
		if prioItem.priority >= highest.priority &&
			prioItem.feePerKB > highest.feePerKB {

			highest = prioItem
		}
		heap.Push(priorityQueue, prioItem)
	}

	for i := 0; i < len(testItems); i++ {
		prioItem := heap.Pop(priorityQueue).(*txPrioItem)
		if prioItem.priority >= highest.priority &&
			prioItem.feePerKB > highest.feePerKB {

			t.Fatalf("priority sort: item (fee per KB: %v, "+
				"priority: %v) higher than than prev "+
				"(fee per KB: %v, priority %v)",
				prioItem.feePerKB, prioItem.priority,
				highest.feePerKB, highest.priority)
		}
		highest = prioItem
	}
}

type fakeTxSource struct {
	txDescs []*TxDesc
}

func (txs *fakeTxSource) LastUpdated() time.Time {
	return time.Unix(0, 0)
}

func (txs *fakeTxSource) MiningDescs() []*TxDesc {
	return txs.txDescs
}

func (txs *fakeTxSource) HaveTransaction(txID *daghash.TxID) bool {
	for _, desc := range txs.txDescs {
		if *desc.Tx.ID() == *txID {
			return true
		}
	}
	return false
}

func TestNewBlockTemplate(t *testing.T) {
	params := dagconfig.SimNetParams
	params.CoinbaseMaturity = 0

	pkScript, err := txscript.NewScriptBuilder().AddOp(txscript.OpTrue).Script()
	if err != nil {
		t.Fatalf("Failed to create pkScript: %v", err)
	}

	dag, teardownFunc, err := blockdag.DAGSetup("TestNewBlockTemplate", blockdag.Config{
		DAGParams: &params,
	})
	if err != nil {
		t.Fatalf("Failed to setup DAG instance: %v", err)
	}
	defer teardownFunc()
	policy := Policy{
		BlockMaxSize:      50000,
		BlockPrioritySize: 750000,
		TxMinFreeFee:      util.Amount(0),
	}

	txSource := &fakeTxSource{
		txDescs: []*TxDesc{},
	}

	var createCoinbaseTxPatch *monkey.PatchGuard
	createCoinbaseTxPatch = monkey.Patch(createCoinbaseTx, func(params *dagconfig.Params, coinbaseScript []byte, nextBlockHeight int32, addr util.Address) (*util.Tx, error) {
		createCoinbaseTxPatch.Unpatch()
		defer createCoinbaseTxPatch.Restore()
		tx, err := createCoinbaseTx(params, coinbaseScript, nextBlockHeight, addr)
		if err != nil {
			return nil, err
		}
		msgTx := tx.MsgTx()
		out := msgTx.TxOut[0]
		out.Value /= 10
		for i := 0; i < 9; i++ {
			msgTx.AddTxOut(&*out)
		}
		return tx, nil
	})
	defer createCoinbaseTxPatch.Unpatch()

	blockTemplateGenerator := NewBlkTmplGenerator(&policy,
		&params, txSource, dag, blockdag.NewMedianTime(), txscript.NewSigCache(100000))

	template1, err := blockTemplateGenerator.NewBlockTemplate(nil)
	createCoinbaseTxPatch.Unpatch()
	if err != nil {
		t.Fatalf("NewBlockTemplate: %v", err)
	}

	isOrphan, err := dag.ProcessBlock(util.NewBlock(template1.Block), blockdag.BFNoPoWCheck)
	if err != nil {
		t.Fatalf("ProcessBlock: %v", err)
	}

	if isOrphan {
		t.Fatalf("ProcessBlock: template1 got unexpectedly orphan")
	}

	cbScript, err := standardCoinbaseScript(dag.Height()+1, 0)
	if err != nil {
		t.Fatalf("standardCoinbaseScript: %v", err)
	}

	cbTx, err := createCoinbaseTx(&params, cbScript, dag.Height()+1, nil)
	if err != nil {
		t.Fatalf("createCoinbaseTx: %v", err)
	}

	template1CbTx := template1.Block.Transactions[0]

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			TxID:  template1CbTx.TxID(),
			Index: 0,
		},
		Sequence: wire.MaxTxInSequenceNum,
	})
	tx.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    1,
	})

	nonFinalizedTx := wire.NewMsgTx(wire.TxVersion)
	nonFinalizedTx.LockTime = uint64(dag.Height() + 2)
	nonFinalizedTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			TxID:  template1CbTx.TxID(),
			Index: 1,
		},
		Sequence: 0,
	})
	nonFinalizedTx.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    1,
	})

	existingSubnetwork := subnetworkid.SubnetworkID{0xff}
	nonExistingSubnetwork := subnetworkid.SubnetworkID{0xfe}

	nonExistingSubnetworkTx := wire.NewMsgTx(wire.TxVersion)
	nonExistingSubnetworkTx.SubnetworkID = nonExistingSubnetwork
	nonExistingSubnetworkTx.Gas = 1
	nonExistingSubnetworkTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			TxID:  template1CbTx.TxID(),
			Index: 2,
		},
		Sequence: 0,
	})
	nonExistingSubnetworkTx.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    1,
	})

	subnetworkTx1 := wire.NewMsgTx(wire.TxVersion)
	subnetworkTx1.SubnetworkID = existingSubnetwork
	subnetworkTx1.Gas = 1
	subnetworkTx1.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			TxID:  template1CbTx.TxID(),
			Index: 3,
		},
		Sequence: 0,
	})
	subnetworkTx1.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    1,
	})

	subnetworkTx2 := wire.NewMsgTx(wire.TxVersion)
	subnetworkTx2.SubnetworkID = existingSubnetwork
	subnetworkTx2.Gas = 100
	subnetworkTx2.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			TxID:  template1CbTx.TxID(),
			Index: 4,
		},
		Sequence: 0,
	})
	subnetworkTx2.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    1,
	})

	txSource.txDescs = []*TxDesc{
		{
			Tx: cbTx,
		},
		{
			Tx: util.NewTx(tx),
		},
		{
			Tx: util.NewTx(nonFinalizedTx),
		},
		{
			Tx: util.NewTx(subnetworkTx1),
		},
		{
			Tx: util.NewTx(subnetworkTx2),
		},
		{
			Tx: util.NewTx(nonExistingSubnetworkTx),
		},
	}

	standardCoinbaseScriptErrString := "standardCoinbaseScript err"

	var standardCoinbaseScriptPatch *monkey.PatchGuard
	standardCoinbaseScriptPatch = monkey.Patch(standardCoinbaseScript, func(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
		return nil, errors.New(standardCoinbaseScriptErrString)
	})
	defer standardCoinbaseScriptPatch.Unpatch()

	_, err = blockTemplateGenerator.NewBlockTemplate(nil)
	standardCoinbaseScriptPatch.Unpatch()

	if err == nil || err.Error() != standardCoinbaseScriptErrString {
		t.Errorf("expected an error \"%v\" but got \"%v\"", standardCoinbaseScriptErrString, err)
	}
	if err == nil {
		t.Errorf("expected an error but got <nil>")
	}

	popCalled := 0
	popReturnedUnexpectedValue := false
	expectedPops := map[daghash.TxID]bool{
		tx.TxID():                      false,
		subnetworkTx1.TxID():           false,
		subnetworkTx2.TxID():           false,
		nonExistingSubnetworkTx.TxID(): false,
	}
	var popPatch *monkey.PatchGuard
	popPatch = monkey.Patch((*txPriorityQueue).Pop, func(pq *txPriorityQueue) interface{} {
		popPatch.Unpatch()
		defer popPatch.Restore()

		item, ok := pq.Pop().(*txPrioItem)
		if _, expected := expectedPops[*item.tx.ID()]; expected && ok {
			expectedPops[*item.tx.ID()] = true
		} else {
			popReturnedUnexpectedValue = true
		}
		popCalled++
		return item
	})
	defer popPatch.Unpatch()

	gasLimitPatch := monkey.Patch((*blockdag.SubnetworkStore).GasLimit, func(_ *blockdag.SubnetworkStore, subnetworkID *subnetworkid.SubnetworkID) (uint64, error) {
		if *subnetworkID == nonExistingSubnetwork {
			return 0, errors.New("not found")
		}
		return 90, nil
	})
	defer gasLimitPatch.Unpatch()

	template2, err := blockTemplateGenerator.NewBlockTemplate(nil)
	popPatch.Unpatch()
	gasLimitPatch.Unpatch()

	if err != nil {
		t.Errorf("NewBlockTemplate: unexpected error: %v", err)
	}

	if popCalled == 0 {
		t.Errorf("(*txPriorityQueue).Pop wasn't called")
	}

	if popReturnedUnexpectedValue {
		t.Errorf("(*txPriorityQueue).Pop returned unexpected value")
	}

	for id, popped := range expectedPops {
		if !popped {
			t.Errorf("tx %v was expected to pop, but wasn't", id)
		}
	}

	expectedTxs := map[daghash.TxID]bool{
		tx.TxID():            false,
		subnetworkTx1.TxID(): false,
	}

	for _, tx := range template2.Block.Transactions[1:] {
		id := tx.TxID()
		if _, ok := expectedTxs[id]; !ok {
			t.Errorf("Unexpected tx %v in template2's candidate block", id)
		}
		expectedTxs[id] = true
	}

	for id, exists := range expectedTxs {
		if !exists {
			t.Errorf("tx %v was expected to be in template2's candidate block, but wasn't", id)
		}
	}
}
