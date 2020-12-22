package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionid"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"net/http"

	"github.com/kaspanet/go-secp256k1"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kasparov/apimodels"
	"github.com/pkg/errors"
)

const feeSompis uint64 = 1000

func send(conf *sendConfig) error {
	toAddress, err := util.DecodeAddress(conf.ToAddress, util.Bech32PrefixUnknown)
	if err != nil {
		return err
	}

	keyPair, publicKey, err := parsePrivateKey(conf.PrivateKey)
	if err != nil {
		return err
	}

	serializedPublicKey, err := publicKey.Serialize()
	if err != nil {
		return err
	}
	fromAddress, err := util.NewAddressPubKeyHashFromPublicKey(serializedPublicKey[:], toAddress.Prefix())
	if err != nil {
		return err
	}

	utxos, err := getUTXOs(conf.KasparovAddress, fromAddress.String())
	if err != nil {
		return err
	}

	sendAmountSompi := uint64(conf.SendAmount * util.SompiPerKaspa)
	totalToSend := sendAmountSompi + feeSompis

	selectedUTXOs, changeSompi, err := selectUTXOs(utxos, totalToSend)
	if err != nil {
		return err
	}

	msgTx, err := generateTx(keyPair, selectedUTXOs, sendAmountSompi, changeSompi, toAddress, fromAddress)
	if err != nil {
		return err
	}

	err = sendTx(conf, msgTx)
	if err != nil {
		return err
	}

	fmt.Println("Transaction was sent successfully")
	fmt.Printf("Transaction ID: \t%s", msgTx.TxID())

	return nil
}

func parsePrivateKey(privateKeyHex string) (*secp256k1.SchnorrKeyPair, *secp256k1.SchnorrPublicKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error parsing private key hex")
	}
	keyPair, err := secp256k1.DeserializePrivateKeyFromSlice(privateKeyBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error deserializing private key")
	}
	publicKey, err := keyPair.SchnorrPublicKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error generating public key")
	}
	return keyPair, publicKey, nil
}

func selectUTXOs(utxos []*apimodels.TransactionOutputResponse, totalToSpend uint64) (
	selectedUTXOs []*apimodels.TransactionOutputResponse, changeSompi uint64, err error) {

	selectedUTXOs = []*apimodels.TransactionOutputResponse{}
	totalValue := uint64(0)

	for _, utxo := range utxos {
		if utxo.IsSpendable == nil || !*utxo.IsSpendable {
			continue
		}

		selectedUTXOs = append(selectedUTXOs, utxo)
		totalValue += utxo.Value

		if totalValue >= totalToSpend {
			break
		}
	}

	if totalValue < totalToSpend {
		return nil, 0, errors.Errorf("Insufficient funds for send: %f required, while only %f available",
			float64(totalToSpend)/util.SompiPerKaspa, float64(totalValue)/util.SompiPerKaspa)
	}

	return selectedUTXOs, totalValue - totalToSpend, nil
}

func generateTx(keyPair *secp256k1.SchnorrKeyPair, selectedUTXOs []*apimodels.TransactionOutputResponse, sompisToSend uint64, change uint64,
	toAddress util.Address, fromAddress util.Address) (*appmessage.MsgTx, error) {

	txIns := make([]*appmessage.TxIn, len(selectedUTXOs))
	for i, utxo := range selectedUTXOs {
		txID, err := transactionid.FromString(utxo.TransactionID)
		if err != nil {
			return nil, err
		}

		txIns[i] = appmessage.NewTxIn(appmessage.NewOutpoint(txID, utxo.Index), []byte{})
	}

	toScript, err := txscript.PayToAddrScript(toAddress)
	if err != nil {
		return nil, err
	}
	mainTxOut := appmessage.NewTxOut(sompisToSend, toScript)

	fromScript, err := txscript.PayToAddrScript(fromAddress)
	if err != nil {
		return nil, err
	}
	changeTxOut := appmessage.NewTxOut(change, fromScript)

	txOuts := []*appmessage.TxOut{mainTxOut, changeTxOut}

	msgTx := appmessage.NewNativeMsgTx(constants.TransactionVersion, txIns, txOuts)
	domainTransaction := appmessage.MsgTxToDomainTransaction(msgTx)

	for i, txIn := range msgTx.TxIn {
		signatureScript, err := txscript.SignatureScript(domainTransaction, i, fromScript, txscript.SigHashAll, keyPair)
		if err != nil {
			return nil, err
		}
		txIn.SignatureScript = signatureScript
	}

	return msgTx, nil
}

func sendTx(conf *sendConfig, msgTx *appmessage.MsgTx) error {
	txBuffer := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	if err := msgTx.KaspaEncode(txBuffer, 0); err != nil {
		return err
	}

	txHex := hex.EncodeToString(txBuffer.Bytes())
	rawTx := &apimodels.RawTransaction{
		RawTransaction: txHex,
	}
	txBytes, err := json.Marshal(rawTx)
	if err != nil {
		return errors.Wrap(err, "Error marshalling transaction to json")
	}

	requestURL, err := resourceURL(conf.KasparovAddress, sendTransactionEndpoint)
	if err != nil {
		return err
	}
	response, err := http.Post(requestURL, "application/json", bytes.NewBuffer(txBytes))
	if err != nil {
		return errors.Wrap(err, "Error posting transaction to server")
	}
	_, err = readResponse(response)
	if err != nil {
		return errors.Wrap(err, "Error reading send transaction response")
	}

	return err
}
