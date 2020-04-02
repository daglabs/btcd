package ldb

import (
	"bytes"
	"encoding/hex"
	"github.com/kaspanet/kaspad/database"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDBCursor is a thin wrapper around native leveldb iterators.
type LevelDBCursor struct {
	ldbIterator iterator.Iterator
	prefix      []byte

	isClosed bool
}

// Cursor begins a new cursor over the given prefix.
func (db *LevelDB) Cursor(prefix []byte) *LevelDBCursor {
	ldbIterator := db.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	return &LevelDBCursor{
		ldbIterator: ldbIterator,
		prefix:      prefix,
		isClosed:    false,
	}
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted. Returns false if the cursor is closed.
func (c *LevelDBCursor) Next() bool {
	if c.isClosed {
		return false
	}
	return c.ldbIterator.Next()
}

// First moves the iterator to the first key/value pair. It returns false if
// such a pair does not exist or if the cursor is closed.
func (c *LevelDBCursor) First() bool {
	if c.isClosed {
		return false
	}
	return c.ldbIterator.First()
}

// Seek moves the iterator to the first key/value pair whose key is greater
// than or equal to the given key. It returns ErrNotFound if such pair does not
// exist.
func (c *LevelDBCursor) Seek(key []byte) error {
	if c.isClosed {
		return errors.New("cannot seek a closed cursor")
	}

	notFoundErr := errors.Wrapf(database.ErrNotFound, "key %s not "+
		"found", hex.EncodeToString(key))
	found := c.ldbIterator.Seek(key)
	if !found {
		return notFoundErr
	}

	// Use c.ldbIterator.Key because c.Key removes the prefix from the key
	currentKey := c.ldbIterator.Key()
	if currentKey == nil {
		return notFoundErr
	}
	if !bytes.Equal(currentKey, key) {
		return notFoundErr
	}

	return nil
}

// Key returns the key of the current key/value pair, or ErrNotFound if done.
// Note that the key is trimmed to not include the prefix the cursor was opened
// with. The caller should not modify the contents of the returned slice, and
// its contents may change on the next call to Next.
func (c *LevelDBCursor) Key() ([]byte, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the key of a closed cursor")
	}
	fullKeyPath := c.ldbIterator.Key()
	if fullKeyPath == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the "+
			"key of a done cursor")
	}
	key := bytes.TrimPrefix(fullKeyPath, c.prefix)
	return key, nil
}

// Value returns the value of the current key/value pair, or ErrNotFound if done.
// The caller should not modify the contents of the returned slice, and its
// contents may change on the next call to Next.
func (c *LevelDBCursor) Value() ([]byte, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the value of a closed cursor")
	}
	value := c.ldbIterator.Value()
	if value == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the "+
			"value of a done cursor")
	}
	return value, nil
}

// Close releases associated resources.
func (c *LevelDBCursor) Close() error {
	if c.isClosed {
		return errors.New("cannot close an already closed cursor")
	}
	c.isClosed = true

	c.ldbIterator.Release()
	return nil
}
