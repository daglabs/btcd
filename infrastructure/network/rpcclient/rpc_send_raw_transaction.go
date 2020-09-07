package rpcclient

import (
	"bytes"
	"encoding/hex"
	"github.com/kaspanet/kaspad/app/appmessage"
)

// SubmitTransaction sends an RPC request respective to the function's name and returns the RPC server's response
func (c *RPCClient) SubmitTransaction(msgTx *appmessage.MsgTx) (*appmessage.SubmitTransactionResponseMessage, error) {
	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	if err := msgTx.Serialize(buf); err != nil {
		return nil, err
	}
	transactionHex := hex.EncodeToString(buf.Bytes())
	err := c.rpcRouter.outgoingRoute().Enqueue(appmessage.NewSubmitTransactionRequestMessage(transactionHex))
	if err != nil {
		return nil, err
	}
	response, err := c.route(appmessage.CmdSubmitTransactionResponseMessage).DequeueWithTimeout(c.timeout)
	if err != nil {
		return nil, err
	}
	submitTransactionResponse := response.(*appmessage.SubmitTransactionResponseMessage)
	if submitTransactionResponse.Error != nil {
		return nil, c.convertRPCError(submitTransactionResponse.Error)
	}

	return submitTransactionResponse, nil
}
