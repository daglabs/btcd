package acceptancedatastore

import (
	"github.com/kaspanet/kaspad/domain/consensus/database/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/dbkeys"
	"github.com/kaspanet/kaspad/domain/consensus/utils/lrucache"
	"google.golang.org/protobuf/proto"
)

var bucket = dbkeys.MakeBucket([]byte("acceptance-data"))

// acceptanceDataStore represents a store of AcceptanceData
type acceptanceDataStore struct {
	staging  map[externalapi.DomainHash]externalapi.AcceptanceData
	toDelete map[externalapi.DomainHash]struct{}
	cache    *lrucache.LRUCache
}

// New instantiates a new AcceptanceDataStore
func New(cacheSize int) model.AcceptanceDataStore {
	return &acceptanceDataStore{
		staging:  make(map[externalapi.DomainHash]externalapi.AcceptanceData),
		toDelete: make(map[externalapi.DomainHash]struct{}),
		cache:    lrucache.New(cacheSize),
	}
}

// Stage stages the given acceptanceData for the given blockHash
func (ads *acceptanceDataStore) Stage(blockHash *externalapi.DomainHash, acceptanceData externalapi.AcceptanceData) {
	ads.staging[*blockHash] = acceptanceData.Clone()
}

func (ads *acceptanceDataStore) IsStaged() bool {
	return len(ads.staging) != 0 || len(ads.toDelete) != 0
}

func (ads *acceptanceDataStore) Discard() {
	ads.staging = make(map[externalapi.DomainHash]externalapi.AcceptanceData)
	ads.toDelete = make(map[externalapi.DomainHash]struct{})
}

func (ads *acceptanceDataStore) Commit(dbTx model.DBTransaction) error {
	for hash, acceptanceData := range ads.staging {
		acceptanceDataBytes, err := ads.serializeAcceptanceData(acceptanceData)
		if err != nil {
			return err
		}
		err = dbTx.Put(ads.hashAsKey(&hash), acceptanceDataBytes)
		if err != nil {
			return err
		}
		ads.cache.Add(&hash, acceptanceData)
	}

	for hash := range ads.toDelete {
		err := dbTx.Delete(ads.hashAsKey(&hash))
		if err != nil {
			return err
		}
		ads.cache.Remove(&hash)
	}

	ads.Discard()
	return nil
}

// Get gets the acceptanceData associated with the given blockHash
func (ads *acceptanceDataStore) Get(dbContext model.DBReader, blockHash *externalapi.DomainHash) (externalapi.AcceptanceData, error) {
	if acceptanceData, ok := ads.staging[*blockHash]; ok {
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
func (ads *acceptanceDataStore) Delete(blockHash *externalapi.DomainHash) {
	if _, ok := ads.staging[*blockHash]; ok {
		delete(ads.staging, *blockHash)
		return
	}
	ads.toDelete[*blockHash] = struct{}{}
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
	return bucket.Key(hash[:])
}
