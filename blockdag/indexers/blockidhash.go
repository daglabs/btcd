package indexers

import (
	"fmt"
	"github.com/daglabs/btcd/database"
	"github.com/daglabs/btcd/util/daghash"
)

var(

// idByHashIndexBucketName is the name of the db bucket used to house
// the block hash -> block id index.
idByHashIndexBucketName = []byte("idbyhashidx")

// hashByIDIndexBucketName is the name of the db bucket used to house
// the block id -> block hash index.
hashByIDIndexBucketName = []byte("hashbyididx")

)

const(
	blockIDSize = 8 // 8 bytes for block ID
	blueScoreSize = 8 // 8 bytes for blue score
)

// dbFetchBlockIDByHash uses an existing database transaction to retrieve the
// block id for the provided hash from the index.
func dbFetchBlockIDByHash(dbTx database.Tx, hash *daghash.Hash) (uint64, error) {
	hashIndex := dbTx.Metadata().Bucket(idByHashIndexBucketName)
	serializedID := hashIndex.Get(hash[:])
	if serializedID == nil {
		return 0, fmt.Errorf("no entry in the block ID index for block with hash %s", hash)
	}

	return deserializeBlockID(serializedID), nil
}

// dropBlockIDIndex drops the internal block id index.
func dropBlockIDIndex(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(idByHashIndexBucketName)
		if err != nil {
			return err
		}

		return meta.DeleteBucket(hashByIDIndexBucketName)
	})
}

// dbFetchBlockHashBySerializedID uses an existing database transaction to
// retrieve the hash for the provided serialized block id from the index.
func dbFetchBlockHashBySerializedID(dbTx database.Tx, serializedID []byte) (*daghash.Hash, error) {
	idIndex := dbTx.Metadata().Bucket(hashByIDIndexBucketName)
	hashBytes := idIndex.Get(serializedID)
	if hashBytes == nil {
		return nil, fmt.Errorf("no entry in the block ID index for block with id %d", byteOrder.Uint64(serializedID))
	}

	var hash daghash.Hash
	copy(hash[:], hashBytes)
	return &hash, nil
}

// dbPutBlockIDIndexEntry uses an existing database transaction to update or add
// the index entries for the hash to id and id to hash mappings for the provided
// values.
func dbPutBlockIDIndexEntry(dbTx database.Tx, hash *daghash.Hash, serializedID []byte) error {
	// Add the block hash to ID mapping to the index.
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(idByHashIndexBucketName)
	if err := hashIndex.Put(hash[:], serializedID[:]); err != nil {
		return err
	}

	// Add the block ID to hash mapping to the index.
	idIndex := meta.Bucket(hashByIDIndexBucketName)
	return idIndex.Put(serializedID[:], hash[:])
}

func dbFetchCurrentBlockID(dbTx database.Tx) uint64 {
	serializedID := dbTx.Metadata().Get(currentBlockIDKey)
	if serializedID == nil{
		return 0
	}
	return deserializeBlockID(serializedID)
}

func deserializeBlockID(serializedID []byte) uint64{
	return byteOrder.Uint64(serializedID)
}

func serializeBlockID(blockID uint64) []byte{
	serializedBlockID := make([]byte, blockIDSize)
	byteOrder.PutUint64(serializedBlockID, blockID)
	return serializedBlockID
}

// dbFetchBlockHashByID uses an existing database transaction to retrieve the
// hash for the provided block id from the index.
func dbFetchBlockHashByID(dbTx database.Tx, id uint64) (*daghash.Hash, error) {
	return dbFetchBlockHashBySerializedID(dbTx, serializeBlockID(id))
}