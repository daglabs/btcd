package consensus

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

// Consensus maintains the current core state of the node
type Consensus interface {
	BuildBlock(coinbaseData *externalapi.DomainCoinbaseData, transactions []*externalapi.DomainTransaction) (*externalapi.DomainBlock, error)
	ValidateAndInsertBlock(block *externalapi.DomainBlock) error
	ValidateTransactionAndPopulateWithConsensusData(transaction *externalapi.DomainTransaction) error

	GetBlock(blockHash *externalapi.DomainHash) (*externalapi.DomainBlock, error)
	GetBlockHeader(blockHash *externalapi.DomainHash) (*externalapi.DomainBlockHeader, error)
	GetBlockInfo(blockHash *externalapi.DomainHash) (*externalapi.BlockInfo, error)

	GetHashesBetween(lowHash, highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error)
	GetMissingBlockBodyHashes(highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error)
	GetPruningPointUTXOSet() ([]byte, error)
	SetPruningPointUTXOSet(serializedUTXOSet []byte) error
	GetVirtualSelectedParent() (*externalapi.DomainBlock, error)
	CreateBlockLocator(lowHash, highHash *externalapi.DomainHash) (externalapi.BlockLocator, error)
	FindNextBlockLocatorBoundaries(blockLocator externalapi.BlockLocator) (lowHash, highHash *externalapi.DomainHash, err error)
	GetSyncInfo() (*externalapi.SyncInfo, error)
}

type consensus struct {
	databaseContext model.DBReader

	blockProcessor        model.BlockProcessor
	consensusStateManager model.ConsensusStateManager
	transactionValidator  model.TransactionValidator
	syncManager           model.SyncManager
	pastMedianTimeManager model.PastMedianTimeManager

	blockStore        model.BlockStore
	blockHeaderStore  model.BlockHeaderStore
	pruningStore      model.PruningStore
	ghostdagDataStore model.GHOSTDAGDataStore
	blockStatusStore  model.BlockStatusStore
}

// BuildBlock builds a block over the current state, with the transactions
// selected by the given transactionSelector
func (s *consensus) BuildBlock(coinbaseData *externalapi.DomainCoinbaseData,
	transactions []*externalapi.DomainTransaction) (*externalapi.DomainBlock, error) {

	return s.blockProcessor.BuildBlock(coinbaseData, transactions)
}

// ValidateAndInsertBlock validates the given block and, if valid, applies it
// to the current state
func (s *consensus) ValidateAndInsertBlock(block *externalapi.DomainBlock) error {
	return s.blockProcessor.ValidateAndInsertBlock(block)
}

// ValidateTransactionAndPopulateWithConsensusData validates the given transaction
// and populates it with any missing consensus data
func (s *consensus) ValidateTransactionAndPopulateWithConsensusData(transaction *externalapi.DomainTransaction) error {
	err := s.transactionValidator.ValidateTransactionInIsolation(transaction)
	if err != nil {
		return err
	}

	err = s.consensusStateManager.PopulateTransactionWithUTXOEntries(transaction)
	if err != nil {
		return err
	}

	virtualGHOSTDAGData, err := s.ghostdagDataStore.Get(s.databaseContext, model.VirtualBlockHash)
	if err != nil {
		return err
	}
	virtualSelectedParentMedianTime, err := s.pastMedianTimeManager.PastMedianTime(virtualGHOSTDAGData.SelectedParent)
	if err != nil {
		return err
	}

	return s.transactionValidator.ValidateTransactionInContextAndPopulateMassAndFee(transaction,
		model.VirtualBlockHash, virtualSelectedParentMedianTime)
}

func (s *consensus) GetBlock(blockHash *externalapi.DomainHash) (*externalapi.DomainBlock, error) {
	return s.blockStore.Block(s.databaseContext, blockHash)
}

func (s *consensus) GetBlockHeader(blockHash *externalapi.DomainHash) (*externalapi.DomainBlockHeader, error) {
	return s.blockHeaderStore.BlockHeader(s.databaseContext, blockHash)
}

func (s *consensus) GetBlockInfo(blockHash *externalapi.DomainHash) (*externalapi.BlockInfo, error) {
	blockInfo := &externalapi.BlockInfo{}

	exists, err := s.blockStatusStore.Exists(s.databaseContext, blockHash)
	if err != nil {
		return nil, err
	}
	blockInfo.Exists = exists
	if !exists {
		return blockInfo, nil
	}

	blockStatus, err := s.blockStatusStore.Get(s.databaseContext, blockHash)
	if err != nil {
		return nil, err
	}
	blockInfo.BlockStatus = &blockStatus

	isBlockInHeaderPruningPointFuture, err := s.syncManager.IsBlockInHeaderPruningPointFuture(blockHash)
	if err != nil {
		return nil, err
	}
	blockInfo.IsBlockInHeaderPruningPointFuture = isBlockInHeaderPruningPointFuture

	return blockInfo, nil
}

func (s *consensus) GetHashesBetween(lowHash, highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	return s.syncManager.GetHashesBetween(lowHash, highHash)
}

func (s *consensus) GetMissingBlockBodyHashes(highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	return s.syncManager.GetMissingBlockBodyHashes(highHash)
}

func (s *consensus) GetPruningPointUTXOSet() ([]byte, error) {
	return s.pruningStore.PruningPointSerializedUTXOSet(s.databaseContext)
}

func (s *consensus) SetPruningPointUTXOSet(serializedUTXOSet []byte) error {
	return s.consensusStateManager.SetPruningPointUTXOSet(serializedUTXOSet)
}

func (s *consensus) GetVirtualSelectedParent() (*externalapi.DomainBlock, error) {
	virtualGHOSTDAGData, err := s.ghostdagDataStore.Get(s.databaseContext, model.VirtualBlockHash)
	if err != nil {
		return nil, err
	}
	return s.GetBlock(virtualGHOSTDAGData.SelectedParent)
}

func (s *consensus) CreateBlockLocator(lowHash, highHash *externalapi.DomainHash) (externalapi.BlockLocator, error) {
	return s.syncManager.CreateBlockLocator(lowHash, highHash)
}

func (s *consensus) FindNextBlockLocatorBoundaries(blockLocator externalapi.BlockLocator) (lowHash, highHash *externalapi.DomainHash, err error) {
	return s.syncManager.FindNextBlockLocatorBoundaries(blockLocator)
}

func (s *consensus) GetSyncInfo() (*externalapi.SyncInfo, error) {
	return s.syncManager.GetSyncInfo()
}
