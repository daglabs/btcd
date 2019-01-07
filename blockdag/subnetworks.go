package blockdag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/database"
	"github.com/daglabs/btcd/wire"
)

// validateAndExtractSubNetworkRegistryTxs filters the given input and extracts a list
// of valid sub-network registry transactions.
func validateAndExtractSubNetworkRegistryTxs(txs []*TxWithBlockHash) ([]*wire.MsgTx, error) {
	validSubNetworkRegistryTxs := make([]*wire.MsgTx, 0)
	for _, txData := range txs {
		tx := txData.Tx.MsgTx()
		if tx.SubNetworkID == wire.SubNetworkRegistry {
			err := validateSubNetworkRegistryTransaction(tx)
			if err != nil {
				return nil, err
			}
			validSubNetworkRegistryTxs = append(validSubNetworkRegistryTxs, tx)
		}
	}

	return validSubNetworkRegistryTxs, nil
}

// validateSubNetworkRegistryTransaction makes sure that a given sub-network registry
// transaction is valid. Such a transaction is valid iff:
// - Its entire payload is a uint64 (8 bytes)
func validateSubNetworkRegistryTransaction(tx *wire.MsgTx) error {
	if len(tx.Payload) != 8 {
		return ruleError(ErrSubNetworkRegistry, fmt.Sprintf("validation failed: subnetwork registry"+
			"tx '%s' has an invalid payload", tx.TxHash()))
	}

	return nil
}

// registerPendingSubNetworks attempts to register all the pending sub-networks that
// had previously been defined between the initial finality point and the new one.
// Note: transactions within newFinalityPoint itself are not registered. Instead, they will
// be registered when the next finality point is chosen; then it will be the
// initialFinalityPoint.
func (dag *BlockDAG) registerPendingSubNetworks(dbTx database.Tx, initialFinalityPoint *blockNode, newFinalityPoint *blockNode) error {
	var stack []*blockNode
	for currentNode := newFinalityPoint; currentNode != initialFinalityPoint; currentNode = currentNode.selectedParent {
		stack = append(stack, currentNode)
	}

	// Register a pending sub-network for all blues. The block itself is not explicitly
	// registered since it will be one of the blues of the next block.
	// Note that this means that the very last block in the selected parent chain is not
	// registered. This is intentional.
	for i := len(stack) - 1; i >= 0; i-- {
		currentNode := stack[i]
		for _, blue := range currentNode.blues {
			err := dag.registerPendingSubNetworksInBlock(dbTx, blue.hash)
			if err != nil {
				return fmt.Errorf("failed to register pending sub-networks: %s", err)
			}
		}
	}

	return nil
}

// registerPendingSubNetworksInBlock attempts to register all the sub-networks
// that had been defined in a given block.
func (dag *BlockDAG) registerPendingSubNetworksInBlock(dbTx database.Tx, blockHash daghash.Hash) error {
	pendingSubNetworkTxs, err := dbGetPendingSubNetworkTxs(dbTx, blockHash)
	if err != nil {
		return fmt.Errorf("failed to retrieve pending sub-network txs in block '%s': %s", blockHash, err)
	}
	for _, tx := range pendingSubNetworkTxs {
		if !dbIsRegisteredSubNetworkTx(dbTx, tx.TxHash()) {
			createdSubNetwork := newSubNetwork(tx)
			err := dbRegisterSubNetwork(dbTx, dag.lastSubNetworkID, tx.TxHash(), createdSubNetwork)
			if err != nil {
				return fmt.Errorf("failed registering sub-network"+
					"for tx '%s' in block '%s': %s", tx.TxHash(), blockHash, err)
			}

			dag.lastSubNetworkID++
		}
	}

	err = dbRemovePendingSubNetworkTxs(dbTx, blockHash)
	if err != nil {
		return fmt.Errorf("failed to remove block '%s'"+
			"from pending sub-networks: %s", blockHash, err)
	}

	return nil
}

// subNetwork returns a registered-and-finalized sub-network. If the sub-network
// does not exist this method returns an error.
func (dag *BlockDAG) subNetwork(subNetworkID uint64) (*subNetwork, error) {
	var sNet *subNetwork
	var err error
	dbErr := dag.db.View(func(dbTx database.Tx) error {
		sNet, err = dbGetSubNetwork(dbTx, subNetworkID)
		return nil
	})
	if dbErr != nil {
		return nil, fmt.Errorf("could not retrieve sub-network '%d': %s", subNetworkID, dbErr)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve sub-network '%d': %s", subNetworkID, err)
	}

	return sNet, nil
}

// GasLimit returns the gas limit of a registered-and-finalized sub-network. If
// the sub-network does not exist this method returns an error.
func (dag *BlockDAG) GasLimit(subNetworkID uint64) (uint64, error) {
	sNet, err := dag.subNetwork(subNetworkID)
	if err != nil {
		return 0, err
	}

	return sNet.gasLimit, nil
}

// -----------------------------------------------------------------------------
// The sub-network registry consists of three buckets:
// a. The pending sub-network bucket
// b. The registered transaction bucket
// c. The sub-network bucket
//
// All newly-discovered sub-network registry transactions are stored in the
// pending sub-network bucket. These transactions are withheld until the
// blocks that contain them become final.
//
// Once the block of a sub-network registry transaction becomes final, all the
// transactions within that block are retrieved and checked for validity.
// Valid transactions are then:
// 1. Assigned a sub-network ID
// 2. Added to the registered transaction bucket
// 3. Added to the sub-network bucket
// -----------------------------------------------------------------------------

// dbPutPendingSubNetworkTxs stores mappings from a block (via its hash) to an
// array of sub-network transactions.
func dbPutPendingSubNetworkTxs(dbTx database.Tx, blockHash *daghash.Hash, subNetworkRegistryTxs []*wire.MsgTx) error {
	// Empty blocks are not written
	if len(subNetworkRegistryTxs) == 0 {
		return nil
	}

	serializedTxs, err := serializeSubNetworkRegistryTxs(subNetworkRegistryTxs)
	if err != nil {
		return fmt.Errorf("failed to serialize pending sub-network txs in block '%s': %s", blockHash, err)
	}

	// Store the serialized transactions
	bucket := dbTx.Metadata().Bucket(pendingSubNetworksBucketName)
	err = bucket.Put(blockHash[:], serializedTxs)
	if err != nil {
		return fmt.Errorf("failed to write pending sub-network txs in block '%s': %s", blockHash, err)
	}

	return nil
}

// dbGetPendingSubNetworkTxs retrieves an array of sub-network transactions,
// associated with a block's hash, that was previously stored with
// dbPutPendingSubNetworkTxs.
// Returns an empty slice if the hash was not previously stored.
func dbGetPendingSubNetworkTxs(dbTx database.Tx, blockHash daghash.Hash) ([]*wire.MsgTx, error) {
	bucket := dbTx.Metadata().Bucket(pendingSubNetworksBucketName)
	serializedTxsBytes := bucket.Get(blockHash[:])
	if serializedTxsBytes == nil {
		return []*wire.MsgTx{}, nil
	}

	txs, err := deserializeSubNetworkRegistryTxs(serializedTxsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize pending sub-network txs for block '%s': %s", blockHash, err)
	}

	return txs, nil
}

// serializeSubNetworkRegistryTxs serializes a slice of MsgTxs by serializing each transaction
// individually and appending it to one long byte slice.
func serializeSubNetworkRegistryTxs(subNetworkRegistryTxs []*wire.MsgTx) ([]byte, error) {
	// Calculate the length in bytes of the serialized transactions
	serializedTxsLength := uint64(0)
	for _, tx := range subNetworkRegistryTxs {
		serializedTxsLength += uint64(tx.SerializeSize())
	}
	serializedTxs := bytes.NewBuffer(make([]byte, 0, serializedTxsLength))

	// Write each transaction in the order it appears in
	for _, tx := range subNetworkRegistryTxs {
		err := tx.Serialize(serializedTxs)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize tx '%s': %s", tx.TxHash(), err)
		}
	}

	return serializedTxs.Bytes(), nil
}

// deserializeSubNetworkRegistryTxs deserializes a byte slice into a slice of MsgTxs.
func deserializeSubNetworkRegistryTxs(serializedTxsBytes []byte) ([]*wire.MsgTx, error) {
	serializedTxs := bytes.NewBuffer(serializedTxsBytes)

	// Read each transaction and store it in txs until the end of the buffer
	txs := make([]*wire.MsgTx, 0)
	for serializedTxs.Len() > 0 {
		var tx wire.MsgTx
		err := tx.Deserialize(serializedTxs)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize pending sub-network txs: %s", err)
		}

		txs = append(txs, &tx)
	}

	return txs, nil
}

// dbRemovePendingSubNetworkTxs deletes an array of sub-network transactions,
// associated with a block's hash, that was previously stored with
// dbPutPendingSubNetworkTxs.
// This function does not return an error if the hash was not previously stored.
func dbRemovePendingSubNetworkTxs(dbTx database.Tx, blockHash daghash.Hash) error {
	bucket := dbTx.Metadata().Bucket(pendingSubNetworksBucketName)

	err := bucket.Delete(blockHash[:])
	if err != nil {
		return fmt.Errorf("failed to remove pending sub-network txs in block '%s': %s", blockHash, err)
	}

	return nil
}

// dbIsRegisteredSubNetworkTx checks whether a sub-network registry transaction
// had successfully registered a sub-network.
func dbIsRegisteredSubNetworkTx(dbTx database.Tx, txHash daghash.Hash) bool {
	bucket := dbTx.Metadata().Bucket(registeredSubNetworkTxsBucketName)
	subNetworkIDBytes := bucket.Get(txHash[:])

	return subNetworkIDBytes != nil
}

// dbRegisterSubNetwork stores mappings:
// a. from the ID of the sub-network to the sub-network data.
// b. from the hash of a sub-network registry transaction to the sub-network ID.
func dbRegisterSubNetwork(dbTx database.Tx, subNetworkID uint64, txHash daghash.Hash, network *subNetwork) error {
	// Serialize the sub-network ID
	subNetworkIDBytes := make([]byte, 8)
	byteOrder.PutUint64(subNetworkIDBytes, subNetworkID)

	// Serialize the sub-network
	serializedSubNetwork, err := serializeSubNetwork(network)
	if err != nil {
		return fmt.Errorf("failed to serialize sub-netowrk of tx '%s': %s", network.txHash, err)
	}

	// Store the sub-network
	subNetworksBucket := dbTx.Metadata().Bucket(subNetworksBucketName)
	err = subNetworksBucket.Put(subNetworkIDBytes, serializedSubNetwork)
	if err != nil {
		return fmt.Errorf("failed to write sub-netowrk of tx '%s': %s", network.txHash, err)
	}

	// Store the mapping between txHash and subNetworkID
	registeredSubNetworkTxs := dbTx.Metadata().Bucket(registeredSubNetworkTxsBucketName)
	err = registeredSubNetworkTxs.Put(txHash[:], subNetworkIDBytes)
	if err != nil {
		return fmt.Errorf("failed to put registered sub-networkTx '%s': %s", txHash, err)
	}

	return nil
}

func dbGetSubNetwork(dbTx database.Tx, subNetworkID uint64) (*subNetwork, error) {
	// Serialize the sub-network ID
	subNetworkIDBytes := make([]byte, 8)
	byteOrder.PutUint64(subNetworkIDBytes, subNetworkID)

	// Get the sub-network
	bucket := dbTx.Metadata().Bucket(subNetworksBucketName)
	serializedSubNetwork := bucket.Get(subNetworkIDBytes)
	if serializedSubNetwork == nil {
		return nil, fmt.Errorf("sub-network '%d' not found", subNetworkID)
	}

	return deserializeSubNetwork(serializedSubNetwork)
}

type subNetwork struct {
	txHash   daghash.Hash
	gasLimit uint64
}

func newSubNetwork(tx *wire.MsgTx) *subNetwork {
	txHash := tx.TxHash()
	gasLimit := binary.LittleEndian.Uint64(tx.Payload[:8])

	return &subNetwork{
		txHash:   txHash,
		gasLimit: gasLimit,
	}
}

// serializeSubNetwork serializes a subNetwork into the following binary format:
// | txHash (32 bytes) | gasLimit (8 bytes) |
func serializeSubNetwork(sNet *subNetwork) ([]byte, error) {
	serializedSNet := bytes.NewBuffer(make([]byte, 0, 40))

	// Write the tx hash
	err := binary.Write(serializedSNet, byteOrder, sNet.txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize sub-network for tx '%s': %s", sNet.txHash, err)
	}

	// Write the gas limit
	err = binary.Write(serializedSNet, byteOrder, sNet.gasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize sub-network for tx '%s': %s", sNet.txHash, err)
	}

	return serializedSNet.Bytes(), nil
}

// deserializeSubNetwork deserializes a byte slice into a subNetwork.
// See serializeSubNetwork for the binary format.
func deserializeSubNetwork(serializedSNetBytes []byte) (*subNetwork, error) {
	serializedSNet := bytes.NewBuffer(serializedSNetBytes)

	// Read the tx hash
	var txHash daghash.Hash
	err := binary.Read(serializedSNet, byteOrder, &txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize sub-network: %s", err)
	}

	// Read the gas limit
	var gasLimit uint64
	err = binary.Read(serializedSNet, byteOrder, &gasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize sub-network: %s", err)
	}

	return &subNetwork{
		txHash:   txHash,
		gasLimit: gasLimit,
	}, nil
}
