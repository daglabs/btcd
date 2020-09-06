package rpcclient

import "github.com/kaspanet/kaspad/app/appmessage"

func (c *RPCClient) GetSubnetwork(subnetworkID string) (*appmessage.GetSubnetworkResponseMessage, error) {
	err := c.rpcRouter.outgoingRoute().Enqueue(appmessage.NewGetSubnetworkRequestMessage(subnetworkID))
	if err != nil {
		return nil, err
	}
	response, err := c.route(appmessage.CmdGetSubnetworkResponseMessage).DequeueWithTimeout(c.timeout)
	if err != nil {
		return nil, err
	}
	getSubnetworkResponse := response.(*appmessage.GetSubnetworkResponseMessage)
	if getSubnetworkResponse.Error != nil {
		return nil, c.convertRPCError(getSubnetworkResponse.Error)
	}
	return getSubnetworkResponse, nil
}
