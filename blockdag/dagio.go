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
	"github.com/pkg/errors"
	"io"
	"math"
	"sync"

	"github.com/kaspanet/kaspad/database"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/binaryserializer"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
)

const (
	// blockHdrSize is the size of a block header. This is simply the
	// constant from wire and is only provided here for convenience since
	// wire.MaxBlockHeaderPayload is quite long.
	blockHdrSize = wire.MaxBlockHeaderPayload

	// latestUTXOSetBucketVersion is the current version of the UTXO set
	// bucket that is used to track all unspent outputs.
	latestUTXOSetBucketVersion = 1
)

var (
	// blockIndexBucketName is the name of the database bucket used to house the
	// block headers and contextual information.
	blockIndexBucketName = []byte("blockheaderidx")

	// dagStateKeyName is the name of the db key used to store the DAG
	// tip hashes.
	dagStateKeyName = []byte("dagstate")

	// utxoSetVersionKeyName is the name of the db key used to store the
	// version of the utxo set currently in the database.
	utxoSetVersionKeyName = []byte("utxosetversion")

	// utxoSetBucketName is the name of the database bucket used to house the
	// unspent transaction output set.
	utxoSetBucketName = []byte("utxoset")

	// utxoDiffsBucketName is the name of the database bucket used to house the
	// diffs and diff children of blocks.
	utxoDiffsBucketName = []byte("utxodiffs")

	// reachabilityDataBucketName is the name of the database bucket used to house the
	// reachability tree nodes and future covering sets of blocks.
	reachabilityDataBucketName = []byte("reachability")

	// subnetworksBucketName is the name of the database bucket used to store the
	// subnetwork registry.
	subnetworksBucketName = []byte("subnetworks")

	// localSubnetworkKeyName is the name of the db key used to store the
	// node's local subnetwork ID.
	localSubnetworkKeyName = []byte("localsubnetworkidkey")

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

// dbPutVersion uses an existing database transaction to update the provided
// key in the metadata bucket to the given version. It is primarily used to
// track versions on entities such as buckets.
func dbPutVersion(dbTx database.Tx, key []byte, version uint32) error {
	var serialized [4]byte
	byteOrder.PutUint32(serialized[:], version)
	return dbTx.Metadata().Put(key, serialized[:])
}

// -----------------------------------------------------------------------------
// The unspent transaction output (UTXO) set consists of an entry for each
// unspent output using a format that is optimized to reduce space using domain
// specific compression algorithms.
//
// Each entry is keyed by an outpoint as specified below. It is important to
// note that the key encoding uses a varint big-endian encoding, which employs
// an MSB encoding so iteration of UTXOs when doing byte-wise comparisons will
// produce them in order.
//
// The serialized key format is:
//   <hash><output index>
//
//   Field                Type             Size
//   hash                 daghash.Hash     daghash.HashSize
//   output index         varint           variable
//
// The serialized value format is:
//
//   <header code><compressed txout>
//
//   Field                Type     Size
//   header code          varint   variable
//   compressed txout
//     compressed amount  varint   variable
//     compressed script  []byte   variable
//
// The serialized header code format is:
//   bit 0 - containing transaction is a coinbase
//   bits 1-x - blue score of the block that accepted the unspent txout
//
// Example 1:
// 4a16969aa4764dd7507fc1de7f0baa4850a246de90c45e59a3207f9a26b5036f:2
//
//    8cf3168900b8025be1b3efc63b0ad48e7f9f10e87544528d58
//    <----><------------------------------------------>
//      |                             |
//   header code             compressed txout
//
//  - header code: 0x8cf316 (not coinbase, height 113931)
//  - compressed txout:
//    - 0x89: Varint-encoded compressed amount for 15000000 (0.15 KAS)
//    - 0x00: special script type pay-to-pubkey-hash
//    - 0xb8...58: pubkey hash
//
// Example 2:
// 1b02d1c8cfef60a189017b9a420c682cf4a0028175f2f563209e4ff61c8c3620:22
//
//    a8a258fe63b4cec4011dd46a006572d820e448e12d2bbb38640bc718e6
//    <----><-------------------------------------------------->
//      |                             |
//   header code             compressed txout
//
//  - header code: 0xa8a258 (not coinbase, blue score 338156)
//  - compressed txout:
//    - 0xfe63b4cec4: Varint-encoded compressed amount for 366875659 (3.66875659 KAS)
//    - 0x01: special script type pay-to-script-hash
//    - 0x1d...e6: script hash
// -----------------------------------------------------------------------------

// outpointKeyPool defines a concurrent safe free list of byte slices used to
// provide temporary buffers for outpoint database keys.
var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{} // Pointer to a buffer to avoid boxing alloc.
	},
}

func serializeOutpoint(w io.Writer, outpoint *wire.Outpoint) error {
	_, err := w.Write(outpoint.TxID[:])
	if err != nil {
		return err
	}

	return wire.WriteVarIntBigEndian(w, uint64(outpoint.Index))
}

var outpointMaxSerializeSize = daghash.TxIDSize + wire.VarIntSerializeSize(math.MaxUint32)

// deserializeOutpoint decodes an outpoint from the passed serialized byte
// slice into a new wire.Outpoint using a format that is suitable for long-
// term storage. this format is described in detail above.
func deserializeOutpoint(r io.Reader) (*wire.Outpoint, error) {
	outpoint := &wire.Outpoint{}
	_, err := r.Read(outpoint.TxID[:])
	if err != nil {
		return nil, err
	}

	idx, err := wire.ReadVarIntBigEndian(r)
	if err != nil {
		return nil, err
	}

	if idx > math.MaxUint32 {
		return nil, errors.Errorf("%d is not a valid outpoint index", idx)
	}

	outpoint.Index = uint32(idx)

	return outpoint, nil
}

// dbPutUTXODiff uses an existing database transaction to update the UTXO set
// in the database based on the provided UTXO view contents and state. In
// particular, only the entries that have been marked as modified are written
// to the database.
func dbPutUTXODiff(dbTx database.Tx, diff *UTXODiff) error {
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	for outpoint := range diff.toRemove {
		w := outpointKeyPool.Get().(*bytes.Buffer)
		w.Reset()
		err := serializeOutpoint(w, &outpoint)
		if err != nil {
			return err
		}

		key := w.Bytes()
		err = utxoBucket.Delete(key)
		if err != nil {
			return err
		}
		outpointKeyPool.Put(w)
	}

	// We are preallocating for P2PKH entries because they are the most common ones.
	// If we have entries with a compressed script bigger than P2PKH's, the buffer will grow.
	bytesToPreallocate := (p2pkhUTXOEntryMaxSerializeSize + outpointMaxSerializeSize) * len(diff.toAdd)
	buff := bytes.NewBuffer(make([]byte, bytesToPreallocate))
	for outpoint, entry := range diff.toAdd {
		// Serialize and store the UTXO entry.
		sBuff := newSubBuffer(buff)
		err := serializeUTXOEntry(sBuff, entry)
		if err != nil {
			return err
		}
		serializedEntry := sBuff.bytes()

		sBuff = newSubBuffer(buff)
		err = serializeOutpoint(sBuff, &outpoint)
		if err != nil {
			return err
		}

		key := sBuff.bytes()
		err = utxoBucket.Put(key, serializedEntry)
		// NOTE: The key is intentionally not recycled here since the
		// database interface contract prohibits modifications. It will
		// be garbage collected normally when the database is done with
		// it.
		if err != nil {
			return err
		}
	}

	return nil
}

type subBuffer struct {
	buff       *bytes.Buffer
	start, end int
}

func (s *subBuffer) bytes() []byte {
	return s.buff.Bytes()[s.start:s.end]
}

func (s *subBuffer) Write(p []byte) (int, error) {
	if s.buff.Len() > s.end || s.buff.Len() < s.start {
		return 0, errors.New("a sub buffer cannot be written after another entity wrote or read from its " +
			"underlying buffer")
	}

	n, err := s.buff.Write(p)
	if err != nil {
		return 0, err
	}

	s.end += n

	return n, nil
}

func newSubBuffer(buff *bytes.Buffer) *subBuffer {
	return &subBuffer{
		buff:  buff,
		start: buff.Len(),
		end:   buff.Len(),
	}
}

type dagState struct {
	TipHashes         []*daghash.Hash
	LastFinalityPoint *daghash.Hash
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
func dbPutDAGState(dbTx database.Tx, state *dagState) error {
	serializedData, err := serializeDAGState(state)

	if err != nil {
		return err
	}

	return dbTx.Metadata().Put(dagStateKeyName, serializedData)
}

// createDAGState initializes both the database and the DAG state to the
// genesis block. This includes creating the necessary buckets, so it
// must only be called on an uninitialized database.
func (dag *BlockDAG) createDAGState() error {
	// Create the initial the database DAG state including creating the
	// necessary index buckets and inserting the genesis block.
	err := dag.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

		// Create the bucket that houses the block index data.
		_, err := meta.CreateBucket(blockIndexBucketName)
		if err != nil {
			return err
		}

		// Create the buckets that house the utxo set, the utxo diffs, and their
		// version.
		_, err = meta.CreateBucket(utxoSetBucketName)
		if err != nil {
			return err
		}

		_, err = meta.CreateBucket(utxoDiffsBucketName)
		if err != nil {
			return err
		}

		_, err = meta.CreateBucket(reachabilityDataBucketName)
		if err != nil {
			return err
		}

		err = dbPutVersion(dbTx, utxoSetVersionKeyName,
			latestUTXOSetBucketVersion)
		if err != nil {
			return err
		}

		// Create the bucket that houses the registered subnetworks.
		_, err = meta.CreateBucket(subnetworksBucketName)
		if err != nil {
			return err
		}

		if err := dbPutLocalSubnetworkID(dbTx, dag.subnetworkID); err != nil {
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

		err := meta.DeleteBucket(blockIndexBucketName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(utxoSetBucketName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(utxoDiffsBucketName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(reachabilityDataBucketName)
		if err != nil {
			return err
		}

		err = dbTx.Metadata().Delete(utxoSetVersionKeyName)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(subnetworksBucketName)
		if err != nil {
			return err
		}

		err = dbTx.Metadata().Delete(localSubnetworkKeyName)
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

func dbPutLocalSubnetworkID(dbTx database.Tx, subnetworkID *subnetworkid.SubnetworkID) error {
	if subnetworkID == nil {
		return dbTx.Metadata().Put(localSubnetworkKeyName, []byte{})
	}
	return dbTx.Metadata().Put(localSubnetworkKeyName, subnetworkID[:])
}

// initDAGState attempts to load and initialize the DAG state from the
// database. When the db does not yet contain any DAG state, both it and the
// DAG state are initialized to the genesis block.
func (dag *BlockDAG) initDAGState() error {
	// Determine the state of the DAG database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	var initialized bool
	err := dag.db.View(func(dbTx database.Tx) error {
		initialized = dbTx.Metadata().Get(dagStateKeyName) != nil
		if initialized {
			var localSubnetworkID *subnetworkid.SubnetworkID
			localSubnetworkIDBytes := dbTx.Metadata().Get(localSubnetworkKeyName)
			if len(localSubnetworkIDBytes) != 0 {
				localSubnetworkID = &subnetworkid.SubnetworkID{}
				localSubnetworkID.SetBytes(localSubnetworkIDBytes)
			}
			if !localSubnetworkID.IsEqual(dag.subnetworkID) {
				return errors.Errorf("Cannot start kaspad with subnetwork ID %s because"+
					" its database is already built with subnetwork ID %s. If you"+
					" want to switch to a new database, please reset the"+
					" database by starting kaspad with --reset-db flag", dag.subnetworkID, localSubnetworkID)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if !initialized {
		// At this point the database has not already been initialized, so
		// initialize both it and the DAG state to the genesis block.
		return dag.createDAGState()
	}

	// Attempt to load the DAG state from the database.
	return dag.db.View(func(dbTx database.Tx) error {
		// Fetch the stored DAG tipHashes from the database metadata.
		// When it doesn't exist, it means the database hasn't been
		// initialized for use with the DAG yet, so break out now to allow
		// that to happen under a writable database transaction.
		serializedData := dbTx.Metadata().Get(dagStateKeyName)
		log.Tracef("Serialized DAG tip hashes: %x", serializedData)
		state, err := deserializeDAGState(serializedData)
		if err != nil {
			return err
		}

		// Load all of the headers from the data for the known DAG
		// and construct the block index accordingly. Since the
		// number of nodes are already known, perform a single alloc
		// for them versus a whole bunch of little ones to reduce
		// pressure on the GC.
		log.Infof("Loading block index...")

		blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)

		var unprocessedBlockNodes []*blockNode
		cursor := blockIndexBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			node, err := dag.deserializeBlockNode(cursor.Value())
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

		// Load all of the known UTXO entries and construct the full
		// UTXO set accordingly. Since the number of entries is already
		// known, perform a single alloc for them versus a whole bunch
		// of little ones to reduce pressure on the GC.
		log.Infof("Loading UTXO set...")

		utxoEntryBucket := dbTx.Metadata().Bucket(utxoSetBucketName)

		// Determine how many UTXO entries will be loaded into the index so we can
		// allocate the right amount.
		var utxoEntryCount int32
		cursor = utxoEntryBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			utxoEntryCount++
		}

		fullUTXOCollection := make(utxoCollection, utxoEntryCount)
		for ok := cursor.First(); ok; ok = cursor.Next() {
			// Deserialize the outpoint
			outpoint, err := deserializeOutpoint(bytes.NewReader(cursor.Key()))
			if err != nil {
				return err
			}

			// Deserialize the utxo entry
			entry, err := deserializeUTXOEntry(bytes.NewReader(cursor.Value()))
			if err != nil {
				return err
			}

			fullUTXOCollection[*outpoint] = entry
		}

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
		for _, tipHash := range state.TipHashes {
			tip := dag.index.LookupNode(tipHash)
			if tip == nil {
				return AssertError(fmt.Sprintf("initDAGState: cannot find "+
					"DAG tip %s in block index", state.TipHashes))
			}
			tips.add(tip)
		}
		dag.virtual.SetTips(tips)

		// Set the last finality point
		dag.lastFinalityPoint = dag.index.LookupNode(state.LastFinalityPoint)
		dag.finalizeNodesBelowFinalityPoint(false)

		// Go over any unprocessed blockNodes and process them now.
		for _, node := range unprocessedBlockNodes {
			// Check to see if the block exists in the block DB. If it
			// doesn't, the database has certainly been corrupted.
			blockExists, err := dbTx.HasBlock(node.hash)
			if err != nil {
				return AssertError(fmt.Sprintf("initDAGState: HasBlock "+
					"for block %s failed: %s", node.hash, err))
			}
			if !blockExists {
				return AssertError(fmt.Sprintf("initDAGState: block %s "+
					"exists in block index but not in block db", node.hash))
			}

			// Attempt to accept the block.
			block, err := dbFetchBlockByNode(dbTx, node)
			if err != nil {
				return err
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

	bluesCount, err := wire.ReadVarIntLittleEndian(buffer)
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

	bluesAnticoneSizesLen, err := wire.ReadVarIntLittleEndian(buffer)
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

// dbFetchBlockByNode uses an existing database transaction to retrieve the
// raw block for the provided node, deserialize it, and return a util.Block
// of it.
func dbFetchBlockByNode(dbTx database.Tx, node *blockNode) (*util.Block, error) {
	// Load the raw block bytes from the database.
	blockBytes, err := dbTx.FetchBlock(node.hash)
	if err != nil {
		return nil, err
	}

	// Create the encapsulated block.
	block, err := util.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func serializeBlockNode(node *blockNode) ([]byte, error) {
	w := bytes.NewBuffer(make([]byte, 0, blockHdrSize+1))
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

	err = wire.WriteVarIntLittleEndian(w, uint64(len(node.blues)))
	if err != nil {
		return nil, err
	}

	for _, blue := range node.blues {
		_, err = w.Write(blue.hash[:])
		if err != nil {
			return nil, err
		}
	}

	err = wire.WriteVarIntLittleEndian(w, uint64(len(node.bluesAnticoneSizes)))
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

// dbStoreBlockNode stores the block node data into the block
// index bucket. This overwrites the current entry if there exists one.
func dbStoreBlockNode(dbTx database.Tx, node *blockNode) error {
	serializedNode, err := serializeBlockNode(node)
	if err != nil {
		return err
	}
	// Write block header data to block index bucket.
	blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)
	key := BlockIndexKey(node.hash, node.blueScore)
	return blockIndexBucket.Put(key, serializedNode)
}

// dbStoreBlock stores the provided block in the database if it is not already
// there. The full block data is written to ffldb.
func dbStoreBlock(dbTx database.Tx, block *util.Block) error {
	hasBlock, err := dbTx.HasBlock(block.Hash())
	if err != nil {
		return err
	}
	if hasBlock {
		return nil
	}
	return dbTx.StoreBlock(block)
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

	// Load the block from the database and return it.
	var block *util.Block
	err := dag.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
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

	err = dag.index.db.View(func(dbTx database.Tx) error {
		blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)
		lowKey := BlockIndexKey(lowHash, blueScore)

		cursor := blockIndexBucket.Cursor()
		cursor.Seek(lowKey)
		for ok := cursor.Next(); ok; ok = cursor.Next() {
			key := cursor.Key()
			blockHash, err := blockHashFromBlockIndexKey(key)
			if err != nil {
				return err
			}
			blockHashes = append(blockHashes, blockHash)
			if len(blockHashes) == limit {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return blockHashes, nil
}
