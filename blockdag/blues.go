package blockdag

func blues(block *blockNode) (blues []*blockNode, selectedParent *blockNode, score int64) {
	bestScore := int64(-1)
	var bestParent *blockNode
	var bestBlues []*blockNode
	for _, parent := range block.parents.toSlice(true) {
		chainStart := digToChainStart(block, parent)
		candidates := blueCandidates(chainStart)
		blues := traverseCandidates(block, candidates, parent)
		score := int64(len(blues)) + parent.blueScore

		if score > bestScore {
			bestScore = score
			bestBlues = blues
			bestParent = parent
		}
	}

	return bestBlues, bestParent, bestScore
}

// digToChainStart digs through the chain and returns the block in depth k+1
func digToChainStart(block *blockNode, parent *blockNode) *blockNode {
	current := parent

	for i := uint(0); i < phantomK; i++ {
		if current.isGenesis() {
			break
		}
		current = current.selectedParent
	}

	return current
}

func blueCandidates(chainStart *blockNode) blockSet {
	candidates := newSet()
	candidates.add(chainStart)

	queue := []*blockNode{chainStart}
	for len(queue) > 0 {
		var current *blockNode
		current, queue = queue[0], queue[1:]

		children := current.children
		for _, child := range children {
			if !candidates.contains(child) {
				candidates.add(child)
				queue = append(queue, child)
			}
		}
	}

	return candidates
}

func traverseCandidates(newBlock *blockNode, candidates blockSet, selectedParent *blockNode) []*blockNode {
	blues := []*blockNode{}
	selectedParentPast := newSet()
	queue := NewHeap()
	visited := newSet()

	for _, parent := range newBlock.parents {
		queue.Push(parent)
	}

	for queue.Len() > 0 {
		current := queue.Pop()
		if candidates.contains(current) {
			if current == selectedParent || selectedParentPast.anyChildInSet(current) {
				selectedParentPast.add(current)
			} else {
				blues = append(blues, current)
			}
			for _, parent := range current.parents {
				if !visited.contains(parent) {
					visited.add(parent)
					queue.Push(parent)
				}
			}
		}
	}

	return append(blues, selectedParent)
}

var phantomK uint = 1
