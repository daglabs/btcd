package model

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

// BlockTemplateBuilder builds block templates for miners to consume
type BlockTemplateBuilder interface {
	GetBlockTemplate(payAddress DomainAddress, extraData []byte) *externalapi.DomainBlock
}