package consensusstatestore

import (
	"github.com/kaspanet/kaspad/domain/consensus/database"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/pkg/errors"
)

var importingPruningPointUTXOSetKey = database.MakeBucket(nil).Key([]byte("importing-pruning-point-utxo-set"))

func (css *consensusStateStore) StartImportingPruningPointUTXOSet() error {
	return css.databaseContext.Put(importingPruningPointUTXOSetKey, []byte{0})
}

func (css *consensusStateStore) HadStartedImportingPruningPointUTXOSet() (bool, error) {
	return css.databaseContext.Has(importingPruningPointUTXOSetKey)
}

func (css *consensusStateStore) ImportPruningPointUTXOSetIntoVirtualUTXOSet(pruningPointUTXOSetIterator model.ReadOnlyUTXOSetIterator) error {
	if css.virtualUTXODiffStaging != nil {
		return errors.New("cannot import virtual UTXO set while virtual UTXO diff is staged")
	}

	// Clear the cache
	css.virtualUTXOSetCache.Clear()

	// Delete all the old UTXOs from the database
	deleteCursor, err := css.databaseContext.Cursor(utxoSetBucket)
	if err != nil {
		return err
	}
	for deleteCursor.Next() {
		key, err := deleteCursor.Key()
		if err != nil {
			return err
		}
		err = css.databaseContext.Delete(key)
		if err != nil {
			return err
		}
	}

	// Insert all the new UTXOs into the database
	pruningPointUTXOSetIterator.First()
	for pruningPointUTXOSetIterator.Next() {
		outpoint, entry, err := pruningPointUTXOSetIterator.Get()
		if err != nil {
			return err
		}

		key, err := utxoKey(outpoint)
		if err != nil {
			return err
		}
		serializedUTXOEntry, err := serializeUTXOEntry(entry)
		if err != nil {
			return err
		}

		err = css.databaseContext.Put(key, serializedUTXOEntry)
		if err != nil {
			return err
		}
	}

	return nil
}

func (css *consensusStateStore) FinishImportingPruningPointUTXOSet() error {
	return css.databaseContext.Delete(importingPruningPointUTXOSetKey)
}
