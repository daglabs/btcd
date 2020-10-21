package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// GHOSTDAGManager resolves and manages GHOSTDAG block data
type GHOSTDAGManager interface {
	GHOSTDAG(blockHash *externalapi.DomainHash) error
	ChooseSelectedParent(
		blockHashA *externalapi.DomainHash, blockAGHOSTDAGData *BlockGHOSTDAGData,
		blockHashB *externalapi.DomainHash, blockBGHOSTDAGData *BlockGHOSTDAGData) *externalapi.DomainHash
}
