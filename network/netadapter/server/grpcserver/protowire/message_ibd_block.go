package protowire

import "github.com/kaspanet/kaspad/network/appmessage"

func (x *KaspadMessage_IbdBlock) toAppMessage() (appmessage.Message, error) {
	msgBlock, err := x.IbdBlock.toAppMessage()
	if err != nil {
		return nil, err
	}
	return &appmessage.MsgIBDBlock{MsgBlock: msgBlock.(*appmessage.MsgBlock)}, nil
}

func (x *KaspadMessage_IbdBlock) fromAppMessage(msgIBDBlock *appmessage.MsgIBDBlock) error {
	x.IbdBlock = new(BlockMessage)
	return x.IbdBlock.fromAppMessage(msgIBDBlock.MsgBlock)
}
