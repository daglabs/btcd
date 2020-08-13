// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/pkg/errors"
	"time"

	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
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
	Added mstime.Time

	// Fee is the total fee the transaction associated with the entry pays.
	Fee uint64

	// FeePerMegaGram is the fee the transaction pays in sompi per million gram.
	FeePerMegaGram uint64
}

// TxSource represents a source of transactions to consider for inclusion in
// new blocks.
//
// The interface contract requires that all of these methods are safe for
// concurrent access with respect to the source.
type TxSource interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() mstime.Time

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
	// Block is a block that is ready to be solved by miners. Thus, it is
	// completely valid with the exception of satisfying the proof-of-work
	// requirement.
	Block *domainmessage.MsgBlock

	// TxMasses contains the mass of each transaction in the generated
	// template performs.
	TxMasses []uint64

	// Fees contains the amount of fees each transaction in the generated
	// template pays in base units. Since the first transaction is the
	// coinbase, the first entry (offset 0) will contain the negative of the
	// sum of the fees of all other transactions.
	Fees []uint64

	// Height is the height at which the block template connects to the DAG
	Height uint64
}

// BlkTmplGenerator provides a type that can be used to generate block templates
// based on a given mining policy and source of transactions to choose from.
// It also houses additional state required in order to ensure the templates
// are built on top of the current DAG and adhere to the consensus rules.
type BlkTmplGenerator struct {
	policy   *Policy
	txSource TxSource
	dag      *blockdag.BlockDAG
	sigCache *txscript.SigCache
}

// NewBlkTmplGenerator returns a new block template generator for the given
// policy using transactions from the provided transaction source.
//
// The additional state-related fields are required in order to ensure the
// templates are built on top of the current DAG and adhere to the
// consensus rules.
func NewBlkTmplGenerator(policy *Policy,
	txSource TxSource, dag *blockdag.BlockDAG,
	sigCache *txscript.SigCache) *BlkTmplGenerator {

	return &BlkTmplGenerator{
		policy:   policy,
		txSource: txSource,
		dag:      dag,
		sigCache: sigCache,
	}
}

// NewBlockTemplate returns a new block template that is ready to be solved
// using the transactions from the passed transaction source pool and a coinbase
// that either pays to the passed address if it is not nil, or a coinbase that
// is redeemable by anyone if the passed address is nil. The nil address
// functionality is useful since there are cases such as the getblocktemplate
// RPC where external mining software is responsible for creating their own
// coinbase which will replace the one generated for the block template. Thus
// the need to have configured address can be avoided.
//
// The transactions selected and included are prioritized according to several
// factors. First, each transaction has a priority calculated based on its
// value, age of inputs, and size. Transactions which consist of larger
// amounts, older inputs, and small sizes have the highest priority. Second, a
// fee per kilobyte is calculated for each transaction. Transactions with a
// higher fee per kilobyte are preferred. Finally, the block generation related
// policy settings are all taken into account.
//
// Transactions which only spend outputs from other transactions already in the
// block DAG are immediately added to a priority queue which either
// prioritizes based on the priority (then fee per kilobyte) or the fee per
// kilobyte (then priority) depending on whether or not the BlockPrioritySize
// policy setting allots space for high-priority transactions. Transactions
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
func (g *BlkTmplGenerator) NewBlockTemplate(payToAddress util.Address, extraNonce uint64) (*BlockTemplate, error) {
	g.dag.Lock()
	defer g.dag.Unlock()

	txsForBlockTemplate, err := g.selectTxs(payToAddress, extraNonce)
	if err != nil {
		return nil, errors.Errorf("failed to select transactions: %s", err)
	}

	msgBlock, err := g.dag.BlockForMining(txsForBlockTemplate.selectedTxs)
	if err != nil {
		return nil, err
	}

	// Finally, perform a full check on the created block against the DAG
	// consensus rules to ensure it properly connects to the DAG with no
	// issues.
	block := util.NewBlock(msgBlock)

	if err := g.dag.CheckConnectBlockTemplateNoLock(block); err != nil {
		return nil, err
	}

	log.Debugf("Created new block template (%d transactions, %d in fees, "+
		"%d mass, target difficulty %064x)",
		len(msgBlock.Transactions), txsForBlockTemplate.totalFees,
		txsForBlockTemplate.totalMass, util.CompactToBig(msgBlock.Header.Bits))

	return &BlockTemplate{
		Block:    msgBlock,
		TxMasses: txsForBlockTemplate.txMasses,
		Fees:     txsForBlockTemplate.txFees,
	}, nil
}

// UpdateBlockTime updates the timestamp in the header of the passed block to
// the current time while taking into account the median time of the last
// several blocks to ensure the new time is after that time per the DAG
// consensus rules. Finally, it will update the target difficulty if needed
// based on the new time for the test networks since their target difficulty can
// change based upon time.
func (g *BlkTmplGenerator) UpdateBlockTime(msgBlock *domainmessage.MsgBlock) error {
	// The new timestamp is potentially adjusted to ensure it comes after
	// the median time of the last several blocks per the DAG consensus
	// rules.
	msgBlock.Header.Timestamp = g.dag.NextBlockTime()
	msgBlock.Header.Bits = g.dag.NextRequiredDifficulty(msgBlock.Header.Timestamp)

	return nil
}

// TxSource returns the associated transaction source.
//
// This function is safe for concurrent access.
func (g *BlkTmplGenerator) TxSource() TxSource {
	return g.txSource
}

// IsSynced checks if the node is synced enough based upon its worldview.
// This is used to determine if the node can support mining and requesting newly-mined blocks.
// To do that, first it checks if the selected tip timestamp is not older than maxTipAge. If that's the case, it means
// the node is synced since blocks' timestamps are not allowed to deviate too much into the future.
// If that's not the case it checks the rate it added new blocks to the DAG recently. If it's faster than
// blockRate * maxSyncRateDeviation it means the node is not synced, since when the node is synced it shouldn't add
// blocks to the DAG faster than the block rate.
func (g *BlkTmplGenerator) IsSynced() bool {
	const maxTipAge = 5 * time.Minute
	isCloseToCurrentTime := g.dag.Now().Sub(g.dag.SelectedTipHeader().Timestamp) <= maxTipAge
	if isCloseToCurrentTime {
		return true
	}

	const maxSyncRateDeviation = 1.05
	return g.dag.IsSyncRateBelowThreshold(maxSyncRateDeviation)
}
