package protowire

import "github.com/kaspanet/kaspad/app/appmessage"

func (x *KaspadMessage_SubmitBlockRequest) toAppMessage() (appmessage.Message, error) {
	blockAppMessage, err := x.SubmitBlockRequest.Block.toAppMessage()
	if err != nil {
		return nil, err
	}

	return &appmessage.SubmitBlockRequestMessage{
		Block: blockAppMessage,
	}, nil
}

func (x *KaspadMessage_SubmitBlockRequest) fromAppMessage(message *appmessage.SubmitBlockRequestMessage) error {
	x.SubmitBlockRequest = &SubmitBlockRequestMessage{Block: &BlockMessage{}}
	return x.SubmitBlockRequest.Block.fromAppMessage(message.Block)
}

func (x *KaspadMessage_SubmitBlockResponse) toAppMessage() (appmessage.Message, error) {
	var err *appmessage.RPCError
	if x.SubmitBlockResponse.Error != nil {
		err = &appmessage.RPCError{Message: x.SubmitBlockResponse.Error.Message}
	}
	return &appmessage.SubmitBlockResponseMessage{
		RejectReason: appmessage.RejectReason(x.SubmitBlockResponse.RejectReason),
		Error:        err,
	}, nil
}

func (x *KaspadMessage_SubmitBlockResponse) fromAppMessage(message *appmessage.SubmitBlockResponseMessage) error {
	var err *RPCError
	if message.Error != nil {
		err = &RPCError{Message: message.Error.Message}
	}
	x.SubmitBlockResponse = &SubmitBlockResponseMessage{
		RejectReason: SubmitBlockResponseMessage_RejectReason(message.RejectReason),
		Error:        err,
	}
	return nil
}
