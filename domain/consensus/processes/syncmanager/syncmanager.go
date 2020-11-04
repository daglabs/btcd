package syncmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/infrastructure/logger"
)

type syncManager struct {
	databaseContext  model.DBReader
	genesisBlockHash *externalapi.DomainHash

	dagTraversalManager   model.DAGTraversalManager
	dagTopologyManager    model.DAGTopologyManager
	consensusStateManager model.ConsensusStateManager

	ghostdagDataStore model.GHOSTDAGDataStore
	blockStatusStore  model.BlockStatusStore
}

// New instantiates a new SyncManager
func New(
	databaseContext model.DBReader,
	genesisBlockHash *externalapi.DomainHash,
	dagTraversalManager model.DAGTraversalManager,
	dagTopologyManager model.DAGTopologyManager,
	consensusStateManager model.ConsensusStateManager,

	ghostdagDataStore model.GHOSTDAGDataStore,
	blockStatusStore model.BlockStatusStore) model.SyncManager {

	return &syncManager{
		databaseContext:  databaseContext,
		genesisBlockHash: genesisBlockHash,

		dagTraversalManager:   dagTraversalManager,
		dagTopologyManager:    dagTopologyManager,
		consensusStateManager: consensusStateManager,

		ghostdagDataStore: ghostdagDataStore,
		blockStatusStore:  blockStatusStore,
	}
}

func (sm *syncManager) GetHashesBetween(lowHash, highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "GetHashesBetween")
	defer onEnd()

	return sm.antiPastHashesBetween(lowHash, highHash)
}

func (sm *syncManager) GetMissingBlockBodyHashes(highHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "GetMissingBlockBodyHashes")
	defer onEnd()

	return sm.missingBlockBodyHashes(highHash)
}

func (sm *syncManager) IsBlockInHeaderPruningPointFutureAndVirtualPast(blockHash *externalapi.DomainHash) (bool, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "IsBlockInHeaderPruningPointFutureAndVirtualPast")
	defer onEnd()

	return sm.isBlockInHeaderPruningPointFutureAndVirtualPast(blockHash)
}

func (sm *syncManager) CreateBlockLocator(lowHash, highHash *externalapi.DomainHash) (externalapi.BlockLocator, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "CreateBlockLocator")
	defer onEnd()

	return sm.createBlockLocator(lowHash, highHash)
}

func (sm *syncManager) FindNextBlockLocatorBoundaries(blockLocator externalapi.BlockLocator) (lowHash, highHash *externalapi.DomainHash, err error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "FindNextBlockLocatorBoundaries")
	defer onEnd()

	return sm.findNextBlockLocatorBoundaries(blockLocator)
}

func (sm *syncManager) GetSyncInfo() (*externalapi.SyncInfo, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "GetSyncInfo")
	defer onEnd()

	panic("implement me")
}
