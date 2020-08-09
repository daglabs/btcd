// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"
	"time"

	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
)

// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block DAG. It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block DAG.
//
// When no errors occurred during processing, the first return value indicates
// whether or not the block is an orphan.
//
// This function is safe for concurrent access.
func (dag *BlockDAG) ProcessBlock(block *util.Block, flags BehaviorFlags) (isOrphan bool, isDelayed bool, err error) {
	dag.dagLock.Lock()
	defer dag.dagLock.Unlock()
	return dag.processBlockNoLock(block, flags)
}

func (dag *BlockDAG) processBlockNoLock(block *util.Block, flags BehaviorFlags) (isOrphan bool, isDelayed bool, err error) {
	isAfterDelay := flags&BFAfterDelay == BFAfterDelay
	wasBlockStored := flags&BFWasStored == BFWasStored
	disallowDelay := flags&BFDisallowDelay == BFDisallowDelay
	disallowOrphans := flags&BFDisallowOrphans == BFDisallowOrphans

	blockHash := block.Hash()
	log.Tracef("Processing block %s", blockHash)

	// The block must not already exist in the DAG.
	if dag.IsInDAG(blockHash) && !wasBlockStored {
		str := fmt.Sprintf("already have block %s", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	// The block must not already exist as an orphan.
	if _, exists := dag.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %s", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	if dag.isKnownDelayedBlock(blockHash) {
		str := fmt.Sprintf("already have block (delayed) %s", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	if !isAfterDelay {
		// Perform preliminary sanity checks on the block and its transactions.
		delay, err := dag.checkBlockSanity(block, flags)
		if err != nil {
			return false, false, err
		}

		if delay != 0 && disallowDelay {
			str := fmt.Sprintf("Cannot process blocks beyond the allowed time offset while the BFDisallowDelay flag is raised %s", blockHash)
			return false, true, ruleError(ErrDelayedBlockIsNotAllowed, str)
		}

		if delay != 0 {
			err = dag.addDelayedBlock(block, delay)
			if err != nil {
				return false, false, err
			}
			return false, true, nil
		}
	}

	var missingParents []*daghash.Hash
	for _, parentHash := range block.MsgBlock().Header.ParentHashes {
		if !dag.IsInDAG(parentHash) {
			missingParents = append(missingParents, parentHash)
		}
	}
	if len(missingParents) > 0 && disallowOrphans {
		str := fmt.Sprintf("Cannot process orphan blocks while the BFDisallowOrphans flag is raised %s", blockHash)
		return false, false, ruleError(ErrOrphanBlockIsNotAllowed, str)
	}

	// Handle the case of a block with a valid timestamp(non-delayed) which points to a delayed block.
	delay, isParentDelayed := dag.maxDelayOfParents(missingParents)
	if isParentDelayed {
		// Add Millisecond to ensure that parent process time will be after its child.
		delay += time.Millisecond
		err := dag.addDelayedBlock(block, delay)
		if err != nil {
			return false, false, err
		}
		return false, true, err
	}

	// Handle orphan blocks.
	if len(missingParents) > 0 {
		// Some orphans during netsync are a normal part of the process, since the anticone
		// of the chain-split is never explicitly requested.
		// Therefore, if we are during netsync - don't report orphans to default logs.
		//
		// The number K*2 was chosen since in peace times anticone is limited to K blocks,
		// while some red block can make it a bit bigger, but much more than that indicates
		// there might be some problem with the netsync process.
		if flags&BFIsSync == BFIsSync && dagconfig.KType(len(dag.orphans)) < dag.Params.K*2 {
			log.Debugf("Adding orphan block %s. This is normal part of netsync process", blockHash)
		} else {
			log.Infof("Adding orphan block %s", blockHash)
		}
		dag.addOrphanBlock(block)

		return true, false, nil
	}

	// The block has passed all context independent checks and appears sane
	// enough to potentially accept it into the block DAG.
	err = dag.maybeAcceptBlock(block, flags)
	if err != nil {
		return false, false, err
	}

	// Accept any orphan blocks that depend on this block (they are
	// no longer orphans) and repeat for those accepted blocks until
	// there are no more.
	err = dag.processOrphans(blockHash, flags)
	if err != nil {
		return false, false, err
	}

	if !isAfterDelay {
		err = dag.processDelayedBlocks()
		if err != nil {
			return false, false, err
		}
	}

	dag.addBlockProcessingTimestamp()

	log.Debugf("Accepted block %s", blockHash)

	return false, false, nil
}
