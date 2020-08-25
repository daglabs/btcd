package grpcserver

import (
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/server/grpcserver/protowire"
	"github.com/pkg/errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/server"
	"google.golang.org/grpc"
)

type gRPCConnection struct {
	server     *gRPCServer
	address    *net.TCPAddr
	isOutbound bool
	stream     grpcStream
	router     *router.Router

	// streamLock protects concurrent access to stream.
	// Note that it's an RWMutex. Despite what the name
	// implies, we use it to RLock() send() and receive() because
	// they can work perfectly fine in parallel, and Lock()
	// closeSend() because it must run alone.
	streamLock sync.RWMutex

	stopChan                chan struct{}
	clientConn              grpc.ClientConn
	onDisconnectedHandler   server.OnDisconnectedHandler
	onInvalidMessageHandler server.OnInvalidMessageHandler

	isConnected uint32
}

type grpcStream interface {
	Send(*protowire.KaspadMessage) error
	Recv() (*protowire.KaspadMessage, error)
}

func newConnection(server *gRPCServer, address *net.TCPAddr, isOutbound bool, stream grpcStream) *gRPCConnection {
	connection := &gRPCConnection{
		server:      server,
		address:     address,
		isOutbound:  isOutbound,
		stream:      stream,
		stopChan:    make(chan struct{}),
		isConnected: 1,
	}

	return connection
}

func (c *gRPCConnection) Start(router *router.Router) {
	if c.onDisconnectedHandler == nil {
		panic(errors.New("onDisconnectedHandler is nil"))
	}

	if c.onInvalidMessageHandler == nil {
		panic(errors.New("onInvalidMessageHandler is nil"))
	}

	c.router = router

	spawn("gRPCConnection.Start-connectionLoops", func() {
		err := c.connectionLoops()
		if err != nil {
			log.Errorf("error from connectionLoops for %s: %s", c.address, err)
		}
	})
}

func (c *gRPCConnection) String() string {
	return c.Address().String()
}

func (c *gRPCConnection) IsConnected() bool {
	return atomic.LoadUint32(&c.isConnected) != 0
}

func (c *gRPCConnection) SetOnDisconnectedHandler(onDisconnectedHandler server.OnDisconnectedHandler) {
	c.onDisconnectedHandler = onDisconnectedHandler
}

func (c *gRPCConnection) SetOnInvalidMessageHandler(onInvalidMessageHandler server.OnInvalidMessageHandler) {
	c.onInvalidMessageHandler = onInvalidMessageHandler
}

func (c *gRPCConnection) IsOutbound() bool {
	return c.isOutbound
}

// Disconnect disconnects the connection
// Calling this function a second time doesn't do anything
//
// This is part of the Connection interface
func (c *gRPCConnection) Disconnect() {
	if !c.IsConnected() {
		return
	}
	atomic.StoreUint32(&c.isConnected, 0)

	close(c.stopChan)

	if c.isOutbound {
		c.closeSend()
		log.Debugf("Disconnected from %s", c)
	}

	log.Debugf("Disconnecting from %s", c)
	if c.onDisconnectedHandler != nil {
		c.onDisconnectedHandler()
	}
}

func (c *gRPCConnection) Address() *net.TCPAddr {
	return c.address
}

func (c *gRPCConnection) receive() (*protowire.KaspadMessage, error) {
	// We use RLock here and in send() because they can work
	// in parallel. closeSend(), however, must not have either
	// receive() nor send() running while it's running.
	c.streamLock.RLock()
	defer c.streamLock.RUnlock()

	return c.stream.Recv()
}

func (c *gRPCConnection) send(message *protowire.KaspadMessage) error {
	// We use RLock here and in receive() because they can work
	// in parallel. closeSend(), however, must not have either
	// receive() nor send() running while it's running.
	c.streamLock.RLock()
	defer c.streamLock.RUnlock()

	return c.stream.Send(message)
}

func (c *gRPCConnection) closeSend() {
	c.streamLock.Lock()
	defer c.streamLock.Unlock()

	clientStream := c.stream.(grpc.ClientStream)
	_ = clientStream.CloseSend() // ignore error because we don't really know what's the status of the connection
}
