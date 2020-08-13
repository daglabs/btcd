package protowire

import (
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/pkg/errors"
)

func (x *KaspadMessage_InvTransactions) toDomainMessage() (domainmessage.Message, error) {
	if len(x.InvTransactions.Ids) > domainmessage.MaxInvPerTxInvMsg {
		return nil, errors.Errorf("too many hashes for message "+
			"[count %d, max %d]", len(x.InvTransactions.Ids), domainmessage.MaxInvPerTxInvMsg)
	}

	ids, err := protoTransactionIDsToWire(x.InvTransactions.Ids)
	if err != nil {
		return nil, err
	}
	return &domainmessage.MsgInvTransaction{TxIDs: ids}, nil
}

func (x *KaspadMessage_InvTransactions) fromDomainMessage(msgInvTransaction *domainmessage.MsgInvTransaction) error {
	if len(msgInvTransaction.TxIDs) > domainmessage.MaxInvPerTxInvMsg {
		return errors.Errorf("too many hashes for message "+
			"[count %d, max %d]", len(msgInvTransaction.TxIDs), domainmessage.MaxInvPerTxInvMsg)
	}

	x.InvTransactions = &InvTransactionsMessage{
		Ids: wireTransactionIDsToProto(msgInvTransaction.TxIDs),
	}
	return nil
}
