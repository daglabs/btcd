package ghostdagmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"sort"
)

func (gm *ghostdagManager) mergeSet(selecteParent *model.DomainHash,
	blockParents []*model.DomainHash) ([]*model.DomainHash, error) {

	mergeSetMap := make(map[model.DomainHash]struct{}, gm.k)
	mergeSetSlice := make([]*model.DomainHash, 0, gm.k)
	selectedParentPast := make(map[model.DomainHash]struct{})
	queue := []*model.DomainHash{}
	// Queueing all parents (other than the selected parent itself) for processing.
	for _, parent := range blockParents {
		if *parent == *selecteParent {
			continue
		}
		mergeSetMap[*parent] = struct{}{}
		mergeSetSlice = append(mergeSetSlice, parent)
		queue = append(queue, parent)
	}

	for len(queue) > 0 {
		var current *model.DomainHash
		current, queue = queue[0], queue[1:]
		// For each parent of the current block we check whether it is in the past of the selected parent. If not,
		// we add the it to the resulting anticone-set and queue it for further processing.
		currentParents, err := gm.dagTopologyManager.Parents(current)
		if err != nil {
			return nil, err
		}
		for _, parent := range currentParents {
			if _, ok := mergeSetMap[*parent]; ok {
				continue
			}

			if _, ok := selectedParentPast[*parent]; ok {
				continue
			}

			isAncestorOfSelectedParent, err := gm.dagTopologyManager.IsAncestorOf(parent, selecteParent)
			if err != nil {
				return nil, err
			}

			if isAncestorOfSelectedParent {
				selectedParentPast[*parent] = struct{}{}
				continue
			}

			mergeSetMap[*parent] = struct{}{}
			mergeSetSlice = append(mergeSetSlice, parent)
			queue = append(queue, parent)
		}
	}

	err := gm.sortMergeSet(mergeSetSlice)
	if err != nil {
		return nil, err
	}

	return mergeSetSlice, nil
}

func (gm *ghostdagManager) sortMergeSet(mergeSetSlice []*model.DomainHash) error {
	var err error
	sort.Slice(mergeSetSlice, func(i, j int) bool {
		if err != nil {
			return false
		}
		isLess, lessErr := gm.less(mergeSetSlice[i], mergeSetSlice[j])
		if lessErr != nil {
			err = lessErr
			return false
		}
		return isLess
	})
	return err
}
