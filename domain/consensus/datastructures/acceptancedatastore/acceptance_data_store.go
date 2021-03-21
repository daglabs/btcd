package acceptancedatastore

import (
	"github.com/kaspanet/kaspad/domain/consensus/database"
	"github.com/kaspanet/kaspad/domain/consensus/database/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/lrucache"
	"google.golang.org/protobuf/proto"
)

var bucket = database.MakeBucket([]byte("acceptance-data"))

// acceptanceDataStore represents a store of AcceptanceData
type acceptanceDataStore struct {
	cache *lrucache.LRUCache
}

// New instantiates a new AcceptanceDataStore
func New(cacheSize int, preallocate bool) model.AcceptanceDataStore {
	return &acceptanceDataStore{
		cache: lrucache.New(cacheSize, preallocate),
	}
}

// Stage stages the given acceptanceData for the given blockHash
func (ads *acceptanceDataStore) Stage(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash, acceptanceData externalapi.AcceptanceData) {
	stagingShard := ads.stagingShard(stagingArea)
	stagingShard.toAdd[*blockHash] = acceptanceData.Clone()
}

func (ads *acceptanceDataStore) IsStaged(stagingArea *model.StagingArea) bool {
	stagingShard := ads.stagingShard(stagingArea)
	return len(stagingShard.toAdd) != 0 || len(stagingShard.toDelete) != 0
}

// Get gets the acceptanceData associated with the given blockHash
func (ads *acceptanceDataStore) Get(dbContext model.DBReader, stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) (externalapi.AcceptanceData, error) {
	stagingShard := ads.stagingShard(stagingArea)

	if acceptanceData, ok := stagingShard.toAdd[*blockHash]; ok {
		return acceptanceData.Clone(), nil
	}

	if acceptanceData, ok := ads.cache.Get(blockHash); ok {
		return acceptanceData.(externalapi.AcceptanceData).Clone(), nil
	}

	acceptanceDataBytes, err := dbContext.Get(ads.hashAsKey(blockHash))
	if err != nil {
		return nil, err
	}

	acceptanceData, err := ads.deserializeAcceptanceData(acceptanceDataBytes)
	if err != nil {
		return nil, err
	}
	ads.cache.Add(blockHash, acceptanceData)
	return acceptanceData.Clone(), nil
}

// Delete deletes the acceptanceData associated with the given blockHash
func (ads *acceptanceDataStore) Delete(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) {
	stagingShard := ads.stagingShard(stagingArea)

	if _, ok := stagingShard.toAdd[*blockHash]; ok {
		delete(stagingShard.toAdd, *blockHash)
		return
	}
	stagingShard.toDelete[*blockHash] = struct{}{}
}

func (ads *acceptanceDataStore) serializeAcceptanceData(acceptanceData externalapi.AcceptanceData) ([]byte, error) {
	dbAcceptanceData := serialization.DomainAcceptanceDataToDbAcceptanceData(acceptanceData)
	return proto.Marshal(dbAcceptanceData)
}

func (ads *acceptanceDataStore) deserializeAcceptanceData(acceptanceDataBytes []byte) (externalapi.AcceptanceData, error) {
	dbAcceptanceData := &serialization.DbAcceptanceData{}
	err := proto.Unmarshal(acceptanceDataBytes, dbAcceptanceData)
	if err != nil {
		return nil, err
	}
	return serialization.DbAcceptanceDataToDomainAcceptanceData(dbAcceptanceData)
}

func (ads *acceptanceDataStore) hashAsKey(hash *externalapi.DomainHash) model.DBKey {
	return bucket.Key(hash.ByteSlice())
}