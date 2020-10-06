package consensus

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/processes"
	"github.com/kaspanet/kaspad/util"
)

// Consensus maintains the current core state of the node
type Consensus interface {
	BuildBlock(scriptPublicKey []byte, extraData []byte, transactionSelector model.TransactionSelector) *appmessage.MsgBlock
	ValidateAndInsertBlock(block *appmessage.MsgBlock) error

	UTXOByOutpoint(outpoint *appmessage.Outpoint) *model.UTXOEntry
	ValidateTransaction(transaction *util.Tx, utxoEntries []*model.UTXOEntry) error

	SetOnBlockAddedToDAGHandler(onBlockAddedToDAGHandler model.OnBlockAddedToDAGHandler)
	SetOnChainChangedHandler(onChainChangedHandler model.OnChainChangedHandler)
	SetOnFinalityConflictHandler(onFinalityConflictHandler model.OnFinalityConflictHandler)
	SetOnFinalityConflictResolvedHandler(onFinalityConflictResolvedHandler model.OnFinalityConflictResolvedHandler)
}

type consensus struct {
	blockProcessor        processes.BlockProcessor
	consensusStateManager processes.ConsensusStateManager
}

// BuildBlock builds a block over the current state, with the transactions
// selected by the given transactionSelector
func (s *consensus) BuildBlock(coinbaseScriptPublicKey []byte, coinbaseExtraData []byte,
	transactionSelector model.TransactionSelector) *appmessage.MsgBlock {

	return s.blockProcessor.BuildBlock(coinbaseScriptPublicKey, coinbaseExtraData, transactionSelector)
}

// ValidateAndInsertBlock validates the given block and, if valid, applies it
// to the current state
func (s *consensus) ValidateAndInsertBlock(block *appmessage.MsgBlock) error {
	return s.blockProcessor.ValidateAndInsertBlock(block)
}

// UTXOByOutpoint returns a UTXOEntry matching the given outpoint
func (s *consensus) UTXOByOutpoint(outpoint *appmessage.Outpoint) *model.UTXOEntry {
	return s.consensusStateManager.UTXOByOutpoint(outpoint)
}

// ValidateTransaction validates the given transaction using
// the given utxoEntries
func (s *consensus) ValidateTransaction(transaction *util.Tx, utxoEntries []*model.UTXOEntry) error {
	return s.consensusStateManager.ValidateTransaction(transaction, utxoEntries)
}

// SetOnBlockAddedToDAGHandler set the onBlockAddedToDAGHandler for the consensus
func (s *consensus) SetOnBlockAddedToDAGHandler(onBlockAddedToDAGHandler model.OnBlockAddedToDAGHandler) {
	s.blockProcessor.SetOnBlockAddedToDAGHandler(onBlockAddedToDAGHandler)
}

// SetOnChainChangedHandler set the onBlockAddedToDAGHandler for the consensus
func (s *consensus) SetOnChainChangedHandler(onChainChangedHandler model.OnChainChangedHandler) {
	s.blockProcessor.SetOnChainChangedHandler(onChainChangedHandler)
}

// SetOnFinalityConflictHandler set the onBlockAddedToDAGHandler for the consensus
func (s *consensus) SetOnFinalityConflictHandler(onFinalityConflictHandler model.OnFinalityConflictHandler) {
	s.blockProcessor.SetOnFinalityConflictHandler(onFinalityConflictHandler)
}

// SetOnFinalityConflictResolvedHandler set the onBlockAddedToDAGHandler for the consensus
func (s *consensus) SetOnFinalityConflictResolvedHandler(onFinalityConflictResolvedHandler model.OnFinalityConflictResolvedHandler) {
	s.consensusStateManager.SetOnFinalityConflictResolvedHandler(onFinalityConflictResolvedHandler)
}
