// Copyright (c) 2015-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/dbaccess"
	"github.com/pkg/errors"
	"io"
	"sync"

	"github.com/kaspanet/kaspad/database"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/binaryserializer"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
)

var (
	// utxoDiffsBucketName is the name of the database bucket used to house the
	// diffs and diff children of blocks.
	utxoDiffsBucketName = []byte("utxodiffs")

	// reachabilityDataBucketName is the name of the database bucket used to house the
	// reachability tree nodes and future covering sets of blocks.
	reachabilityDataBucketName = []byte("reachability")

	// subnetworksBucketName is the name of the database bucket used to store the
	// subnetwork registry.
	subnetworksBucketName = []byte("subnetworks")

	// byteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	byteOrder = binary.LittleEndian
)

// errNotInDAG signifies that a block hash or height that is not in the
// DAG was requested.
type errNotInDAG string

// Error implements the error interface.
func (e errNotInDAG) Error() string {
	return string(e)
}

// isNotInDAGErr returns whether or not the passed error is an
// errNotInDAG error.
func isNotInDAGErr(err error) bool {
	var notInDAGErr errNotInDAG
	return errors.As(err, &notInDAGErr)
}

// errDeserialize signifies that a problem was encountered when deserializing
// data.
type errDeserialize string

// Error implements the error interface.
func (e errDeserialize) Error() string {
	return string(e)
}

// isDeserializeErr returns whether or not the passed error is an errDeserialize
// error.
func isDeserializeErr(err error) bool {
	var deserializeErr errDeserialize
	return errors.As(err, &deserializeErr)
}

// -----------------------------------------------------------------------------
// The unspent transaction output (UTXO) set consists of an entry for each
// unspent output using a format that is optimized to reduce space using domain
// specific compression algorithms.
//
// Each entry is keyed by an outpoint as specified below. It is important to
// note that the key encoding uses a VLQ, which employs an MSB encoding so
// iteration of UTXOs when doing byte-wise comparisons will produce them in
// order.
//
// The serialized key format is:
//   <hash><output index>
//
//   Field                Type             Size
//   hash                 daghash.Hash   daghash.HashSize
//   output index         VLQ              variable
//
// The serialized value format is:
//
//   <header code><compressed txout>
//
//   Field                Type     Size
//   header code          VLQ      variable
//   compressed txout
//     compressed amount  VLQ      variable
//     compressed script  []byte   variable
//
// The serialized header code format is:
//   bit 0 - containing transaction is a coinbase
//   bits 1-x - height of the block that contains the unspent txout
//
// Example 1:
// b7c3332bc138e2c9429818f5fed500bcc1746544218772389054dc8047d7cd3f:0
//
//    03320496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52
//    <><------------------------------------------------------------------>
//     |                                          |
//   header code                         compressed txout
//
//  - header code: 0x03 (coinbase, height 1)
//  - compressed txout:
//    - 0x32: VLQ-encoded compressed amount for 5000000000 (50 KAS)
//    - 0x04: special script type pay-to-pubkey
//    - 0x96...52: x-coordinate of the pubkey
//
// Example 2:
// 4a16969aa4764dd7507fc1de7f0baa4850a246de90c45e59a3207f9a26b5036f:2
//
//    8cf316800900b8025be1b3efc63b0ad48e7f9f10e87544528d58
//    <----><------------------------------------------>
//      |                             |
//   header code             compressed txout
//
//  - header code: 0x8cf316 (not coinbase, height 113931)
//  - compressed txout:
//    - 0x8009: VLQ-encoded compressed amount for 15000000 (0.15 KAS)
//    - 0x00: special script type pay-to-pubkey-hash
//    - 0xb8...58: pubkey hash
//
// Example 3:
// 1b02d1c8cfef60a189017b9a420c682cf4a0028175f2f563209e4ff61c8c3620:22
//
//    a8a2588ba5b9e763011dd46a006572d820e448e12d2bbb38640bc718e6
//    <----><-------------------------------------------------->
//      |                             |
//   header code             compressed txout
//
//  - header code: 0xa8a258 (not coinbase, height 338156)
//  - compressed txout:
//    - 0x8ba5b9e763: VLQ-encoded compressed amount for 366875659 (3.66875659 KAS)
//    - 0x01: special script type pay-to-script-hash
//    - 0x1d...e6: script hash
// -----------------------------------------------------------------------------

// maxUint32VLQSerializeSize is the maximum number of bytes a max uint32 takes
// to serialize as a VLQ.
var maxUint32VLQSerializeSize = serializeSizeVLQ(1<<32 - 1)

// outpointKeyPool defines a concurrent safe free list of byte slices used to
// provide temporary buffers for outpoint database keys.
var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, daghash.HashSize+maxUint32VLQSerializeSize)
		return &b // Pointer to slice to avoid boxing alloc.
	},
}

// outpointKey returns a key suitable for use as a database key in the UTXO set
// while making use of a free list. A new buffer is allocated if there are not
// already any available on the free list. The returned byte slice should be
// returned to the free list by using the recycleOutpointKey function when the
// caller is done with it _unless_ the slice will need to live for longer than
// the caller can calculate such as when used to write to the database.
func outpointKey(outpoint wire.Outpoint) *[]byte {
	// A VLQ employs an MSB encoding, so they are useful not only to reduce
	// the amount of storage space, but also so iteration of UTXOs when
	// doing byte-wise comparisons will produce them in order.
	key := outpointKeyPool.Get().(*[]byte)
	idx := uint64(outpoint.Index)
	*key = (*key)[:daghash.HashSize+serializeSizeVLQ(idx)]
	copy(*key, outpoint.TxID[:])
	putVLQ((*key)[daghash.HashSize:], idx)
	return key
}

// recycleOutpointKey puts the provided byte slice, which should have been
// obtained via the outpointKey function, back on the free list.
func recycleOutpointKey(key *[]byte) {
	outpointKeyPool.Put(key)
}

// dbUpdateUTXOSet updates the UTXO set in the database based on the provided
// UTXO diff.
func dbUpdateUTXOSet(context dbaccess.Context, virtualUTXODiff *UTXODiff) error {
	for outpoint := range virtualUTXODiff.toRemove {
		key := outpointKey(outpoint)
		err := dbaccess.RemoveFromUTXOSet(context, *key)
		recycleOutpointKey(key)
		if err != nil {
			return err
		}
	}

	for outpoint, entry := range virtualUTXODiff.toAdd {
		serialized := serializeUTXOEntry(entry)
		key := outpointKey(outpoint)
		err := dbaccess.AddToUTXOSet(context, *key, serialized)
		recycleOutpointKey(key)
		if err != nil {
			return err
		}
	}

	return nil
}

type dagState struct {
	TipHashes         []*daghash.Hash
	LastFinalityPoint *daghash.Hash
	localSubnetworkID *subnetworkid.SubnetworkID
}

// serializeDAGState returns the serialization of the DAG state.
// This is data to be stored in the DAG state bucket.
func serializeDAGState(state *dagState) ([]byte, error) {
	return json.Marshal(state)
}

// deserializeDAGState deserializes the passed serialized DAG state.
// This is data stored in the DAG state bucket and is updated after
// every block is connected to the DAG.
func deserializeDAGState(serializedData []byte) (*dagState, error) {
	var state *dagState
	err := json.Unmarshal(serializedData, &state)
	if err != nil {
		return nil, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt DAG state",
		}
	}

	return state, nil
}

// dbPutDAGState uses an existing database transaction to store the latest
// tip hashes of the DAG.
func dbPutDAGState(context dbaccess.Context, state *dagState) error {
	serializedDAGState, err := serializeDAGState(state)
	if err != nil {
		return err
	}

	return dbaccess.StoreDAGState(context, serializedDAGState)
}

// createDAGState initializes both the database and the DAG state to the
// genesis block. This includes creating the necessary buckets, so it
// must only be called on an uninitialized database.
func (dag *BlockDAG) createDAGState() error {
	// Create the initial the database DAG state including creating the
	// necessary index buckets and inserting the genesis block.
	err := dag.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

		// Create the buckets that house the utxo diffs.
		_, err := meta.CreateBucket(utxoDiffsBucketName)
		if err != nil {
			return err
		}

		_, err = meta.CreateBucket(reachabilityDataBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the registered subnetworks.
		_, err = meta.CreateBucket(subnetworksBucketName)
		if err != nil {
			return err
		}

		if _, err := meta.CreateBucketIfNotExists(idByHashIndexBucketName); err != nil {
			return err
		}
		if _, err := meta.CreateBucketIfNotExists(hashByIDIndexBucketName); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (dag *BlockDAG) removeDAGState() error {
	err := dag.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

		err := meta.DeleteBucket(utxoDiffsBucketName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(reachabilityDataBucketName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(subnetworksBucketName)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// initDAGState attempts to load and initialize the DAG state from the
// database. When the db does not yet contain any DAG state, both it and the
// DAG state are initialized to the genesis block.
func (dag *BlockDAG) initDAGState() error {
	// Fetch the stored DAG state from the database metadata.
	// When it doesn't exist, it means the database hasn't been
	// initialized for use with the DAG yet.
	serializedDAGState, found, err := dbaccess.FetchDAGState(dbaccess.NoTx())
	if err != nil {
		return err
	}
	if !found {
		// At this point the database has not already been initialized, so
		// initialize both it and the DAG state to the genesis block.
		return dag.createDAGState()
	}

	dagState, err := deserializeDAGState(serializedDAGState)
	if err != nil {
		return err
	}
	if !dagState.localSubnetworkID.IsEqual(dag.subnetworkID) {
		return errors.Errorf("Cannot start kaspad with subnetwork ID %s because"+
			" its database is already built with subnetwork ID %s. If you"+
			" want to switch to a new database, please reset the"+
			" database by starting kaspad with --reset-db flag", dag.subnetworkID, dagState.localSubnetworkID)
	}

	log.Infof("Loading block index...")
	var unprocessedBlockNodes []*blockNode
	blockIndexCursor, err := dbaccess.BlockIndexCursor(dbaccess.NoTx())
	if err != nil {
		return err
	}
	for blockIndexCursor.Next() {
		serializedDBNode, err := blockIndexCursor.Value()
		if err != nil {
			return err
		}
		node, err := dag.deserializeBlockNode(serializedDBNode)
		if err != nil {
			return err
		}

		// Check to see if this node had been stored in the the block DB
		// but not yet accepted. If so, add it to a slice to be processed later.
		if node.status == statusDataStored {
			unprocessedBlockNodes = append(unprocessedBlockNodes, node)
			continue
		}

		// If the node is known to be invalid add it as-is to the block
		// index and continue.
		if node.status.KnownInvalid() {
			dag.index.addNode(node)
			continue
		}

		if dag.blockCount == 0 {
			if !node.hash.IsEqual(dag.dagParams.GenesisHash) {
				return AssertError(fmt.Sprintf("initDAGState: Expected "+
					"first entry in block index to be genesis block, "+
					"found %s", node.hash))
			}
		} else {
			if len(node.parents) == 0 {
				return AssertError(fmt.Sprintf("initDAGState: Could "+
					"not find any parent for block %s", node.hash))
			}
		}

		// Add the node to its parents children, connect it,
		// and add it to the block index.
		node.updateParentsChildren()
		dag.index.addNode(node)

		dag.blockCount++
	}

	// Load all of the headers from the data for the known DAG
	// and construct the block index accordingly. Since the
	// number of nodes are already known, perform a single alloc
	// for them versus a whole bunch of little ones to reduce
	// pressure on the GC.

	// Load all of the known UTXO entries and construct the full
	// UTXO set accordingly. Since the number of entries is already
	// known, perform a single alloc for them versus a whole bunch
	// of little ones to reduce pressure on the GC.
	log.Infof("Loading UTXO set...")

	// Determine how many UTXO entries will be loaded into the index so we can
	// allocate the right amount.
	var utxoEntryCount int32
	cursor, err := dbaccess.UTXOSetCursor(dbaccess.NoTx())
	if err != nil {
		return err
	}
	for cursor.Next() {
		utxoEntryCount++
	}

	fullUTXOCollection := make(utxoCollection, utxoEntryCount)
	for ok := cursor.First(); ok; ok = cursor.Next() {
		key, err := cursor.Key()
		if err != nil {
			return err
		}

		// Deserialize the outpoint
		outpoint, err := deserializeOutpoint(key)
		if err != nil {
			// Ensure any deserialization errors are returned as database
			// corruption errors.
			if isDeserializeErr(err) {
				return database.Error{
					ErrorCode:   database.ErrCorruption,
					Description: fmt.Sprintf("corrupt outpoint: %s", err),
				}
			}

			return err
		}

		value, err := cursor.Value()
		if err != nil {
			return err
		}

		// Deserialize the utxo entry
		entry, err := deserializeUTXOEntry(value)
		if err != nil {
			// Ensure any deserialization errors are returned as database
			// corruption errors.
			if isDeserializeErr(err) {
				return database.Error{
					ErrorCode:   database.ErrCorruption,
					Description: fmt.Sprintf("corrupt utxo entry: %s", err),
				}
			}

			return err
		}

		fullUTXOCollection[*outpoint] = entry
	}

	// Attempt to load the DAG state from the database.
	return dag.db.View(func(dbTx database.Tx) error {
		// Initialize the reachability store
		err = dag.reachabilityStore.init(dbTx)
		if err != nil {
			return err
		}

		// Apply the loaded utxoCollection to the virtual block.
		dag.virtual.utxoSet, err = newFullUTXOSetFromUTXOCollection(fullUTXOCollection)
		if err != nil {
			return AssertError(fmt.Sprintf("Error loading UTXOSet: %s", err))
		}

		// Apply the stored tips to the virtual block.
		tips := newBlockSet()
		for _, tipHash := range dagState.TipHashes {
			tip := dag.index.LookupNode(tipHash)
			if tip == nil {
				return AssertError(fmt.Sprintf("initDAGState: cannot find "+
					"DAG tip %s in block index", dagState.TipHashes))
			}
			tips.add(tip)
		}
		dag.virtual.SetTips(tips)

		// Set the last finality point
		dag.lastFinalityPoint = dag.index.LookupNode(dagState.LastFinalityPoint)
		dag.finalizeNodesBelowFinalityPoint(false)

		// Go over any unprocessed blockNodes and process them now.
		for _, node := range unprocessedBlockNodes {
			// Check to see if the block exists in the block DB. If it
			// doesn't, the database has certainly been corrupted.
			blockExists, err := dbaccess.HasBlock(dbaccess.NoTx(), node.hash)
			if err != nil {
				return AssertError(fmt.Sprintf("initDAGState: HasBlock "+
					"for block %s failed: %s", node.hash, err))
			}
			if !blockExists {
				return AssertError(fmt.Sprintf("initDAGState: block %s "+
					"exists in block index but not in block db", node.hash))
			}

			// Attempt to accept the block.
			block, found, err := dbFetchBlockByHash(dbaccess.NoTx(), node.hash)
			if err != nil {
				return err
			}
			if !found {
				return errors.Errorf("block %s not found",
					node.hash)
			}
			isOrphan, isDelayed, err := dag.ProcessBlock(block, BFWasStored)
			if err != nil {
				log.Warnf("Block %s, which was not previously processed, "+
					"failed to be accepted to the DAG: %s", node.hash, err)
				continue
			}

			// If the block is an orphan or is delayed then it couldn't have
			// possibly been written to the block index in the first place.
			if isOrphan {
				return AssertError(fmt.Sprintf("Block %s, which was not "+
					"previously processed, turned out to be an orphan, which is "+
					"impossible.", node.hash))
			}
			if isDelayed {
				return AssertError(fmt.Sprintf("Block %s, which was not "+
					"previously processed, turned out to be delayed, which is "+
					"impossible.", node.hash))
			}
		}

		return nil
	})
}

// deserializeBlockNode parses a value in the block index bucket and returns a block node.
func (dag *BlockDAG) deserializeBlockNode(blockRow []byte) (*blockNode, error) {
	buffer := bytes.NewReader(blockRow)

	var header wire.BlockHeader
	err := header.Deserialize(buffer)
	if err != nil {
		return nil, err
	}

	node := &blockNode{
		hash:                 header.BlockHash(),
		version:              header.Version,
		bits:                 header.Bits,
		nonce:                header.Nonce,
		timestamp:            header.Timestamp.Unix(),
		hashMerkleRoot:       header.HashMerkleRoot,
		acceptedIDMerkleRoot: header.AcceptedIDMerkleRoot,
		utxoCommitment:       header.UTXOCommitment,
	}

	node.children = newBlockSet()
	node.parents = newBlockSet()

	for _, hash := range header.ParentHashes {
		parent := dag.index.LookupNode(hash)
		if parent == nil {
			return nil, AssertError(fmt.Sprintf("deserializeBlockNode: Could "+
				"not find parent %s for block %s", hash, header.BlockHash()))
		}
		node.parents.add(parent)
	}

	statusByte, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	node.status = blockStatus(statusByte)

	selectedParentHash := &daghash.Hash{}
	if _, err := io.ReadFull(buffer, selectedParentHash[:]); err != nil {
		return nil, err
	}

	// Because genesis doesn't have selected parent, it's serialized as zero hash
	if !selectedParentHash.IsEqual(&daghash.ZeroHash) {
		node.selectedParent = dag.index.LookupNode(selectedParentHash)
	}

	node.blueScore, err = binaryserializer.Uint64(buffer, byteOrder)
	if err != nil {
		return nil, err
	}

	bluesCount, err := wire.ReadVarInt(buffer)
	if err != nil {
		return nil, err
	}

	node.blues = make([]*blockNode, bluesCount)
	for i := uint64(0); i < bluesCount; i++ {
		hash := &daghash.Hash{}
		if _, err := io.ReadFull(buffer, hash[:]); err != nil {
			return nil, err
		}
		node.blues[i] = dag.index.LookupNode(hash)
	}

	bluesAnticoneSizesLen, err := wire.ReadVarInt(buffer)
	if err != nil {
		return nil, err
	}

	node.bluesAnticoneSizes = make(map[*blockNode]dagconfig.KType)
	for i := uint64(0); i < bluesAnticoneSizesLen; i++ {
		hash := &daghash.Hash{}
		if _, err := io.ReadFull(buffer, hash[:]); err != nil {
			return nil, err
		}
		bluesAnticoneSize, err := binaryserializer.Uint8(buffer)
		if err != nil {
			return nil, err
		}
		blue := dag.index.LookupNode(hash)
		if blue == nil {
			return nil, errors.Errorf("couldn't find block with hash %s", hash)
		}
		node.bluesAnticoneSizes[blue] = dagconfig.KType(bluesAnticoneSize)
	}

	return node, nil
}

// dbFetchBlockByHash retrieves the raw block for the provided hash,
// deserialize it, and return a util.Block of it.
func dbFetchBlockByHash(context dbaccess.Context, hash *daghash.Hash) (block *util.Block, found bool, err error) {
	blockBytes, found, err := dbaccess.FetchBlock(context, hash)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	// Create the encapsulated block.
	block, err = util.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, false, err
	}
	return block, true, nil
}

func dbStoreBlock(context dbaccess.Context, block *util.Block) error {
	blockBytes, err := block.Bytes()
	if err != nil {
		return err
	}
	return dbaccess.StoreBlock(context, block.Hash(), blockBytes)
}

func serializeBlockNode(node *blockNode) ([]byte, error) {
	w := bytes.NewBuffer(make([]byte, 0, wire.MaxBlockHeaderPayload+1))
	header := node.Header()
	err := header.Serialize(w)
	if err != nil {
		return nil, err
	}

	err = w.WriteByte(byte(node.status))
	if err != nil {
		return nil, err
	}

	// Because genesis doesn't have selected parent, it's serialized as zero hash
	selectedParentHash := &daghash.ZeroHash
	if node.selectedParent != nil {
		selectedParentHash = node.selectedParent.hash
	}
	_, err = w.Write(selectedParentHash[:])
	if err != nil {
		return nil, err
	}

	err = binaryserializer.PutUint64(w, byteOrder, node.blueScore)
	if err != nil {
		return nil, err
	}

	err = wire.WriteVarInt(w, uint64(len(node.blues)))
	if err != nil {
		return nil, err
	}

	for _, blue := range node.blues {
		_, err = w.Write(blue.hash[:])
		if err != nil {
			return nil, err
		}
	}

	err = wire.WriteVarInt(w, uint64(len(node.bluesAnticoneSizes)))
	if err != nil {
		return nil, err
	}
	for blue, blueAnticoneSize := range node.bluesAnticoneSizes {
		_, err = w.Write(blue.hash[:])
		if err != nil {
			return nil, err
		}

		err = binaryserializer.PutUint8(w, uint8(blueAnticoneSize))
		if err != nil {
			return nil, err
		}
	}
	return w.Bytes(), nil
}

// BlockIndexKey generates the binary key for an entry in the block index
// bucket. The key is composed of the block blue score encoded as a big-endian
// 64-bit unsigned int followed by the 32 byte block hash.
// The blue score component is important for iteration order.
func BlockIndexKey(blockHash *daghash.Hash, blueScore uint64) []byte {
	indexKey := make([]byte, daghash.HashSize+8)
	binary.BigEndian.PutUint64(indexKey[0:8], blueScore)
	copy(indexKey[8:daghash.HashSize+8], blockHash[:])
	return indexKey
}

func blockHashFromBlockIndexKey(BlockIndexKey []byte) (*daghash.Hash, error) {
	return daghash.NewHash(BlockIndexKey[8 : daghash.HashSize+8])
}

// BlockByHash returns the block from the DAG with the given hash.
//
// This function is safe for concurrent access.
func (dag *BlockDAG) BlockByHash(hash *daghash.Hash) (*util.Block, error) {
	// Lookup the block hash in block index and ensure it is in the DAG
	node := dag.index.LookupNode(hash)
	if node == nil {
		str := fmt.Sprintf("block %s is not in the DAG", hash)
		return nil, errNotInDAG(str)
	}

	block, found, err := dbFetchBlockByHash(dbaccess.NoTx(), node.hash)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("block %s not found",
			node.hash)
	}
	return block, err
}

// BlockHashesFrom returns a slice of blocks starting from lowHash
// ordered by blueScore. If lowHash is nil then the genesis block is used.
//
// This method MUST be called with the DAG lock held
func (dag *BlockDAG) BlockHashesFrom(lowHash *daghash.Hash, limit int) ([]*daghash.Hash, error) {
	blockHashes := make([]*daghash.Hash, 0, limit)
	if lowHash == nil {
		lowHash = dag.genesis.hash

		// If we're starting from the beginning we should include the
		// genesis hash in the result
		blockHashes = append(blockHashes, dag.genesis.hash)
	}
	if !dag.IsInDAG(lowHash) {
		return nil, errors.Errorf("block %s not found", lowHash)
	}
	blueScore, err := dag.BlueScoreByBlockHash(lowHash)
	if err != nil {
		return nil, err
	}

	key := BlockIndexKey(lowHash, blueScore)
	cursor, found, err := dbaccess.BlockIndexCursorFrom(dbaccess.NoTx(), key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("block %s not in block index", lowHash)
	}

	for cursor.Next() && len(blockHashes) < limit {
		key, err := cursor.Key()
		if err != nil {
			return nil, err
		}
		blockHash, err := blockHashFromBlockIndexKey(key)
		if err != nil {
			return nil, err
		}
		blockHashes = append(blockHashes, blockHash)
	}

	return blockHashes, nil
}
