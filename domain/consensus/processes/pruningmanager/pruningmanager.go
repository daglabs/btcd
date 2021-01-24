package pruningmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/multiset"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"github.com/kaspanet/kaspad/infrastructure/db/database"
	"github.com/kaspanet/kaspad/infrastructure/logger"
	"github.com/pkg/errors"
)

// pruningManager resolves and manages the current pruning point
type pruningManager struct {
	databaseContext model.DBManager

	dagTraversalManager    model.DAGTraversalManager
	dagTopologyManager     model.DAGTopologyManager
	consensusStateManager  model.ConsensusStateManager
	consensusStateStore    model.ConsensusStateStore
	ghostdagDataStore      model.GHOSTDAGDataStore
	pruningStore           model.PruningStore
	blockStatusStore       model.BlockStatusStore
	headerSelectedTipStore model.HeaderSelectedTipStore

	multiSetStore       model.MultisetStore
	acceptanceDataStore model.AcceptanceDataStore
	blocksStore         model.BlockStore
	blockHeaderStore    model.BlockHeaderStore
	utxoDiffStore       model.UTXODiffStore

	isArchivalNode   bool
	genesisHash      *externalapi.DomainHash
	finalityInterval uint64
	pruningDepth     uint64
}

// New instantiates a new PruningManager
func New(
	databaseContext model.DBManager,

	dagTraversalManager model.DAGTraversalManager,
	dagTopologyManager model.DAGTopologyManager,
	consensusStateManager model.ConsensusStateManager,
	consensusStateStore model.ConsensusStateStore,
	ghostdagDataStore model.GHOSTDAGDataStore,
	pruningStore model.PruningStore,
	blockStatusStore model.BlockStatusStore,
	headerSelectedTipStore model.HeaderSelectedTipStore,

	multiSetStore model.MultisetStore,
	acceptanceDataStore model.AcceptanceDataStore,
	blocksStore model.BlockStore,
	blockHeaderStore model.BlockHeaderStore,
	utxoDiffStore model.UTXODiffStore,

	isArchivalNode bool,
	genesisHash *externalapi.DomainHash,
	finalityInterval uint64,
	pruningDepth uint64,
) model.PruningManager {

	return &pruningManager{
		databaseContext:        databaseContext,
		dagTraversalManager:    dagTraversalManager,
		dagTopologyManager:     dagTopologyManager,
		consensusStateManager:  consensusStateManager,
		consensusStateStore:    consensusStateStore,
		ghostdagDataStore:      ghostdagDataStore,
		pruningStore:           pruningStore,
		blockStatusStore:       blockStatusStore,
		multiSetStore:          multiSetStore,
		acceptanceDataStore:    acceptanceDataStore,
		blocksStore:            blocksStore,
		blockHeaderStore:       blockHeaderStore,
		utxoDiffStore:          utxoDiffStore,
		headerSelectedTipStore: headerSelectedTipStore,
		isArchivalNode:         isArchivalNode,
		genesisHash:            genesisHash,
		pruningDepth:           pruningDepth,
		finalityInterval:       finalityInterval,
	}
}

// FindNextPruningPoint finds the next pruning point from the
// given blockHash
func (pm *pruningManager) UpdatePruningPointByVirtual() error {
	hasPruningPoint, err := pm.pruningStore.HasPruningPoint(pm.databaseContext)
	if err != nil {
		return err
	}

	if !hasPruningPoint {
		err = pm.savePruningPoint(pm.genesisHash)
		if err != nil {
			return err
		}
	}

	currentCandidate, err := pm.pruningPointCandidate()
	if err != nil {
		return err
	}

	currentCandidateGHOSTDAGData, err := pm.ghostdagDataStore.Get(pm.databaseContext, currentCandidate)
	if err != nil {
		return err
	}

	virtual, err := pm.ghostdagDataStore.Get(pm.databaseContext, model.VirtualBlockHash)
	if err != nil {
		return err
	}

	virtualSelectedParent, err := pm.ghostdagDataStore.Get(pm.databaseContext, virtual.SelectedParent())
	if err != nil {
		return err
	}

	currentPruningPoint, err := pm.pruningStore.PruningPoint(pm.databaseContext)
	if err != nil {
		return err
	}

	currentPruningPointGHOSTDAGData, err := pm.ghostdagDataStore.Get(pm.databaseContext, currentPruningPoint)
	if err != nil {
		return err
	}

	iterator, err := pm.dagTraversalManager.SelectedChildIterator(virtual.SelectedParent(), currentCandidate)
	if err != nil {
		return err
	}

	// Finding the next pruning point candidate: look for the latest
	// selected child of the current candidate that is in depth of at
	// least pm.pruningDepth blocks from the virtual selected parent.
	//
	// Note: Sometimes the current candidate is less than pm.pruningDepth
	// from the virtual. This can happen only if the virtual blue score
	// got smaller, because virtual blue score is not guaranteed to always
	// increase (because sometimes a block with higher blue work can have
	// lower blue score).
	// In such cases we still keep the same candidate because it's guaranteed
	// that a block that was once in depth of pm.pruningDepth cannot be
	// reorged without causing a finality conflict first.
	newCandidate := currentCandidate
	newCandidateGHOSTDAGData := currentCandidateGHOSTDAGData

	newPruningPoint := currentPruningPoint
	newPruningPointGHOSTDAGData := currentPruningPointGHOSTDAGData
	for iterator.Next() {
		selectedChild := iterator.Get()
		selectedChildGHOSTDAGData, err := pm.ghostdagDataStore.Get(pm.databaseContext, selectedChild)
		if err != nil {
			return err
		}

		if virtualSelectedParent.BlueScore()-selectedChildGHOSTDAGData.BlueScore() < pm.pruningDepth {
			break
		}

		newCandidate = selectedChild
		newCandidateGHOSTDAGData = selectedChildGHOSTDAGData

		// We move the pruning point every time the candidate's finality score is
		// bigger than the current pruning point finality score.
		if pm.finalityScore(newCandidateGHOSTDAGData.BlueScore()) > pm.finalityScore(newPruningPointGHOSTDAGData.BlueScore()) {
			newPruningPoint = newCandidate
			newPruningPointGHOSTDAGData = newCandidateGHOSTDAGData
		}
	}

	if !newCandidate.Equal(currentCandidate) {
		log.Debugf("Staged a new pruning candidate, old: %s, new: %s", currentCandidate, newCandidate)
		pm.pruningStore.StagePruningPointCandidate(newCandidate)
	}

	// We move the pruning point every time the candidate's finality score is
	// bigger than the current pruning point finality score.
	if pm.finalityScore(newCandidateGHOSTDAGData.BlueScore()) <= pm.finalityScore(currentPruningPointGHOSTDAGData.BlueScore()) {
		return nil
	}

	if !newPruningPoint.Equal(currentPruningPoint) {
		log.Debugf("Moving pruning point from %s to %s", currentPruningPoint, newPruningPoint)
		err = pm.savePruningPoint(newPruningPoint)
		if err != nil {
			return err
		}
		return pm.deletePastBlocks(newPruningPoint)
	}

	return nil
}

func (pm *pruningManager) isInPruningFutureOrInVirtualPast(block *externalapi.DomainHash, pruningPoint *externalapi.DomainHash, virtualParents []*externalapi.DomainHash) (bool, error) {
	hasPruningPointInPast, err := pm.dagTopologyManager.IsAncestorOf(pruningPoint, block)
	if err != nil {
		return false, err
	}
	if hasPruningPointInPast {
		return true, nil
	}
	// Because virtual doesn't have reachability data, we need to check reachability
	// using it parents.
	isInVirtualPast, err := pm.dagTopologyManager.IsAncestorOfAny(block, virtualParents)
	if err != nil {
		return false, err
	}
	if isInVirtualPast {
		return true, nil
	}

	return false, nil
}

func (pm *pruningManager) deletePastBlocks(pruningPoint *externalapi.DomainHash) error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "pruningManager.deletePastBlocks")
	defer onEnd()

	// Go over all P.Past and P.AC that's not in V.Past
	queue := pm.dagTraversalManager.NewDownHeap()

	// Find P.AC that's not in V.Past
	dagTips, err := pm.consensusStateStore.Tips(pm.databaseContext)
	if err != nil {
		return err
	}
	newTips := make([]*externalapi.DomainHash, 0, len(dagTips))
	virtualParents, err := pm.dagTopologyManager.Parents(model.VirtualBlockHash)
	if err != nil {
		return err
	}
	for _, tip := range dagTips {
		isInPruningFutureOrInVirtualPast, err := pm.isInPruningFutureOrInVirtualPast(tip, pruningPoint, virtualParents)
		if err != nil {
			return err
		}
		if !isInPruningFutureOrInVirtualPast {
			// Add them to the queue so they and their past will be pruned
			err := queue.Push(tip)
			if err != nil {
				return err
			}
		} else {
			newTips = append(newTips, tip)
		}
	}
	pm.consensusStateStore.StageTips(newTips)
	// Add P.Parents
	parents, err := pm.dagTopologyManager.Parents(pruningPoint)
	if err != nil {
		return err
	}
	for _, parent := range parents {
		err = queue.Push(parent)
		if err != nil {
			return err
		}
	}

	visited := map[externalapi.DomainHash]struct{}{}
	// Prune everything in the queue including its past
	for queue.Len() > 0 {
		current := queue.Pop()
		if _, ok := visited[*current]; ok {
			continue
		}
		visited[*current] = struct{}{}

		alreadyPruned, err := pm.deleteBlock(current)
		if err != nil {
			return err
		}
		if !alreadyPruned {
			parents, err := pm.dagTopologyManager.Parents(current)
			if err != nil {
				return err
			}
			for _, parent := range parents {
				err = queue.Push(parent)
				if err != nil {
					return err
				}
			}
		}
	}

	// Delete virtual diff parents that are in PruningPoint's Anticone and not in Virtual's Past
	virtualDiffParents, err := pm.consensusStateStore.VirtualDiffParents(pm.databaseContext)
	if err != nil {
		return err
	}
	validVirtualDiffParents := make([]*externalapi.DomainHash, 0, len(virtualParents))
	for _, parent := range virtualDiffParents {
		isInPruningFutureOrInVirtualPast, err := pm.isInPruningFutureOrInVirtualPast(parent, pruningPoint, virtualParents)
		if err != nil {
			return err
		}
		if isInPruningFutureOrInVirtualPast {
			validVirtualDiffParents = append(validVirtualDiffParents, parent)
		}
	}
	pm.consensusStateStore.StageVirtualDiffParents(validVirtualDiffParents)

	return nil
}

func (pm *pruningManager) savePruningPoint(pruningPointHash *externalapi.DomainHash) error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "pruningManager.savePruningPoint")
	defer onEnd()

	// TODO: This is an assert that takes ~30 seconds to run
	// It must be removed or optimized before launching testnet
	err := pm.validateUTXOSetFitsCommitment(pruningPointHash)
	if err != nil {
		return err
	}

	pm.pruningStore.StagePruningPoint(pruningPointHash)
	pm.pruningStore.StageStartUpdatingPruningPointUTXOSet()

	return nil
}

func (pm *pruningManager) deleteBlock(blockHash *externalapi.DomainHash) (alreadyPruned bool, err error) {
	status, err := pm.blockStatusStore.Get(pm.databaseContext, blockHash)
	if err != nil {
		return false, err
	}
	if status == externalapi.StatusHeaderOnly {
		return true, nil
	}

	pm.blockStatusStore.Stage(blockHash, externalapi.StatusHeaderOnly)
	if pm.isArchivalNode {
		return false, nil
	}

	pm.multiSetStore.Delete(blockHash)
	pm.acceptanceDataStore.Delete(blockHash)
	pm.blocksStore.Delete(blockHash)
	pm.utxoDiffStore.Delete(blockHash)

	return false, nil
}

func (pm *pruningManager) IsValidPruningPoint(blockHash *externalapi.DomainHash) (bool, error) {
	if *pm.genesisHash == *blockHash {
		return true, nil
	}

	headersSelectedTip, err := pm.headerSelectedTipStore.HeadersSelectedTip(pm.databaseContext)
	if err != nil {
		return false, err
	}

	// A pruning point has to be in the selected chain of the headers selected tip.
	headersSelectedTipGHOSTDAGData, err := pm.ghostdagDataStore.Get(pm.databaseContext, headersSelectedTip)
	if err != nil {
		return false, err
	}

	isInSelectedParentChainOfHeadersSelectedTip, err := pm.dagTopologyManager.IsInSelectedParentChainOf(blockHash,
		headersSelectedTip)
	if err != nil {
		return false, err
	}

	if !isInSelectedParentChainOfHeadersSelectedTip {
		return false, nil
	}

	ghostdagData, err := pm.ghostdagDataStore.Get(pm.databaseContext, blockHash)
	if err != nil {
		return false, err
	}

	// A pruning point has to be at depth of at least pm.pruningDepth
	if headersSelectedTipGHOSTDAGData.BlueScore()-ghostdagData.BlueScore() < pm.pruningDepth {
		return false, nil
	}

	selectedParentGHOSTDAGData, err := pm.ghostdagDataStore.Get(pm.databaseContext, ghostdagData.SelectedParent())
	if err != nil {
		return false, err
	}

	// A pruning point has to be the lowest chain block with a certain finality score, so
	// if the block selected parent has the same finality score it means it cannot be a
	// pruning point.
	if pm.finalityScore(ghostdagData.BlueScore()) == pm.finalityScore(selectedParentGHOSTDAGData.BlueScore()) {
		return false, nil
	}

	return true, nil
}

func (pm *pruningManager) pruningPointCandidate() (*externalapi.DomainHash, error) {
	hasPruningPointCandidate, err := pm.pruningStore.HasPruningPointCandidate(pm.databaseContext)
	if err != nil {
		return nil, err
	}

	if !hasPruningPointCandidate {
		return pm.genesisHash, nil
	}

	return pm.pruningStore.PruningPointCandidate(pm.databaseContext)
}

// validateUTXOSetFitsCommitment makes sure that the calculated UTXOSet of the new pruning point fits the commitment.
// This is a sanity test, to make sure that kaspad doesn't store, and subsequently sends syncing peers the wrong UTXOSet.
func (pm *pruningManager) validateUTXOSetFitsCommitment(pruningPointHash *externalapi.DomainHash) error {
	utxoSetIterator, err := pm.consensusStateManager.RestorePastUTXOSetIterator(pruningPointHash)
	if err != nil {
		return err
	}

	utxoSetMultiset := multiset.New()
	for ok := utxoSetIterator.First(); ok; ok = utxoSetIterator.Next() {
		outpoint, entry, err := utxoSetIterator.Get()
		if err != nil {
			return err
		}
		serializedUTXO, err := utxo.SerializeUTXO(entry, outpoint)
		if err != nil {
			return err
		}
		utxoSetMultiset.Add(serializedUTXO)
	}
	utxoSetHash := utxoSetMultiset.Hash()

	header, err := pm.blockHeaderStore.BlockHeader(pm.databaseContext, pruningPointHash)
	if err != nil {
		return err
	}
	expectedUTXOCommitment := header.UTXOCommitment()

	if !expectedUTXOCommitment.Equal(utxoSetHash) {
		return errors.Errorf("Calculated UTXOSet for next pruning point %s doesn't match it's UTXO commitment\n"+
			"Calculated UTXOSet hash: %s. Commitment: %s",
			pruningPointHash, utxoSetHash, expectedUTXOCommitment)
	}

	log.Debugf("Validated the pruning point %s UTXO commitment: %s", pruningPointHash, utxoSetHash)

	return nil
}

// finalityScore is the number of finality intervals passed since
// the given block.
func (pm *pruningManager) finalityScore(blueScore uint64) uint64 {
	return blueScore / pm.finalityInterval
}

func (pm *pruningManager) ClearImportedPruningPointData() error {
	err := pm.pruningStore.ClearImportedPruningPointMultiset(pm.databaseContext)
	if err != nil {
		return err
	}
	return pm.pruningStore.ClearImportedPruningPointUTXOs(pm.databaseContext)
}

func (pm *pruningManager) AppendImportedPruningPointUTXOs(
	outpointAndUTXOEntryPairs []*externalapi.OutpointAndUTXOEntryPair) error {

	dbTx, err := pm.databaseContext.Begin()
	if err != nil {
		return err
	}
	defer dbTx.RollbackUnlessClosed()

	importedMultiset, err := pm.pruningStore.ImportedPruningPointMultiset(dbTx)
	if err != nil {
		if !database.IsNotFoundError(err) {
			return err
		}
		importedMultiset = multiset.New()
	}
	for _, outpointAndUTXOEntryPair := range outpointAndUTXOEntryPairs {
		serializedUTXO, err := utxo.SerializeUTXO(outpointAndUTXOEntryPair.UTXOEntry, outpointAndUTXOEntryPair.Outpoint)
		if err != nil {
			return err
		}
		importedMultiset.Add(serializedUTXO)
	}
	err = pm.pruningStore.UpdateImportedPruningPointMultiset(dbTx, importedMultiset)
	if err != nil {
		return err
	}

	err = pm.pruningStore.AppendImportedPruningPointUTXOs(dbTx, outpointAndUTXOEntryPairs)
	if err != nil {
		return err
	}

	return dbTx.Commit()
}

func (pm *pruningManager) UpdatePruningPointUTXOSetIfRequired() error {
	hadStartedUpdatingPruningPointUTXOSet, err := pm.pruningStore.HadStartedUpdatingPruningPointUTXOSet(pm.databaseContext)
	if err != nil {
		return err
	}
	if !hadStartedUpdatingPruningPointUTXOSet {
		return nil
	}

	log.Debugf("Pruning point UTXO set update is required")
	err = pm.updatePruningPointUTXOSet()
	if err != nil {
		return err
	}
	log.Debugf("Pruning point UTXO set updated")

	return nil
}

func (pm *pruningManager) updatePruningPointUTXOSet() error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "updatePruningPointUTXOSet")
	defer onEnd()

	logger.LogMemoryStats(log, "updatePruningPointUTXOSet start")
	defer logger.LogMemoryStats(log, "updatePruningPointUTXOSet end")

	log.Debugf("Getting the pruning point")
	pruningPoint, err := pm.pruningStore.PruningPoint(pm.databaseContext)
	if err != nil {
		return err
	}

	log.Debugf("Restoring the pruning point UTXO set")
	utxoSetIterator, err := pm.consensusStateManager.RestorePastUTXOSetIterator(pruningPoint)
	if err != nil {
		return err
	}

	log.Debugf("Updating the pruning point UTXO set")
	err = pm.pruningStore.UpdatePruningPointUTXOSet(pm.databaseContext, utxoSetIterator)
	if err != nil {
		return err
	}

	log.Debugf("Finishing updating the pruning point UTXO set")
	return pm.pruningStore.FinishUpdatingPruningPointUTXOSet(pm.databaseContext)
}
