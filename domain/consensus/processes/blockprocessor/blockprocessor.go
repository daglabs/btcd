package blockprocessor

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/infrastructure/logger"
	"time"
)

// blockProcessor is responsible for processing incoming blocks
// and creating blocks from the current state
type blockProcessor struct {
	genesisHash        *externalapi.DomainHash
	targetTimePerBlock time.Duration
	databaseContext    model.DBManager

	consensusStateManager model.ConsensusStateManager
	pruningManager        model.PruningManager
	blockValidator        model.BlockValidator
	dagTopologyManager    model.DAGTopologyManager
	reachabilityManager   model.ReachabilityManager
	difficultyManager     model.DifficultyManager
	ghostdagManager       model.GHOSTDAGManager
	pastMedianTimeManager model.PastMedianTimeManager
	coinbaseManager       model.CoinbaseManager
	headerTipsManager     model.HeadersSelectedTipManager
	syncManager           model.SyncManager

	acceptanceDataStore       model.AcceptanceDataStore
	blockStore                model.BlockStore
	blockStatusStore          model.BlockStatusStore
	blockRelationStore        model.BlockRelationStore
	multisetStore             model.MultisetStore
	ghostdagDataStore         model.GHOSTDAGDataStore
	consensusStateStore       model.ConsensusStateStore
	pruningStore              model.PruningStore
	reachabilityDataStore     model.ReachabilityDataStore
	utxoDiffStore             model.UTXODiffStore
	blockHeaderStore          model.BlockHeaderStore
	headersSelectedTipStore   model.HeaderSelectedTipStore
	finalityStore             model.FinalityStore
	headersSelectedChainStore model.HeadersSelectedChainStore

	stores []model.Store
}

// New instantiates a new BlockProcessor
func New(
	genesisHash *externalapi.DomainHash,
	targetTimePerBlock time.Duration,
	databaseContext model.DBManager,
	consensusStateManager model.ConsensusStateManager,
	pruningManager model.PruningManager,
	blockValidator model.BlockValidator,
	dagTopologyManager model.DAGTopologyManager,
	reachabilityManager model.ReachabilityManager,
	difficultyManager model.DifficultyManager,
	pastMedianTimeManager model.PastMedianTimeManager,
	ghostdagManager model.GHOSTDAGManager,
	coinbaseManager model.CoinbaseManager,
	headerTipsManager model.HeadersSelectedTipManager,
	syncManager model.SyncManager,

	acceptanceDataStore model.AcceptanceDataStore,
	blockStore model.BlockStore,
	blockStatusStore model.BlockStatusStore,
	blockRelationStore model.BlockRelationStore,
	multisetStore model.MultisetStore,
	ghostdagDataStore model.GHOSTDAGDataStore,
	consensusStateStore model.ConsensusStateStore,
	pruningStore model.PruningStore,
	reachabilityDataStore model.ReachabilityDataStore,
	utxoDiffStore model.UTXODiffStore,
	blockHeaderStore model.BlockHeaderStore,
	headersSelectedTipStore model.HeaderSelectedTipStore,
	finalityStore model.FinalityStore,
	headersSelectedChainStore model.HeadersSelectedChainStore,
) model.BlockProcessor {

	return &blockProcessor{
		genesisHash:           genesisHash,
		targetTimePerBlock:    targetTimePerBlock,
		databaseContext:       databaseContext,
		pruningManager:        pruningManager,
		blockValidator:        blockValidator,
		dagTopologyManager:    dagTopologyManager,
		reachabilityManager:   reachabilityManager,
		difficultyManager:     difficultyManager,
		pastMedianTimeManager: pastMedianTimeManager,
		ghostdagManager:       ghostdagManager,
		coinbaseManager:       coinbaseManager,
		headerTipsManager:     headerTipsManager,
		syncManager:           syncManager,

		consensusStateManager:     consensusStateManager,
		acceptanceDataStore:       acceptanceDataStore,
		blockStore:                blockStore,
		blockStatusStore:          blockStatusStore,
		blockRelationStore:        blockRelationStore,
		multisetStore:             multisetStore,
		ghostdagDataStore:         ghostdagDataStore,
		consensusStateStore:       consensusStateStore,
		pruningStore:              pruningStore,
		reachabilityDataStore:     reachabilityDataStore,
		utxoDiffStore:             utxoDiffStore,
		blockHeaderStore:          blockHeaderStore,
		headersSelectedTipStore:   headersSelectedTipStore,
		finalityStore:             finalityStore,
		headersSelectedChainStore: headersSelectedChainStore,

		stores: []model.Store{
			consensusStateStore,
			acceptanceDataStore,
			blockStore,
			blockStatusStore,
			blockRelationStore,
			multisetStore,
			ghostdagDataStore,
			consensusStateStore,
			pruningStore,
			reachabilityDataStore,
			utxoDiffStore,
			blockHeaderStore,
			headersSelectedTipStore,
			finalityStore,
			headersSelectedChainStore,
		},
	}
}

// ValidateAndInsertBlock validates the given block and, if valid, applies it
// to the current state
func (bp *blockProcessor) ValidateAndInsertBlock(block *externalapi.DomainBlock) (*externalapi.BlockInsertionResult, error) {
	onEnd := logger.LogAndMeasureExecutionTime(log, "ValidateAndInsertBlock")
	defer onEnd()

	return bp.validateAndInsertBlock(block, false)
}

func (bp *blockProcessor) ValidateAndInsertPruningPoint(newPruningPoint *externalapi.DomainBlock, serializedUTXOSet []byte) error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "ValidateAndInsertPruningPoint")
	defer onEnd()

	return bp.validateAndInsertPruningPoint(newPruningPoint, serializedUTXOSet)
}
