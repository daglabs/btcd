package main

import (
	"github.com/daglabs/btcd/btcec"
	"github.com/daglabs/btcd/txscript"
	"github.com/daglabs/btcd/wire"
	"github.com/pkg/errors"
)

func buildSubnetworkRegistryTx(cfg *configFlags, fundingOutpoint *wire.Outpoint, fundingTx *wire.MsgTx, privateKey *btcec.PrivateKey) (*wire.MsgTx, error) {
	txIn := &wire.TxIn{
		PreviousOutpoint: *fundingOutpoint,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	txOut := &wire.TxOut{
		ScriptPubKey: fundingTx.TxOut[fundingOutpoint.Index].ScriptPubKey,
		Value:        fundingTx.TxOut[fundingOutpoint.Index].Value - cfg.RegistryTxFee,
	}
	registryTx := wire.NewRegistryMsgTx(1, []*wire.TxIn{txIn}, []*wire.TxOut{txOut}, cfg.GasLimit)

	SignatureScript, err := txscript.SignatureScript(registryTx, 0, fundingTx.TxOut[fundingOutpoint.Index].ScriptPubKey,
		txscript.SigHashAll, privateKey, true)
	if err != nil {
		return nil, errors.Errorf("failed to build signature script: %s", err)
	}
	txIn.SignatureScript = SignatureScript

	return registryTx, nil
}
