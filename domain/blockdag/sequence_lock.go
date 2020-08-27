package blockdag

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/util"
)

// SequenceLock represents the converted relative lock-time in seconds, and
// absolute block-blue-score for a transaction input's relative lock-times.
// According to SequenceLock, after the referenced input has been confirmed
// within a block, a transaction spending that input can be included into a
// block either after 'seconds' (according to past median time), or once the
// 'BlockBlueScore' has been reached.
type SequenceLock struct {
	Milliseconds   int64
	BlockBlueScore int64
}

// CalcSequenceLock computes a relative lock-time SequenceLock for the passed
// transaction using the passed UTXOSet to obtain the past median time
// for blocks in which the referenced inputs of the transactions were included
// within. The generated SequenceLock lock can be used in conjunction with a
// block height, and adjusted median block time to determine if all the inputs
// referenced within a transaction have reached sufficient maturity allowing
// the candidate transaction to be included in a block.
//
// This function is safe for concurrent access.
func (dag *BlockDAG) CalcSequenceLock(tx *util.Tx, utxoSet UTXOSet) (*SequenceLock, error) {
	dag.dagLock.RLock()
	defer dag.dagLock.RUnlock()

	return dag.calcTxSequenceLock(dag.selectedTip(), tx, utxoSet)
}

// CalcSequenceLockNoLock is lock free version of CalcSequenceLockWithLock
// This function is unsafe for concurrent access.
func (dag *BlockDAG) CalcSequenceLockNoLock(tx *util.Tx, utxoSet UTXOSet) (*SequenceLock, error) {
	return dag.calcTxSequenceLock(dag.selectedTip(), tx, utxoSet)
}

// calcTxSequenceLock computes the relative lock-times for the passed
// transaction. See the exported version, CalcSequenceLock for further details.
//
// This function MUST be called with the DAG state lock held (for writes).
func (dag *BlockDAG) calcTxSequenceLock(node *blockNode, tx *util.Tx, utxoSet UTXOSet) (*SequenceLock, error) {
	inputsWithReferencedUTXOEntries, err := dag.GetReferencedUTXOEntries(tx, utxoSet)
	if err != nil {
		return nil, err
	}

	return dag.calcTxSequenceLockFromInputsWithReferencedEntries(node, tx, inputsWithReferencedUTXOEntries)
}

func (dag *BlockDAG) calcTxSequenceLockFromInputsWithReferencedEntries(
	node *blockNode, tx *util.Tx, inputsWithReferencedUTXOEntries []*txInputAndReferencedUTXOEntry) (*SequenceLock, error) {

	// A value of -1 for each relative lock type represents a relative time
	// lock value that will allow a transaction to be included in a block
	// at any given height or time.
	sequenceLock := &SequenceLock{Milliseconds: -1, BlockBlueScore: -1}

	// Sequence locks don't apply to coinbase transactions Therefore, we
	// return sequence lock values of -1 indicating that this transaction
	// can be included within a block at any given height or time.
	if tx.IsCoinBase() {
		return sequenceLock, nil
	}

	for _, txInAndReferencedUTXOEntry := range inputsWithReferencedUTXOEntries {
		txIn := txInAndReferencedUTXOEntry.txIn
		utxoEntry := txInAndReferencedUTXOEntry.utxoEntry

		// If the input blue score is set to the mempool blue score, then we
		// assume the transaction makes it into the next block when
		// evaluating its sequence blocks.
		inputBlueScore := utxoEntry.BlockBlueScore()
		if utxoEntry.IsUnaccepted() {
			inputBlueScore = dag.virtual.blueScore
		}

		// Given a sequence number, we apply the relative time lock
		// mask in order to obtain the time lock delta required before
		// this input can be spent.
		sequenceNum := txIn.Sequence
		relativeLock := int64(sequenceNum & appmessage.SequenceLockTimeMask)

		switch {
		// Relative time locks are disabled for this input, so we can
		// skip any further calculation.
		case sequenceNum&appmessage.SequenceLockTimeDisabled == appmessage.SequenceLockTimeDisabled:
			continue
		case sequenceNum&appmessage.SequenceLockTimeIsSeconds == appmessage.SequenceLockTimeIsSeconds:
			// This input requires a relative time lock expressed
			// in seconds before it can be spent. Therefore, we
			// need to query for the block prior to the one in
			// which this input was accepted within so we can
			// compute the past median time for the block prior to
			// the one which accepted this referenced output.
			blockNode := node
			for blockNode.selectedParent.blueScore > inputBlueScore {
				blockNode = blockNode.selectedParent
			}
			medianTime := blockNode.PastMedianTime()

			// Time based relative time-locks have a time granularity of
			// appmessage.SequenceLockTimeGranularity, so we shift left by this
			// amount to convert to the proper relative time-lock. We also
			// subtract one from the relative lock to maintain the original
			// lockTime semantics.
			timeLockMilliseconds := (relativeLock << appmessage.SequenceLockTimeGranularity) - 1
			timeLock := medianTime.UnixMilliseconds() + timeLockMilliseconds
			if timeLock > sequenceLock.Milliseconds {
				sequenceLock.Milliseconds = timeLock
			}
		default:
			// The relative lock-time for this input is expressed
			// in blocks so we calculate the relative offset from
			// the input's blue score as its converted absolute
			// lock-time. We subtract one from the relative lock in
			// order to maintain the original lockTime semantics.
			blockBlueScore := int64(inputBlueScore) + relativeLock - 1
			if blockBlueScore > sequenceLock.BlockBlueScore {
				sequenceLock.BlockBlueScore = blockBlueScore
			}
		}
	}

	return sequenceLock, nil
}

// LockTimeToSequence converts the passed relative locktime to a sequence
// number.
func LockTimeToSequence(isMilliseconds bool, locktime uint64) uint64 {
	// If we're expressing the relative lock time in blocks, then the
	// corresponding sequence number is simply the desired input age.
	if !isMilliseconds {
		return locktime
	}

	// Set the 22nd bit which indicates the lock time is in milliseconds, then
	// shift the locktime over by 19 since the time granularity is in
	// 524288-millisecond intervals (2^19). This results in a max lock-time of
	// 34,359,214,080 seconds, or 1.1 years.
	return appmessage.SequenceLockTimeIsSeconds |
		locktime>>appmessage.SequenceLockTimeGranularity
}
