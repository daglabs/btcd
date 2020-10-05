package rpchandlers

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/rpc/rpccontext"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
)

// HandleGetMempoolEntries handles the respectively named RPC command
func HandleGetMempoolEntries(context *rpccontext.Context, _ *router.Router, _ appmessage.Message) (appmessage.Message, error) {
	txDescs := context.Mempool.TxDescs()
	entries := make([]*appmessage.MempoolEntry, len(txDescs))
	for i, txDesc := range txDescs {
		transactionVerboseData, err := context.BuildTransactionVerboseData(txDesc.Tx.MsgTx(), txDesc.Tx.ID().String(),
			nil, "", nil, true)
		if err != nil {
			return nil, err
		}
		entries[i] = &appmessage.MempoolEntry{
			Fee:                    txDesc.Fee,
			TransactionVerboseData: transactionVerboseData,
		}
	}
	return appmessage.NewGetMempoolEntriesResponseMessage(entries), nil
}
