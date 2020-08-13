package protowire

import (
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/pkg/errors"
)

func (x *KaspadMessage_Transaction) toDomainMessage() (domainmessage.Message, error) {
	return x.Transaction.toDomainMessage()
}

func (x *KaspadMessage_Transaction) fromDomainMessage(msgTx *domainmessage.MsgTx) error {
	x.Transaction = new(TransactionMessage)
	x.Transaction.fromDomainMessage(msgTx)
	return nil
}

func (x *TransactionMessage) toDomainMessage() (domainmessage.Message, error) {
	inputs := make([]*domainmessage.TxIn, len(x.Inputs))
	for i, protoInput := range x.Inputs {
		prevTxID, err := protoInput.PreviousOutpoint.TransactionID.toWire()
		if err != nil {
			return nil, err
		}

		outpoint := domainmessage.NewOutpoint(prevTxID, protoInput.PreviousOutpoint.Index)
		inputs[i] = domainmessage.NewTxIn(outpoint, protoInput.SignatureScript)
	}

	outputs := make([]*domainmessage.TxOut, len(x.Outputs))
	for i, protoOutput := range x.Outputs {
		outputs[i] = &domainmessage.TxOut{
			Value:        protoOutput.Value,
			ScriptPubKey: protoOutput.ScriptPubKey,
		}
	}

	if x.SubnetworkID == nil {
		return nil, errors.New("transaction subnetwork field cannot be nil")
	}

	subnetworkID, err := x.SubnetworkID.toWire()
	if err != nil {
		return nil, err
	}

	var payloadHash *daghash.Hash
	if x.PayloadHash != nil {
		payloadHash, err = x.PayloadHash.toWire()
		if err != nil {
			return nil, err
		}
	}

	return &domainmessage.MsgTx{
		Version:      x.Version,
		TxIn:         inputs,
		TxOut:        outputs,
		LockTime:     x.LockTime,
		SubnetworkID: *subnetworkID,
		Gas:          x.Gas,
		PayloadHash:  payloadHash,
		Payload:      x.Payload,
	}, nil
}

func (x *TransactionMessage) fromDomainMessage(msgTx *domainmessage.MsgTx) {
	protoInputs := make([]*TransactionInput, len(msgTx.TxIn))
	for i, input := range msgTx.TxIn {
		protoInputs[i] = &TransactionInput{
			PreviousOutpoint: &Outpoint{
				TransactionID: wireTransactionIDToProto(&input.PreviousOutpoint.TxID),
				Index:         input.PreviousOutpoint.Index,
			},
			SignatureScript: input.SignatureScript,
			Sequence:        input.Sequence,
		}
	}

	protoOutputs := make([]*TransactionOutput, len(msgTx.TxOut))
	for i, output := range msgTx.TxOut {
		protoOutputs[i] = &TransactionOutput{
			Value:        output.Value,
			ScriptPubKey: output.ScriptPubKey,
		}
	}

	var payloadHash *Hash
	if msgTx.PayloadHash != nil {
		payloadHash = wireHashToProto(msgTx.PayloadHash)
	}
	*x = TransactionMessage{
		Version:      msgTx.Version,
		Inputs:       protoInputs,
		Outputs:      protoOutputs,
		LockTime:     msgTx.LockTime,
		SubnetworkID: wireSubnetworkIDToProto(&msgTx.SubnetworkID),
		Gas:          msgTx.Gas,
		PayloadHash:  payloadHash,
		Payload:      msgTx.Payload,
	}

}
