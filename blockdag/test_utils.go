package blockdag

// This file functions are not considered safe for regular use, and should be used for test purposes only.

import (
	"compress/bzip2"
	"encoding/binary"
	"github.com/daglabs/kaspad/util"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/daglabs/kaspad/util/subnetworkid"

	"github.com/daglabs/kaspad/database"
	_ "github.com/daglabs/kaspad/database/ffldb" // blank import ffldb so that its init() function runs before tests
	"github.com/daglabs/kaspad/txscript"
	"github.com/daglabs/kaspad/util/daghash"
	"github.com/daglabs/kaspad/wire"
)

const (
	// testDbType is the database backend type to use for the tests.
	testDbType = "ffldb"

	// testDbRoot is the root directory used to create all test databases.
	testDbRoot = "testdbs"

	// blockDataNet is the expected network in the test block data.
	blockDataNet = wire.MainNet
)

// isSupportedDbType returns whether or not the passed database type is
// currently supported.
func isSupportedDbType(dbType string) bool {
	supportedDrivers := database.SupportedDrivers()
	for _, driver := range supportedDrivers {
		if dbType == driver {
			return true
		}
	}

	return false
}

// FileExists returns whether or not the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// DAGSetup is used to create a new db and chain instance with the genesis
// block already inserted.  In addition to the new chain instance, it returns
// a teardown function the caller should invoke when done testing to clean up.
func DAGSetup(dbName string, config Config) (*BlockDAG, func(), error) {
	if !isSupportedDbType(testDbType) {
		return nil, nil, errors.Errorf("unsupported db type %s", testDbType)
	}

	var teardown func()

	// To make sure that the teardown function is not called before any goroutines finished to run -
	// overwrite `spawn` to count the number of running goroutines
	spawnWaitGroup := sync.WaitGroup{}
	realSpawn := spawn
	spawn = func(f func()) {
		spawnWaitGroup.Add(1)
		realSpawn(func() {
			f()
			spawnWaitGroup.Done()
		})
	}

	if config.DB == nil {
		// Create the root directory for test databases.
		if !FileExists(testDbRoot) {
			if err := os.MkdirAll(testDbRoot, 0700); err != nil {
				err := errors.Errorf("unable to create test db "+
					"root: %s", err)
				return nil, nil, err
			}
		}

		dbPath := filepath.Join(testDbRoot, dbName)
		_ = os.RemoveAll(dbPath)
		var err error
		config.DB, err = database.Create(testDbType, dbPath, blockDataNet)
		if err != nil {
			return nil, nil, errors.Errorf("error creating db: %s", err)
		}

		// Setup a teardown function for cleaning up.  This function is
		// returned to the caller to be invoked when it is done testing.
		teardown = func() {
			spawnWaitGroup.Wait()
			spawn = realSpawn
			config.DB.Close()
			os.RemoveAll(dbPath)
			os.RemoveAll(testDbRoot)
		}
	} else {
		teardown = func() {
			spawnWaitGroup.Wait()
			spawn = realSpawn
			config.DB.Close()
		}
	}

	config.TimeSource = NewMedianTime()
	config.SigCache = txscript.NewSigCache(1000)

	// Create the DAG instance.
	dag, err := New(&config)
	if err != nil {
		teardown()
		err := errors.Errorf("failed to create dag instance: %s", err)
		return nil, nil, err
	}
	return dag, teardown, nil
}

// OpTrueScript is script returning TRUE
var OpTrueScript = []byte{txscript.OpTrue}

type txSubnetworkData struct {
	subnetworkID *subnetworkid.SubnetworkID
	Gas          uint64
	Payload      []byte
}

func createTxForTest(numInputs uint32, numOutputs uint32, outputValue uint64, subnetworkData *txSubnetworkData) *wire.MsgTx {
	txIns := []*wire.TxIn{}
	txOuts := []*wire.TxOut{}

	for i := uint32(0); i < numInputs; i++ {
		txIns = append(txIns, &wire.TxIn{
			PreviousOutpoint: *wire.NewOutpoint(&daghash.TxID{}, i),
			SignatureScript:  []byte{},
			Sequence:         wire.MaxTxInSequenceNum,
		})
	}

	for i := uint32(0); i < numOutputs; i++ {
		txOuts = append(txOuts, &wire.TxOut{
			ScriptPubKey: OpTrueScript,
			Value:        outputValue,
		})
	}

	if subnetworkData != nil {
		return wire.NewSubnetworkMsgTx(wire.TxVersion, txIns, txOuts, subnetworkData.subnetworkID, subnetworkData.Gas, subnetworkData.Payload)
	}

	return wire.NewNativeMsgTx(wire.TxVersion, txIns, txOuts)
}

// VirtualForTest is an exported version for virtualBlock, so that it can be returned by exported test_util methods
type VirtualForTest *virtualBlock

// SetVirtualForTest replaces the dag's virtual block. This function is used for test purposes only
func SetVirtualForTest(dag *BlockDAG, virtual VirtualForTest) VirtualForTest {
	oldVirtual := dag.virtual
	dag.virtual = virtual
	return VirtualForTest(oldVirtual)
}

// GetVirtualFromParentsForTest generates a virtual block with the given parents.
func GetVirtualFromParentsForTest(dag *BlockDAG, parentHashes []*daghash.Hash) (VirtualForTest, error) {
	parents := newSet()
	for _, hash := range parentHashes {
		parent := dag.index.LookupNode(hash)
		if parent == nil {
			return nil, errors.Errorf("GetVirtualFromParentsForTest: didn't found node for hash %s", hash)
		}
		parents.add(parent)
	}
	virtual := newVirtualBlock(parents, dag.dagParams.K)

	pastUTXO, acceptanceData, err := dag.pastUTXO(&virtual.blockNode)
	if err != nil {
		return nil, err
	}
	diffFromAcceptanceData, err := virtual.blockNode.diffFromAcceptanceData(pastUTXO, acceptanceData)
	if err != nil {
		return nil, err
	}
	utxo, err := pastUTXO.WithDiff(diffFromAcceptanceData)
	if err != nil {
		return nil, err
	}
	diffUTXO := utxo.clone().(*DiffUTXOSet)
	err = diffUTXO.meldToBase()
	if err != nil {
		return nil, err
	}
	virtual.utxoSet = diffUTXO.base

	return VirtualForTest(virtual), nil
}

// LoadBlocks reads files containing bitcoin block data (gzipped but otherwise
// in the format bitcoind writes) from disk and returns them as an array of
// util.Block.  This is largely borrowed from the test code in btcdb.
func LoadBlocks(filename string) (blocks []*util.Block, err error) {
	var network = wire.MainNet
	var dr io.Reader
	var fi io.ReadCloser

	fi, err = os.Open(filename)
	if err != nil {
		return
	}

	if strings.HasSuffix(filename, ".bz2") {
		dr = bzip2.NewReader(fi)
	} else {
		dr = fi
	}
	defer fi.Close()

	var block *util.Block

	err = nil
	for height := uint64(0); err == nil; height++ {
		var rintbuf uint32
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		if err == io.EOF {
			// hit end of file at expected offset: no warning
			height--
			err = nil
			break
		}
		if err != nil {
			break
		}
		if rintbuf != uint32(network) {
			break
		}
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		blocklen := rintbuf

		rbytes := make([]byte, blocklen)

		// read block
		dr.Read(rbytes)

		block, err = util.NewBlockFromBytes(rbytes)
		if err != nil {
			return
		}
		blocks = append(blocks, block)
	}

	return
}
