package dagtraversalmanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/hashset"
)

// AnticoneFromContext returns blocks in (context.Past ⋂ block.Anticone)
func (dtm *dagTraversalManager) AnticoneFromContext(context, block *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	anticone := []*externalapi.DomainHash{}
	queue := []*externalapi.DomainHash{context}
	visited := hashset.New()

	for len(queue) > 0 {
		var current *externalapi.DomainHash
		current, queue = queue[0], queue[1:]

		if visited.Contains(current) {
			continue
		}

		visited.Add(current)

		currentIsAncestorOfBlock, err := dtm.dagTopologyManager.IsAncestorOf(current, block)
		if err != nil {
			return nil, err
		}

		if currentIsAncestorOfBlock {
			continue
		}

		blockIsAncestorOfCurrent, err := dtm.dagTopologyManager.IsAncestorOf(block, current)
		if err != nil {
			return nil, err
		}

		if !blockIsAncestorOfCurrent {
			anticone = append(anticone, current)
		}

		currentParents, err := dtm.dagTopologyManager.Parents(current)
		if err != nil {
			return nil, err
		}

		for _, parent := range currentParents {
			queue = append(queue, parent)
		}
	}

	return anticone, nil
}
