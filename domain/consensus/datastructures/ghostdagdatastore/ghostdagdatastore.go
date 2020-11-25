package ghostdagdatastore

import (
	"github.com/golang/protobuf/proto"
	"github.com/kaspanet/kaspad/domain/consensus/database/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/dbkeys"
	"github.com/kaspanet/kaspad/domain/consensus/utils/lrucache"
)

var bucket = dbkeys.MakeBucket([]byte("block-ghostdag-data"))

// ghostdagDataStore represents a store of BlockGHOSTDAGData
type ghostdagDataStore struct {
	staging map[externalapi.DomainHash]*model.BlockGHOSTDAGData
	cache   *lrucache.LRUCache
}

// New instantiates a new GHOSTDAGDataStore
func New(cacheSize int) model.GHOSTDAGDataStore {
	return &ghostdagDataStore{
		staging: make(map[externalapi.DomainHash]*model.BlockGHOSTDAGData),
		cache:   lrucache.New(cacheSize),
	}
}

// Stage stages the given blockGHOSTDAGData for the given blockHash
func (gds *ghostdagDataStore) Stage(blockHash *externalapi.DomainHash, blockGHOSTDAGData *model.BlockGHOSTDAGData) {
	gds.staging[*blockHash] = blockGHOSTDAGData.Clone()
}

func (gds *ghostdagDataStore) IsStaged() bool {
	return len(gds.staging) != 0
}

func (gds *ghostdagDataStore) Discard() {
	gds.staging = make(map[externalapi.DomainHash]*model.BlockGHOSTDAGData)
}

func (gds *ghostdagDataStore) Commit(dbTx model.DBTransaction) error {
	for hash, blockGHOSTDAGData := range gds.staging {
		blockGhostdagDataBytes, err := gds.serializeBlockGHOSTDAGData(blockGHOSTDAGData)
		if err != nil {
			return err
		}
		err = dbTx.Put(gds.hashAsKey(&hash), blockGhostdagDataBytes)
		if err != nil {
			return err
		}
		gds.cache.Add(&hash, blockGHOSTDAGData)
	}

	gds.Discard()
	return nil
}

// Get gets the blockGHOSTDAGData associated with the given blockHash
func (gds *ghostdagDataStore) Get(dbContext model.DBReader, blockHash *externalapi.DomainHash) (*model.BlockGHOSTDAGData, error) {
	if blockGHOSTDAGData, ok := gds.staging[*blockHash]; ok {
		return blockGHOSTDAGData.Clone(), nil
	}

	if blockGHOSTDAGData, ok := gds.cache.Get(blockHash); ok {
		return blockGHOSTDAGData.(*model.BlockGHOSTDAGData).Clone(), nil
	}

	blockGHOSTDAGDataBytes, err := dbContext.Get(gds.hashAsKey(blockHash))
	if err != nil {
		return nil, err
	}

	blockGHOSTDAGData, err := gds.deserializeBlockGHOSTDAGData(blockGHOSTDAGDataBytes)
	if err != nil {
		return nil, err
	}
	gds.cache.Add(blockHash, blockGHOSTDAGData)
	return blockGHOSTDAGData.Clone(), nil
}

func (gds *ghostdagDataStore) hashAsKey(hash *externalapi.DomainHash) model.DBKey {
	return bucket.Key(hash[:])
}

func (gds *ghostdagDataStore) serializeBlockGHOSTDAGData(blockGHOSTDAGData *model.BlockGHOSTDAGData) ([]byte, error) {
	return proto.Marshal(serialization.BlockGHOSTDAGDataToDBBlockGHOSTDAGData(blockGHOSTDAGData))
}

func (gds *ghostdagDataStore) deserializeBlockGHOSTDAGData(blockGHOSTDAGDataBytes []byte) (*model.BlockGHOSTDAGData, error) {
	dbBlockGHOSTDAGData := &serialization.DbBlockGhostdagData{}
	err := proto.Unmarshal(blockGHOSTDAGDataBytes, dbBlockGHOSTDAGData)
	if err != nil {
		return nil, err
	}

	return serialization.DBBlockGHOSTDAGDataToBlockGHOSTDAGData(dbBlockGHOSTDAGData)
}
