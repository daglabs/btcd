package domainmessage

import (
	"github.com/kaspanet/kaspad/util/daghash"
)

// MsgSelectedTip implements the Message interface and represents a kaspa
// selectedtip message. It is used to answer getseltip messages and tell
// the asking peer what is the selected tip of this peer.
type MsgSelectedTip struct {
	baseMessage
	// The selected tip hash of the generator of the message.
	SelectedTipHash *daghash.Hash
}

// Command returns the protocol command string for the message. This is part
// of the Message interface implementation.
func (msg *MsgSelectedTip) Command() MessageCommand {
	return CmdSelectedTip
}

// NewMsgSelectedTip returns a new kaspa selectedtip message that conforms to the
// Message interface.
func NewMsgSelectedTip(selectedTipHash *daghash.Hash) *MsgSelectedTip {
	return &MsgSelectedTip{
		SelectedTipHash: selectedTipHash,
	}
}
