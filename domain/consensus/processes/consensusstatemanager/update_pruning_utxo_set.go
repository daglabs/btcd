package consensusstatemanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"github.com/kaspanet/kaspad/infrastructure/logger"
	"github.com/pkg/errors"
)

func (csm *consensusStateManager) UpdatePruningPoint(newPruningPoint *externalapi.DomainBlock) error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "UpdatePruningPoint")
	defer onEnd()

	err := csm.consensusStateStore.BeginOverwritingVirtualUTXOSet()
	if err != nil {
		return err
	}

	err = csm.updatePruningPoint(newPruningPoint)
	if err != nil {
		csm.discardSetPruningPointUTXOSetChanges()
		return err
	}

	err = csm.commitSetPruningPointUTXOSetAll()
	if err != nil {
		return err
	}

	return csm.consensusStateStore.FinishOverwritingVirtualUTXOSet()
}

func (csm *consensusStateManager) updatePruningPoint(newPruningPoint *externalapi.DomainBlock) error {
	log.Debugf("updatePruningPoint start")
	defer log.Debugf("updatePruningPoint end")

	newPruningPointHash := consensushashing.BlockHash(newPruningPoint)

	// We ignore the shouldSendNotification return value because we always want to send finality conflict notification
	// in case the new pruning point violates finality
	isViolatingFinality, _, err := csm.isViolatingFinality(newPruningPointHash)
	if err != nil {
		return err
	}

	if isViolatingFinality {
		log.Warnf("Finality Violation Detected! The suggest pruning point %s violates finality!", newPruningPointHash)
		return errors.Wrapf(ruleerrors.ErrSuggestedPruningViolatesFinality, "%s cannot be a pruning point because "+
			"it violates finality", newPruningPointHash)
	}

	utxoSetMultiset, err := csm.pruningStore.CandidatePruningPointMultiset(csm.databaseContext)
	if err != nil {
		return err
	}

	newPruningPointHeader, err := csm.blockHeaderStore.BlockHeader(csm.databaseContext, newPruningPointHash)
	if err != nil {
		return err
	}
	log.Debugf("The UTXO commitment of the pruning point: %s",
		newPruningPointHeader.UTXOCommitment())

	if !newPruningPointHeader.UTXOCommitment().Equal(utxoSetMultiset.Hash()) {
		return errors.Wrapf(ruleerrors.ErrBadPruningPointUTXOSet, "the expected multiset hash of the pruning "+
			"point UTXO set is %s but got %s", newPruningPointHeader.UTXOCommitment(), *utxoSetMultiset.Hash())
	}
	log.Debugf("The new pruning point UTXO commitment validation passed")

	log.Debugf("Staging the the pruning point as the only DAG tip")
	newTips := []*externalapi.DomainHash{newPruningPointHash}
	csm.consensusStateStore.StageTips(newTips)

	log.Debugf("Setting the pruning point as the only virtual parent")
	err = csm.dagTopologyManager.SetParents(model.VirtualBlockHash, newTips)
	if err != nil {
		return err
	}

	log.Debugf("Calculating GHOSTDAG for the new virtual")
	err = csm.ghostdagManager.GHOSTDAG(model.VirtualBlockHash)
	if err != nil {
		return err
	}

	pruningPointUTXOSetIterator, err := csm.pruningStore.CandidatePruningPointUTXOIterator(csm.databaseContext)
	if err != nil {
		return err
	}

	log.Debugf("Overwriting the virtual UTXO set")
	err = csm.consensusStateStore.OverwriteVirtualUTXOSet(pruningPointUTXOSetIterator)
	if err != nil {
		return err
	}

	log.Debugf("Deleting all the existing virtual diff parents")
	csm.consensusStateStore.StageVirtualDiffParents(nil)

	log.Debugf("Updating the new pruning point to be the new virtual diff parent with an empty diff")
	err = csm.stageDiff(newPruningPointHash, utxo.NewUTXODiff(), nil)
	if err != nil {
		return err
	}

	log.Debugf("Staging the new pruning point")
	csm.pruningStore.StagePruningPoint(newPruningPointHash)

	log.Debugf("Committing the new pruning point UTXO set")
	err = csm.pruningStore.CommitCandidatePruningPointUTXOSet()
	if err != nil {
		return err
	}

	// Before we manually mark the new pruning point as valid, we validate that all of its transactions are valid
	// against the provided UTXO set.
	log.Debugf("Validating that the pruning point is UTXO valid")

	// validateBlockTransactionsAgainstPastUTXO pre-fills the block's transactions inputs, which
	// are assumed to not be pre-filled during further validations.
	// Therefore - clone newPruningPoint before passing it to validateBlockTransactionsAgainstPastUTXO
	err = csm.validateBlockTransactionsAgainstPastUTXO(newPruningPoint.Clone(), utxo.NewUTXODiff())
	if err != nil {
		return err
	}

	log.Debugf("Staging the new pruning point as %s", externalapi.StatusUTXOValid)
	csm.blockStatusStore.Stage(newPruningPointHash, externalapi.StatusUTXOValid)

	log.Debugf("Staging the new pruning point multiset")
	csm.multisetStore.Stage(newPruningPointHash, utxoSetMultiset)
	return nil
}

func (csm *consensusStateManager) discardSetPruningPointUTXOSetChanges() {
	for _, store := range csm.stores {
		store.Discard()
	}
}

func (csm *consensusStateManager) commitSetPruningPointUTXOSetAll() error {
	dbTx, err := csm.databaseContext.Begin()
	if err != nil {
		return err
	}

	for _, store := range csm.stores {
		err = store.Commit(dbTx)
		if err != nil {
			return err
		}
	}

	return dbTx.Commit()
}
