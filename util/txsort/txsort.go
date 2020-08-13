// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Provides functions for sorting tx inputs and outputs according to BIP 69
// (https://github.com/bitcoin/bips/blob/master/bip-0069.mediawiki)

package txsort

import (
	"bytes"
	"sort"

	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util/daghash"
)

// InPlaceSort modifies the passed transaction inputs and outputs to be sorted
// based on BIP 69.
//
// WARNING: This function must NOT be called with published transactions since
// it will mutate the transaction if it's not already sorted. This can cause
// issues if you mutate a tx in a block, for example, which would invalidate the
// block. It could also cause cached hashes, such as in a util.Tx to become
// invalidated.
//
// The function should only be used if the caller is creating the transaction or
// is otherwise 100% positive mutating will not cause adverse affects due to
// other dependencies.
func InPlaceSort(tx *domainmessage.MsgTx) {
	sort.Sort(sortableInputSlice(tx.TxIn))
	sort.Sort(sortableOutputSlice(tx.TxOut))
}

// Sort returns a new transaction with the inputs and outputs sorted based on
// BIP 69. The passed transaction is not modified and the new transaction
// might have a different hash if any sorting was done.
func Sort(tx *domainmessage.MsgTx) *domainmessage.MsgTx {
	txCopy := tx.Copy()
	sort.Sort(sortableInputSlice(txCopy.TxIn))
	sort.Sort(sortableOutputSlice(txCopy.TxOut))
	return txCopy
}

// IsSorted checks whether tx has inputs and outputs sorted according to BIP
// 69.
func IsSorted(tx *domainmessage.MsgTx) bool {
	if !sort.IsSorted(sortableInputSlice(tx.TxIn)) {
		return false
	}
	if !sort.IsSorted(sortableOutputSlice(tx.TxOut)) {
		return false
	}
	return true
}

type sortableInputSlice []*domainmessage.TxIn
type sortableOutputSlice []*domainmessage.TxOut

// For SortableInputSlice and SortableOutputSlice, three functions are needed
// to make it sortable with sort.Sort() -- Len, Less, and Swap
// Len and Swap are trivial. Less is BIP 69 specific.
func (s sortableInputSlice) Len() int       { return len(s) }
func (s sortableOutputSlice) Len() int      { return len(s) }
func (s sortableOutputSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableInputSlice) Swap(i, j int)  { s[i], s[j] = s[j], s[i] }

// Input comparison function.
// First sort based on input hash (reversed / rpc-style), then index.
func (s sortableInputSlice) Less(i, j int) bool {
	// Input hashes are the same, so compare the index.
	iTxID := s[i].PreviousOutpoint.TxID
	jTxID := s[j].PreviousOutpoint.TxID
	if iTxID == jTxID {
		return s[i].PreviousOutpoint.Index < s[j].PreviousOutpoint.Index
	}

	// At this point, the hashes are not equal, so reverse them to
	// big-endian and return the result of the comparison.
	const txIDSize = daghash.TxIDSize
	for b := 0; b < txIDSize/2; b++ {
		iTxID[b], iTxID[txIDSize-1-b] = iTxID[txIDSize-1-b], iTxID[b]
		jTxID[b], jTxID[txIDSize-1-b] = jTxID[txIDSize-1-b], jTxID[b]
	}
	return bytes.Compare(iTxID[:], jTxID[:]) == -1
}

// Output comparison function.
// First sort based on amount (smallest first), then ScriptPubKey.
func (s sortableOutputSlice) Less(i, j int) bool {
	if s[i].Value == s[j].Value {
		return bytes.Compare(s[i].ScriptPubKey, s[j].ScriptPubKey) < 0
	}
	return s[i].Value < s[j].Value
}
