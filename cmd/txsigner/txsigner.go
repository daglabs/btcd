package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/kaspanet/go-secp256k1"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
	"github.com/pkg/errors"
	"os"
)

func main() {
	cfg, err := parseCommandLine()
	if err != nil {
		printErrorAndExit(err, "Failed to parse arguments")
	}

	privateKey, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		printErrorAndExit(err, "Failed to decode private key")
	}

	transaction, err := parseTransaction(cfg.Transaction)
	if err != nil {
		printErrorAndExit(err, "Failed to decode transaction")
	}

	pubkey, err := privateKey.SchnorrPublicKey()
	if err != nil {
		printErrorAndExit(err, "Failed to generate a public key")
	}
	scriptPubKey, err := createScriptPubKey(pubkey)
	if err != nil {
		printErrorAndExit(err, "Failed to create scriptPubKey")
	}

	err = signTransaction(transaction, privateKey, scriptPubKey)
	if err != nil {
		printErrorAndExit(err, "Failed to sign transaction")
	}

	serializedTransaction, err := serializeTransaction(transaction)
	if err != nil {
		printErrorAndExit(err, "Failed to serialize transaction")
	}

	fmt.Printf("Signed Transaction (hex): %s\n\n", serializedTransaction)
}

func parsePrivateKey(privateKeyHex string) (*secp256k1.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, errors.Errorf("'%s' isn't a valid hex. err: '%s' ", privateKeyHex, err)
	}
	return secp256k1.DeserializePrivateKeyFromSlice(privateKeyBytes)
}

func parseTransaction(transactionHex string) (*domainmessage.MsgTx, error) {
	serializedTx, err := hex.DecodeString(transactionHex)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't decode transaction hex")
	}
	var transaction domainmessage.MsgTx
	err = transaction.Deserialize(bytes.NewReader(serializedTx))
	return &transaction, err
}

func createScriptPubKey(publicKey *secp256k1.SchnorrPublicKey) ([]byte, error) {
	serializedKey, err := publicKey.SerializeCompressed()
	if err != nil {
		return nil, err
	}
	p2pkhAddress, err := util.NewAddressPubKeyHashFromPublicKey(serializedKey, ActiveConfig().NetParams().Prefix)
	if err != nil {
		return nil, err
	}
	scriptPubKey, err := txscript.PayToAddrScript(p2pkhAddress)
	return scriptPubKey, err
}

func signTransaction(transaction *domainmessage.MsgTx, privateKey *secp256k1.PrivateKey, scriptPubKey []byte) error {
	for i, transactionInput := range transaction.TxIn {
		signatureScript, err := txscript.SignatureScript(transaction, i, scriptPubKey, txscript.SigHashAll, privateKey, true)
		if err != nil {
			return err
		}
		transactionInput.SignatureScript = signatureScript
	}
	return nil
}

func serializeTransaction(transaction *domainmessage.MsgTx) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, transaction.SerializeSize()))
	err := transaction.Serialize(buf)
	serializedTransaction := hex.EncodeToString(buf.Bytes())
	return serializedTransaction, err
}

func printErrorAndExit(err error, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s", message, err)
	os.Exit(1)
}
