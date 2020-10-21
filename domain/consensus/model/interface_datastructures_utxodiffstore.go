package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// UTXODiffStore represents a store of UTXODiffs
type UTXODiffStore interface {
	Stage(blockHash *externalapi.DomainHash, utxoDiff *UTXODiff, utxoDiffChild *externalapi.DomainHash)
	IsStaged() bool
	Discard()
	Commit(dbTx DBTxProxy) error
	UTXODiff(dbContext DBContextProxy, blockHash *externalapi.DomainHash) (*UTXODiff, error)
	UTXODiffChild(dbContext DBContextProxy, blockHash *externalapi.DomainHash) (*externalapi.DomainHash, error)
	Delete(dbTx DBTxProxy, blockHash *externalapi.DomainHash) error
}
