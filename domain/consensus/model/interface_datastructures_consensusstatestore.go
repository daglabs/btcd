package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// ConsensusStateStore represents a store for the current consensus state
type ConsensusStateStore interface {
	Update(dbTx DBTxProxy, consensusStateChanges *ConsensusStateChanges) error
	UTXOByOutpoint(dbContext DBContextProxy, outpoint *externalapi.DomainOutpoint) (*externalapi.UTXOEntry, error)
	Tips(dbContext DBContextProxy) ([]*externalapi.DomainHash, error)
}
