package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// ConsensusStateManager manages the node's consensus state
type ConsensusStateManager interface {
	AddBlockToVirtual(blockHash *externalapi.DomainHash) error
	VirtualData() (medianTime int64, blueScore uint64, err error)
}
