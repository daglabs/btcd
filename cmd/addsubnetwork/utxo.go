package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/daglabs/btcd/btcjson"
	"github.com/daglabs/btcd/rpcclient"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/wire"
)

const (
	resultsCount     = 1000
	minConfirmations = 10
)

func findUnspentTXO(client *rpcclient.Client, addrPubKeyHash *util.AddressPubKeyHash) (*wire.OutPoint, *wire.MsgTx, error) {
	txs, err := collectTransactions(client, addrPubKeyHash)
	if err != nil {
		return nil, nil, err
	}

	utxos := buildUTXOs(txs)
	for outPoint, tx := range utxos {
		return &outPoint, tx, nil
	}

	return nil, nil, nil
}

func collectTransactions(client *rpcclient.Client, addrPubKeyHash *util.AddressPubKeyHash) ([]*wire.MsgTx, error) {
	txs := make([]*wire.MsgTx, 0)
	skip := 0
	for {
		results, err := client.SearchRawTransactionsVerbose(addrPubKeyHash, skip, resultsCount, true, false, nil)
		if err != nil {
			// Break when there are no further txs
			if rpcError, ok := err.(*btcjson.RPCError); ok && rpcError.Code == btcjson.ErrRPCNoTxInfo {
				break
			}

			return nil, err
		}

		for _, result := range results {
			tx, err := parseRawTransactionResult(result)
			if err != nil {
				return nil, fmt.Errorf("failed to process SearchRawTransactionResult: %s", err)
			}
			if tx == nil {
				continue
			}
			if !isTxMatured(tx, *result.Confirmations) {
				continue
			}

			txs = append(txs, tx)
		}

		skip += resultsCount
	}
	return txs, nil
}

func parseRawTransactionResult(result *btcjson.SearchRawTransactionsResult) (*wire.MsgTx, error) {
	txBytes, err := hex.DecodeString(result.Hex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction bytes: %s", err)
	}
	var tx wire.MsgTx
	reader := bytes.NewReader(txBytes)
	err = tx.Deserialize(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %s", err)
	}
	return &tx, nil
}

func isTxMatured(tx *wire.MsgTx, confirmations uint64) bool {
	if !tx.IsBlockReward() {
		return confirmations >= minConfirmations
	}
	return confirmations >= activeNetParams.BlockRewardMaturity
}

func buildUTXOs(txs []*wire.MsgTx) map[wire.OutPoint]*wire.MsgTx {
	utxos := make(map[wire.OutPoint]*wire.MsgTx)
	for _, tx := range txs {
		for i := range tx.TxOut {
			outPoint := wire.NewOutPoint(tx.TxID(), uint32(i))
			utxos[*outPoint] = tx
		}
	}
	for _, tx := range txs {
		for _, input := range tx.TxIn {
			delete(utxos, input.PreviousOutPoint)
		}
	}
	return utxos
}
