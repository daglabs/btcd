package protowire

import "github.com/kaspanet/kaspad/app/appmessage"

func (x *KaspadMessage_NotifyChainChangedRequest) toAppMessage() (appmessage.Message, error) {
	return &appmessage.NotifyChainChangedRequestMessage{}, nil
}

func (x *KaspadMessage_NotifyChainChangedRequest) fromAppMessage(_ *appmessage.NotifyChainChangedRequestMessage) error {
	x.NotifyChainChangedRequest = &NotifyChainChangedRequestMessage{}
	return nil
}

func (x *KaspadMessage_NotifyChainChangedResponse) toAppMessage() (appmessage.Message, error) {
	var err *appmessage.RPCError
	if x.NotifyChainChangedResponse.Error != nil {
		err = &appmessage.RPCError{Message: x.NotifyChainChangedResponse.Error.Message}
	}
	return &appmessage.NotifyChainChangedResponseMessage{
		Error: err,
	}, nil
}

func (x *KaspadMessage_NotifyChainChangedResponse) fromAppMessage(message *appmessage.NotifyChainChangedResponseMessage) error {
	var err *RPCError
	if message.Error != nil {
		err = &RPCError{Message: message.Error.Message}
	}
	x.NotifyChainChangedResponse = &NotifyChainChangedResponseMessage{
		Error: err,
	}
	return nil
}

func (x *KaspadMessage_ChainChangedNotification) toAppMessage() (appmessage.Message, error) {
	addedChainBlocks := make([]*appmessage.ChainChangedChainBlock, len(x.ChainChangedNotification.AddedChainBlocks))
	for i, addedChainBlock := range x.ChainChangedNotification.AddedChainBlocks {
		appAddedChainBlock, err := addedChainBlock.toAppMessage()
		if err != nil {
			return nil, err
		}
		addedChainBlocks[i] = appAddedChainBlock
	}
	return &appmessage.ChainChangedNotificationMessage{
		RemovedChainBlockHashes: x.ChainChangedNotification.RemovedChainBlockHashes,
		AddedChainBlocks:        addedChainBlocks,
	}, nil
}

func (x *KaspadMessage_ChainChangedNotification) fromAppMessage(message *appmessage.ChainChangedNotificationMessage) error {
	addedChainBlocks := make([]*ChainChangedChainBlock, len(message.AddedChainBlocks))
	for i, addedChainBlock := range message.AddedChainBlocks {
		protoAddedChainBlock := &ChainChangedChainBlock{}
		err := protoAddedChainBlock.fromAppMessage(addedChainBlock)
		if err != nil {
			return err
		}
		addedChainBlocks[i] = protoAddedChainBlock
	}
	x.ChainChangedNotification = &ChainChangedNotificationMessage{
		RemovedChainBlockHashes: message.RemovedChainBlockHashes,
		AddedChainBlocks:        addedChainBlocks,
	}
	return nil
}

func (x *ChainChangedChainBlock) toAppMessage() (*appmessage.ChainChangedChainBlock, error) {
	acceptedBlocks := make([]*appmessage.ChainChangedAcceptedBlock, len(x.AcceptedBlocks))
	for j, acceptedBlock := range x.AcceptedBlocks {
		acceptedBlocks[j] = &appmessage.ChainChangedAcceptedBlock{
			Hash:          acceptedBlock.Hash,
			AcceptedTxIDs: acceptedBlock.AcceptedTxIds,
		}
	}
	return &appmessage.ChainChangedChainBlock{
		Hash:           x.Hash,
		AcceptedBlocks: acceptedBlocks,
	}, nil
}

func (x *ChainChangedChainBlock) fromAppMessage(message *appmessage.ChainChangedChainBlock) error {
	acceptedBlocks := make([]*ChainChangedAcceptedBlock, len(message.AcceptedBlocks))
	for j, acceptedBlock := range message.AcceptedBlocks {
		acceptedBlocks[j] = &ChainChangedAcceptedBlock{
			Hash:          acceptedBlock.Hash,
			AcceptedTxIds: acceptedBlock.AcceptedTxIDs,
		}
	}
	*x = ChainChangedChainBlock{
		Hash:           message.Hash,
		AcceptedBlocks: acceptedBlocks,
	}
	return nil
}
