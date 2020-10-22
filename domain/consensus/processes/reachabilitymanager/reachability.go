package reachabilitymanager

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

// IsDAGAncestorOf returns true if blockHashA is an ancestor of
// blockHashB in the DAG.
//
// Note: this method will return true if blockHashA == blockHashA
// The complexity of this method is O(log(|this.futureCoveringTreeNodeSet|))
func (rt *reachabilityTreeManager) IsDAGAncestorOf(blockHashA, blockHashB *externalapi.DomainHash) (bool, error) {
	// Check if this node is a reachability tree ancestor of the
	// other node
	isReachabilityTreeAncestor, err := rt.IsReachabilityTreeAncestorOf(blockHashA, blockHashB)
	if err != nil {
		return false, err
	}
	if isReachabilityTreeAncestor {
		return true, nil
	}

	// Otherwise, use previously registered future blocks to complete the
	// reachability test
	return rt.futureCoveringSetHasAncestorOf(blockHashA, blockHashB)
}

func (rt *reachabilityTreeManager) UpdateReindexRoot(selectedTip *externalapi.DomainHash) error {
	return rt.updateReindexRoot(selectedTip)
}
