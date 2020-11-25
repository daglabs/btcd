package headertipsmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

type headerTipsManager struct {
	databaseContext    model.DBReader
	dagTopologyManager model.DAGTopologyManager
	ghostdagManager    model.GHOSTDAGManager
	headerTipsStore    model.HeaderTipsStore
}

// New instantiates a new HeaderTipsManager
func New(databaseContext model.DBReader,
	dagTopologyManager model.DAGTopologyManager,
	ghostdagManager model.GHOSTDAGManager,
	headerTipsStore model.HeaderTipsStore) model.HeaderTipsManager {
	return &headerTipsManager{
		databaseContext:    databaseContext,
		dagTopologyManager: dagTopologyManager,
		ghostdagManager:    ghostdagManager,
		headerTipsStore:    headerTipsStore,
	}
}

func (h headerTipsManager) AddHeaderTip(hash *externalapi.DomainHash) error {
	tips := []*externalapi.DomainHash{}
	hasTips, err := h.headerTipsStore.HasTips(h.databaseContext)
	if err != nil {
		return err
	}

	if hasTips {
		var err error
		tips, err = h.headerTipsStore.Tips(h.databaseContext)
		if err != nil {
			return err
		}
	}

	newTips := make([]*externalapi.DomainHash, 0, len(tips)+1)
	for _, tip := range tips {
		isAncestorOf, err := h.dagTopologyManager.IsAncestorOf(tip, hash)
		if err != nil {
			return err
		}

		if !isAncestorOf {
			newTips = append(newTips, tip)
		}
	}

	newTips = append(newTips, hash)
	h.headerTipsStore.Stage(newTips)

	return nil
}
