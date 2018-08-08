// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"sync"
)

// virtualBlock is a virtual block whose parents are the tip of the DAG.
type virtualBlock struct {
	mtx      sync.Mutex
	phantomK uint32
	blockNode
}

// newVirtualBlock creates and returns a new virtualBlock.
func newVirtualBlock(tips blockSet, phantomK uint32) *virtualBlock {
	// The mutex is intentionally not held since this is a constructor.
	var virtual virtualBlock
	virtual.phantomK = phantomK
	if tips != nil {
		virtual.setTips(tips)
	}

	return &virtual
}

func (v *virtualBlock) setTips(tips blockSet) {
	v.blockNode = *newBlockNode(nil, tips, v.phantomK)
}

func (v *virtualBlock) SetTips(tips blockSet) {
	v.mtx.Lock()
	v.setTips(tips)
	v.mtx.Unlock()
}

// Tips returns the current tip block nodes for the chain view.  It will return
// an empty slice if there is no tip.
//
// This function is safe for concurrent access.
func (v *virtualBlock) Tips() blockSet {
	v.mtx.Lock()
	defer func() {
		v.mtx.Unlock()
	}()

	return v.parents
}

// SelectedTip returns the current selected tip for the DAG.
// It will return nil if there is no tip.
//
// This function is safe for concurrent access.
func (v *virtualBlock) SelectedTip() *blockNode {
	return v.selectedParent
}