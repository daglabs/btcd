package ffldb

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/btcsuite/goleveldb/leveldb"
	"github.com/btcsuite/goleveldb/leveldb/filter"
	"github.com/btcsuite/goleveldb/leveldb/opt"
	"github.com/daglabs/kaspad/wire"
)

func newTestDb(testName string, t *testing.T) *db {
	dbPath := path.Join(os.TempDir(), "db_test", testName)
	err := os.RemoveAll(dbPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("%s: Error deleting database folder before starting: %s", testName, err)
	}

	network := wire.SimNet

	opts := opt.Options{
		ErrorIfExist: true,
		Strict:       opt.DefaultStrict,
		Compression:  opt.NoCompression,
		Filter:       filter.NewBloomFilter(10),
	}
	metadataDbPath := filepath.Join(dbPath, metadataDbName)
	ldb, err := leveldb.OpenFile(metadataDbPath, &opts)
	if err != nil {
		t.Errorf("%s: Error opening metadataDbPath: %s", testName, err)
	}
	err = initDB(ldb)
	if err != nil {
		t.Errorf("%s: Error initializing metadata Db: %s", testName, err)
	}

	store := newBlockStore(dbPath, network)
	cache := newDbCache(ldb, store, defaultCacheSize, defaultFlushSecs)
	return &db{store: store, cache: cache}
}
