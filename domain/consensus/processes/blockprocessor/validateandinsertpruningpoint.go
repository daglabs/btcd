package blockprocessor

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/pkg/errors"
)

func (bp *blockProcessor) validateAndInsertPruningPoint(newPruningPoint *externalapi.DomainBlock, serializedUTXOSet []byte) error {
	log.Info("Checking that the given pruning point is the expected pruning point")

	newPruningPointHash := consensushashing.BlockHash(newPruningPoint)
	isValidPruningPoint, err := bp.pruningManager.IsValidPruningPoint(newPruningPointHash)
	if err != nil {
		return err
	}

	if !isValidPruningPoint {
		return errors.Wrapf(ruleerrors.ErrUnexpectedPruningPoint, "%s is not a valid pruning point",
			newPruningPointHash)
	}

	// We have to validate the pruning point block before we set the new pruning point in consensusStateManager.
	log.Infof("Validating the new pruning point %s", newPruningPointHash)
	err = bp.validateBlockAndDiscardChanges(newPruningPoint, true)
	if err != nil {
		return err
	}

	log.Infof("Updating consensus state manager according to the new pruning point %s", newPruningPointHash)
	err = bp.consensusStateManager.UpdatePruningPoint(newPruningPoint, serializedUTXOSet)
	if err != nil {
		return err
	}

	log.Infof("Inserting the new pruning point %s", newPruningPointHash)
	_, err = bp.validateAndInsertBlock(newPruningPoint, true)
	if err != nil && errors.As(err, &ruleerrors.RuleError{}) {
		// This should never happen because we already validated the block with bp.validateBlockAndDiscardChanges.
		// We use Errorf so it won't be identified later on to be a rule error and will eventually cause
		// the program to panic.
		return errors.Errorf("validateAndInsertBlock returned unexpected rule error while processing "+
			"the pruning point: %+v", err)
	}
	return err
}
