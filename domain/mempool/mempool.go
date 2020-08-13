// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mempool

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/mining"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/infrastructure/logger"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
)

const (
	// orphanTTL is the maximum amount of time an orphan is allowed to
	// stay in the orphan pool before it expires and is evicted during the
	// next scan.
	orphanTTL = time.Minute * 15

	// orphanExpireScanInterval is the minimum amount of time in between
	// scans of the orphan pool to evict expired transactions.
	orphanExpireScanInterval = time.Minute * 5
)

// NewBlockMsg is the type that is used in NewBlockMsg to transfer
// data about transaction removed and added to the mempool
type NewBlockMsg struct {
	AcceptedTxs []*TxDesc
	Tx          *util.Tx
}

// Config is a descriptor containing the memory pool configuration.
type Config struct {
	// Policy defines the various mempool configuration options related
	// to policy.
	Policy Policy

	// CalcSequenceLockNoLock defines the function to use in order to generate
	// the current sequence lock for the given transaction using the passed
	// utxo set.
	CalcSequenceLockNoLock func(*util.Tx, blockdag.UTXOSet) (*blockdag.SequenceLock, error)

	// SigCache defines a signature cache to use.
	SigCache *txscript.SigCache

	// DAG is the BlockDAG we want to use (mainly for UTXO checks)
	DAG *blockdag.BlockDAG
}

// Policy houses the policy (configuration parameters) which is used to
// control the mempool.
type Policy struct {
	// MaxTxVersion is the transaction version that the mempool should
	// accept. All transactions above this version are rejected as
	// non-standard.
	MaxTxVersion int32

	// AcceptNonStd defines whether to accept non-standard transactions. If
	// true, non-standard transactions will be accepted into the mempool.
	// Otherwise, all non-standard transactions will be rejected.
	AcceptNonStd bool

	// MaxOrphanTxs is the maximum number of orphan transactions
	// that can be queued.
	MaxOrphanTxs int

	// MaxOrphanTxSize is the maximum size allowed for orphan transactions.
	// This helps prevent memory exhaustion attacks from sending a lot of
	// of big orphans.
	MaxOrphanTxSize int

	// MinRelayTxFee defines the minimum transaction fee in KAS/kB to be
	// considered a non-zero fee.
	MinRelayTxFee util.Amount
}

// TxDesc is a descriptor containing a transaction in the mempool along with
// additional metadata.
type TxDesc struct {
	mining.TxDesc

	// depCount is not 0 for dependent transaction. Dependent transaction is
	// one that is accepted to pool, but cannot be mined in next block because it
	// depends on outputs of accepted, but still not mined transaction
	depCount int
}

// orphanTx is normal transaction that references an ancestor transaction
// that is not yet available. It also contains additional information related
// to it such as an expiration time to help prevent caching the orphan forever.
type orphanTx struct {
	tx         *util.Tx
	expiration mstime.Time
}

// TxPool is used as a source of transactions that need to be mined into blocks
// and relayed to other peers. It is safe for concurrent access from multiple
// peers.
type TxPool struct {
	// The following variables must only be used atomically.
	lastUpdated int64 // last time pool was updated

	mtx           sync.RWMutex
	cfg           Config
	pool          map[daghash.TxID]*TxDesc
	depends       map[daghash.TxID]*TxDesc
	dependsByPrev map[domainmessage.Outpoint]map[daghash.TxID]*TxDesc
	orphans       map[daghash.TxID]*orphanTx
	orphansByPrev map[domainmessage.Outpoint]map[daghash.TxID]*util.Tx
	outpoints     map[domainmessage.Outpoint]*util.Tx

	// nextExpireScan is the time after which the orphan pool will be
	// scanned in order to evict orphans. This is NOT a hard deadline as
	// the scan will only run when an orphan is added to the pool as opposed
	// to on an unconditional timer.
	nextExpireScan mstime.Time

	mpUTXOSet blockdag.UTXOSet
}

// Ensure the TxPool type implements the mining.TxSource interface.
var _ mining.TxSource = (*TxPool)(nil)

// removeOrphan is the internal function which implements the public
// RemoveOrphan. See the comment for RemoveOrphan for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeOrphan(tx *util.Tx, removeRedeemers bool) {
	// Nothing to do if passed tx is not an orphan.
	txID := tx.ID()
	otx, exists := mp.orphans[*txID]
	if !exists {
		return
	}

	// Remove the reference from the previous orphan index.
	for _, txIn := range otx.tx.MsgTx().TxIn {
		orphans, exists := mp.orphansByPrev[txIn.PreviousOutpoint]
		if exists {
			delete(orphans, *txID)

			// Remove the map entry altogether if there are no
			// longer any orphans which depend on it.
			if len(orphans) == 0 {
				delete(mp.orphansByPrev, txIn.PreviousOutpoint)
			}
		}
	}

	// Remove any orphans that redeem outputs from this one if requested.
	if removeRedeemers {
		prevOut := domainmessage.Outpoint{TxID: *txID}
		for txOutIdx := range tx.MsgTx().TxOut {
			prevOut.Index = uint32(txOutIdx)
			for _, orphan := range mp.orphansByPrev[prevOut] {
				mp.removeOrphan(orphan, true)
			}
		}
	}

	// Remove the transaction from the orphan pool.
	delete(mp.orphans, *txID)
}

// RemoveOrphan removes the passed orphan transaction from the orphan pool and
// previous orphan index.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveOrphan(tx *util.Tx) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	mp.removeOrphan(tx, false)
}

// limitNumOrphans limits the number of orphan transactions by evicting a random
// orphan if adding a new one would cause it to overflow the max allowed.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) limitNumOrphans() error {
	// Scan through the orphan pool and remove any expired orphans when it's
	// time. This is done for efficiency so the scan only happens
	// periodically instead of on every orphan added to the pool.
	if now := mstime.Now(); now.After(mp.nextExpireScan) {
		origNumOrphans := len(mp.orphans)
		for _, otx := range mp.orphans {
			if now.After(otx.expiration) {
				// Remove redeemers too because the missing
				// parents are very unlikely to ever materialize
				// since the orphan has already been around more
				// than long enough for them to be delivered.
				mp.removeOrphan(otx.tx, true)
			}
		}

		// Set next expiration scan to occur after the scan interval.
		mp.nextExpireScan = now.Add(orphanExpireScanInterval)

		numOrphans := len(mp.orphans)
		if numExpired := origNumOrphans - numOrphans; numExpired > 0 {
			log.Debugf("Expired %d %s (remaining: %d)", numExpired,
				logger.PickNoun(uint64(numExpired), "orphan", "orphans"),
				numOrphans)
		}
	}

	// Nothing to do if adding another orphan will not cause the pool to
	// exceed the limit.
	if len(mp.orphans)+1 <= mp.cfg.Policy.MaxOrphanTxs {
		return nil
	}

	// Remove a random entry from the map. For most compilers, Go's
	// range statement iterates starting at a random item although
	// that is not 100% guaranteed by the spec. The iteration order
	// is not important here because an adversary would have to be
	// able to pull off preimage attacks on the hashing function in
	// order to target eviction of specific entries anyways.
	for _, otx := range mp.orphans {
		// Don't remove redeemers in the case of a random eviction since
		// it is quite possible it might be needed again shortly.
		mp.removeOrphan(otx.tx, false)
		break
	}

	return nil
}

// addOrphan adds an orphan transaction to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addOrphan(tx *util.Tx) {
	// Nothing to do if no orphans are allowed.
	if mp.cfg.Policy.MaxOrphanTxs <= 0 {
		return
	}

	// Limit the number orphan transactions to prevent memory exhaustion.
	// This will periodically remove any expired orphans and evict a random
	// orphan if space is still needed.
	mp.limitNumOrphans()

	mp.orphans[*tx.ID()] = &orphanTx{
		tx:         tx,
		expiration: mstime.Now().Add(orphanTTL),
	}
	for _, txIn := range tx.MsgTx().TxIn {
		if _, exists := mp.orphansByPrev[txIn.PreviousOutpoint]; !exists {
			mp.orphansByPrev[txIn.PreviousOutpoint] =
				make(map[daghash.TxID]*util.Tx)
		}
		mp.orphansByPrev[txIn.PreviousOutpoint][*tx.ID()] = tx
	}

	log.Debugf("Stored orphan transaction %s (total: %d)", tx.ID(),
		len(mp.orphans))
}

// maybeAddOrphan potentially adds an orphan to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAddOrphan(tx *util.Tx) error {
	// Ignore orphan transactions that are too large. This helps avoid
	// a memory exhaustion attack based on sending a lot of really large
	// orphans. In the case there is a valid transaction larger than this,
	// it will ultimtely be rebroadcast after the parent transactions
	// have been mined or otherwise received.
	//
	// Note that the number of orphan transactions in the orphan pool is
	// also limited, so this equates to a maximum memory used of
	// mp.cfg.Policy.MaxOrphanTxSize * mp.cfg.Policy.MaxOrphanTxs (which is ~5MB
	// using the default values at the time this comment was written).
	serializedLen := tx.MsgTx().SerializeSize()
	if serializedLen > mp.cfg.Policy.MaxOrphanTxSize {
		str := fmt.Sprintf("orphan transaction size of %d bytes is "+
			"larger than max allowed size of %d bytes",
			serializedLen, mp.cfg.Policy.MaxOrphanTxSize)
		return txRuleError(RejectNonstandard, str)
	}

	// Add the orphan if the none of the above disqualified it.
	mp.addOrphan(tx)

	return nil
}

// removeOrphanDoubleSpends removes all orphans which spend outputs spent by the
// passed transaction from the orphan pool. Removing those orphans then leads
// to removing all orphans which rely on them, recursively. This is necessary
// when a transaction is added to the main pool because it may spend outputs
// that orphans also spend.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeOrphanDoubleSpends(tx *util.Tx) {
	msgTx := tx.MsgTx()
	for _, txIn := range msgTx.TxIn {
		for _, orphan := range mp.orphansByPrev[txIn.PreviousOutpoint] {
			mp.removeOrphan(orphan, true)
		}
	}
}

// isTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isTransactionInPool(txID *daghash.TxID) bool {
	if _, exists := mp.pool[*txID]; exists {
		return true
	}
	return mp.isInDependPool(txID)
}

// IsTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsTransactionInPool(hash *daghash.TxID) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	inPool := mp.isTransactionInPool(hash)

	return inPool
}

// isInDependPool returns whether or not the passed transaction already
// exists in the depend pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isInDependPool(hash *daghash.TxID) bool {
	if _, exists := mp.depends[*hash]; exists {
		return true
	}

	return false
}

// IsInDependPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsInDependPool(hash *daghash.TxID) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	return mp.isInDependPool(hash)
}

// isOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isOrphanInPool(txID *daghash.TxID) bool {
	if _, exists := mp.orphans[*txID]; exists {
		return true
	}

	return false
}

// IsOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsOrphanInPool(hash *daghash.TxID) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	inPool := mp.isOrphanInPool(hash)

	return inPool
}

// haveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) haveTransaction(txID *daghash.TxID) bool {
	return mp.isTransactionInPool(txID) || mp.isOrphanInPool(txID)
}

// HaveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) HaveTransaction(txID *daghash.TxID) bool {
	// Protect concurrent access.
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	haveTx := mp.haveTransaction(txID)

	return haveTx
}

// removeTransactions is the internal function which implements the public
// RemoveTransactions. See the comment for RemoveTransactions for more details.
//
// This method, in contrast to removeTransaction (singular), creates one utxoDiff
// and calls removeTransactionWithDiff on it for every transaction. This is an
// optimization to save us a good amount of allocations (specifically in
// UTXODiff.WithDiff) every time we accept a block.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeTransactions(txs []*util.Tx) error {
	diff := blockdag.NewUTXODiff()

	for _, tx := range txs {
		txID := tx.ID()

		if _, exists := mp.fetchTxDesc(txID); !exists {
			continue
		}

		err := mp.removeTransactionWithDiff(tx, diff, false)
		if err != nil {
			return err
		}
	}

	var err error
	mp.mpUTXOSet, err = mp.mpUTXOSet.WithDiff(diff)
	if err != nil {
		return err
	}
	atomic.StoreInt64(&mp.lastUpdated, mstime.Now().UnixMilliseconds())

	return nil
}

// removeTransaction is the internal function which implements the public
// RemoveTransaction. See the comment for RemoveTransaction for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeTransaction(tx *util.Tx, removeDependants bool, restoreInputs bool) error {
	txID := tx.ID()
	if removeDependants {
		// Remove any transactions which rely on this one.
		for i := uint32(0); i < uint32(len(tx.MsgTx().TxOut)); i++ {
			prevOut := domainmessage.Outpoint{TxID: *txID, Index: i}
			if txRedeemer, exists := mp.outpoints[prevOut]; exists {
				err := mp.removeTransaction(txRedeemer, true, false)
				if err != nil {
					return err
				}
			}
		}
	}

	if _, exists := mp.fetchTxDesc(txID); !exists {
		return nil
	}

	diff := blockdag.NewUTXODiff()
	err := mp.removeTransactionWithDiff(tx, diff, restoreInputs)
	if err != nil {
		return err
	}

	mp.mpUTXOSet, err = mp.mpUTXOSet.WithDiff(diff)
	if err != nil {
		return err
	}
	atomic.StoreInt64(&mp.lastUpdated, mstime.Now().UnixMilliseconds())

	return nil
}

// removeTransactionWithDiff removes the transaction tx from the mempool while
// updating the UTXODiff diff with appropriate changes. diff is later meant to
// be withDiff'd against the mempool UTXOSet to update it.
//
// This method assumes that tx exists in the mempool.
func (mp *TxPool) removeTransactionWithDiff(tx *util.Tx, diff *blockdag.UTXODiff, restoreInputs bool) error {
	txID := tx.ID()

	err := mp.removeTransactionUTXOEntriesFromDiff(tx, diff)
	if err != nil {
		return errors.Errorf("could not remove UTXOEntry from diff: %s", err)
	}

	err = mp.markTransactionOutputsUnspent(tx, diff, restoreInputs)
	if err != nil {
		return errors.Errorf("could not mark transaction output as unspent: %s", err)
	}

	txDesc, _ := mp.fetchTxDesc(txID)
	if txDesc.depCount == 0 {
		delete(mp.pool, *txID)
	} else {
		delete(mp.depends, *txID)
	}

	mp.processRemovedTransactionDependencies(tx)

	return nil
}

// removeTransactionUTXOEntriesFromDiff removes tx's UTXOEntries from the diff
func (mp *TxPool) removeTransactionUTXOEntriesFromDiff(tx *util.Tx, diff *blockdag.UTXODiff) error {
	for idx := range tx.MsgTx().TxOut {
		outpoint := *domainmessage.NewOutpoint(tx.ID(), uint32(idx))
		entry, exists := mp.mpUTXOSet.Get(outpoint)
		if exists {
			err := diff.RemoveEntry(outpoint, entry)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// markTransactionOutputsUnspent updates the mempool so that tx's TXOs are unspent
// Iff restoreInputs is true then the inputs are restored back into the supplied diff
func (mp *TxPool) markTransactionOutputsUnspent(tx *util.Tx, diff *blockdag.UTXODiff, restoreInputs bool) error {
	for _, txIn := range tx.MsgTx().TxIn {
		if restoreInputs {
			if prevTxDesc, exists := mp.pool[txIn.PreviousOutpoint.TxID]; exists {
				prevOut := prevTxDesc.Tx.MsgTx().TxOut[txIn.PreviousOutpoint.Index]
				entry := blockdag.NewUTXOEntry(prevOut, false, blockdag.UnacceptedBlueScore)
				err := diff.AddEntry(txIn.PreviousOutpoint, entry)
				if err != nil {
					return err
				}
			}
			if prevTxDesc, exists := mp.depends[txIn.PreviousOutpoint.TxID]; exists {
				prevOut := prevTxDesc.Tx.MsgTx().TxOut[txIn.PreviousOutpoint.Index]
				entry := blockdag.NewUTXOEntry(prevOut, false, blockdag.UnacceptedBlueScore)
				err := diff.AddEntry(txIn.PreviousOutpoint, entry)
				if err != nil {
					return err
				}
			}
		}
		delete(mp.outpoints, txIn.PreviousOutpoint)
	}
	return nil
}

// processRemovedTransactionDependencies processes the dependencies of a
// transaction tx that was just now removed from the mempool
func (mp *TxPool) processRemovedTransactionDependencies(tx *util.Tx) {
	prevOut := domainmessage.Outpoint{TxID: *tx.ID()}
	for txOutIdx := range tx.MsgTx().TxOut {
		// Skip to the next available output if there are none.
		prevOut.Index = uint32(txOutIdx)
		depends, exists := mp.dependsByPrev[prevOut]
		if !exists {
			continue
		}

		// Move independent transactions into main pool
		for _, txD := range depends {
			txD.depCount--
			if txD.depCount == 0 {
				// Transaction may be already removed by recursive calls, if removeRedeemers is true.
				// So avoid moving it into main pool
				if _, ok := mp.depends[*txD.Tx.ID()]; ok {
					delete(mp.depends, *txD.Tx.ID())
					mp.pool[*txD.Tx.ID()] = txD
				}
			}
		}
		delete(mp.dependsByPrev, prevOut)
	}
}

// RemoveTransaction removes the passed transaction from the mempool. When the
// removeDependants flag is set, any transactions that depend on the removed
// transaction (that is to say, redeem outputs from it) will also be removed
// recursively from the mempool, as they would otherwise become orphans.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveTransaction(tx *util.Tx, removeDependants bool, restoreInputs bool) error {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	return mp.removeTransaction(tx, removeDependants, restoreInputs)
}

// RemoveTransactions removes the passed transactions from the mempool.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveTransactions(txs []*util.Tx) error {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	return mp.removeTransactions(txs)
}

// RemoveDoubleSpends removes all transactions which spend outputs spent by the
// passed transaction from the memory pool. Removing those transactions then
// leads to removing all transactions which rely on them, recursively. This is
// necessary when a block is connected to the DAG because the block may
// contain transactions which were previously unknown to the memory pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveDoubleSpends(tx *util.Tx) error {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	return mp.removeDoubleSpends(tx)
}

func (mp *TxPool) removeDoubleSpends(tx *util.Tx) error {
	for _, txIn := range tx.MsgTx().TxIn {
		if txRedeemer, ok := mp.outpoints[txIn.PreviousOutpoint]; ok {
			if !txRedeemer.ID().IsEqual(tx.ID()) {
				err := mp.removeTransaction(txRedeemer, true, false)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// addTransaction adds the passed transaction to the memory pool. It should
// not be called directly as it doesn't perform any validation. This is a
// helper for maybeAcceptTransaction.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addTransaction(tx *util.Tx, fee uint64, parentsInPool []*domainmessage.Outpoint) (*TxDesc, error) {
	// Add the transaction to the pool and mark the referenced outpoints
	// as spent by the pool.
	mass, err := blockdag.CalcTxMassFromUTXOSet(tx, mp.mpUTXOSet)
	if err != nil {
		return nil, err
	}
	txD := &TxDesc{
		TxDesc: mining.TxDesc{
			Tx:             tx,
			Added:          mstime.Now(),
			Fee:            fee,
			FeePerMegaGram: fee * 1e6 / mass,
		},
		depCount: len(parentsInPool),
	}

	if len(parentsInPool) == 0 {
		mp.pool[*tx.ID()] = txD
	} else {
		mp.depends[*tx.ID()] = txD
		for _, previousOutpoint := range parentsInPool {
			if _, exists := mp.dependsByPrev[*previousOutpoint]; !exists {
				mp.dependsByPrev[*previousOutpoint] = make(map[daghash.TxID]*TxDesc)
			}
			mp.dependsByPrev[*previousOutpoint][*tx.ID()] = txD
		}
	}

	for _, txIn := range tx.MsgTx().TxIn {
		mp.outpoints[txIn.PreviousOutpoint] = tx
	}
	if isAccepted, err := mp.mpUTXOSet.AddTx(tx.MsgTx(), blockdag.UnacceptedBlueScore); err != nil {
		return nil, err
	} else if !isAccepted {
		return nil, errors.Errorf("unexpectedly failed to add tx %s to the mempool utxo set", tx.ID())
	}
	atomic.StoreInt64(&mp.lastUpdated, mstime.Now().UnixMilliseconds())

	return txD, nil
}

// checkPoolDoubleSpend checks whether or not the passed transaction is
// attempting to spend coins already spent by other transactions in the pool.
// Note it does not check for double spends against transactions already in the
// DAG.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) checkPoolDoubleSpend(tx *util.Tx) error {
	for _, txIn := range tx.MsgTx().TxIn {
		if txR, exists := mp.outpoints[txIn.PreviousOutpoint]; exists {
			str := fmt.Sprintf("output %s already spent by "+
				"transaction %s in the memory pool",
				txIn.PreviousOutpoint, txR.ID())
			return txRuleError(RejectDuplicate, str)
		}
	}

	return nil
}

// CheckSpend checks whether the passed outpoint is already spent by a
// transaction in the mempool. If that's the case the spending transaction will
// be returned, if not nil will be returned.
func (mp *TxPool) CheckSpend(op domainmessage.Outpoint) *util.Tx {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	txR := mp.outpoints[op]

	return txR
}

// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) fetchTxDesc(txID *daghash.TxID) (*TxDesc, bool) {
	txDesc, exists := mp.pool[*txID]
	if !exists {
		txDesc, exists = mp.depends[*txID]
	}
	return txDesc, exists
}

// FetchTxDesc returns the requested TxDesc from the transaction pool.
// This only fetches from the main transaction pool and does not include
// orphans.
// returns false in the second return parameter if transaction was not found
//
// This function is safe for concurrent access.
func (mp *TxPool) FetchTxDesc(txID *daghash.TxID) (*TxDesc, bool) {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if txDesc, exists := mp.fetchTxDesc(txID); exists {
		return txDesc, true
	}

	return nil, false
}

// FetchTransaction returns the requested transaction from the transaction pool.
// This only fetches from the main transaction pool and does not include
// orphans.
// returns false in the second return parameter if transaction was not found
//
// This function is safe for concurrent access.
func (mp *TxPool) FetchTransaction(txID *daghash.TxID) (*util.Tx, bool) {
	// Protect concurrent access.
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if txDesc, exists := mp.fetchTxDesc(txID); exists {
		return txDesc.Tx, true
	}

	return nil, false
}

// checkTransactionMassSanity checks that a transaction must not exceed the maximum allowed block mass when serialized.
func checkTransactionMassSanity(tx *util.Tx) error {
	serializedTxSize := tx.MsgTx().SerializeSize()
	if serializedTxSize*blockdag.MassPerTxByte > domainmessage.MaxMassPerTx {
		str := fmt.Sprintf("serialized transaction is too big - got "+
			"%d, max %d", serializedTxSize, domainmessage.MaxMassPerBlock)
		return txRuleError(RejectInvalid, str)
	}
	return nil
}

// maybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool. It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, detecting orphan transactions, and insertion into the memory pool.
//
// If the transaction is an orphan (missing parent transactions), the
// transaction is NOT added to the orphan pool, but each unknown referenced
// parent is returned. Use ProcessTransaction instead if new orphans should
// be added to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAcceptTransaction(tx *util.Tx, rejectDupOrphans bool) ([]*daghash.TxID, *TxDesc, error) {
	txID := tx.ID()

	// Don't accept the transaction if it already exists in the pool. This
	// applies to orphan transactions as well when the reject duplicate
	// orphans flag is set. This check is intended to be a quick check to
	// weed out duplicates.
	if mp.isTransactionInPool(txID) || (rejectDupOrphans &&
		mp.isOrphanInPool(txID)) {

		str := fmt.Sprintf("already have transaction %s", txID)
		return nil, nil, txRuleError(RejectDuplicate, str)
	}

	// Don't accept the transaction if it's from an incompatible subnetwork.
	subnetworkID := mp.cfg.DAG.SubnetworkID()
	if !tx.MsgTx().IsSubnetworkCompatible(subnetworkID) {
		str := fmt.Sprintf("tx %s belongs to an invalid subnetwork %s, DAG subnetwork %s", tx.ID(),
			tx.MsgTx().SubnetworkID, subnetworkID)
		return nil, nil, txRuleError(RejectInvalid, str)
	}

	// Disallow non-native/coinbase subnetworks in networks that don't allow them
	if !mp.cfg.DAG.Params.EnableNonNativeSubnetworks {
		if !(tx.MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDNative) ||
			tx.MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDCoinbase)) {
			return nil, nil, txRuleError(RejectInvalid, "non-native/coinbase subnetworks are not allowed")
		}
	}

	err := checkTransactionMassSanity(tx)
	if err != nil {
		return nil, nil, err
	}

	// Perform preliminary sanity checks on the transaction. This makes
	// use of blockDAG which contains the invariant rules for what
	// transactions are allowed into blocks.
	err = blockdag.CheckTransactionSanity(tx, subnetworkID)
	if err != nil {
		var ruleErr blockdag.RuleError
		if ok := errors.As(err, &ruleErr); ok {
			return nil, nil, dagRuleError(ruleErr)
		}
		return nil, nil, err
	}

	// Check that transaction does not overuse GAS
	msgTx := tx.MsgTx()
	if !msgTx.SubnetworkID.IsBuiltInOrNative() {
		gasLimit, err := mp.cfg.DAG.GasLimit(&msgTx.SubnetworkID)
		if err != nil {
			return nil, nil, err
		}
		if msgTx.Gas > gasLimit {
			str := fmt.Sprintf("transaction wants more gas %d, than allowed %d",
				msgTx.Gas, gasLimit)
			return nil, nil, dagRuleError(blockdag.RuleError{
				ErrorCode:   blockdag.ErrInvalidGas,
				Description: str})
		}
	}

	// A standalone transaction must not be a coinbase transaction.
	if tx.IsCoinBase() {
		str := fmt.Sprintf("transaction %s is an individual coinbase transaction",
			txID)
		return nil, nil, txRuleError(RejectInvalid, str)
	}

	// We take the blue score of the current virtual block to validate
	// the transaction as though it was mined on top of the current tips
	nextBlockBlueScore := mp.cfg.DAG.VirtualBlueScore()

	medianTimePast := mp.cfg.DAG.CalcPastMedianTime()

	// Don't allow non-standard transactions if the network parameters
	// forbid their acceptance.
	if !mp.cfg.Policy.AcceptNonStd {
		err = checkTransactionStandard(tx, nextBlockBlueScore,
			medianTimePast, &mp.cfg.Policy)
		if err != nil {
			// Attempt to extract a reject code from the error so
			// it can be retained. When not possible, fall back to
			// a non standard error.
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = RejectNonstandard
			}
			str := fmt.Sprintf("transaction %s is not standard: %s",
				txID, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}

	// The transaction may not use any of the same outputs as other
	// transactions already in the pool as that would ultimately result in a
	// double spend. This check is intended to be quick and therefore only
	// detects double spends within the transaction pool itself. The
	// transaction could still be double spending coins from the DAG
	// at this point. There is a more in-depth check that happens later
	// after fetching the referenced transaction inputs from the DAG
	// which examines the actual spend data and prevents double spends.
	err = mp.checkPoolDoubleSpend(tx)
	if err != nil {
		return nil, nil, err
	}

	// Don't allow the transaction if it exists in the DAG and is
	// not already fully spent.
	prevOut := domainmessage.Outpoint{TxID: *txID}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)
		_, ok := mp.mpUTXOSet.Get(prevOut)
		if ok {
			return nil, nil, txRuleError(RejectDuplicate,
				"transaction already exists")
		}
	}

	// Transaction is an orphan if any of the referenced transaction outputs
	// don't exist or are already spent. Adding orphans to the orphan pool
	// is not handled by this function, and the caller should use
	// maybeAddOrphan if this behavior is desired.
	var missingParents []*daghash.TxID
	var parentsInPool []*domainmessage.Outpoint
	for _, txIn := range tx.MsgTx().TxIn {
		if _, ok := mp.mpUTXOSet.Get(txIn.PreviousOutpoint); !ok {
			// Must make a copy of the hash here since the iterator
			// is replaced and taking its address directly would
			// result in all of the entries pointing to the same
			// memory location and thus all be the final hash.
			txIDCopy := txIn.PreviousOutpoint.TxID
			missingParents = append(missingParents, &txIDCopy)
		}
		if mp.isTransactionInPool(&txIn.PreviousOutpoint.TxID) {
			parentsInPool = append(parentsInPool, &txIn.PreviousOutpoint)
		}
	}
	if len(missingParents) > 0 {
		return missingParents, nil, nil
	}

	// Don't allow the transaction into the mempool unless its sequence
	// lock is active, meaning that it'll be allowed into the next block
	// with respect to its defined relative lock times.
	sequenceLock, err := mp.cfg.CalcSequenceLockNoLock(tx, mp.mpUTXOSet)
	if err != nil {
		var dagRuleErr blockdag.RuleError
		if ok := errors.As(err, &dagRuleErr); ok {
			return nil, nil, dagRuleError(dagRuleErr)
		}
		return nil, nil, err
	}
	if !blockdag.SequenceLockActive(sequenceLock, nextBlockBlueScore,
		medianTimePast) {
		return nil, nil, txRuleError(RejectNonstandard,
			"transaction's sequence locks on inputs not met")
	}

	// Don't allow transactions that exceed the maximum allowed
	// transaction mass.
	err = blockdag.ValidateTxMass(tx, mp.mpUTXOSet)
	if err != nil {
		var ruleError blockdag.RuleError
		if ok := errors.As(err, &ruleError); ok {
			return nil, nil, dagRuleError(ruleError)
		}
		return nil, nil, err
	}

	// Perform several checks on the transaction inputs using the invariant
	// rules in blockDAG for what transactions are allowed into blocks.
	// Also returns the fees associated with the transaction which will be
	// used later.
	txFee, err := blockdag.CheckTransactionInputsAndCalulateFee(tx, nextBlockBlueScore,
		mp.mpUTXOSet, mp.cfg.DAG.Params, false)
	if err != nil {
		var dagRuleErr blockdag.RuleError
		if ok := errors.As(err, &dagRuleErr); ok {
			return nil, nil, dagRuleError(dagRuleErr)
		}
		return nil, nil, err
	}

	// Don't allow transactions with non-standard inputs if the network
	// parameters forbid their acceptance.
	if !mp.cfg.Policy.AcceptNonStd {
		err := checkInputsStandard(tx, mp.mpUTXOSet)
		if err != nil {
			// Attempt to extract a reject code from the error so
			// it can be retained. When not possible, fall back to
			// a non standard error.
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = RejectNonstandard
			}
			str := fmt.Sprintf("transaction %s has a non-standard "+
				"input: %s", txID, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}

	// NOTE: if you modify this code to accept non-standard transactions,
	// you should add code here to check that the transaction does a
	// reasonable number of ECDSA signature verifications.

	// Don't allow transactions with 0 fees.
	if txFee == 0 {
		str := fmt.Sprintf("transaction %s has 0 fees", txID)
		return nil, nil, txRuleError(RejectInsufficientFee, str)
	}

	// Don't allow transactions with fees too low to get into a mined block.
	//
	// Most miners allow a free transaction area in blocks they mine to go
	// alongside the area used for high-priority transactions as well as
	// transactions with fees. A transaction size of up to 1000 bytes is
	// considered safe to go into this section. Further, the minimum fee
	// calculated below on its own would encourage several small
	// transactions to avoid fees rather than one single larger transaction
	// which is more desirable. Therefore, as long as the size of the
	// transaction does not exceeed 1000 less than the reserved space for
	// high-priority transactions, don't require a fee for it.
	serializedSize := int64(tx.MsgTx().SerializeSize())
	minFee := uint64(calcMinRequiredTxRelayFee(serializedSize,
		mp.cfg.Policy.MinRelayTxFee))
	if txFee < minFee {
		str := fmt.Sprintf("transaction %s has %d fees which is under "+
			"the required amount of %d", txID, txFee,
			minFee)
		return nil, nil, txRuleError(RejectInsufficientFee, str)
	}

	// Verify crypto signatures for each input and reject the transaction if
	// any don't verify.
	err = blockdag.ValidateTransactionScripts(tx, mp.mpUTXOSet,
		txscript.StandardVerifyFlags, mp.cfg.SigCache)
	if err != nil {
		var dagRuleErr blockdag.RuleError
		if ok := errors.As(err, &dagRuleErr); ok {
			return nil, nil, dagRuleError(dagRuleErr)
		}
		return nil, nil, err
	}

	// Add to transaction pool.
	txD, err := mp.addTransaction(tx, txFee, parentsInPool)
	if err != nil {
		return nil, nil, err
	}

	log.Debugf("Accepted transaction %s (pool size: %d)", txID,
		len(mp.pool))

	return nil, txD, nil
}

// processOrphans is the internal function which implements the public
// ProcessOrphans. See the comment for ProcessOrphans for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) processOrphans(acceptedTx *util.Tx) []*TxDesc {
	var acceptedTxns []*TxDesc

	// Start with processing at least the passed transaction.
	processList := list.New()
	processList.PushBack(acceptedTx)
	for processList.Len() > 0 {
		// Pop the transaction to process from the front of the list.
		firstElement := processList.Remove(processList.Front())
		processItem := firstElement.(*util.Tx)

		prevOut := domainmessage.Outpoint{TxID: *processItem.ID()}
		for txOutIdx := range processItem.MsgTx().TxOut {
			// Look up all orphans that redeem the output that is
			// now available. This will typically only be one, but
			// it could be multiple if the orphan pool contains
			// double spends. While it may seem odd that the orphan
			// pool would allow this since there can only possibly
			// ultimately be a single redeemer, it's important to
			// track it this way to prevent malicious actors from
			// being able to purposely constructing orphans that
			// would otherwise make outputs unspendable.
			//
			// Skip to the next available output if there are none.
			prevOut.Index = uint32(txOutIdx)
			orphans, exists := mp.orphansByPrev[prevOut]
			if !exists {
				continue
			}

			// Potentially accept an orphan into the tx pool.
			for _, tx := range orphans {
				missing, txD, err := mp.maybeAcceptTransaction(
					tx, false)
				if err != nil {
					// The orphan is now invalid, so there
					// is no way any other orphans which
					// redeem any of its outputs can be
					// accepted. Remove them.
					mp.removeOrphan(tx, true)
					break
				}

				// Transaction is still an orphan. Try the next
				// orphan which redeems this output.
				if len(missing) > 0 {
					continue
				}

				// Transaction was accepted into the main pool.
				//
				// Add it to the list of accepted transactions
				// that are no longer orphans, remove it from
				// the orphan pool, and add it to the list of
				// transactions to process so any orphans that
				// depend on it are handled too.
				acceptedTxns = append(acceptedTxns, txD)
				mp.removeOrphan(tx, false)
				processList.PushBack(tx)

				// Only one transaction for this outpoint can be
				// accepted, so the rest are now double spends
				// and are removed later.
				break
			}
		}
	}

	// Recursively remove any orphans that also redeem any outputs redeemed
	// by the accepted transactions since those are now definitive double
	// spends.
	mp.removeOrphanDoubleSpends(acceptedTx)
	for _, txD := range acceptedTxns {
		mp.removeOrphanDoubleSpends(txD.Tx)
	}

	return acceptedTxns
}

// ProcessOrphans determines if there are any orphans which depend on the passed
// transaction hash (it is possible that they are no longer orphans) and
// potentially accepts them to the memory pool. It repeats the process for the
// newly accepted transactions (to detect further orphans which may no longer be
// orphans) until there are no more.
//
// It returns a slice of transactions added to the mempool. A nil slice means
// no transactions were moved from the orphan pool to the mempool.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessOrphans(acceptedTx *util.Tx) []*TxDesc {
	mp.cfg.DAG.RLock()
	defer mp.cfg.DAG.RUnlock()
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	acceptedTxns := mp.processOrphans(acceptedTx)

	return acceptedTxns
}

// ProcessTransaction is the main workhorse for handling insertion of new
// free-standing transactions into the memory pool. It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, orphan transaction handling, and insertion into the memory pool.
//
// It returns a slice of transactions added to the mempool. When the
// error is nil, the list will include the passed transaction itself along
// with any additional orphan transaactions that were added as a result of
// the passed one being accepted.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessTransaction(tx *util.Tx, allowOrphan bool) ([]*TxDesc, error) {
	log.Tracef("Processing transaction %s", tx.ID())

	// Protect concurrent access.
	mp.cfg.DAG.RLock()
	defer mp.cfg.DAG.RUnlock()
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	// Potentially accept the transaction to the memory pool.
	missingParents, txD, err := mp.maybeAcceptTransaction(tx, true)
	if err != nil {
		return nil, err
	}

	if len(missingParents) == 0 {
		// Accept any orphan transactions that depend on this
		// transaction (they may no longer be orphans if all inputs
		// are now available) and repeat for those accepted
		// transactions until there are no more.
		newTxs := mp.processOrphans(tx)
		acceptedTxs := make([]*TxDesc, len(newTxs)+1)

		// Add the parent transaction first so remote nodes
		// do not add orphans.
		acceptedTxs[0] = txD
		copy(acceptedTxs[1:], newTxs)

		return acceptedTxs, nil
	}

	// The transaction is an orphan (has inputs missing). Reject
	// it if the flag to allow orphans is not set.
	if !allowOrphan {
		// Only use the first missing parent transaction in
		// the error message.
		//
		// NOTE: RejectDuplicate is really not an accurate
		// reject code here, but it matches the reference
		// implementation and there isn't a better choice due
		// to the limited number of reject codes. Missing
		// inputs is assumed to mean they are already spent
		// which is not really always the case.
		str := fmt.Sprintf("orphan transaction %s references "+
			"outputs of unknown or fully-spent "+
			"transaction %s", tx.ID(), missingParents[0])
		return nil, txRuleError(RejectDuplicate, str)
	}

	// Potentially add the orphan transaction to the orphan pool.
	err = mp.maybeAddOrphan(tx)
	return nil, err
}

// Count returns the number of transactions in the main pool. It does not
// include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	count := len(mp.pool)

	return count
}

// DepCount returns the number of dependent transactions in the main pool. It does not
// include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) DepCount() int {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	return len(mp.depends)
}

// TxIDs returns a slice of IDs for all of the transactions in the memory
// pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) TxIDs() []*daghash.TxID {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	ids := make([]*daghash.TxID, len(mp.pool))
	i := 0
	for txID := range mp.pool {
		idCopy := txID
		ids[i] = &idCopy
		i++
	}

	return ids
}

// TxDescs returns a slice of descriptors for all the transactions in the pool.
// The descriptors are to be treated as read only.
//
// This function is safe for concurrent access.
func (mp *TxPool) TxDescs() []*TxDesc {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	descs := make([]*TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = desc
		i++
	}

	return descs
}

// MiningDescs returns a slice of mining descriptors for all the transactions
// in the pool.
//
// This is part of the mining.TxSource interface implementation and is safe for
// concurrent access as required by the interface contract.
func (mp *TxPool) MiningDescs() []*mining.TxDesc {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	descs := make([]*mining.TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = &desc.TxDesc
		i++
	}

	return descs
}

// LastUpdated returns the last time a transaction was added to or removed from
// the main pool. It does not include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) LastUpdated() mstime.Time {
	return mstime.UnixMilliseconds(atomic.LoadInt64(&mp.lastUpdated))
}

// HandleNewBlock removes all the transactions in the new block
// from the mempool and the orphan pool, and it also removes
// from the mempool transactions that double spend a
// transaction that is already in the DAG
func (mp *TxPool) HandleNewBlock(block *util.Block) ([]*util.Tx, error) {
	// Protect concurrent access.
	mp.cfg.DAG.RLock()
	defer mp.cfg.DAG.RUnlock()
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	oldUTXOSet := mp.mpUTXOSet

	// Remove all of the transactions (except the coinbase) in the
	// connected block from the transaction pool. Secondly, remove any
	// transactions which are now double spends as a result of these
	// new transactions. Finally, remove any transaction that is
	// no longer an orphan. Transactions which depend on a confirmed
	// transaction are NOT removed recursively because they are still
	// valid.
	err := mp.removeTransactions(block.Transactions()[util.CoinbaseTransactionIndex+1:])
	if err != nil {
		mp.mpUTXOSet = oldUTXOSet
		return nil, err
	}
	acceptedTxs := make([]*util.Tx, 0)
	for _, tx := range block.Transactions()[util.CoinbaseTransactionIndex+1:] {
		err := mp.removeDoubleSpends(tx)
		if err != nil {
			return nil, err
		}
		mp.removeOrphan(tx, false)
		acceptedOrphans := mp.processOrphans(tx)
		for _, acceptedOrphan := range acceptedOrphans {
			acceptedTxs = append(acceptedTxs, acceptedOrphan.Tx)
		}
	}
	return acceptedTxs, nil
}

// New returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func New(cfg *Config) *TxPool {
	virtualUTXO := cfg.DAG.UTXOSet()
	mpUTXO := blockdag.NewDiffUTXOSet(virtualUTXO, blockdag.NewUTXODiff())
	return &TxPool{
		cfg:            *cfg,
		pool:           make(map[daghash.TxID]*TxDesc),
		depends:        make(map[daghash.TxID]*TxDesc),
		dependsByPrev:  make(map[domainmessage.Outpoint]map[daghash.TxID]*TxDesc),
		orphans:        make(map[daghash.TxID]*orphanTx),
		orphansByPrev:  make(map[domainmessage.Outpoint]map[daghash.TxID]*util.Tx),
		nextExpireScan: mstime.Now().Add(orphanExpireScanInterval),
		outpoints:      make(map[domainmessage.Outpoint]*util.Tx),
		mpUTXOSet:      mpUTXO,
	}
}
