package blockrelationstore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/util/daghash"
)

// BlockRelationStore represents a store of BlockRelations
type BlockRelationStore struct {
}

// New instantiates a new BlockRelationStore
func New() *BlockRelationStore {
	return &BlockRelationStore{}
}

// Insert inserts the given blockRelationData for the given blockHash
func (brs *BlockRelationStore) Insert(dbTx model.TxContextProxy, blockHash *daghash.Hash, blockRelationData *model.BlockRelations) {

}

// Get gets the blockRelationData associated with the given blockHash
func (brs *BlockRelationStore) Get(dbContext model.ContextProxy, blockHash *daghash.Hash) *model.BlockRelations {
	return nil
}
