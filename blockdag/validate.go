// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/txscript"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/util/daghash"
	"github.com/daglabs/btcd/util/subnetworkid"
	"github.com/daglabs/btcd/wire"
)

const (
	// MaxSigOpsPerBlock is the maximum number of signature operations
	// allowed for a block.  It is a fraction of the max block payload size.
	MaxSigOpsPerBlock = wire.MaxBlockPayload / 50

	// MaxCoinbasePayloadLen is the maximum length a coinbase payload can be.
	MaxCoinbasePayloadLen = 150

	// baseSubsidy is the starting subsidy amount for mined blocks.  This
	// value is halved every SubsidyHalvingInterval blocks.
	baseSubsidy = 50 * util.SatoshiPerBitcoin

	// the following are used when calculating a transaction's mass
	massPerTxSize       = 1
	massPerPKScriptSize = 10
	massPerSigOp        = 10000

	// maxMassPerBlock is the maximum total transaction mass a block may contain.
	maxMassPerBlock = 10000000
)

// isNullOutpoint determines whether or not a previous transaction outpoint
// is set.
func isNullOutpoint(outpoint *wire.Outpoint) bool {
	if outpoint.Index == math.MaxUint32 && outpoint.TxID == daghash.ZeroTxID {
		return true
	}
	return false
}

// SequenceLockActive determines if a transaction's sequence locks have been
// met, meaning that all the inputs of a given transaction have reached a
// blue score or time sufficient for their relative lock-time maturity.
func SequenceLockActive(sequenceLock *SequenceLock, blockBlueScore uint64,
	medianTimePast time.Time) bool {

	// If either the seconds, or blue score relative-lock time has not yet
	// reached, then the transaction is not yet mature according to its
	// sequence locks.
	if sequenceLock.Seconds >= medianTimePast.Unix() ||
		sequenceLock.BlockBlueScore >= int64(blockBlueScore) {
		return false
	}

	return true
}

// IsFinalizedTransaction determines whether or not a transaction is finalized.
func IsFinalizedTransaction(tx *util.Tx, blockBlueScore uint64, blockTime time.Time) bool {
	msgTx := tx.MsgTx()

	// Lock time of zero means the transaction is finalized.
	lockTime := msgTx.LockTime
	if lockTime == 0 {
		return true
	}

	// The lock time field of a transaction is either a block blue score at
	// which the transaction is finalized or a timestamp depending on if the
	// value is before the txscript.LockTimeThreshold.  When it is under the
	// threshold it is a block blue score.
	blockTimeOrBlueScore := int64(0)
	if lockTime < txscript.LockTimeThreshold {
		blockTimeOrBlueScore = int64(blockBlueScore)
	} else {
		blockTimeOrBlueScore = blockTime.Unix()
	}
	if int64(lockTime) < blockTimeOrBlueScore {
		return true
	}

	// At this point, the transaction's lock time hasn't occurred yet, but
	// the transaction might still be finalized if the sequence number
	// for all transaction inputs is maxed out.
	for _, txIn := range msgTx.TxIn {
		if txIn.Sequence != math.MaxUint64 {
			return false
		}
	}
	return true
}

// CalcBlockSubsidy returns the subsidy amount a block at the provided blue score
// should have. This is mainly used for determining how much the coinbase for
// newly generated blocks awards as well as validating the coinbase for blocks
// has the expected value.
//
// The subsidy is halved every SubsidyReductionInterval blocks.  Mathematically
// this is: baseSubsidy / 2^(blueScore/SubsidyReductionInterval)
//
// At the target block generation rate for the main network, this is
// approximately every 4 years.
func CalcBlockSubsidy(blueScore uint64, dagParams *dagconfig.Params) uint64 {
	if dagParams.SubsidyReductionInterval == 0 {
		return baseSubsidy
	}

	// Equivalent to: baseSubsidy / 2^(blueScore/subsidyHalvingInterval)
	return baseSubsidy >> uint(blueScore/dagParams.SubsidyReductionInterval)
}

// CheckTransactionSanity performs some preliminary checks on a transaction to
// ensure it is sane.  These checks are context free.
func CheckTransactionSanity(tx *util.Tx, subnetworkID *subnetworkid.SubnetworkID) error {
	isCoinbase := tx.IsCoinBase()
	// A transaction must have at least one input.
	msgTx := tx.MsgTx()
	if !isCoinbase && len(msgTx.TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

	// A transaction must not exceed the maximum allowed block payload when
	// serialized.
	serializedTxSize := msgTx.SerializeSize()
	if serializedTxSize > wire.MaxBlockPayload {
		str := fmt.Sprintf("serialized transaction is too big - got "+
			"%d, max %d", serializedTxSize, wire.MaxBlockPayload)
		return ruleError(ErrTxTooBig, str)
	}

	// Ensure the transaction amounts are in range.  Each transaction
	// output must not be negative or more than the max allowed per
	// transaction.  Also, the total of all outputs must abide by the same
	// restrictions.  All amounts in a transaction are in a unit value known
	// as a satoshi.  One bitcoin is a quantity of satoshi as defined by the
	// SatoshiPerBitcoin constant.
	var totalSatoshi uint64
	for _, txOut := range msgTx.TxOut {
		satoshi := txOut.Value
		if satoshi > util.MaxSatoshi {
			str := fmt.Sprintf("transaction output value of %d is "+
				"higher than max allowed value of %d", satoshi,
				util.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}

		// Binary arithmetic guarantees that any overflow is detected and reported.
		// This is impossible for Bitcoin, but perhaps possible if an alt increases
		// the total money supply.
		newTotalSatoshi := totalSatoshi + satoshi
		if newTotalSatoshi < totalSatoshi {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs exceeds max allowed value of %d",
				util.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}
		totalSatoshi = newTotalSatoshi
		if totalSatoshi > util.MaxSatoshi {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs is %d which is higher than max "+
				"allowed value of %d", totalSatoshi,
				util.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}
	}

	// Check for duplicate transaction inputs.
	existingTxOut := make(map[wire.Outpoint]struct{})
	for _, txIn := range msgTx.TxIn {
		if _, exists := existingTxOut[txIn.PreviousOutpoint]; exists {
			return ruleError(ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutpoint] = struct{}{}
	}

	// Coinbase payload length must not exceed the max length.
	if isCoinbase {
		payloadLen := len(msgTx.Payload)
		if payloadLen > MaxCoinbasePayloadLen {
			str := fmt.Sprintf("coinbase transaction payload length "+
				"of %d is out of range (max: %d)",
				payloadLen, MaxCoinbasePayloadLen)
			return ruleError(ErrBadCoinbasePayloadLen, str)
		}
	} else {
		// Previous transaction outputs referenced by the inputs to this
		// transaction must not be null.
		for _, txIn := range msgTx.TxIn {
			if isNullOutpoint(&txIn.PreviousOutpoint) {
				return ruleError(ErrBadTxInput, "transaction "+
					"input refers to previous output that "+
					"is null")
			}
		}
	}

	// Check payload's hash
	if !msgTx.SubnetworkID.IsEqual(subnetworkid.SubnetworkIDNative) {
		payloadHash := daghash.DoubleHashH(msgTx.Payload)
		if !msgTx.PayloadHash.IsEqual(&payloadHash) {
			return ruleError(ErrInvalidPayloadHash, "invalid payload hash")
		}
	} else if msgTx.PayloadHash != nil {
		return ruleError(ErrInvalidPayloadHash, "unexpected non-empty payload hash in native subnetwork")
	}

	// Transactions in native, registry and coinbase subnetworks must have Gas = 0
	if (msgTx.SubnetworkID.IsEqual(subnetworkid.SubnetworkIDNative) ||
		msgTx.SubnetworkID.IsBuiltIn()) &&
		msgTx.Gas > 0 {

		return ruleError(ErrInvalidGas, "transaction in the native or "+
			"registry subnetworks has gas > 0 ")
	}

	if msgTx.SubnetworkID.IsEqual(subnetworkid.SubnetworkIDRegistry) {
		err := validateSubnetworkRegistryTransaction(msgTx)
		if err != nil {
			return err
		}
	}

	if msgTx.SubnetworkID.IsEqual(subnetworkid.SubnetworkIDNative) &&
		len(msgTx.Payload) > 0 {

		return ruleError(ErrInvalidPayload,
			"transaction in the native subnetwork includes a payload")
	}

	// If we are a partial node, only transactions on built in subnetworks
	// or our own subnetwork may have a payload
	isLocalNodeFull := subnetworkID == nil
	shouldTxBeFull := msgTx.SubnetworkID.IsBuiltIn() ||
		msgTx.SubnetworkID.IsEqual(subnetworkID)
	if !isLocalNodeFull && !shouldTxBeFull && len(msgTx.Payload) > 0 {
		return ruleError(ErrInvalidPayload,
			"transaction that was expected to be partial has a payload "+
				"with length > 0")
	}

	return nil
}

// checkProofOfWork ensures the block header bits which indicate the target
// difficulty is in min/max range and that the block hash is less than the
// target difficulty as claimed.
//
// The flags modify the behavior of this function as follows:
//  - BFNoPoWCheck: The check to ensure the block hash is less than the target
//    difficulty is not performed.
func (dag *BlockDAG) checkProofOfWork(header *wire.BlockHeader, flags BehaviorFlags) error {
	// The target difficulty must be larger than zero.
	target := util.CompactToBig(header.Bits)
	if target.Sign() <= 0 {
		str := fmt.Sprintf("block target difficulty of %064x is too low",
			target)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

	// The target difficulty must be less than the maximum allowed.
	if target.Cmp(dag.dagParams.PowMax) > 0 {
		str := fmt.Sprintf("block target difficulty of %064x is "+
			"higher than max of %064x", target, dag.dagParams.PowMax)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

	// The block hash must be less than the claimed target unless the flag
	// to avoid proof of work checks is set.
	if flags&BFNoPoWCheck != BFNoPoWCheck {
		// The block hash must be less than the claimed target.
		hash := header.BlockHash()
		hashNum := daghash.HashToBig(hash)
		if hashNum.Cmp(target) > 0 {
			str := fmt.Sprintf("block hash of %064x is higher than "+
				"expected max of %064x", hashNum, target)
			return ruleError(ErrHighHash, str)
		}
	}

	return nil
}

// CountSigOps returns the number of signature operations for all transaction
// input and output scripts in the provided transaction.  This uses the
// quicker, but imprecise, signature operation counting mechanism from
// txscript.
func CountSigOps(tx *util.Tx) int {
	msgTx := tx.MsgTx()

	// Accumulate the number of signature operations in all transaction
	// inputs.
	totalSigOps := 0
	for _, txIn := range msgTx.TxIn {
		numSigOps := txscript.GetSigOpCount(txIn.SignatureScript)
		totalSigOps += numSigOps
	}

	// Accumulate the number of signature operations in all transaction
	// outputs.
	for _, txOut := range msgTx.TxOut {
		numSigOps := txscript.GetSigOpCount(txOut.PkScript)
		totalSigOps += numSigOps
	}

	return totalSigOps
}

// CountP2SHSigOps returns the number of signature operations for all input
// transactions which are of the pay-to-script-hash type.  This uses the
// precise, signature operation counting mechanism from the script engine which
// requires access to the input transaction scripts.
func CountP2SHSigOps(tx *util.Tx, isCoinbase bool, utxoSet UTXOSet) (int, error) {
	// Coinbase transactions have no interesting inputs.
	if isCoinbase {
		return 0, nil
	}

	// Accumulate the number of signature operations in all transaction
	// inputs.
	msgTx := tx.MsgTx()
	totalSigOps := 0
	for txInIndex, txIn := range msgTx.TxIn {
		// Ensure the referenced input transaction is available.
		entry, ok := utxoSet.Get(txIn.PreviousOutpoint)
		if !ok {
			str := fmt.Sprintf("output %s referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutpoint,
				tx.ID(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

		// We're only interested in pay-to-script-hash types, so skip
		// this input if it's not one.
		pkScript := entry.PkScript()
		if !txscript.IsPayToScriptHash(pkScript) {
			continue
		}

		// Count the precise number of signature operations in the
		// referenced public key script.
		sigScript := txIn.SignatureScript
		numSigOps := txscript.GetPreciseSigOpCount(sigScript, pkScript,
			true)

		// We could potentially overflow the accumulator so check for
		// overflow.
		lastSigOps := totalSigOps
		totalSigOps += numSigOps
		if totalSigOps < lastSigOps {
			str := fmt.Sprintf("the public key script from output "+
				"%s contains too many signature operations - "+
				"overflow", txIn.PreviousOutpoint)
			return 0, ruleError(ErrTooManySigOps, str)
		}
	}

	return totalSigOps, nil
}

func ValidateTxMass(tx *util.Tx, utxoSet UTXOSet) error {
	txMass, err := CountTxMass(tx, utxoSet)
	if err != nil {
		return err
	}
	if txMass > maxMassPerBlock {
		str := fmt.Sprintf("tx %s has mass %d, which is above the "+
			"allowed limit of %d", tx.ID(), txMass, maxMassPerBlock)
		return ruleError(ErrMassTooHigh, str)
	}
	return nil
}

func validateBlockMass(pastUTXO UTXOSet, transactions []*util.Tx) error {
	totalMass := uint64(0)
	for _, tx := range transactions {
		txMass, err := CountTxMass(tx, pastUTXO)
		if err != nil {
			return err
		}
		totalMass += txMass
	}
	if totalMass > maxMassPerBlock {
		str := fmt.Sprintf("block has total mass %d, which is "+
			"above the allowed limit of %d", totalMass, maxMassPerBlock)
		return ruleError(ErrMassTooHigh, str)
	}
	return nil
}

func CountTxMass(tx *util.Tx, utxoSet UTXOSet) (uint64, error) {
	txSize := tx.MsgTx().SerializeSize()
	txSizeMass := txSize * massPerTxSize

	if tx.IsCoinBase() {
		return uint64(txSizeMass), nil
	}

	pkScriptSize := 0
	for _, txOut := range tx.MsgTx().TxOut {
		pkScriptSize += len(txOut.PkScript)
	}
	pkScriptSizeMass := pkScriptSize * massPerPKScriptSize

	sigOpsCount := 0
	for txInIndex, txIn := range tx.MsgTx().TxIn {
		// Ensure the referenced input transaction is available.
		entry, ok := utxoSet.Get(txIn.PreviousOutpoint)
		if !ok {
			str := fmt.Sprintf("output %s referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutpoint,
				tx.ID(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

		// Count the precise number of signature operations in the
		// referenced public key script.
		pkScript := entry.PkScript()
		sigScript := txIn.SignatureScript
		isP2SH := txscript.IsPayToScriptHash(pkScript)
		sigOpsCount += txscript.GetPreciseSigOpCount(sigScript, pkScript, isP2SH)
	}
	sigOpsCountMass := sigOpsCount * massPerSigOp

	return uint64(txSizeMass + pkScriptSizeMass + sigOpsCountMass), nil
}

// checkBlockHeaderSanity performs some preliminary checks on a block header to
// ensure it is sane before continuing with processing.  These checks are
// context free.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to checkProofOfWork.
func (dag *BlockDAG) checkBlockHeaderSanity(header *wire.BlockHeader, flags BehaviorFlags) (delay time.Duration, err error) {
	// Ensure the proof of work bits in the block header is in min/max range
	// and the block hash is less than the target value described by the
	// bits.
	err = dag.checkProofOfWork(header, flags)
	if err != nil {
		return 0, err
	}

	if len(header.ParentHashes) == 0 {
		if !header.BlockHash().IsEqual(dag.dagParams.GenesisHash) {
			return 0, ruleError(ErrNoParents, "block has no parents")
		}
	} else {
		err = checkBlockParentsOrder(header)
		if err != nil {
			return 0, err
		}
	}

	// A block timestamp must not have a greater precision than one second.
	// This check is necessary because Go time.Time values support
	// nanosecond precision whereas the consensus rules only apply to
	// seconds and it's much nicer to deal with standard Go time values
	// instead of converting to seconds everywhere.
	if !header.Timestamp.Equal(time.Unix(header.Timestamp.Unix(), 0)) {
		str := fmt.Sprintf("block timestamp of %s has a higher "+
			"precision than one second", header.Timestamp)
		return 0, ruleError(ErrInvalidTime, str)
	}

	// Ensure the block time is not too far in the future. If it's too far, return
	// the duration of time that should be waited before the block becomes valid.
	maxTimestamp := dag.timeSource.AdjustedTime().Add(time.Second *
		time.Duration(int64(dag.TimestampDeviationTolerance)*dag.targetTimePerBlock))
	if header.Timestamp.After(maxTimestamp) {
		return header.Timestamp.Sub(maxTimestamp), nil
	}

	return 0, nil
}

//checkBlockParentsOrder ensures that the block's parents are ordered by hash
func checkBlockParentsOrder(header *wire.BlockHeader) error {
	sortedHashes := make([]*daghash.Hash, 0, header.NumParentBlocks())
	for _, hash := range header.ParentHashes {
		sortedHashes = append(sortedHashes, hash)
	}
	sort.Slice(sortedHashes, func(i, j int) bool {
		return daghash.Less(sortedHashes[i], sortedHashes[j])
	})
	if !daghash.AreEqual(header.ParentHashes, sortedHashes) {
		return ruleError(ErrWrongParentsOrder, "block parents are not ordered by hash")
	}
	return nil
}

// checkBlockSanity performs some preliminary checks on a block to ensure it is
// sane before continuing with block processing.  These checks are context free.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to checkBlockHeaderSanity.
func (dag *BlockDAG) checkBlockSanity(block *util.Block, flags BehaviorFlags) (time.Duration, error) {
	msgBlock := block.MsgBlock()
	header := &msgBlock.Header
	delay, err := dag.checkBlockHeaderSanity(header, flags)
	if err != nil {
		return 0, err
	}

	// A block must have at least one transaction.
	numTx := len(msgBlock.Transactions)
	if numTx == 0 {
		return 0, ruleError(ErrNoTransactions, "block does not contain "+
			"any transactions")
	}

	// A block must not have more transactions than the max block payload or
	// else it is certainly over the block size limit.
	if numTx > wire.MaxBlockPayload {
		str := fmt.Sprintf("block contains too many transactions - "+
			"got %d, max %d", numTx, wire.MaxBlockPayload)
		return 0, ruleError(ErrBlockTooBig, str)
	}

	// A block must not exceed the maximum allowed block payload when
	// serialized.
	serializedSize := msgBlock.SerializeSize()
	if serializedSize > wire.MaxBlockPayload {
		str := fmt.Sprintf("serialized block is too big - got %d, "+
			"max %d", serializedSize, wire.MaxBlockPayload)
		return 0, ruleError(ErrBlockTooBig, str)
	}

	// The first transaction in a block must be a coinbase.
	transactions := block.Transactions()
	if !transactions[util.CoinbaseTransactionIndex].IsCoinBase() {
		return 0, ruleError(ErrFirstTxNotCoinbase, "first transaction in "+
			"block is not a coinbase")
	}

	txOffset := util.CoinbaseTransactionIndex + 1

	// A block must not have more than one coinbase. And transactions must be
	// ordered by subnetwork
	for i, tx := range transactions[txOffset:] {
		if tx.IsCoinBase() {
			str := fmt.Sprintf("block contains second coinbase at "+
				"index %d", i+2)
			return 0, ruleError(ErrMultipleCoinbases, str)
		}
		if i != 0 && subnetworkid.Less(&tx.MsgTx().SubnetworkID, &transactions[i].MsgTx().SubnetworkID) {
			return 0, ruleError(ErrTransactionsNotSorted, "transactions must be sorted by subnetwork")
		}
	}

	// Do some preliminary checks on each transaction to ensure they are
	// sane before continuing.
	for _, tx := range transactions {
		err := CheckTransactionSanity(tx, dag.subnetworkID)
		if err != nil {
			return 0, err
		}
	}

	// Build merkle tree and ensure the calculated merkle root matches the
	// entry in the block header.  This also has the effect of caching all
	// of the transaction hashes in the block to speed up future hash
	// checks.  Bitcoind builds the tree here and checks the merkle root
	// after the following checks, but there is no reason not to check the
	// merkle root matches here.
	hashMerkleTree := BuildHashMerkleTreeStore(block.Transactions())
	calculatedHashMerkleRoot := hashMerkleTree.Root()
	if !header.HashMerkleRoot.IsEqual(calculatedHashMerkleRoot) {
		str := fmt.Sprintf("block hash merkle root is invalid - block "+
			"header indicates %s, but calculated value is %s",
			header.HashMerkleRoot, calculatedHashMerkleRoot)
		return 0, ruleError(ErrBadMerkleRoot, str)
	}

	// Check for duplicate transactions.  This check will be fairly quick
	// since the transaction IDs are already cached due to building the
	// merkle tree above.
	existingTxIDs := make(map[daghash.TxID]struct{})
	for _, tx := range transactions {
		id := tx.ID()
		if _, exists := existingTxIDs[*id]; exists {
			str := fmt.Sprintf("block contains duplicate "+
				"transaction %s", id)
			return 0, ruleError(ErrDuplicateTx, str)
		}
		existingTxIDs[*id] = struct{}{}
	}

	// The number of signature operations must be less than the maximum
	// allowed per block.
	totalSigOps := 0
	for _, tx := range transactions {
		// We could potentially overflow the accumulator so check for
		// overflow.
		lastSigOps := totalSigOps
		totalSigOps += CountSigOps(tx)
		if totalSigOps < lastSigOps || totalSigOps > MaxSigOpsPerBlock {
			str := fmt.Sprintf("block contains too many signature "+
				"operations - got %d, max %d", totalSigOps,
				MaxSigOpsPerBlock)
			return 0, ruleError(ErrTooManySigOps, str)
		}
	}

	return delay, nil
}

// checkBlockHeaderContext performs several validation checks on the block header
// which depend on its position within the block dag.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: All checks except those involving comparing the header against
//    the checkpoints are not performed.
//
// This function MUST be called with the dag state lock held (for writes).
func (dag *BlockDAG) checkBlockHeaderContext(header *wire.BlockHeader, bluestParent *blockNode, blockChainHeight uint64, fastAdd bool) error {
	if !fastAdd {
		if err := dag.validateDifficulty(header, bluestParent); err != nil {
			return err
		}

		if err := validateMedianTime(dag, header, bluestParent); err != nil {
			return err
		}
	}

	return dag.validateCheckpoints(header, blockChainHeight)
}

func (dag *BlockDAG) validateCheckpoints(header *wire.BlockHeader, blockChainHeight uint64) error {
	// Ensure DAG matches up to predetermined checkpoints.
	blockHash := header.BlockHash()
	if !dag.verifyCheckpoint(blockChainHeight, blockHash) {
		str := fmt.Sprintf("block at chain height %d does not match "+
			"checkpoint hash", blockChainHeight)
		return ruleError(ErrBadCheckpoint, str)
	}

	return nil
}

func validateMedianTime(dag *BlockDAG, header *wire.BlockHeader, bluestParent *blockNode) error {
	if !header.IsGenesis() {
		// Ensure the timestamp for the block header is not before the
		// median time of the last several blocks (medianTimeBlocks).
		medianTime := bluestParent.PastMedianTime(dag)
		if header.Timestamp.Before(medianTime) {
			str := fmt.Sprintf("block timestamp of %s is not after expected %s", header.Timestamp, medianTime)
			return ruleError(ErrTimeTooOld, str)
		}
	}

	return nil
}

func (dag *BlockDAG) validateDifficulty(header *wire.BlockHeader, bluestParent *blockNode) error {
	// Ensure the difficulty specified in the block header matches
	// the calculated difficulty based on the previous block and
	// difficulty retarget rules.
	expectedDifficulty := dag.requiredDifficulty(bluestParent,
		header.Timestamp)
	blockDifficulty := header.Bits
	if blockDifficulty != expectedDifficulty {
		str := fmt.Sprintf("block difficulty of %d is not the expected value of %d", blockDifficulty, expectedDifficulty)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

	return nil
}

// validateParents validates that no parent is an ancestor of another parent, and no parent is finalized
func validateParents(blockHeader *wire.BlockHeader, parents blockSet) error {
	minHeight := uint64(math.MaxUint64)
	queue := newDownHeap()
	visited := newSet()
	for _, parent := range parents {
		// isFinalized might be false-negative because node finality status is
		// updated in a separate goroutine. This is why later the block is
		// checked more thoroughly on the finality rules in dag.checkFinalityRules.
		if parent.isFinalized {
			return ruleError(ErrFinality, fmt.Sprintf("block %s is a finalized parent of block %s", parent.hash, blockHeader.BlockHash()))
		}
		if parent.height < minHeight {
			minHeight = parent.height
		}
		for _, grandParent := range parent.parents {
			if !visited.contains(grandParent) {
				queue.Push(grandParent)
				visited.add(grandParent)
			}
		}
	}
	for queue.Len() > 0 {
		current := queue.pop()
		if parents.contains(current) {
			return ruleError(ErrInvalidParentsRelation, fmt.Sprintf("block %s is both a parent of %s and an"+
				" ancestor of another parent",
				current.hash,
				blockHeader.BlockHash()))
		}
		if current.height > minHeight {
			for _, parent := range current.parents {
				if !visited.contains(parent) {
					queue.Push(parent)
					visited.add(parent)
				}
			}
		}
	}
	return nil
}

// checkBlockContext peforms several validation checks on the block which depend
// on its position within the block DAG.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: The transaction are not checked to see if they are finalized
//    and the somewhat expensive BIP0034 validation is not performed.
//
// The flags are also passed to checkBlockHeaderContext.  See its documentation
// for how the flags modify its behavior.
//
// This function MUST be called with the dag state lock held (for writes).
func (dag *BlockDAG) checkBlockContext(block *util.Block, parents blockSet, flags BehaviorFlags) error {
	bluestParent := parents.bluest()
	fastAdd := flags&BFFastAdd == BFFastAdd

	err := validateParents(&block.MsgBlock().Header, parents)
	if err != nil {
		return err
	}

	// Perform all block header related validation checks.
	header := &block.MsgBlock().Header
	if err = dag.checkBlockHeaderContext(header, bluestParent, block.ChainHeight(), fastAdd); err != nil {
		return err
	}

	return nil
}

func (dag *BlockDAG) validateAllTxsFinalized(block *util.Block, node *blockNode, bluestParent *blockNode) error {
	blockTime := block.MsgBlock().Header.Timestamp
	if !block.IsGenesis() {
		blockTime = bluestParent.PastMedianTime(dag)
	}

	// Ensure all transactions in the block are finalized.
	for _, tx := range block.Transactions() {
		if !IsFinalizedTransaction(tx, node.blueScore, blockTime) {
			str := fmt.Sprintf("block contains unfinalized "+
				"transaction %s", tx.ID())
			return ruleError(ErrUnfinalizedTx, str)
		}
	}

	return nil
}

// ensureNoDuplicateTx ensures blocks do not contain duplicate transactions which
// 'overwrite' older transactions that are not fully spent.  This prevents an
// attack where a coinbase and all of its dependent transactions could be
// duplicated to effectively revert the overwritten transactions to a single
// confirmation thereby making them vulnerable to a double spend.
//
// For more details, see
// https://github.com/bitcoin/bips/blob/master/bip-0030.mediawiki and
// http://r6.ca/blog/20120206T005236Z.html.
//
// This function MUST be called with the dag state lock held (for reads).
func ensureNoDuplicateTx(utxoSet UTXOSet, transactions []*util.Tx) error {
	// Fetch utxos for all of the transaction ouputs in this block.
	// Typically, there will not be any utxos for any of the outputs.
	fetchSet := make(map[wire.Outpoint]struct{})
	for _, tx := range transactions {
		prevOut := wire.Outpoint{TxID: *tx.ID()}
		for txOutIdx := range tx.MsgTx().TxOut {
			prevOut.Index = uint32(txOutIdx)
			fetchSet[prevOut] = struct{}{}
		}
	}

	// Duplicate transactions are only allowed if the previous transaction
	// is fully spent.
	for outpoint := range fetchSet {
		if _, ok := utxoSet.Get(outpoint); ok {
			str := fmt.Sprintf("tried to overwrite transaction %s "+
				"that is not fully spent", outpoint.TxID)
			return ruleError(ErrOverwriteTx, str)
		}
	}

	return nil
}

// CheckTransactionInputsAndCalulateFee performs a series of checks on the inputs to a
// transaction to ensure they are valid.  An example of some of the checks
// include verifying all inputs exist, ensuring the block reward seasoning
// requirements are met, detecting double spends, validating all values and fees
// are in the legal range and the total output amount doesn't exceed the input
// amount.  As it checks the inputs, it also calculates the total fees for the
// transaction and returns that value.
//
// NOTE: The transaction MUST have already been sanity checked with the
// CheckTransactionSanity function prior to calling this function.
func CheckTransactionInputsAndCalulateFee(tx *util.Tx, txBlueScore uint64, utxoSet UTXOSet, dagParams *dagconfig.Params, fastAdd bool) (
	txFeeInSatoshi uint64, err error) {

	// Coinbase transactions have no standard inputs to validate.
	if tx.IsCoinBase() {
		return 0, nil
	}

	txID := tx.ID()
	var totalSatoshiIn uint64
	for txInIndex, txIn := range tx.MsgTx().TxIn {
		// Ensure the referenced input transaction is available.
		entry, ok := utxoSet.Get(txIn.PreviousOutpoint)
		if !ok {
			str := fmt.Sprintf("output %s referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutpoint,
				tx.ID(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

		if !fastAdd {
			if err = validateCoinbaseMaturity(dagParams, entry, txBlueScore, txIn); err != nil {
				return 0, err
			}
		}

		// Ensure the transaction amounts are in range.  Each of the
		// output values of the input transactions must not be negative
		// or more than the max allowed per transaction.  All amounts in
		// a transaction are in a unit value known as a satoshi.  One
		// bitcoin is a quantity of satoshi as defined by the
		// SatoshiPerBitcoin constant.
		originTxSatoshi := entry.Amount()
		if originTxSatoshi > util.MaxSatoshi {
			str := fmt.Sprintf("transaction output value of %s is "+
				"higher than max allowed value of %d",
				util.Amount(originTxSatoshi),
				util.MaxSatoshi)
			return 0, ruleError(ErrBadTxOutValue, str)
		}

		// The total of all outputs must not be more than the max
		// allowed per transaction.  Also, we could potentially overflow
		// the accumulator so check for overflow.
		lastSatoshiIn := totalSatoshiIn
		totalSatoshiIn += originTxSatoshi
		if totalSatoshiIn < lastSatoshiIn ||
			totalSatoshiIn > util.MaxSatoshi {
			str := fmt.Sprintf("total value of all transaction "+
				"inputs is %d which is higher than max "+
				"allowed value of %d", totalSatoshiIn,
				util.MaxSatoshi)
			return 0, ruleError(ErrBadTxOutValue, str)
		}
	}

	// Calculate the total output amount for this transaction.  It is safe
	// to ignore overflow and out of range errors here because those error
	// conditions would have already been caught by checkTransactionSanity.
	var totalSatoshiOut uint64
	for _, txOut := range tx.MsgTx().TxOut {
		totalSatoshiOut += txOut.Value
	}

	// Ensure the transaction does not spend more than its inputs.
	if totalSatoshiIn < totalSatoshiOut {
		str := fmt.Sprintf("total value of all transaction inputs for "+
			"transaction %s is %d which is less than the amount "+
			"spent of %d", txID, totalSatoshiIn, totalSatoshiOut)
		return 0, ruleError(ErrSpendTooHigh, str)
	}

	// NOTE: bitcoind checks if the transaction fees are < 0 here, but that
	// is an impossible condition because of the check above that ensures
	// the inputs are >= the outputs.
	txFeeInSatoshi = totalSatoshiIn - totalSatoshiOut
	return txFeeInSatoshi, nil
}

func validateCoinbaseMaturity(dagParams *dagconfig.Params, entry *UTXOEntry, txBlueScore uint64, txIn *wire.TxIn) error {
	// Ensure the transaction is not spending coins which have not
	// yet reached the required coinbase maturity.
	if entry.IsCoinbase() {
		originBlueScore := entry.BlockBlueScore()
		blueScoreSincePrev := txBlueScore - originBlueScore
		if blueScoreSincePrev < dagParams.BlockCoinbaseMaturity {
			str := fmt.Sprintf("tried to spend coinbase "+
				"transaction output %s from blue score %d "+
				"to blue score %d before required maturity "+
				"of %d", txIn.PreviousOutpoint,
				originBlueScore, txBlueScore,
				dagParams.BlockCoinbaseMaturity)
			return ruleError(ErrImmatureSpend, str)
		}
	}
	return nil
}

// checkConnectToPastUTXO performs several checks to confirm connecting the passed
// block to the DAG represented by the passed view does not violate any rules.
//
// An example of some of the checks performed are ensuring connecting the block
// would not cause any duplicate transaction hashes for old transactions that
// aren't already fully spent, double spends, exceeding the maximum allowed
// signature operations per block, invalid values in relation to the expected
// block subsidy, or fail transaction script validation.
//
// It also returns the feeAccumulator for this block.
//
// This function MUST be called with the dag state lock held (for writes).
func (dag *BlockDAG) checkConnectToPastUTXO(block *blockNode, pastUTXO UTXOSet,
	transactions []*util.Tx, fastAdd bool) (compactFeeData, error) {

	if !fastAdd {
		err := ensureNoDuplicateTx(pastUTXO, transactions)
		if err != nil {
			return nil, err
		}

		if err := validateSigopsCount(pastUTXO, transactions); err != nil {
			return nil, err
		}

		if err := validateBlockMass(pastUTXO, transactions); err != nil {
			return nil, err
		}
	}

	// Perform several checks on the inputs for each transaction.  Also
	// accumulate the total fees.  This could technically be combined with
	// the loop above instead of running another loop over the transactions,
	// but by separating it we can avoid running the more expensive (though
	// still relatively cheap as compared to running the scripts) checks
	// against all the inputs when the signature operations are out of
	// bounds.
	// In addition - add all fees into a fee accumulator, to be stored and checked
	// when validating descendants' coinbase transactions.
	var totalFees uint64
	compactFeeFactory := newCompactFeeFactory()

	for _, tx := range transactions {
		txFee, err := CheckTransactionInputsAndCalulateFee(tx, block.blueScore, pastUTXO,
			dag.dagParams, fastAdd)
		if err != nil {
			return nil, err
		}

		// Sum the total fees and ensure we don't overflow the
		// accumulator.
		lastTotalFees := totalFees
		totalFees += txFee
		if totalFees < lastTotalFees {
			return nil, ruleError(ErrBadFees, "total fees for block "+
				"overflows accumulator")
		}

		err = compactFeeFactory.add(txFee)
		if err != nil {
			return nil, fmt.Errorf("error adding tx %s fee to compactFeeFactory: %s", tx.ID(), err)
		}
	}
	feeData, err := compactFeeFactory.data()
	if err != nil {
		return nil, fmt.Errorf("error getting bytes of fee data: %s", err)
	}

	if !fastAdd {

		// Don't run scripts if this node is before the latest known good
		// checkpoint since the validity is verified via the checkpoints (all
		// transactions are included in the merkle root hash and any changes
		// will therefore be detected by the next checkpoint).  This is a huge
		// optimization because running the scripts is the most time consuming
		// portion of block handling.
		checkpoint := dag.LatestCheckpoint()
		runScripts := true
		if checkpoint != nil && block.chainHeight <= checkpoint.ChainHeight {
			runScripts = false
		}

		scriptFlags := txscript.ScriptNoFlags

		// We obtain the MTP of the *previous* block (unless it's genesis block)
		// in order to determine if transactions in the current block are final.
		medianTime := block.Header().Timestamp
		if !block.isGenesis() {
			medianTime = block.selectedParent.PastMedianTime(dag)
		}

		// We also enforce the relative sequence number based
		// lock-times within the inputs of all transactions in this
		// candidate block.
		for _, tx := range transactions {
			// A transaction can only be included within a block
			// once the sequence locks of *all* its inputs are
			// active.
			sequenceLock, err := dag.calcSequenceLock(block, pastUTXO, tx, false)
			if err != nil {
				return nil, err
			}
			if !SequenceLockActive(sequenceLock, block.blueScore,
				medianTime) {
				str := fmt.Sprintf("block contains " +
					"transaction whose input sequence " +
					"locks are not met")
				return nil, ruleError(ErrUnfinalizedTx, str)
			}
		}

		// Now that the inexpensive checks are done and have passed, verify the
		// transactions are actually allowed to spend the coins by running the
		// expensive ECDSA signature check scripts.  Doing this last helps
		// prevent CPU exhaustion attacks.
		if runScripts {
			err := checkBlockScripts(block, pastUTXO, transactions, scriptFlags, dag.sigCache)
			if err != nil {
				return nil, err
			}
		}

	}
	return feeData, nil
}

func validateSigopsCount(pastUTXO UTXOSet, transactions []*util.Tx) error {
	// The number of signature operations must be less than the maximum
	// allowed per block.  Note that the preliminary sanity checks on a
	// block also include a check similar to this one, but this check
	// expands the count to include a precise count of pay-to-script-hash
	// signature operations in each of the input transaction public key
	// scripts.
	totalSigOps := 0
	for i, tx := range transactions {
		numsigOps := CountSigOps(tx)
		// Since the first transaction has already been verified to be a
		// coinbase transaction, use i != util.CoinbaseTransactionIndex
		// as an optimization for the flag to countP2SHSigOps for whether
		// or not the transaction is a coinbase transaction rather than
		// having to do a full coinbase check again.
		numP2SHSigOps, err := CountP2SHSigOps(tx, i == util.CoinbaseTransactionIndex, pastUTXO)
		if err != nil {
			return err
		}
		numsigOps += numP2SHSigOps

		// Check for overflow or going over the limits.  We have to do
		// this on every loop iteration to avoid overflow.
		lastSigops := totalSigOps
		totalSigOps += numsigOps
		if totalSigOps < lastSigops || totalSigOps > MaxSigOpsPerBlock {
			str := fmt.Sprintf("block contains too many "+
				"signature operations - got %d, max %d",
				totalSigOps, MaxSigOpsPerBlock)
			return ruleError(ErrTooManySigOps, str)
		}
	}

	return nil
}

// CheckConnectBlockTemplate fully validates that connecting the passed block to
// the DAG does not violate any consensus rules, aside from the proof of
// work requirement.
//
// This function is safe for concurrent access.
func (dag *BlockDAG) CheckConnectBlockTemplate(block *util.Block) error {
	dag.dagLock.RLock()
	defer dag.dagLock.RUnlock()
	return dag.CheckConnectBlockTemplateNoLock(block)
}

// CheckConnectBlockTemplateNoLock fully validates that connecting the passed block to
// the DAG does not violate any consensus rules, aside from the proof of
// work requirement. The block must connect to the current tip of the main dag.
func (dag *BlockDAG) CheckConnectBlockTemplateNoLock(block *util.Block) error {

	// Skip the proof of work check as this is just a block template.
	flags := BFNoPoWCheck

	header := block.MsgBlock().Header

	delay, err := dag.checkBlockSanity(block, flags)
	if err != nil {
		return err
	}

	if delay != 0 {
		return fmt.Errorf("Block timestamp is too far in the future")
	}

	parents, err := lookupParentNodes(block, dag)
	if err != nil {
		return err
	}

	err = dag.checkBlockContext(block, parents, flags)
	if err != nil {
		return err
	}

	templateNode := newBlockNode(&header, dag.virtual.tips(), dag.dagParams.K)

	_, err = dag.checkConnectToPastUTXO(templateNode,
		dag.UTXOSet(), block.Transactions(), false)

	return err
}
