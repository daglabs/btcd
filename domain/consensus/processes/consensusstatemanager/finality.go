package consensusstatemanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

func (csm *consensusStateManager) checkFinalityViolation(
	blockHash *externalapi.DomainHash) error {

	if *blockHash == *csm.genesisHash {
		return nil
	}

	isViolatingFinality, err := csm.isViolatingFinality(blockHash)
	if err != nil {
		return err
	}

	if isViolatingFinality {
		csm.blockStatusStore.Stage(blockHash, externalapi.StatusUTXOPendingVerification)
		log.Warnf("Finality Violation Detected! Block %s violates finality!", blockHash)
	}

	return nil
}

func (csm *consensusStateManager) virtualFinalityPoint(virtualGHOSTDAGData *model.BlockGHOSTDAGData) (
	*externalapi.DomainHash, error) {

	blueScore := virtualGHOSTDAGData.BlueScore - csm.finalityDepth
	if virtualGHOSTDAGData.BlueScore < csm.finalityDepth {
		blueScore = 0
	}

	return csm.dagTraversalManager.HighestChainBlockBelowBlueScore(
		model.VirtualBlockHash, blueScore)
}

func (csm *consensusStateManager) isViolatingFinality(
	blockHash *externalapi.DomainHash) (bool, error) {

	virtualGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, model.VirtualBlockHash)
	if err != nil {
		return false, err
	}

	virtualFinalityPoint, err := csm.virtualFinalityPoint(virtualGHOSTDAGData)
	if err != nil {
		return false, err
	}

	return csm.dagTopologyManager.IsInSelectedParentChainOf(virtualFinalityPoint, blockHash)
}
