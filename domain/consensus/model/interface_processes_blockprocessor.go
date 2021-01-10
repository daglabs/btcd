package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// BlockProcessor is responsible for processing incoming blocks
type BlockProcessor interface {
	ValidateAndInsertBlock(block *externalapi.DomainBlock) (*externalapi.BlockInsertionResult, error)
	ValidateAndInsertPruningPoint(newPruningPoint *externalapi.DomainBlock, serializedUTXOSet []byte) error
}
