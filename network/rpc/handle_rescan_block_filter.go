package rpc

import (
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
)

// rescanBlockFilter rescans a block for any relevant transactions for the
// passed lookup keys. Any discovered transactions are returned hex encoded as
// a string slice.
//
// NOTE: This extension is ported from github.com/decred/dcrd
func rescanBlockFilter(filter *wsClientFilter, block *util.Block, params *dagconfig.Params) []string {
	var transactions []string

	filter.mu.Lock()
	defer filter.mu.Unlock()
	for _, tx := range block.Transactions() {
		msgTx := tx.MsgTx()

		// Keep track of whether the transaction has already been added
		// to the result. It shouldn't be added twice.
		added := false

		// Scan inputs if not a coinbase transaction.
		if !msgTx.IsCoinBase() {
			for _, input := range msgTx.TxIn {
				if !filter.existsUnspentOutpointNoLock(&input.PreviousOutpoint) {
					continue
				}
				if !added {
					transactions = append(
						transactions,
						txHexString(msgTx))
					added = true
				}
			}
		}

		// Scan outputs.
		for i, output := range msgTx.TxOut {
			_, addr, err := txscript.ExtractScriptPubKeyAddress(
				output.ScriptPubKey, params)
			if err != nil {
				continue
			}
			if addr != nil {
				if !filter.existsAddress(addr) {
					continue
				}

				op := domainmessage.Outpoint{
					TxID:  *tx.ID(),
					Index: uint32(i),
				}
				filter.addUnspentOutpoint(&op)

				if !added {
					transactions = append(
						transactions,
						txHexString(msgTx))
					added = true
				}
			}
		}
	}

	return transactions
}
