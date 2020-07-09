package grpcserver

import (
	"github.com/kaspanet/kaspad/netadapter/server/grpcserver/protowire"
	"github.com/kaspanet/kaspad/wire"
	"google.golang.org/grpc"
)

type gRPCConnection struct {
	address        string
	sendChan       chan *protowire.KaspadMessage
	receiveChan    chan *protowire.KaspadMessage
	disconnectChan chan struct{}
	errChan        chan error
	clientConn     grpc.ClientConn
}

func newConnection(address string) *gRPCConnection {
	return &gRPCConnection{
		address:        address,
		sendChan:       make(chan *protowire.KaspadMessage),
		receiveChan:    make(chan *protowire.KaspadMessage),
		disconnectChan: make(chan struct{}),
		errChan:        make(chan error),
	}
}

// Send sends the given message through the connection
// This is part of the Connection interface
func (c *gRPCConnection) Send(message wire.Message) error {
	messageProto, err := protowire.FromWireMessage(message)
	if err != nil {
		return err
	}

	c.sendChan <- messageProto
	return <-c.errChan
}

// Receive receives the next message from the connection
// This is part of the Connection interface
func (c *gRPCConnection) Receive() (wire.Message, error) {
	protoMessage := <-c.receiveChan

	return protoMessage.ToWireMessage()
}

// Disconnect disconnects the connection
// This is part of the Connection interface
func (c *gRPCConnection) Disconnect() error {
	c.disconnectChan <- struct{}{}

	return <-c.errChan
}
