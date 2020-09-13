package appmessage

// SubmitTransactionRequestMessage is an appmessage corresponding to
// its respective RPC message
type SubmitTransactionRequestMessage struct {
	baseMessage
	TransactionHex string
}

// Command returns the protocol command string for the message
func (msg *SubmitTransactionRequestMessage) Command() MessageCommand {
	return CmdSubmitTransactionRequestMessage
}

// NewSubmitTransactionRequestMessage returns a instance of the message
func NewSubmitTransactionRequestMessage(transactionHex string) *SubmitTransactionRequestMessage {
	return &SubmitTransactionRequestMessage{
		TransactionHex: transactionHex,
	}
}

// SubmitTransactionResponseMessage is an appmessage corresponding to
// its respective RPC message
type SubmitTransactionResponseMessage struct {
	baseMessage
	TxID string

	Error *RPCError
}

// Command returns the protocol command string for the message
func (msg *SubmitTransactionResponseMessage) Command() MessageCommand {
	return CmdSubmitTransactionResponseMessage
}

// NewSubmitTransactionResponseMessage returns a instance of the message
func NewSubmitTransactionResponseMessage(txID string) *SubmitTransactionResponseMessage {
	return &SubmitTransactionResponseMessage{
		TxID: txID,
	}
}
