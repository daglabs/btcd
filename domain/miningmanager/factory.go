package miningmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/domain/miningmanager/blocktemplatebuilder"
	mempoolpkg "github.com/kaspanet/kaspad/domain/miningmanager/mempool"
)

// Factory instantiates new mining managers
type Factory interface {
	NewMiningManager(consensus externalapi.Consensus, params *dagconfig.Params) MiningManager
}

type factory struct{}

// NewMiningManager instantiate a new mining manager
func (f *factory) NewMiningManager(consensus externalapi.Consensus, params *dagconfig.Params) MiningManager {
	mempool := mempoolpkg.New(consensus, params)
	blockTemplateBuilder := blocktemplatebuilder.New(consensus, mempool, params.MaxMassAcceptedByBlock)

	return &miningManager{
		mempool:              mempool,
		blockTemplateBuilder: blockTemplateBuilder,
	}
}

// NewFactory creates a new mining manager factory
func NewFactory() Factory {
	return &factory{}
}
