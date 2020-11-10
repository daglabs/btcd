package blockheaderstore

import (
	"github.com/golang/protobuf/proto"
	"github.com/kaspanet/kaspad/domain/consensus/database/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/dbkeys"
)

var bucket = dbkeys.MakeBucket([]byte("block-headers"))

// blockHeaderStore represents a store of blocks
type blockHeaderStore struct {
	staging  map[externalapi.DomainHash]*externalapi.DomainBlockHeader
	toDelete map[externalapi.DomainHash]struct{}
}

// New instantiates a new BlockHeaderStore
func New() model.BlockHeaderStore {
	return &blockHeaderStore{
		staging:  make(map[externalapi.DomainHash]*externalapi.DomainBlockHeader),
		toDelete: make(map[externalapi.DomainHash]struct{}),
	}
}

// Stage stages the given block header for the given blockHash
func (bms *blockHeaderStore) Stage(blockHash *externalapi.DomainHash, blockHeader *externalapi.DomainBlockHeader) error {
	clone, err := bms.cloneHeader(blockHeader)
	if err != nil {
		return err
	}

	bms.staging[*blockHash] = clone
	return nil
}

func (bms *blockHeaderStore) IsStaged() bool {
	return len(bms.staging) != 0 || len(bms.toDelete) != 0
}

func (bms *blockHeaderStore) Discard() {
	bms.staging = make(map[externalapi.DomainHash]*externalapi.DomainBlockHeader)
	bms.toDelete = make(map[externalapi.DomainHash]struct{})
}

func (bms *blockHeaderStore) Commit(dbTx model.DBTransaction) error {
	for hash, header := range bms.staging {
		headerBytes, err := bms.serializeHeader(header)
		if err != nil {
			return err
		}
		err = dbTx.Put(bms.hashAsKey(&hash), headerBytes)
		if err != nil {
			return err
		}
	}

	for hash := range bms.toDelete {
		err := dbTx.Delete(bms.hashAsKey(&hash))
		if err != nil {
			return err
		}
	}

	bms.Discard()
	return nil
}

// BlockHeader gets the block header associated with the given blockHash
func (bms *blockHeaderStore) BlockHeader(dbContext model.DBReader, blockHash *externalapi.DomainHash) (*externalapi.DomainBlockHeader, error) {
	if header, ok := bms.staging[*blockHash]; ok {
		return header, nil
	}

	headerBytes, err := dbContext.Get(bms.hashAsKey(blockHash))
	if err != nil {
		return nil, err
	}

	return bms.deserializeHeader(headerBytes)
}

// HasBlock returns whether a block header with a given hash exists in the store.
func (bms *blockHeaderStore) HasBlockHeader(dbContext model.DBReader, blockHash *externalapi.DomainHash) (bool, error) {
	if _, ok := bms.staging[*blockHash]; ok {
		return true, nil
	}

	exists, err := dbContext.Has(bms.hashAsKey(blockHash))
	if err != nil {
		return false, err
	}

	return exists, nil
}

// BlockHeaders gets the block headers associated with the given blockHashes
func (bms *blockHeaderStore) BlockHeaders(dbContext model.DBReader, blockHashes []*externalapi.DomainHash) ([]*externalapi.DomainBlockHeader, error) {
	headers := make([]*externalapi.DomainBlockHeader, len(blockHashes))
	for i, hash := range blockHashes {
		var err error
		headers[i], err = bms.BlockHeader(dbContext, hash)
		if err != nil {
			return nil, err
		}
	}
	return headers, nil
}

// Delete deletes the block associated with the given blockHash
func (bms *blockHeaderStore) Delete(blockHash *externalapi.DomainHash) {
	if _, ok := bms.staging[*blockHash]; ok {
		delete(bms.staging, *blockHash)
		return
	}
	bms.toDelete[*blockHash] = struct{}{}
}

func (bms *blockHeaderStore) hashAsKey(hash *externalapi.DomainHash) model.DBKey {
	return bucket.Key(hash[:])
}

func (bms *blockHeaderStore) serializeHeader(header *externalapi.DomainBlockHeader) ([]byte, error) {
	dbBlockHeader := serialization.DomainBlockHeaderToDbBlockHeader(header)
	return proto.Marshal(dbBlockHeader)
}

func (bms *blockHeaderStore) deserializeHeader(headerBytes []byte) (*externalapi.DomainBlockHeader, error) {
	dbBlockHeader := &serialization.DbBlockHeader{}
	err := proto.Unmarshal(headerBytes, dbBlockHeader)
	if err != nil {
		return nil, err
	}
	return serialization.DbBlockHeaderToDomainBlockHeader(dbBlockHeader)
}

func (bms *blockHeaderStore) cloneHeader(header *externalapi.DomainBlockHeader) (*externalapi.DomainBlockHeader, error) {
	serialized, err := bms.serializeHeader(header)
	if err != nil {
		return nil, err
	}

	return bms.deserializeHeader(serialized)
}
