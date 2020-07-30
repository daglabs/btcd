package wire

import (
	"github.com/kaspanet/kaspad/util/daghash"
)

// MaxInvPerGetTransactionsMsg is the maximum number of hashes that can
// be in a single CmdInvTransaction message.
const MaxInvPerGetTransactionsMsg = MaxInvPerMsg

// MsgRequestTransactions implements the Message interface and represents a kaspa
// RequestTransactions message. It is used to request transactions as part of the
// transactions relay protocol.
type MsgRequestTransactions struct {
	IDs []*daghash.TxID
}

// Command returns the protocol command string for the message. This is part
// of the Message interface implementation.
func (msg *MsgRequestTransactions) Command() MessageCommand {
	return CmdRequestTransactions
}

// NewMsgGetTransactions returns a new kaspa RequestTransactions message that conforms to
// the Message interface. See MsgRequestTransactions for details.
func NewMsgGetTransactions(ids []*daghash.TxID) *MsgRequestTransactions {
	return &MsgRequestTransactions{
		IDs: ids,
	}
}
