package pruningstore

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

type multisetStagingShard struct {
	store *pruningStore

	newPruningPoint                  *externalapi.DomainHash
	newPruningPointCandidate         *externalapi.DomainHash
	startUpdatingPruningPointUTXOSet bool
}

func (ps *pruningStore) stagingShard(stagingArea *model.StagingArea) *multisetStagingShard {
	return stagingArea.GetOrCreateShard("BlockStore", func() model.StagingShard {
		return &multisetStagingShard{
			store:                            ps,
			newPruningPoint:                  nil,
			newPruningPointCandidate:         nil,
			startUpdatingPruningPointUTXOSet: false,
		}
	}).(*multisetStagingShard)
}

func (mss multisetStagingShard) Commit(dbTx model.DBTransaction) error {
	if mss.newPruningPoint != nil {
		pruningPointBytes, err := mss.store.serializeHash(mss.newPruningPoint)
		if err != nil {
			return err
		}
		err = dbTx.Put(pruningBlockHashKey, pruningPointBytes)
		if err != nil {
			return err
		}
		mss.store.pruningPointCache = mss.newPruningPoint
	}

	if mss.newPruningPointCandidate != nil {
		candidateBytes, err := mss.store.serializeHash(mss.newPruningPointCandidate)
		if err != nil {
			return err
		}
		err = dbTx.Put(candidatePruningPointHashKey, candidateBytes)
		if err != nil {
			return err
		}
		mss.store.pruningPointCandidateCache = mss.newPruningPointCandidate
	}

	if mss.startUpdatingPruningPointUTXOSet {
		err := dbTx.Put(updatingPruningPointUTXOSetKey, []byte{0})
		if err != nil {
			return err
		}
	}

	return nil
}