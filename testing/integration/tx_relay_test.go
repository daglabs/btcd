package integration

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/kaspanet/go-secp256k1"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
)

func TestTxRelay(t *testing.T) {
	payer, mediator, payee, teardown := standardSetup(t)
	defer teardown()

	// Connect nodes in chain: payer <--> mediator <--> payee
	// So that payee doesn't directly get transactions from payer
	connect(t, payer, mediator)
	connect(t, mediator, payee)

	payeeBlockAddedChan := make(chan *domainmessage.BlockHeader)
	setOnBlockAddedHandler(t, payee, func(header *domainmessage.BlockHeader) {
		payeeBlockAddedChan <- header
	})
	// skip the first block because it's paying to genesis script
	mineNextBlock(t, payer)
	waitForPayeeToReceiveBlock(t, payeeBlockAddedChan)
	// use the second block to get money to pay with
	secondBlock := mineNextBlock(t, payer)
	waitForPayeeToReceiveBlock(t, payeeBlockAddedChan)

	// Mine BlockCoinbaseMaturity more blocks for our money to mature
	for i := uint64(0); i < payer.config.ActiveNetParams.BlockCoinbaseMaturity; i++ {
		mineNextBlock(t, payer)
		waitForPayeeToReceiveBlock(t, payeeBlockAddedChan)
	}

	tx := generateTx(t, secondBlock.CoinbaseTransaction().MsgTx(), payer, payee)
	txID, err := payer.rpcClient.SendRawTransaction(tx, true)
	if err != nil {
		t.Fatalf("Error submitting transaction: %+v", err)
	}

	txAddedToMempoolChan := make(chan struct{})

	spawn("TestTxRelay-WaitForTransactionPropagation", func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			_, err := payee.rpcClient.GetMempoolEntry(txID.String())
			if err != nil {
				if strings.Contains(err.Error(), "-32603: transaction is not in the pool") {
					continue
				}

				t.Fatalf("Error getting mempool entry: %+v", err)
			}
			close(txAddedToMempoolChan)
		}
	})

	select {
	case <-txAddedToMempoolChan:
	case <-time.After(defaultTimeout):
		t.Fatalf("Timeout waiting for transaction to be accepted into mempool")
	}
}

func waitForPayeeToReceiveBlock(t *testing.T, payeeBlockAddedChan chan *domainmessage.BlockHeader) {
	select {
	case <-payeeBlockAddedChan:
	case <-time.After(defaultTimeout):
		t.Fatalf("Timeout waiting for block added")
	}
}

func generateTx(t *testing.T, firstBlockCoinbase *domainmessage.MsgTx, payer, payee *appHarness) *domainmessage.MsgTx {
	txIns := make([]*domainmessage.TxIn, 1)
	txIns[0] = domainmessage.NewTxIn(domainmessage.NewOutpoint(firstBlockCoinbase.TxID(), 0), []byte{})

	payeeAddress, err := util.DecodeAddress(payee.miningAddress, util.Bech32PrefixKaspaSim)
	if err != nil {
		t.Fatalf("Error decoding payeeAddress: %+v", err)
	}
	toScript, err := txscript.PayToAddrScript(payeeAddress)
	if err != nil {
		t.Fatalf("Error generating script: %+v", err)
	}

	txOuts := []*domainmessage.TxOut{domainmessage.NewTxOut(firstBlockCoinbase.TxOut[0].Value-1, toScript)}

	fromScript := firstBlockCoinbase.TxOut[0].ScriptPubKey

	tx := domainmessage.NewNativeMsgTx(domainmessage.TxVersion, txIns, txOuts)

	privateKeyBytes, err := hex.DecodeString(payer.miningAddressPrivateKey)
	if err != nil {
		t.Fatalf("Error decoding private key: %+v", err)
	}
	privateKey, err := secp256k1.DeserializePrivateKeyFromSlice(privateKeyBytes)
	if err != nil {
		t.Fatalf("Error deserializing private key: %+v", err)
	}

	signatureScript, err := txscript.SignatureScript(tx, 0, fromScript, txscript.SigHashAll, privateKey, true)
	if err != nil {
		t.Fatalf("Error signing transaction: %+v", err)
	}
	tx.TxIn[0].SignatureScript = signatureScript

	return tx
}
