package ghostdagmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

func (gm *ghostdagManager) findSelectedParent(parentHashes []*externalapi.DomainHash) (*externalapi.DomainHash, error) {
	var selectedParent *externalapi.DomainHash
	for _, hash := range parentHashes {
		if selectedParent == nil {
			selectedParent = hash
			continue
		}
		isHashBiggerThanSelectedParent, err := gm.less(selectedParent, hash)
		if err != nil {
			return nil, err
		}
		if isHashBiggerThanSelectedParent {
			selectedParent = hash
		}
	}
	return selectedParent, nil
}

func (gm *ghostdagManager) less(blockHashA *externalapi.DomainHash, blockHashB *externalapi.DomainHash) (bool, error) {
	chosenSelectedParent, err := gm.ChooseSelectedParent(blockHashA, blockHashB)
	if err != nil {
		return false, err
	}
	return chosenSelectedParent == blockHashB, nil
}

func (gm *ghostdagManager) ChooseSelectedParent(blockHashA *externalapi.DomainHash,
	blockHashB *externalapi.DomainHash) (*externalapi.DomainHash, error) {

	blockAGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, blockHashA)
	if err != nil {
		return nil, err
	}
	blockBGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, blockHashB)
	if err != nil {
		return nil, err
	}

	blockABlueScore := blockAGHOSTDAGData.BlueScore
	blockBBlueScore := blockBGHOSTDAGData.BlueScore
	if blockABlueScore == blockBBlueScore {
		if hashesLess(blockHashA, blockHashB) {
			return blockHashB, nil
		}
		return blockHashA, nil
	}
	if blockABlueScore < blockBBlueScore {
		return blockHashB, nil
	}
	return blockHashA, nil
}

func hashesLess(a, b *externalapi.DomainHash) bool {
	// We compare the hashes backwards because Hash is stored as a little endian byte array.
	for i := len(a) - 1; i >= 0; i-- {
		switch {
		case a[i] < b[i]:
			return true
		case a[i] > b[i]:
			return false
		}
	}
	return false
}
