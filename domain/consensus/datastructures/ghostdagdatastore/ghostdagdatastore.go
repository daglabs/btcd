package ghostdagdatastore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
)

// ghostdagDataStore represents a store of BlockGHOSTDAGData
type ghostdagDataStore struct {
}

// New instantiates a new ghostdagDataStore
func New() model.GHOSTDAGDataStore {
	return &ghostdagDataStore{}
}

// Insert inserts the given blockGHOSTDAGData for the given blockHash
func (gds *ghostdagDataStore) Insert(dbTx model.DBTxProxy, blockHash *model.DomainHash, blockGHOSTDAGData *model.BlockGHOSTDAGData) error {
	return nil
}

// Get gets the blockGHOSTDAGData associated with the given blockHash
func (gds *ghostdagDataStore) Get(dbContext model.DBContextProxy, blockHash *model.DomainHash) (*model.BlockGHOSTDAGData, error) {
	return nil, nil
}
