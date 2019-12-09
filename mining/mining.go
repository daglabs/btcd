// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"sort"
	"time"

	"github.com/kaspanet/kaspad/blockdag"
	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/txscript"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
)

const (
	// CoinbaseFlags is added to the coinbase script of a generated block
	// and is used to monitor BIP16 support as well as blocks that are
	// generated via kaspad.
	CoinbaseFlags = "/kaspad/"
)

// TxDesc is a descriptor about a transaction in a transaction source along with
// additional metadata.
type TxDesc struct {
	// Tx is the transaction associated with the entry.
	Tx *util.Tx

	// Added is the time when the entry was added to the source pool.
	Added time.Time

	// Height is the block height when the entry was added to the the source
	// pool.
	Height uint64

	// Fee is the total fee the transaction associated with the entry pays.
	Fee uint64

	// FeePerKB is the fee the transaction pays in Satoshi per 1000 bytes.
	FeePerKB uint64
}

// TxSource represents a source of transactions to consider for inclusion in
// new blocks.
//
// The interface contract requires that all of these methods are safe for
// concurrent access with respect to the source.
type TxSource interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() time.Time

	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	MiningDescs() []*TxDesc

	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(txID *daghash.TxID) bool
}

// BlockTemplate houses a block that has yet to be solved along with additional
// details about the fees and the number of signature operations for each
// transaction in the block.
type BlockTemplate struct {
	// Block is a block that is ready to be solved by miners.  Thus, it is
	// completely valid with the exception of satisfying the proof-of-work
	// requirement.
	Block *wire.MsgBlock

	// TxMasses contains the mass of each transaction in the generated
	// template performs.
	TxMasses []uint64

	// Fees contains the amount of fees each transaction in the generated
	// template pays in base units.  Since the first transaction is the
	// coinbase, the first entry (offset 0) will contain the negative of the
	// sum of the fees of all other transactions.
	Fees []uint64

	// Height is the height at which the block template connects to the DAG
	Height uint64

	// ValidPayAddress indicates whether or not the template coinbase pays
	// to an address or is redeemable by anyone.  See the documentation on
	// NewBlockTemplate for details on which this can be useful to generate
	// templates without a coinbase payment address.
	ValidPayAddress bool
}

// MinimumMedianTime returns the minimum allowed timestamp for a block building
// on the end of the DAG.  In particular, it is one second after
// the median timestamp of the last several blocks per the DAG consensus
// rules.
func MinimumMedianTime(dagMedianTime time.Time) time.Time {
	return dagMedianTime.Add(time.Second)
}

// medianAdjustedTime returns the current time adjusted to ensure it is at least
// one second after the median timestamp of the last several blocks per the
// chain consensus rules.
func medianAdjustedTime(dagMedianTime time.Time, timeSource blockdag.MedianTimeSource) time.Time {
	// The timestamp for the block must not be before the median timestamp
	// of the last several blocks.  Thus, choose the maximum between the
	// current time and one second after the past median time.  The current
	// timestamp is truncated to a second boundary before comparison since a
	// block timestamp does not supported a precision greater than one
	// second.
	newTimestamp := timeSource.AdjustedTime()
	minTimestamp := MinimumMedianTime(dagMedianTime)
	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}

// BlkTmplGenerator provides a type that can be used to generate block templates
// based on a given mining policy and source of transactions to choose from.
// It also houses additional state required in order to ensure the templates
// are built on top of the current best chain and adhere to the consensus rules.
type BlkTmplGenerator struct {
	policy     *Policy
	dagParams  *dagconfig.Params
	txSource   TxSource
	dag        *blockdag.BlockDAG
	timeSource blockdag.MedianTimeSource
	sigCache   *txscript.SigCache
}

// NewBlkTmplGenerator returns a new block template generator for the given
// policy using transactions from the provided transaction source.
//
// The additional state-related fields are required in order to ensure the
// templates are built on top of the current best chain and adhere to the
// consensus rules.
func NewBlkTmplGenerator(policy *Policy, params *dagconfig.Params,
	txSource TxSource, dag *blockdag.BlockDAG,
	timeSource blockdag.MedianTimeSource,
	sigCache *txscript.SigCache) *BlkTmplGenerator {

	return &BlkTmplGenerator{
		policy:     policy,
		dagParams:  params,
		txSource:   txSource,
		dag:        dag,
		timeSource: timeSource,
		sigCache:   sigCache,
	}
}

// NewBlockTemplate returns a new block template that is ready to be solved
// using the transactions from the passed transaction source pool and a coinbase
// that either pays to the passed address if it is not nil, or a coinbase that
// is redeemable by anyone if the passed address is nil.  The nil address
// functionality is useful since there are cases such as the getblocktemplate
// RPC where external mining software is responsible for creating their own
// coinbase which will replace the one generated for the block template.  Thus
// the need to have configured address can be avoided.
//
// The transactions selected and included are prioritized according to several
// factors.  First, each transaction has a priority calculated based on its
// value, age of inputs, and size.  Transactions which consist of larger
// amounts, older inputs, and small sizes have the highest priority.  Second, a
// fee per kilobyte is calculated for each transaction.  Transactions with a
// higher fee per kilobyte are preferred.  Finally, the block generation related
// policy settings are all taken into account.
//
// Transactions which only spend outputs from other transactions already in the
// block chain are immediately added to a priority queue which either
// prioritizes based on the priority (then fee per kilobyte) or the fee per
// kilobyte (then priority) depending on whether or not the BlockPrioritySize
// policy setting allots space for high-priority transactions.  Transactions
// which spend outputs from other transactions in the source pool are added to a
// dependency map so they can be added to the priority queue once the
// transactions they depend on have been included.
//
// Once the high-priority area (if configured) has been filled with
// transactions, or the priority falls below what is considered high-priority,
// the priority queue is updated to prioritize by fees per kilobyte (then
// priority).
//
// When the fees per kilobyte drop below the TxMinFreeFee policy setting, the
// transaction will be skipped unless the BlockMinSize policy setting is
// nonzero, in which case the block will be filled with the low-fee/free
// transactions until the block size reaches that minimum size.
//
// Any transactions which would cause the block to exceed the BlockMaxMass
// policy setting, exceed the maximum allowed signature operations per block, or
// otherwise cause the block to be invalid are skipped.
//
// Given the above, a block generated by this function is of the following form:
//
//   -----------------------------------  --  --
//  |      Coinbase Transaction         |   |   |
//  |-----------------------------------|   |   |
//  |                                   |   |   | ----- policy.BlockPrioritySize
//  |   High-priority Transactions      |   |   |
//  |                                   |   |   |
//  |-----------------------------------|   | --
//  |                                   |   |
//  |                                   |   |
//  |                                   |   |--- policy.BlockMaxMass
//  |  Transactions prioritized by fee  |   |
//  |  until <= policy.TxMinFreeFee     |   |
//  |                                   |   |
//  |                                   |   |
//  |                                   |   |
//  |-----------------------------------|   |
//  |  Low-fee/Non high-priority (free) |   |
//  |  transactions (while block size   |   |
//  |  <= policy.BlockMinSize)          |   |
//   -----------------------------------  --
func (g *BlkTmplGenerator) NewBlockTemplate(payToAddress util.Address) (*BlockTemplate, error) {
	g.dag.Lock()
	defer g.dag.Unlock()

	txsForBlockTemplate, err := g.selectTxs(payToAddress)
	if err != nil {
		return nil, errors.Errorf("failed to select txs: %s", err)
	}

	// Calculate the required difficulty for the block.  The timestamp
	// is potentially adjusted to ensure it comes after the median time of
	// the last several blocks per the chain consensus rules.
	ts := medianAdjustedTime(g.dag.CalcPastMedianTime(), g.timeSource)
	requiredDifficulty := g.dag.NextRequiredDifficulty(ts)

	// Calculate the next expected block version based on the state of the
	// rule change deployments.
	nextBlockVersion, err := g.dag.CalcNextBlockVersion()
	if err != nil {
		return nil, err
	}

	// Sort transactions by subnetwork ID before building Merkle tree
	selectedTxs := txsForBlockTemplate.selectedTxs
	sort.Slice(selectedTxs, func(i, j int) bool {
		if selectedTxs[i].MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDCoinbase) {
			return true
		}
		if selectedTxs[j].MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDCoinbase) {
			return false
		}
		return subnetworkid.Less(&selectedTxs[i].MsgTx().SubnetworkID, &selectedTxs[j].MsgTx().SubnetworkID)
	})

	// Create a new block ready to be solved.
	hashMerkleTree := blockdag.BuildHashMerkleTreeStore(selectedTxs)
	acceptedIDMerkleRoot, err := g.dag.NextAcceptedIDMerkleRootNoLock()
	if err != nil {
		return nil, err
	}
	var msgBlock wire.MsgBlock
	for _, tx := range selectedTxs {
		msgBlock.AddTransaction(tx.MsgTx())
	}
	utxoCommitment, err := g.buildUTXOCommitment(msgBlock.Transactions)
	if err != nil {
		return nil, err
	}
	msgBlock.Header = wire.BlockHeader{
		Version:              nextBlockVersion,
		ParentHashes:         g.dag.TipHashes(),
		HashMerkleRoot:       hashMerkleTree.Root(),
		AcceptedIDMerkleRoot: acceptedIDMerkleRoot,
		UTXOCommitment:       utxoCommitment,
		Timestamp:            ts,
		Bits:                 requiredDifficulty,
	}

	// Finally, perform a full check on the created block against the DAG
	// consensus rules to ensure it properly connects to the DAG with no
	// issues.
	block := util.NewBlock(&msgBlock)

	if err := g.dag.CheckConnectBlockTemplateNoLock(block); err != nil {
		return nil, err
	}

	log.Debugf("Created new block template (%d transactions, %d in fees, "+
		"%d mass, target difficulty %064x)",
		len(msgBlock.Transactions), txsForBlockTemplate.totalFees,
		txsForBlockTemplate.totalMass, util.CompactToBig(msgBlock.Header.Bits))

	return &BlockTemplate{
		Block:           &msgBlock,
		TxMasses:        txsForBlockTemplate.txMasses,
		Fees:            txsForBlockTemplate.txFees,
		ValidPayAddress: payToAddress != nil,
	}, nil
}

// CoinbasePayloadExtraData returns coinbase payload extra data parameter
// which is built from extra nonce and coinbase flags.
func CoinbasePayloadExtraData(extraNonce uint64) ([]byte, error) {
	extraNonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(extraNonceBytes, extraNonce)
	w := &bytes.Buffer{}
	_, err := w.Write(extraNonceBytes)
	if err != nil {
		return nil, err
	}
	_, err = w.Write([]byte(CoinbaseFlags))
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (g *BlkTmplGenerator) buildUTXOCommitment(transactions []*wire.MsgTx) (*daghash.Hash, error) {
	utxoWithTransactions, err := g.dag.UTXOSet().WithTransactions(transactions, blockdag.UnacceptedBlueScore, false)
	if err != nil {
		return nil, err
	}

	return utxoWithTransactions.Multiset().Hash(), nil
}

// UpdateBlockTime updates the timestamp in the header of the passed block to
// the current time while taking into account the median time of the last
// several blocks to ensure the new time is after that time per the chain
// consensus rules.  Finally, it will update the target difficulty if needed
// based on the new time for the test networks since their target difficulty can
// change based upon time.
func (g *BlkTmplGenerator) UpdateBlockTime(msgBlock *wire.MsgBlock) error {
	// The new timestamp is potentially adjusted to ensure it comes after
	// the median time of the last several blocks per the chain consensus
	// rules.
	dagMedianTime := g.dag.CalcPastMedianTime()
	newTime := medianAdjustedTime(dagMedianTime, g.timeSource)
	msgBlock.Header.Timestamp = newTime

	return nil
}

// UpdateExtraNonce updates the extra nonce in the coinbase script of the passed
// block by regenerating the coinbase script with the passed value and block
// height.  It also recalculates and updates the new merkle root that results
// from changing the coinbase script.
func (g *BlkTmplGenerator) UpdateExtraNonce(msgBlock *wire.MsgBlock, extraNonce uint64) error {
	coinbasePayloadScriptPubKey, _, err := blockdag.DeserializeCoinbasePayload(msgBlock.Transactions[util.CoinbaseTransactionIndex])
	if err != nil {
		return err
	}

	coinbasePayloadExtraData, err := CoinbasePayloadExtraData(extraNonce)
	if err != nil {
		return err
	}
	coinbasePayload, err := blockdag.SerializeCoinbasePayload(coinbasePayloadScriptPubKey, coinbasePayloadExtraData)
	if err != nil {
		return err
	}
	if len(coinbasePayload) > blockdag.MaxCoinbasePayloadLen {
		return errors.Errorf("coinbase transaction script length "+
			"of %d is out of range (max: %d)",
			len(coinbasePayload),
			blockdag.MaxCoinbasePayloadLen)
	}
	oldCoinbaseTx := msgBlock.Transactions[util.CoinbaseTransactionIndex]
	msgBlock.Transactions[util.CoinbaseTransactionIndex] = wire.NewSubnetworkMsgTx(oldCoinbaseTx.Version, oldCoinbaseTx.TxIn, oldCoinbaseTx.TxOut, &oldCoinbaseTx.SubnetworkID, oldCoinbaseTx.Gas, coinbasePayload)

	// TODO(davec): A util.Block should use saved in the state to avoid
	// recalculating all of the other transaction hashes.
	// block.Transactions[util.CoinbaseTransactionIndex].InvalidateCache()

	// Recalculate the merkle roots with the updated extra nonce.
	block := util.NewBlock(msgBlock)
	hashMerkleTree := blockdag.BuildHashMerkleTreeStore(block.Transactions())
	msgBlock.Header.HashMerkleRoot = hashMerkleTree.Root()

	utxoCommitment, err := g.buildUTXOCommitment(msgBlock.Transactions)
	if err != nil {
		return err
	}

	msgBlock.Header.UTXOCommitment = utxoCommitment

	return nil
}

// VirtualBlueScore returns the virtual block's current blue score
func (g *BlkTmplGenerator) VirtualBlueScore() uint64 {
	return g.dag.VirtualBlueScore()
}

// TipHashes returns the hashes of the DAG's tips
func (g *BlkTmplGenerator) TipHashes() []*daghash.Hash {
	return g.dag.TipHashes()
}

// TxSource returns the associated transaction source.
//
// This function is safe for concurrent access.
func (g *BlkTmplGenerator) TxSource() TxSource {
	return g.txSource
}
