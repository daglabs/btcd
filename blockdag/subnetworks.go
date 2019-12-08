package blockdag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/util"

	"github.com/kaspanet/kaspad/database"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
)

// SubnetworkStore stores the subnetworks data
type SubnetworkStore struct {
	db database.DB
}

func newSubnetworkStore(db database.DB) *SubnetworkStore {
	return &SubnetworkStore{
		db: db,
	}
}

// registerSubnetworks scans a list of transactions, singles out
// subnetwork registry transactions, validates them, and registers a new
// subnetwork based on it.
// This function returns an error if one or more transactions are invalid
func registerSubnetworks(dbTx database.Tx, txs []*util.Tx) error {
	subnetworkRegistryTxs := make([]*wire.MsgTx, 0)
	for _, tx := range txs {
		msgTx := tx.MsgTx()

		if msgTx.SubnetworkID.IsEqual(subnetworkid.SubnetworkIDRegistry) {
			subnetworkRegistryTxs = append(subnetworkRegistryTxs, msgTx)
		}

		if subnetworkid.Less(subnetworkid.SubnetworkIDRegistry, &msgTx.SubnetworkID) {
			// Transactions are ordered by subnetwork, so we can safely assume
			// that the rest of the transactions will not be subnetwork registry
			// transactions.
			break
		}
	}

	for _, registryTx := range subnetworkRegistryTxs {
		subnetworkID, err := TxToSubnetworkID(registryTx)
		if err != nil {
			return err
		}
		sNet, err := dbGetSubnetwork(dbTx, subnetworkID)
		if err != nil {
			return err
		}
		if sNet == nil {
			createdSubnetwork := newSubnetwork(registryTx)
			err := dbRegisterSubnetwork(dbTx, subnetworkID, createdSubnetwork)
			if err != nil {
				return errors.Errorf("failed registering subnetwork"+
					"for tx '%s': %s", registryTx.TxHash(), err)
			}
		}
	}

	return nil
}

// validateSubnetworkRegistryTransaction makes sure that a given subnetwork registry
// transaction is valid. Such a transaction is valid iff:
// - Its entire payload is a uint64 (8 bytes)
func validateSubnetworkRegistryTransaction(tx *wire.MsgTx) error {
	if len(tx.Payload) != 8 {
		return ruleError(ErrSubnetworkRegistry, fmt.Sprintf("validation failed: subnetwork registry"+
			"tx '%s' has an invalid payload", tx.TxHash()))
	}

	return nil
}

// TxToSubnetworkID creates a subnetwork ID from a subnetwork registry transaction
func TxToSubnetworkID(tx *wire.MsgTx) (*subnetworkid.SubnetworkID, error) {
	txHash := tx.TxHash()
	return subnetworkid.New(util.Hash160(txHash[:]))
}

// subnetwork returns a registered subnetwork. If the subnetwork does not exist
// this method returns an error.
func (s *SubnetworkStore) subnetwork(subnetworkID *subnetworkid.SubnetworkID) (*subnetwork, error) {
	var sNet *subnetwork
	var err error
	dbErr := s.db.View(func(dbTx database.Tx) error {
		sNet, err = dbGetSubnetwork(dbTx, subnetworkID)
		return nil
	})
	if dbErr != nil {
		return nil, errors.Errorf("could not retrieve subnetwork '%d': %s", subnetworkID, dbErr)
	}
	if err != nil {
		return nil, errors.Errorf("could not retrieve subnetwork '%d': %s", subnetworkID, err)
	}

	return sNet, nil
}

// GasLimit returns the gas limit of a registered subnetwork. If the subnetwork does not
// exist this method returns an error.
func (s *SubnetworkStore) GasLimit(subnetworkID *subnetworkid.SubnetworkID) (uint64, error) {
	sNet, err := s.subnetwork(subnetworkID)
	if err != nil {
		return 0, err
	}
	if sNet == nil {
		return 0, errors.Errorf("subnetwork '%s' not found", subnetworkID)
	}

	return sNet.gasLimit, nil
}

// dbRegisterSubnetwork stores mappings from ID of the subnetwork to the subnetwork data.
func dbRegisterSubnetwork(dbTx database.Tx, subnetworkID *subnetworkid.SubnetworkID, network *subnetwork) error {
	// Serialize the subnetwork
	serializedSubnetwork, err := serializeSubnetwork(network)
	if err != nil {
		return errors.Errorf("failed to serialize sub-netowrk '%s': %s", subnetworkID, err)
	}

	// Store the subnetwork
	subnetworksBucket := dbTx.Metadata().Bucket(subnetworksBucketName)
	err = subnetworksBucket.Put(subnetworkID[:], serializedSubnetwork)
	if err != nil {
		return errors.Errorf("failed to write sub-netowrk '%s': %s", subnetworkID, err)
	}

	return nil
}

// dbGetSubnetwork returns the subnetwork associated with subnetworkID or nil if the subnetwork was not found.
func dbGetSubnetwork(dbTx database.Tx, subnetworkID *subnetworkid.SubnetworkID) (*subnetwork, error) {
	bucket := dbTx.Metadata().Bucket(subnetworksBucketName)
	serializedSubnetwork := bucket.Get(subnetworkID[:])
	if serializedSubnetwork == nil {
		return nil, nil
	}

	return deserializeSubnetwork(serializedSubnetwork)
}

type subnetwork struct {
	gasLimit uint64
}

func newSubnetwork(tx *wire.MsgTx) *subnetwork {
	return &subnetwork{
		gasLimit: ExtractGasLimit(tx),
	}
}

// ExtractGasLimit extracts the gas limit from the transaction payload
func ExtractGasLimit(tx *wire.MsgTx) uint64 {
	return binary.LittleEndian.Uint64(tx.Payload[:8])
}

// serializeSubnetwork serializes a subnetwork into the following binary format:
// | gasLimit (8 bytes) |
func serializeSubnetwork(sNet *subnetwork) ([]byte, error) {
	serializedSNet := bytes.NewBuffer(make([]byte, 0, 8))

	// Write the gas limit
	err := binary.Write(serializedSNet, byteOrder, sNet.gasLimit)
	if err != nil {
		return nil, errors.Errorf("failed to serialize subnetwork: %s", err)
	}

	return serializedSNet.Bytes(), nil
}

// deserializeSubnetwork deserializes a byte slice into a subnetwork.
// See serializeSubnetwork for the binary format.
func deserializeSubnetwork(serializedSNetBytes []byte) (*subnetwork, error) {
	serializedSNet := bytes.NewBuffer(serializedSNetBytes)

	// Read the gas limit
	var gasLimit uint64
	err := binary.Read(serializedSNet, byteOrder, &gasLimit)
	if err != nil {
		return nil, errors.Errorf("failed to deserialize subnetwork: %s", err)
	}

	return &subnetwork{
		gasLimit: gasLimit,
	}, nil
}
