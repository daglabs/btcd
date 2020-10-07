package utxodiffstore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/util/daghash"
)

// UTXODiffStore represents a store of UTXODiffs
type UTXODiffStore struct {
}

// New instantiates a new UTXODiffStore
func New() *UTXODiffStore {
	return &UTXODiffStore{}
}

// Insert inserts the given utxoDiff for the given blockHash
func (uds *UTXODiffStore) Insert(dbTx model.TxContextProxy, blockHash *daghash.Hash, utxoDiff *model.UTXODiff) {

}

// Get gets the utxoDiff associated with the given blockHash
func (uds *UTXODiffStore) Get(dbContext model.ContextProxy, blockHash *daghash.Hash) *model.UTXODiff {
	return nil
}
