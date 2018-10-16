// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"sync"

	"github.com/daglabs/btcd/wire"
)

// VirtualBlock is a virtual block whose parents are the tips of the DAG.
type VirtualBlock struct {
	mtx      sync.Mutex
	phantomK uint32
	utxoSet  *fullUTXOSet
	blockNode
	// selectedPathSet is a block set that includes all the blocks that belong to the chain of selected parents from the virtual block.
	selectedPathSet blockSet
}

// newVirtualBlock creates and returns a new VirtualBlock.
func newVirtualBlock(tips blockSet, phantomK uint32) *VirtualBlock {
	// The mutex is intentionally not held since this is a constructor.
	var virtual VirtualBlock
	virtual.phantomK = phantomK
	virtual.utxoSet = NewFullUTXOSet()
	virtual.selectedPathSet = newSet()
	virtual.setTips(tips)

	return &virtual
}

// clone creates and returns a clone of the virtual block.
func (v *VirtualBlock) clone() *VirtualBlock {
	return &VirtualBlock{
		phantomK:        v.phantomK,
		utxoSet:         v.utxoSet.clone().(*fullUTXOSet),
		blockNode:       v.blockNode,
		selectedPathSet: v.selectedPathSet,
	}
}

// setTips replaces the tips of the virtual block with the blocks in the
// given blockSet. This only differs from the exported version in that it
// is up to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for writes).
func (v *VirtualBlock) setTips(tips blockSet) {
	oldSelectedParent := v.selectedParent
	v.blockNode = *newBlockNode(nil, tips, v.phantomK)
	var intersectionNode *blockNode
	for node := v.blockNode.selectedParent; intersectionNode == nil && node != nil; node = node.selectedParent {
		if oldSelectedParent != nil && v.selectedPathSet.contains(node) {
			intersectionNode = node
		} else {
			v.selectedPathSet.add(node)
		}
	}

	if intersectionNode != nil {
		for node := oldSelectedParent; !node.hash.IsEqual(&intersectionNode.hash); node = node.selectedParent {
			v.selectedPathSet.remove(node)
		}
	}
}

// SetTips replaces the tips of the virtual block with the blocks in the
// given blockSet.
//
// This function is safe for concurrent access.
func (v *VirtualBlock) SetTips(tips blockSet) {
	v.mtx.Lock()
	v.setTips(tips)
	v.mtx.Unlock()
}

// addTip adds the given tip to the set of tips in the virtual block.
// All former tips that happen to be the given tips parents are removed
// from the set. This only differs from the exported version in that it
// is up to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for writes).
func (v *VirtualBlock) addTip(newTip *blockNode) {
	updatedTips := v.tips().clone()
	for _, parent := range newTip.parents {
		updatedTips.remove(parent)
	}

	updatedTips.add(newTip)
	v.setTips(updatedTips)
}

// AddTip adds the given tip to the set of tips in the virtual block.
// All former tips that happen to be the given tip's parents are removed
// from the set.
//
// This function is safe for concurrent access.
func (v *VirtualBlock) AddTip(newTip *blockNode) {
	v.mtx.Lock()
	v.addTip(newTip)
	v.mtx.Unlock()
}

// tips returns the current tip block nodes for the DAG.  It will return
// an empty blockSet if there is no tip.
//
// This function is safe for concurrent access.
func (v *VirtualBlock) tips() blockSet {
	return v.parents
}

// SelectedTip returns the current selected tip for the DAG.
// It will return nil if there is no tip.
//
// This function is safe for concurrent access.
func (v *VirtualBlock) SelectedTip() *blockNode {
	return v.selectedParent
}

// GetUTXOEntry returns the requested unspent transaction output. The returned
// instance must be treated as immutable since it is shared by all callers.
//
// This function is safe for concurrent access. However, the returned entry (if
// any) is NOT.
func (v *VirtualBlock) GetUTXOEntry(outPoint wire.OutPoint) (*UTXOEntry, bool) {
	return v.utxoSet.get(outPoint)
}
