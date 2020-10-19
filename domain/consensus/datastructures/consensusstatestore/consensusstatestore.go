package consensusstatestore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

// consensusStateStore represents a store for the current consensus state
type consensusStateStore struct {
}

// New instantiates a new ConsensusStateStore
func New() model.ConsensusStateStore {
	return &consensusStateStore{}
}

// Update updates the store with the given consensusStateChanges
func (css *consensusStateStore) Update(dbTx model.DBTxProxy, consensusStateChanges *model.ConsensusStateChanges) error {
	return nil
}

// UTXOByOutpoint gets the utxoEntry associated with the given outpoint
func (css *consensusStateStore) UTXOByOutpoint(dbContext model.DBContextProxy, outpoint *externalapi.DomainOutpoint) (*externalapi.UTXOEntry, error) {
	return nil, nil
}

// Tips returns the current tips
func (css *consensusStateStore) Tips(dbContext model.DBContextProxy) ([]*externalapi.DomainHash, error) {
	return nil, nil
}
