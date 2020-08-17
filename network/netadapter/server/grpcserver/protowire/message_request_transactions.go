package protowire

import (
	"github.com/kaspanet/kaspad/network/appmessage"
	"github.com/pkg/errors"
)

func (x *KaspadMessage_RequestTransactions) toDomainMessage() (appmessage.Message, error) {
	if len(x.RequestTransactions.Ids) > appmessage.MaxInvPerRequestTransactionsMsg {
		return nil, errors.Errorf("too many hashes for message "+
			"[count %d, max %d]", len(x.RequestTransactions.Ids), appmessage.MaxInvPerRequestTransactionsMsg)
	}

	ids, err := protoTransactionIDsToWire(x.RequestTransactions.Ids)
	if err != nil {
		return nil, err
	}
	return &appmessage.MsgRequestTransactions{IDs: ids}, nil
}

func (x *KaspadMessage_RequestTransactions) fromDomainMessage(msgGetTransactions *appmessage.MsgRequestTransactions) error {
	if len(msgGetTransactions.IDs) > appmessage.MaxInvPerRequestTransactionsMsg {
		return errors.Errorf("too many hashes for message "+
			"[count %d, max %d]", len(x.RequestTransactions.Ids), appmessage.MaxInvPerRequestTransactionsMsg)
	}

	x.RequestTransactions = &RequestTransactionsMessage{
		Ids: wireTransactionIDsToProto(msgGetTransactions.IDs),
	}
	return nil
}
