package client

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/client/grpcclient"
	routerpkg "github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
	"time"
)

const defaultTimeout = 30 * time.Second

type RPCClient struct {
	*grpcclient.GRPCClient

	rpcAddress string
	rpcRouter  *rpcRouter

	timeout time.Duration
}

func NewRPCClient(rpcAddress string) (*RPCClient, error) {
	rpcClient, err := grpcclient.Connect(rpcAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to address %s", rpcClient)
	}
	rpcRouter, err := buildRPCRouter()
	if err != nil {
		return nil, errors.Wrapf(err, "error creating the RPC router")
	}
	rpcClient.AttachRouter(rpcRouter.router)

	log.Infof("Connected to server %s", rpcAddress)

	return &RPCClient{
		GRPCClient: rpcClient,
		rpcAddress: rpcAddress,
		rpcRouter:  rpcRouter,
		timeout:    defaultTimeout,
	}, nil
}

func (c *RPCClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *RPCClient) Close() {
	c.rpcRouter.router.Close()
}

func (c *RPCClient) Address() string {
	return c.rpcAddress
}

func (c *RPCClient) route(command appmessage.MessageCommand) *routerpkg.Route {
	return c.rpcRouter.routes[command]
}

func (c *RPCClient) convertRPCError(rpcError *appmessage.RPCError) error {
	return errors.Errorf("received error response from RPC: %s", rpcError.Message)
}
