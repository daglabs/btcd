package blockstatusstore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
)

// BlockStatusStore represents a store of BlockStatuses
type BlockStatusStore struct {
}

// New instantiates a new BlockStatusStore
func New() *BlockStatusStore {
	return &BlockStatusStore{}
}

// Insert inserts the given blockStatus for the given blockHash
func (bss *BlockStatusStore) Insert(dbTx model.DBTxProxy, blockHash *model.DomainHash, blockStatus model.BlockStatus) {

}

// Get gets the blockStatus associated with the given blockHash
func (bss *BlockStatusStore) Get(dbContext model.DBContextProxy, blockHash *model.DomainHash) model.BlockStatus {
	return 0
}

func (bss *BlockStatusStore) Exists(dbContext model.DBContextProxy, blockHash *model.DomainHash) bool {
	return false
}
