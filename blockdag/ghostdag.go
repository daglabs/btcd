package blockdag

import "github.com/pkg/errors"

func (dag *BlockDAG) selectedParentAnticone(node *blockNode) (*blockHeap, error) {
	anticoneSet := newSet()
	anticoneHeap := newUpHeap()
	past := newSet()
	var queue []*blockNode
	for _, parent := range node.parents {
		if parent == node.selectedParent {
			continue
		}
		anticoneSet.add(parent)
		queue = append(queue, parent)
	}
	for len(queue) > 0 {
		var current *blockNode
		current, queue = queue[0], queue[1:]
		for _, parent := range current.parents {
			if anticoneSet.contains(parent) || past.contains(parent) {
				continue
			}
			isAncestorOfSelectedParent, err := dag.isAncestorOf(parent, node.selectedParent)
			if err != nil {
				return nil, err
			}
			if isAncestorOfSelectedParent {
				past.add(parent)
				continue
			}
			anticoneSet.add(parent)
			anticoneHeap.Push(parent)
			queue = append(queue, parent)
		}
	}
	return &anticoneHeap, nil
}

// blueAnticoneSize returns the blue anticone size of 'block' from the worldview of 'context'.
// Expects 'block' to be ∈ blue-set(context)
func (dag *BlockDAG) blueAnticoneSize(block, context *blockNode) (uint32, error) {
	node := context
	for {
		if isAncestorOf, err := dag.isAncestorOf(block, node); err != nil {
			return 0, err
		} else if !isAncestorOf {
			return 0, errors.Errorf("block %s is not in blue-set of %s", block.hash, context.hash)
		}

		if blueAnticoneSize, ok := node.bluesAnticoneSizes[*block.hash]; ok {
			return blueAnticoneSize, nil
		}
	}
}

func (dag *BlockDAG) ghostdag(newNode *blockNode) (*blockHeap, error) {
	newNode.selectedParent = newNode.parents.bluest()
	selectedParentAnticone, err := dag.selectedParentAnticone(newNode)
	if err != nil {
		return nil, err
	}

	for selectedParentAnticone.Len() > 0 {
		blueCandidate := selectedParentAnticone.pop()
		candidateBluesAnticoneSizes := make(map[*blockNode]uint32)
		var candidateAnticoneSize uint32
		possiblyBlue := true

		for chainBlock := newNode; possiblyBlue; chainBlock = chainBlock.selectedParent {
			chainBlockAndItsBlues := make([]*blockNode, 0, len(chainBlock.blues)+1)
			chainBlockAndItsBlues = append(chainBlockAndItsBlues, chainBlock)
			chainBlockAndItsBlues = append(chainBlockAndItsBlues, chainBlock.blues...)

			if isAncestorOf, err := dag.isAncestorOf(chainBlock, blueCandidate); err != nil {
				return nil, err
			} else if isAncestorOf {
				// All remaining blues are in past(chainBlock) and thus in past(blueCandidate)
				break
			}

			for _, block := range chainBlockAndItsBlues {
				// Skip blocks that exists in the past of blueCandidate.
				// We already checked it for chainBlock above, so if the
				// block is chainBlock, there's no need to recheck.
				if block != chainBlock {
					if isAncestorOf, err := dag.isAncestorOf(block, blueCandidate); err != nil {
						return nil, err
					} else if isAncestorOf {
						continue
					}
				}

				candidateBluesAnticoneSizes[block], err = dag.blueAnticoneSize(block, newNode)
				if err != nil {
					return nil, err
				}
				candidateAnticoneSize++
				if candidateAnticoneSize > dag.dagParams.K || candidateBluesAnticoneSizes[block] == dag.dagParams.K {
					// Two possible k-cluster violations here:
					// 	(i) The candidate blue anticone now became larger than k
					//	(ii) A block in candidate's blue anticone already has k blue
					//	blocks in its own anticone
					possiblyBlue = false
					break
				}
				if candidateBluesAnticoneSizes[block] > dag.dagParams.K {
					return nil, errors.New("found blue anticone size larger than k")
				}
			}
		}

		if possiblyBlue {
			// No k-cluster violation found, we can now set the candidate block as blue
			newNode.blues = append(newNode.blues, blueCandidate)
			for blue, blueAnticoneSize := range candidateBluesAnticoneSizes {
				candidateBluesAnticoneSizes[blue] = blueAnticoneSize + 1
			}
			if uint32(len(newNode.blues)) == dag.dagParams.K {
				break
			}
		}
	}

	newNode.blues = append(newNode.blues, newNode.selectedParent)
	newNode.blueScore = newNode.selectedParent.blueScore + uint64(len(newNode.blues))
	return selectedParentAnticone, nil
}
