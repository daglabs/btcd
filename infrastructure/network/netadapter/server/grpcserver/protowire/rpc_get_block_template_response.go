package protowire

import "github.com/kaspanet/kaspad/app/appmessage"

func (x *KaspadMessage_GetBlockTemplateResponse) toAppMessage() (appmessage.Message, error) {
	return &appmessage.GetBlockTemplateRequestMessage{}, nil
}

func (x *KaspadMessage_GetBlockTemplateResponse) fromAppMessage(_ *appmessage.GetBlockTemplateResponseMessage) error {
	return nil
}
