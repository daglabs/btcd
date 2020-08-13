package rpc

import (
	"github.com/kaspanet/kaspad/network/rpc/model"
	"github.com/kaspanet/kaspad/util/network"
)

// handleConnect handles connect commands.
func handleConnect(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*model.ConnectCmd)

	isPermanent := c.IsPermanent != nil && *c.IsPermanent

	address, err := network.NormalizeAddress(c.Address, s.dag.Params.DefaultPort)
	if err != nil {
		return nil, &model.RPCError{
			Code:    model.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

	s.connectionManager.AddConnectionRequest(address, isPermanent)
	return nil, nil
}
