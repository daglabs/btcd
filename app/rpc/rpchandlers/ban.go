package rpchandlers

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/rpc/rpccontext"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
)

// Ban handles the respectively named RPC command
func Ban(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	unbanRequest := request.(*appmessage.BanRequestMessage)
	err := context.ConnectionManager.BanByIP(unbanRequest.IP)
	if err != nil {
		errorMessage := &appmessage.BanResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("Could not ban IP: %s", err)
		return errorMessage, nil
	}
	response := appmessage.NewBanResponseMessage()
	return response, nil
}