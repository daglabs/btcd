// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpctest

import (
	"github.com/pkg/errors"
	"math"
	"math/big"
	"runtime"
	"time"

	"github.com/kaspanet/kaspad/blockdag"
	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/txscript"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/wire"
)

// solveBlock attempts to find a nonce which makes the passed block header hash
// to a value less than the target difficulty. When a successful solution is
// found true is returned and the nonce field of the passed header is updated
// with the solution. False is returned if no solution exists.
func solveBlock(header *wire.BlockHeader, targetDifficulty *big.Int) bool {
	// sbResult is used by the solver goroutines to send results.
	type sbResult struct {
		found bool
		nonce uint64
	}

	// solver accepts a block header and a nonce range to test. It is
	// intended to be run as a goroutine.
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint64) {
		// We need to modify the nonce field of the header, so make sure
		// we work with a copy of the original header.
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				return
			default:
				hdr.Nonce = i
				hash := hdr.BlockHash()
				if daghash.HashToBig(hash).Cmp(targetDifficulty) <= 0 {
					select {
					case results <- sbResult{true, i}:
						return
					case <-quit:
						return
					}
				}
			}
		}
		select {
		case results <- sbResult{false, 0}:
		case <-quit:
			return
		}
	}

	startNonce := uint64(0)
	stopNonce := uint64(math.MaxUint64)
	numCores := uint64(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint64(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	for i := uint64(0); i < numCores; i++ {
		result := <-results
		if result.found {
			close(quit)
			header.Nonce = result.nonce
			return true
		}
	}

	return false
}

// standardCoinbaseScript returns a standard script suitable for use as the
// signature script of the coinbase transaction of a new block. In particular,
// it starts with the block height that is required by version 2 blocks.
func standardCoinbaseScript(nextBlockHeight uint64, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate
// subsidy based on the passed block height to the provided address.
func createCoinbaseTx(coinbaseScript []byte, nextBlueScore uint64,
	addr util.Address, mineTo []wire.TxOut,
	net *dagconfig.Params) (*util.Tx, error) {

	// Create the script to pay to the provided payment address.
	scriptPubKey, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	txIns := []*wire.TxIn{&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutpoint: *wire.NewOutpoint(&daghash.TxID{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	}}
	txOuts := []*wire.TxOut{}
	if len(mineTo) == 0 {
		txOuts = append(txOuts, &wire.TxOut{
			Value:        blockdag.CalcBlockSubsidy(nextBlueScore, net),
			ScriptPubKey: scriptPubKey,
		})
	} else {
		for i := range mineTo {
			txOuts = append(txOuts, &mineTo[i])
		}
	}

	return util.NewTx(wire.NewNativeMsgTx(wire.TxVersion, txIns, txOuts)), nil
}

// CreateBlock creates a new block building from the previous block with a
// specified blockversion and timestamp. If the timestamp passed is zero (not
// initialized), then the timestamp of the previous block will be used plus 1
// second is used. Passing nil for the previous block results in a block that
// builds off of the genesis block for the specified chain.
func CreateBlock(parentBlock *util.Block, inclusionTxs []*util.Tx,
	blockVersion int32, blockTime time.Time, miningAddr util.Address,
	mineTo []wire.TxOut, net *dagconfig.Params, powMaxBits uint32) (*util.Block, error) {

	var (
		parentHash       *daghash.Hash
		blockChainHeight uint64
		parentBlockTime  time.Time
	)

	// If the parent block isn't specified, then we'll construct a block
	// that builds off of the genesis block for the chain.
	if parentBlock == nil {
		parentHash = net.GenesisHash
		blockChainHeight = 1
		parentBlockTime = net.GenesisBlock.Header.Timestamp.Add(time.Minute)
	} else {
		parentHash = parentBlock.Hash()
		blockChainHeight = parentBlock.ChainHeight() + 1
		parentBlockTime = parentBlock.MsgBlock().Header.Timestamp
	}

	// If a target block time was specified, then use that as the header's
	// timestamp. Otherwise, add one second to the parent block unless
	// it's the genesis block in which case use the current time.
	var ts time.Time
	switch {
	case !blockTime.IsZero():
		ts = blockTime
	default:
		ts = parentBlockTime.Add(time.Second)
	}

	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(blockChainHeight, extraNonce)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := createCoinbaseTx(coinbaseScript, blockChainHeight,
		miningAddr, mineTo, net)
	if err != nil {
		return nil, err
	}

	// Create a new block ready to be solved.
	blockTxns := []*util.Tx{coinbaseTx}
	if inclusionTxs != nil {
		blockTxns = append(blockTxns, inclusionTxs...)
	}
	hashMerkleTree := blockdag.BuildHashMerkleTreeStore(blockTxns)
	var block wire.MsgBlock
	block.Header = wire.BlockHeader{
		Version:              blockVersion,
		ParentHashes:         []*daghash.Hash{parentHash},
		HashMerkleRoot:       hashMerkleTree.Root(),
		AcceptedIDMerkleRoot: &daghash.ZeroHash,
		UTXOCommitment:       &daghash.ZeroHash,
		Timestamp:            ts,
		Bits:                 powMaxBits,
	}
	for _, tx := range blockTxns {
		block.AddTransaction(tx.MsgTx())
	}

	found := solveBlock(&block.Header, net.PowMax)
	if !found {
		return nil, errors.New("Unable to solve block")
	}

	utilBlock := util.NewBlock(&block)
	return utilBlock, nil
}
