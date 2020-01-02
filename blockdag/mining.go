package blockdag

import (
	"bytes"
	"encoding/binary"
	"github.com/kaspanet/kaspad/txscript"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
	"sort"
	"time"
)

// BlockForMining returns a block with the given transactions
// that points to the current DAG tips, that is valid from
// all aspects except proof of work.
func (dag *BlockDAG) BlockForMining(transactions []*util.Tx, useMinimalTime bool) (*wire.MsgBlock, error) {
	dag.dagLock.Lock()
	defer dag.dagLock.Unlock()

	var blockTimestamp time.Time
	if useMinimalTime {
		blockTimestamp = MinimumMedianTime(dag.CalcPastMedianTime())
	} else {
		blockTimestamp = MedianAdjustedTime(dag.CalcPastMedianTime(), dag.timeSource)
	}
	requiredDifficulty := dag.NextRequiredDifficulty(blockTimestamp)

	// Calculate the next expected block version based on the state of the
	// rule change deployments.
	nextBlockVersion, err := dag.CalcNextBlockVersion()
	if err != nil {
		return nil, err
	}

	// Sort transactions by subnetwork ID before building Merkle tree
	sort.Slice(transactions, func(i, j int) bool {
		if transactions[i].MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDCoinbase) {
			return true
		}
		if transactions[j].MsgTx().SubnetworkID.IsEqual(subnetworkid.SubnetworkIDCoinbase) {
			return false
		}
		return subnetworkid.Less(&transactions[i].MsgTx().SubnetworkID, &transactions[j].MsgTx().SubnetworkID)
	})

	// Create a new block ready to be solved.
	hashMerkleTree := BuildHashMerkleTreeStore(transactions)
	acceptedIDMerkleRoot, err := dag.NextAcceptedIDMerkleRootNoLock()
	if err != nil {
		return nil, err
	}
	var msgBlock wire.MsgBlock
	for _, tx := range transactions {
		msgBlock.AddTransaction(tx.MsgTx())
	}

	utxoWithTransactions, err := dag.UTXOSet().WithTransactions(msgBlock.Transactions, UnacceptedBlueScore, false)
	if err != nil {
		return nil, err
	}
	utxoCommitment := utxoWithTransactions.Multiset().Hash()

	msgBlock.Header = wire.BlockHeader{
		Version:              nextBlockVersion,
		ParentHashes:         dag.TipHashes(),
		HashMerkleRoot:       hashMerkleTree.Root(),
		AcceptedIDMerkleRoot: acceptedIDMerkleRoot,
		UTXOCommitment:       utxoCommitment,
		Timestamp:            blockTimestamp,
		Bits:                 requiredDifficulty,
	}

	return &msgBlock, nil
}

// CoinbasePayloadExtraData returns coinbase payload extra data parameter
// which is built from extra nonce and coinbase flags.
func CoinbasePayloadExtraData(extraNonce uint64, coinbaseFlags string) ([]byte, error) {
	extraNonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(extraNonceBytes, extraNonce)
	w := &bytes.Buffer{}
	_, err := w.Write(extraNonceBytes)
	if err != nil {
		return nil, err
	}
	_, err = w.Write([]byte(coinbaseFlags))
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// NextCoinbaseFromAddress returns a coinbase transaction for the
// next block with the given address and extra data in its payload.
func (dag *BlockDAG) NextCoinbaseFromAddress(payToAddress util.Address, extraData []byte) (*util.Tx, error) {
	coinbasePayloadScriptPubKey, err := txscript.PayToAddrScript(payToAddress)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := dag.NextBlockCoinbaseTransaction(coinbasePayloadScriptPubKey, extraData)
	if err != nil {
		return nil, err
	}
	return coinbaseTx, nil
}

// MinimumMedianTime returns the minimum allowed timestamp for a block building
// on the end of the DAG. In particular, it is one second after
// the median timestamp of the last several blocks per the DAG consensus
// rules.
func MinimumMedianTime(dagMedianTime time.Time) time.Time {
	return dagMedianTime.Add(time.Second)
}

// MedianAdjustedTime returns the current time adjusted to ensure it is at least
// one second after the median timestamp of the last several blocks per the
// DAG consensus rules.
func MedianAdjustedTime(dagMedianTime time.Time, timeSource MedianTimeSource) time.Time {
	// The timestamp for the block must not be before the median timestamp
	// of the last several blocks. Thus, choose the maximum between the
	// current time and one second after the past median time. The current
	// timestamp is truncated to a second boundary before comparison since a
	// block timestamp does not supported a precision greater than one
	// second.
	newTimestamp := timeSource.AdjustedTime()
	minTimestamp := MinimumMedianTime(dagMedianTime)
	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}
