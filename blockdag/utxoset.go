package blockdag

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/ecc"
	"github.com/kaspanet/kaspad/wire"
)

const (
	// UnacceptedBlueScore is the blue score used for the "block" blueScore
	// field of the contextual transaction information provided in a
	// transaction store when it has not yet been accepted by a block.
	UnacceptedBlueScore uint64 = math.MaxUint64
)

// UTXOEntry houses details about an individual transaction output in a utxo
// set such as whether or not it was contained in a coinbase tx, the blue
// score of the block that accepts the tx, its public key script, and how
// much it pays.
type UTXOEntry struct {
	// NOTE: Additions, deletions, or modifications to the order of the
	// definitions in this struct should not be changed without considering
	// how it affects alignment on 64-bit platforms. The current order is
	// specifically crafted to result in minimal padding. There will be a
	// lot of these in memory, so a few extra bytes of padding adds up.

	amount         uint64
	scriptPubKey   []byte // The public key script for the output.
	blockBlueScore uint64 // Blue score of the block accepting the tx.

	// packedFlags contains additional info about output such as whether it
	// is a coinbase, and whether it has been modified
	// since it was loaded. This approach is used in order to reduce memory
	// usage since there will be a lot of these in memory.
	packedFlags txoFlags
}

// IsCoinbase returns whether or not the output was contained in a block
// reward transaction.
func (entry *UTXOEntry) IsCoinbase() bool {
	return entry.packedFlags&tfCoinbase == tfCoinbase
}

// BlockBlueScore returns the blue score of the block accepting the output.
func (entry *UTXOEntry) BlockBlueScore() uint64 {
	return entry.blockBlueScore
}

// Amount returns the amount of the output.
func (entry *UTXOEntry) Amount() uint64 {
	return entry.amount
}

// ScriptPubKey returns the public key script for the output.
func (entry *UTXOEntry) ScriptPubKey() []byte {
	return entry.scriptPubKey
}

// IsUnaccepted returns true iff this UTXOEntry has been included in a block
// but has not yet been accepted by any block.
func (entry *UTXOEntry) IsUnaccepted() bool {
	return entry.blockBlueScore == UnacceptedBlueScore
}

// txoFlags is a bitmask defining additional information and state for a
// transaction output in a UTXO set.
type txoFlags uint8

const (
	// tfCoinbase indicates that a txout was contained in a coinbase tx.
	tfCoinbase txoFlags = 1 << iota
)

// NewUTXOEntry creates a new utxoEntry representing the given txOut
func NewUTXOEntry(txOut *wire.TxOut, isCoinbase bool, blockBlueScore uint64) *UTXOEntry {
	entry := &UTXOEntry{
		amount:         txOut.Value,
		scriptPubKey:   txOut.ScriptPubKey,
		blockBlueScore: blockBlueScore,
	}

	if isCoinbase {
		entry.packedFlags |= tfCoinbase
	}

	return entry
}

// utxoCollection represents a set of UTXOs indexed by their outpoints
type utxoCollection map[wire.Outpoint]*UTXOEntry

func (uc utxoCollection) String() string {
	utxoStrings := make([]string, len(uc))

	i := 0
	for outpoint, utxoEntry := range uc {
		utxoStrings[i] = fmt.Sprintf("(%s, %d) => %d, blueScore: %d",
			outpoint.TxID, outpoint.Index, utxoEntry.amount, utxoEntry.blockBlueScore)
		i++
	}

	// Sort strings for determinism.
	sort.Strings(utxoStrings)

	return fmt.Sprintf("[ %s ]", strings.Join(utxoStrings, ", "))
}

// add adds a new UTXO entry to this collection
func (uc utxoCollection) add(outpoint wire.Outpoint, entry *UTXOEntry) {
	uc[outpoint] = entry
}

// remove removes a UTXO entry from this collection if it exists
func (uc utxoCollection) remove(outpoint wire.Outpoint) {
	delete(uc, outpoint)
}

// get returns the UTXOEntry represented by provided outpoint,
// and a boolean value indicating if said UTXOEntry is in the set or not
func (uc utxoCollection) get(outpoint wire.Outpoint) (*UTXOEntry, bool) {
	entry, ok := uc[outpoint]
	return entry, ok
}

// contains returns a boolean value indicating whether a UTXO entry is in the set
func (uc utxoCollection) contains(outpoint wire.Outpoint) bool {
	_, ok := uc[outpoint]
	return ok
}

// containsWithBlueScore returns a boolean value indicating whether a UTXOEntry
// is in the set and its blue score is equal to the given blue score.
func (uc utxoCollection) containsWithBlueScore(outpoint wire.Outpoint, blueScore uint64) bool {
	entry, ok := uc.get(outpoint)
	return ok && entry.blockBlueScore == blueScore
}

// clone returns a clone of this collection
func (uc utxoCollection) clone() utxoCollection {
	clone := utxoCollection{}
	for outpoint, entry := range uc {
		clone.add(outpoint, entry)
	}

	return clone
}

// UTXODiff represents a diff between two UTXO Sets.
type UTXODiff struct {
	toAdd        utxoCollection
	toRemove     utxoCollection
	diffMultiset *ecc.Multiset
	useMultiset  bool
}

// NewUTXODiffWithoutMultiset creates a new, empty utxoDiff
// without a multiset.
func NewUTXODiffWithoutMultiset() *UTXODiff {
	return &UTXODiff{
		toAdd:       utxoCollection{},
		toRemove:    utxoCollection{},
		useMultiset: false,
	}
}

// NewUTXODiff creates a new, empty utxoDiff.
func NewUTXODiff() *UTXODiff {
	return &UTXODiff{
		toAdd:        utxoCollection{},
		toRemove:     utxoCollection{},
		useMultiset:  true,
		diffMultiset: ecc.NewMultiset(ecc.S256()),
	}
}

// diffFrom returns a new utxoDiff with the difference between this utxoDiff and another
// Assumes that:
// Both utxoDiffs are from the same base
// If a txOut exists in both utxoDiffs, its underlying values would be the same
//
// diffFrom follows a set of rules represented by the following 3 by 3 table:
//
//          |           | this      |           |
// ---------+-----------+-----------+-----------+-----------
//          |           | toAdd     | toRemove  | None
// ---------+-----------+-----------+-----------+-----------
// other    | toAdd     | -         | X         | toAdd
// ---------+-----------+-----------+-----------+-----------
//          | toRemove  | X         | -         | toRemove
// ---------+-----------+-----------+-----------+-----------
//          | None      | toRemove  | toAdd     | -
//
// Key:
// -		Don't add anything to the result
// X		Return an error
// toAdd	Add the UTXO into the toAdd collection of the result
// toRemove	Add the UTXO into the toRemove collection of the result
//
// Examples:
// 1. This diff contains a UTXO in toAdd, and the other diff contains it in toRemove
//    diffFrom results in an error
// 2. This diff contains a UTXO in toRemove, and the other diff does not contain it
//    diffFrom results in the UTXO being added to toAdd
func (d *UTXODiff) diffFrom(other *UTXODiff) (*UTXODiff, error) {
	result := UTXODiff{
		toAdd:       make(utxoCollection, len(d.toRemove)+len(other.toAdd)),
		toRemove:    make(utxoCollection, len(d.toAdd)+len(other.toRemove)),
		useMultiset: d.useMultiset,
	}

	// Note that the following cases are not accounted for, as they are impossible
	// as long as the base utxoSet is the same:
	// - if utxoEntry is in d.toAdd and other.toRemove
	// - if utxoEntry is in d.toRemove and other.toAdd

	// All transactions in d.toAdd:
	// If they are not in other.toAdd - should be added in result.toRemove
	// If they are in other.toRemove - base utxoSet is not the same
	for outpoint, utxoEntry := range d.toAdd {
		if !other.toAdd.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore) {
			result.toRemove.add(outpoint, utxoEntry)
		} else if (d.toRemove.contains(outpoint) && !other.toRemove.contains(outpoint)) ||
			(!d.toRemove.contains(outpoint) && other.toRemove.contains(outpoint)) {
			return nil, errors.New(
				"diffFrom: outpoint both in d.toAdd, other.toAdd, and only one of d.toRemove and other.toRemove")
		}
		if diffEntry, ok := other.toRemove.get(outpoint); ok {
			// An exception is made for entries with unequal blue scores
			// as long as the appropriate entry exists in either d.toRemove
			// or other.toAdd.
			// These are just "updates" to accepted blue score
			if diffEntry.blockBlueScore != utxoEntry.blockBlueScore &&
				(d.toRemove.containsWithBlueScore(outpoint, diffEntry.blockBlueScore) ||
					other.toAdd.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore)) {
				continue
			}
			return nil, errors.Errorf("diffFrom: outpoint %s both in d.toAdd and in other.toRemove", outpoint)
		}
	}

	// All transactions in d.toRemove:
	// If they are not in other.toRemove - should be added in result.toAdd
	// If they are in other.toAdd - base utxoSet is not the same
	for outpoint, utxoEntry := range d.toRemove {
		diffEntry, ok := other.toRemove.get(outpoint)
		if ok {
			// if have the same entry in d.toRemove - simply don't copy.
			// unless existing entry is with different blue score, in this case - this is an error
			if utxoEntry.blockBlueScore != diffEntry.blockBlueScore {
				return nil, errors.New("diffFrom: outpoint both in d.toRemove and other.toRemove with different " +
					"blue scores, with no corresponding entry in d.toAdd")
			}
		} else { // if no existing entry - add to result.toAdd
			result.toAdd.add(outpoint, utxoEntry)
		}

		if !other.toRemove.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore) {
			result.toAdd.add(outpoint, utxoEntry)
		}
		if diffEntry, ok := other.toAdd.get(outpoint); ok {
			// An exception is made for entries with unequal blue scores
			// as long as the appropriate entry exists in either d.toAdd
			// or other.toRemove.
			// These are just "updates" to accepted blue score
			if diffEntry.blockBlueScore != utxoEntry.blockBlueScore &&
				(d.toAdd.containsWithBlueScore(outpoint, diffEntry.blockBlueScore) ||
					other.toRemove.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore)) {
				continue
			}
			return nil, errors.New("diffFrom: outpoint both in d.toRemove and in other.toAdd")
		}
	}

	// All transactions in other.toAdd:
	// If they are not in d.toAdd - should be added in result.toAdd
	for outpoint, utxoEntry := range other.toAdd {
		if !d.toAdd.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore) {
			result.toAdd.add(outpoint, utxoEntry)
		}
	}

	// All transactions in other.toRemove:
	// If they are not in d.toRemove - should be added in result.toRemove
	for outpoint, utxoEntry := range other.toRemove {
		if !d.toRemove.containsWithBlueScore(outpoint, utxoEntry.blockBlueScore) {
			result.toRemove.add(outpoint, utxoEntry)
		}
	}

	if d.useMultiset {
		// Create a new diffMultiset as the subtraction of the two diffs.
		result.diffMultiset = other.diffMultiset.Subtract(d.diffMultiset)
	}

	return &result, nil
}

// withDiffInPlace applies provided diff to this diff in-place, that would be the result if
// first d, and than diff were applied to the same base
func (d *UTXODiff) withDiffInPlace(diff *UTXODiff) error {
	for outpoint, entryToRemove := range diff.toRemove {
		if d.toAdd.containsWithBlueScore(outpoint, entryToRemove.blockBlueScore) {
			// If already exists in toAdd with the same blueScore - remove from toAdd
			d.toAdd.remove(outpoint)
			continue
		}
		if d.toRemove.contains(outpoint) {
			// If already exists - this is an error
			return ruleError(ErrWithDiff, fmt.Sprintf(
				"withDiffInPlace: outpoint %s both in d.toRemove and in diff.toRemove", outpoint))
		}

		// If not exists neither in toAdd nor in toRemove - add to toRemove
		d.toRemove.add(outpoint, entryToRemove)
	}

	for outpoint, entryToAdd := range diff.toAdd {
		if d.toRemove.containsWithBlueScore(outpoint, entryToAdd.blockBlueScore) {
			// If already exists in toRemove with the same blueScore - remove from toRemove
			if d.toAdd.contains(outpoint) && !diff.toRemove.contains(outpoint) {
				return ruleError(ErrWithDiff, fmt.Sprintf(
					"withDiffInPlace: outpoint %s both in d.toAdd and in diff.toAdd with no "+
						"corresponding entry in diff.toRemove", outpoint))
			}
			d.toRemove.remove(outpoint)
			continue
		}
		if existingEntry, ok := d.toAdd.get(outpoint); ok &&
			(existingEntry.blockBlueScore == entryToAdd.blockBlueScore ||
				!diff.toRemove.containsWithBlueScore(outpoint, existingEntry.blockBlueScore)) {
			// If already exists - this is an error
			return ruleError(ErrWithDiff, fmt.Sprintf(
				"withDiffInPlace: outpoint %s both in d.toAdd and in diff.toAdd", outpoint))
		}

		// If not exists neither in toAdd nor in toRemove, or exists in toRemove with different blueScore - add to toAdd
		d.toAdd.add(outpoint, entryToAdd)
	}

	// Apply diff.diffMultiset to d.diffMultiset
	if d.useMultiset {
		d.diffMultiset = d.diffMultiset.Union(diff.diffMultiset)
	}

	return nil
}

// WithDiff applies provided diff to this diff, creating a new utxoDiff, that would be the result if
// first d, and than diff were applied to some base
func (d *UTXODiff) WithDiff(diff *UTXODiff) (*UTXODiff, error) {
	clone := d.clone()

	err := clone.withDiffInPlace(diff)
	if err != nil {
		return nil, err
	}

	return clone, nil
}

// clone returns a clone of this utxoDiff
func (d *UTXODiff) clone() *UTXODiff {
	clone := &UTXODiff{
		toAdd:       d.toAdd.clone(),
		toRemove:    d.toRemove.clone(),
		useMultiset: d.useMultiset,
	}
	if d.useMultiset {
		clone.diffMultiset = d.diffMultiset.Clone()
	}
	return clone
}

// AddEntry adds a UTXOEntry to the diff
//
// If d.useMultiset is true, this function MUST be
// called with the DAG lock held.
func (d *UTXODiff) AddEntry(outpoint wire.Outpoint, entry *UTXOEntry) error {
	if d.toRemove.containsWithBlueScore(outpoint, entry.blockBlueScore) {
		d.toRemove.remove(outpoint)
	} else if _, exists := d.toAdd[outpoint]; exists {
		return errors.Errorf("AddEntry: Cannot add outpoint %s twice", outpoint)
	} else {
		d.toAdd.add(outpoint, entry)
	}

	if d.useMultiset {
		newMs, err := addUTXOToMultiset(d.diffMultiset, entry, &outpoint)
		if err != nil {
			return err
		}
		d.diffMultiset = newMs
	}
	return nil
}

// RemoveEntry removes a UTXOEntry from the diff.
//
// If d.useMultiset is true, this function MUST be
// called with the DAG lock held.
func (d *UTXODiff) RemoveEntry(outpoint wire.Outpoint, entry *UTXOEntry) error {
	if d.toAdd.containsWithBlueScore(outpoint, entry.blockBlueScore) {
		d.toAdd.remove(outpoint)
	} else if _, exists := d.toRemove[outpoint]; exists {
		return errors.Errorf("removeEntry: Cannot remove outpoint %s twice", outpoint)
	} else {
		d.toRemove.add(outpoint, entry)
	}

	if d.useMultiset {
		newMs, err := removeUTXOFromMultiset(d.diffMultiset, entry, &outpoint)
		if err != nil {
			return err
		}
		d.diffMultiset = newMs
	}
	return nil
}

func (d UTXODiff) String() string {
	if d.useMultiset {
		return fmt.Sprintf("toAdd: %s; toRemove: %s, Multiset-Hash: %s", d.toAdd, d.toRemove, d.diffMultiset.Hash())
	}
	return fmt.Sprintf("toAdd: %s; toRemove: %s", d.toAdd, d.toRemove)
}

// UTXOSet represents a set of unspent transaction outputs
// Every DAG has exactly one fullUTXOSet.
// When a new block arrives, it is validated and applied to the fullUTXOSet in the following manner:
// 1. Get the block's PastUTXO:
// 2. Add all the block's transactions to the block's PastUTXO
// 3. For each of the block's parents,
// 3.1. Rebuild their utxoDiff
// 3.2. Set the block as their diffChild
// 4. Create and initialize a new virtual block
// 5. Get the new virtual's PastUTXO
// 6. Rebuild the utxoDiff for all the tips
// 7. Convert (meld) the new virtual's diffUTXOSet into a fullUTXOSet. This updates the DAG's fullUTXOSet
type UTXOSet interface {
	fmt.Stringer
	diffFrom(other UTXOSet) (*UTXODiff, error)
	WithDiff(utxoDiff *UTXODiff) (UTXOSet, error)
	diffFromTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error)
	diffFromAcceptedTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error)
	AddTx(tx *wire.MsgTx, blockBlueScore uint64) (ok bool, err error)
	clone() UTXOSet
	Get(outpoint wire.Outpoint) (*UTXOEntry, bool)
	Multiset() *ecc.Multiset
	WithTransactions(transactions []*wire.MsgTx, blockBlueScore uint64, ignoreDoubleSpends bool) (UTXOSet, error)
}

// diffFromTx is a common implementation for diffFromTx, that works
// for both diff-based and full UTXO sets
// Returns a diff that is equivalent to provided transaction,
// or an error if provided transaction is not valid in the context of this UTXOSet
func diffFromTx(u UTXOSet, tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	diff := NewUTXODiff()
	isCoinbase := tx.IsCoinBase()
	if !isCoinbase {
		for _, txIn := range tx.TxIn {
			if entry, ok := u.Get(txIn.PreviousOutpoint); ok {
				err := diff.RemoveEntry(txIn.PreviousOutpoint, entry)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, ruleError(ErrMissingTxOut, fmt.Sprintf(
					"Transaction %s is invalid because spends outpoint %s that is not in utxo set",
					tx.TxID(), txIn.PreviousOutpoint))
			}
		}
	}
	for i, txOut := range tx.TxOut {
		entry := NewUTXOEntry(txOut, isCoinbase, acceptingBlueScore)
		outpoint := *wire.NewOutpoint(tx.TxID(), uint32(i))
		err := diff.AddEntry(outpoint, entry)
		if err != nil {
			return nil, err
		}
	}
	return diff, nil
}

// diffFromAcceptedTx is a common implementation for diffFromAcceptedTx, that works
// for both diff-based and full UTXO sets.
// Returns a diff that replaces an entry's blockBlueScore with the given acceptingBlueScore.
// Returns an error if the provided transaction's entry is not valid in the context
// of this UTXOSet.
func diffFromAcceptedTx(u UTXOSet, tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	diff := NewUTXODiff()
	isCoinbase := tx.IsCoinBase()
	for i, txOut := range tx.TxOut {
		// Fetch any unaccepted transaction
		existingOutpoint := *wire.NewOutpoint(tx.TxID(), uint32(i))
		existingEntry, ok := u.Get(existingOutpoint)
		if !ok {
			return nil, errors.Errorf("cannot accept outpoint %s because it doesn't exist in the given UTXO", existingOutpoint)
		}

		// Remove unaccepted entries
		err := diff.RemoveEntry(existingOutpoint, existingEntry)
		if err != nil {
			return nil, err
		}

		// Add new entries with their accepting blue score
		newEntry := NewUTXOEntry(txOut, isCoinbase, acceptingBlueScore)
		err = diff.AddEntry(existingOutpoint, newEntry)
		if err != nil {
			return nil, err
		}
	}
	return diff, nil
}

// FullUTXOSet represents a full list of transaction outputs and their values
type FullUTXOSet struct {
	utxoCollection
	UTXOMultiset *ecc.Multiset
}

// NewFullUTXOSet creates a new utxoSet with full list of transaction outputs and their values
func NewFullUTXOSet() *FullUTXOSet {
	return &FullUTXOSet{
		utxoCollection: utxoCollection{},
		UTXOMultiset:   ecc.NewMultiset(ecc.S256()),
	}
}

// newFullUTXOSetFromUTXOCollection converts a utxoCollection to a FullUTXOSet
func newFullUTXOSetFromUTXOCollection(collection utxoCollection) (*FullUTXOSet, error) {
	var err error
	multiset := ecc.NewMultiset(ecc.S256())
	for outpoint, utxoEntry := range collection {
		multiset, err = addUTXOToMultiset(multiset, utxoEntry, &outpoint)
		if err != nil {
			return nil, err
		}
	}
	return &FullUTXOSet{
		utxoCollection: collection,
		UTXOMultiset:   multiset,
	}, nil
}

// diffFrom returns the difference between this utxoSet and another
// diffFrom can only work when other is a diffUTXOSet, and its base utxoSet is this.
func (fus *FullUTXOSet) diffFrom(other UTXOSet) (*UTXODiff, error) {
	otherDiffSet, ok := other.(*DiffUTXOSet)
	if !ok {
		return nil, errors.New("can't diffFrom two fullUTXOSets")
	}

	if otherDiffSet.base != fus {
		return nil, errors.New("can diffFrom only with diffUTXOSet where this fullUTXOSet is the base")
	}

	return otherDiffSet.UTXODiff, nil
}

// WithDiff returns a utxoSet which is a diff between this and another utxoSet
func (fus *FullUTXOSet) WithDiff(other *UTXODiff) (UTXOSet, error) {
	return NewDiffUTXOSet(fus, other.clone()), nil
}

// AddTx adds a transaction to this utxoSet and returns isAccepted=true iff it's valid in this UTXO's context.
// It returns error if something unexpected happens, such as serialization error (isAccepted=false doesn't
// necessarily means there's an error).
//
// This function MUST be called with the DAG lock held.
func (fus *FullUTXOSet) AddTx(tx *wire.MsgTx, blueScore uint64) (isAccepted bool, err error) {
	isCoinbase := tx.IsCoinBase()
	if !isCoinbase {
		if !fus.containsInputs(tx) {
			return false, nil
		}

		for _, txIn := range tx.TxIn {
			outpoint := *wire.NewOutpoint(&txIn.PreviousOutpoint.TxID, txIn.PreviousOutpoint.Index)
			err := fus.removeAndUpdateMultiset(outpoint)
			if err != nil {
				return false, err
			}
		}
	}

	for i, txOut := range tx.TxOut {
		outpoint := *wire.NewOutpoint(tx.TxID(), uint32(i))
		entry := NewUTXOEntry(txOut, isCoinbase, blueScore)

		err := fus.addAndUpdateMultiset(outpoint, entry)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// diffFromTx returns a diff that is equivalent to provided transaction,
// or an error if provided transaction is not valid in the context of this UTXOSet
func (fus *FullUTXOSet) diffFromTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	return diffFromTx(fus, tx, acceptingBlueScore)
}

func (fus *FullUTXOSet) containsInputs(tx *wire.MsgTx) bool {
	for _, txIn := range tx.TxIn {
		outpoint := *wire.NewOutpoint(&txIn.PreviousOutpoint.TxID, txIn.PreviousOutpoint.Index)
		if !fus.contains(outpoint) {
			return false
		}
	}

	return true
}

func (fus *FullUTXOSet) diffFromAcceptedTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	return diffFromAcceptedTx(fus, tx, acceptingBlueScore)
}

// clone returns a clone of this utxoSet
func (fus *FullUTXOSet) clone() UTXOSet {
	return &FullUTXOSet{utxoCollection: fus.utxoCollection.clone(), UTXOMultiset: fus.UTXOMultiset.Clone()}
}

// Get returns the UTXOEntry associated with the given Outpoint, and a boolean indicating if such entry was found
func (fus *FullUTXOSet) Get(outpoint wire.Outpoint) (*UTXOEntry, bool) {
	utxoEntry, ok := fus.utxoCollection[outpoint]
	return utxoEntry, ok
}

// Multiset returns the ecmh-Multiset of this utxoSet
func (fus *FullUTXOSet) Multiset() *ecc.Multiset {
	return fus.UTXOMultiset
}

// addAndUpdateMultiset adds a UTXOEntry to this utxoSet and updates its multiset accordingly
func (fus *FullUTXOSet) addAndUpdateMultiset(outpoint wire.Outpoint, entry *UTXOEntry) error {
	fus.add(outpoint, entry)
	newMs, err := addUTXOToMultiset(fus.UTXOMultiset, entry, &outpoint)
	if err != nil {
		return err
	}
	fus.UTXOMultiset = newMs
	return nil
}

// removeAndUpdateMultiset removes a UTXOEntry from this utxoSet and updates its multiset accordingly
func (fus *FullUTXOSet) removeAndUpdateMultiset(outpoint wire.Outpoint) error {
	entry, ok := fus.Get(outpoint)
	if !ok {
		return errors.Errorf("Couldn't find outpoint %s", outpoint)
	}
	fus.remove(outpoint)
	var err error
	newMs, err := removeUTXOFromMultiset(fus.UTXOMultiset, entry, &outpoint)
	if err != nil {
		return err
	}
	fus.UTXOMultiset = newMs
	return nil
}

// WithTransactions returns a new UTXO Set with the added transactions.
//
// This function MUST be called with the DAG lock held.
func (fus *FullUTXOSet) WithTransactions(transactions []*wire.MsgTx, blockBlueScore uint64, ignoreDoubleSpends bool) (UTXOSet, error) {
	diffSet := NewDiffUTXOSet(fus, NewUTXODiff())
	for _, tx := range transactions {
		isAccepted, err := diffSet.AddTx(tx, blockBlueScore)
		if err != nil {
			return nil, err
		}
		if !ignoreDoubleSpends && !isAccepted {
			return nil, errors.Errorf("Transaction %s is not valid with the current UTXO set", tx.TxID())
		}
	}
	return UTXOSet(diffSet), nil
}

// DiffUTXOSet represents a utxoSet with a base fullUTXOSet and a UTXODiff
type DiffUTXOSet struct {
	base     *FullUTXOSet
	UTXODiff *UTXODiff
}

// NewDiffUTXOSet Creates a new utxoSet based on a base fullUTXOSet and a UTXODiff
func NewDiffUTXOSet(base *FullUTXOSet, diff *UTXODiff) *DiffUTXOSet {
	return &DiffUTXOSet{
		base:     base,
		UTXODiff: diff,
	}
}

// diffFrom returns the difference between this utxoSet and another.
// diffFrom can work if other is this's base fullUTXOSet, or a diffUTXOSet with the same base as this
func (dus *DiffUTXOSet) diffFrom(other UTXOSet) (*UTXODiff, error) {
	otherDiffSet, ok := other.(*DiffUTXOSet)
	if !ok {
		return nil, errors.New("can't diffFrom diffUTXOSet with fullUTXOSet")
	}

	if otherDiffSet.base != dus.base {
		return nil, errors.New("can't diffFrom with another diffUTXOSet with a different base")
	}

	return dus.UTXODiff.diffFrom(otherDiffSet.UTXODiff)
}

// WithDiff return a new utxoSet which is a diffFrom between this and another utxoSet
func (dus *DiffUTXOSet) WithDiff(other *UTXODiff) (UTXOSet, error) {
	diff, err := dus.UTXODiff.WithDiff(other)
	if err != nil {
		return nil, err
	}

	return NewDiffUTXOSet(dus.base, diff), nil
}

// AddTx adds a transaction to this utxoSet and returns true iff it's valid in this UTXO's context.
//
// If dus.UTXODiff.useMultiset is true, this function MUST be
// called with the DAG lock held.
func (dus *DiffUTXOSet) AddTx(tx *wire.MsgTx, blockBlueScore uint64) (bool, error) {
	isCoinbase := tx.IsCoinBase()
	if !isCoinbase && !dus.containsInputs(tx) {
		return false, nil
	}

	err := dus.appendTx(tx, blockBlueScore, isCoinbase)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (dus *DiffUTXOSet) appendTx(tx *wire.MsgTx, blockBlueScore uint64, isCoinbase bool) error {
	if !isCoinbase {
		for _, txIn := range tx.TxIn {
			outpoint := *wire.NewOutpoint(&txIn.PreviousOutpoint.TxID, txIn.PreviousOutpoint.Index)
			entry, ok := dus.Get(outpoint)
			if !ok {
				return errors.Errorf("Couldn't find entry for outpoint %s", outpoint)
			}
			err := dus.UTXODiff.RemoveEntry(outpoint, entry)
			if err != nil {
				return err
			}
		}
	}

	for i, txOut := range tx.TxOut {
		outpoint := *wire.NewOutpoint(tx.TxID(), uint32(i))
		entry := NewUTXOEntry(txOut, isCoinbase, blockBlueScore)

		err := dus.UTXODiff.AddEntry(outpoint, entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dus *DiffUTXOSet) containsInputs(tx *wire.MsgTx) bool {
	for _, txIn := range tx.TxIn {
		outpoint := *wire.NewOutpoint(&txIn.PreviousOutpoint.TxID, txIn.PreviousOutpoint.Index)
		isInBase := dus.base.contains(outpoint)
		isInDiffToAdd := dus.UTXODiff.toAdd.contains(outpoint)
		isInDiffToRemove := dus.UTXODiff.toRemove.contains(outpoint)
		if (!isInBase && !isInDiffToAdd) || (isInDiffToRemove && !(isInBase && isInDiffToAdd)) {
			return false
		}
	}

	return true
}

// meldToBase updates the base fullUTXOSet with all changes in diff
func (dus *DiffUTXOSet) meldToBase() error {
	for outpoint := range dus.UTXODiff.toRemove {
		if _, ok := dus.base.Get(outpoint); ok {
			dus.base.remove(outpoint)
		} else {
			return errors.Errorf("Couldn't remove outpoint %s because it doesn't exist in the DiffUTXOSet base", outpoint)
		}
	}

	for outpoint, utxoEntry := range dus.UTXODiff.toAdd {
		dus.base.add(outpoint, utxoEntry)
	}

	if dus.UTXODiff.useMultiset {
		dus.base.UTXOMultiset = dus.base.UTXOMultiset.Union(dus.UTXODiff.diffMultiset)
	}

	if dus.UTXODiff.useMultiset {
		dus.UTXODiff = NewUTXODiff()
	} else {
		dus.UTXODiff = NewUTXODiffWithoutMultiset()
	}
	return nil
}

// diffFromTx returns a diff that is equivalent to provided transaction,
// or an error if provided transaction is not valid in the context of this UTXOSet
func (dus *DiffUTXOSet) diffFromTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	return diffFromTx(dus, tx, acceptingBlueScore)
}

func (dus *DiffUTXOSet) diffFromAcceptedTx(tx *wire.MsgTx, acceptingBlueScore uint64) (*UTXODiff, error) {
	return diffFromAcceptedTx(dus, tx, acceptingBlueScore)
}

func (dus *DiffUTXOSet) String() string {
	return fmt.Sprintf("{Base: %s, To Add: %s, To Remove: %s, Multiset-Hash:%s}", dus.base, dus.UTXODiff.toAdd, dus.UTXODiff.toRemove, dus.Multiset().Hash())
}

// clone returns a clone of this UTXO Set
func (dus *DiffUTXOSet) clone() UTXOSet {
	return NewDiffUTXOSet(dus.base.clone().(*FullUTXOSet), dus.UTXODiff.clone())
}

// Get returns the UTXOEntry associated with provided outpoint in this UTXOSet.
// Returns false in second output if this UTXOEntry was not found
func (dus *DiffUTXOSet) Get(outpoint wire.Outpoint) (*UTXOEntry, bool) {
	if toRemoveEntry, ok := dus.UTXODiff.toRemove.get(outpoint); ok {
		// An exception is made for entries with unequal blue scores
		// These are just "updates" to accepted blue score
		if toAddEntry, ok := dus.UTXODiff.toAdd.get(outpoint); ok && toAddEntry.blockBlueScore != toRemoveEntry.blockBlueScore {
			return toAddEntry, true
		}
		return nil, false
	}
	if txOut, ok := dus.base.get(outpoint); ok {
		return txOut, true
	}
	txOut, ok := dus.UTXODiff.toAdd.get(outpoint)
	return txOut, ok
}

// Multiset returns the ecmh-Multiset of this utxoSet
func (dus *DiffUTXOSet) Multiset() *ecc.Multiset {
	return dus.base.UTXOMultiset.Union(dus.UTXODiff.diffMultiset)
}

// WithTransactions returns a new UTXO Set with the added transactions.
//
// If dus.UTXODiff.useMultiset is true, this function MUST be
// called with the DAG lock held.
func (dus *DiffUTXOSet) WithTransactions(transactions []*wire.MsgTx, blockBlueScore uint64, ignoreDoubleSpends bool) (UTXOSet, error) {
	diffSet := NewDiffUTXOSet(dus.base, dus.UTXODiff.clone())
	for _, tx := range transactions {
		isAccepted, err := diffSet.AddTx(tx, blockBlueScore)
		if err != nil {
			return nil, err
		}
		if !ignoreDoubleSpends && !isAccepted {
			return nil, errors.Errorf("Transaction %s is not valid with the current UTXO set", tx.TxID())
		}
	}
	return UTXOSet(diffSet), nil
}

func addUTXOToMultiset(ms *ecc.Multiset, entry *UTXOEntry, outpoint *wire.Outpoint) (*ecc.Multiset, error) {
	utxoMS, err := utxoMultiset(entry, outpoint)
	if err != nil {
		return nil, err
	}
	return ms.Union(utxoMS), nil
}

func removeUTXOFromMultiset(ms *ecc.Multiset, entry *UTXOEntry, outpoint *wire.Outpoint) (*ecc.Multiset, error) {
	utxoMS, err := utxoMultiset(entry, outpoint)
	if err != nil {
		return nil, err
	}
	return ms.Subtract(utxoMS), nil
}
