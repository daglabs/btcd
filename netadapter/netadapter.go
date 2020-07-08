package netadapter

import (
	"github.com/kaspanet/kaspad/netadapter/server"
	"github.com/kaspanet/kaspad/netadapter/server/grpcserver"
	"github.com/kaspanet/kaspad/wire"
	"github.com/pkg/errors"
	"sync/atomic"
)

// RouterInitializer is a function that initializes a new
// router to be used with a new connection
type RouterInitializer func() (*Router, error)

// NetAdapter is an abstraction layer over networking.
// This type expects a RouteInitializer function. This
// function weaves together the various "routes" (messages
// and message handlers) without exposing anything related
// to networking internals.
type NetAdapter struct {
	id                *ID
	server            server.Server
	routerInitializer RouterInitializer
	stop              uint32

	connectionIDs    map[server.Connection]*ID
	idsToConnections map[*ID]server.Connection
	idsToRouters     map[*ID]*Router
}

// NewNetAdapter creates and starts a new NetAdapter on the
// given listeningPort
func NewNetAdapter(listeningAddrs []string) (*NetAdapter, error) {
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}
	s, err := grpcserver.NewGRPCServer(listeningAddrs)
	if err != nil {
		return nil, err
	}
	adapter := NetAdapter{
		id:     id,
		server: s,

		connectionIDs:    make(map[server.Connection]*ID),
		idsToConnections: make(map[*ID]server.Connection),
		idsToRouters:     make(map[*ID]*Router),
	}

	onConnectedHandler := adapter.newOnConnectedHandler()
	adapter.server.SetOnConnectedHandler(onConnectedHandler)

	return &adapter, nil
}

// Start begins the operation of the NetAdapter
func (na *NetAdapter) Start() error {
	return na.server.Start()
}

// Stop safely closes the NetAdapter
func (na *NetAdapter) Stop() error {
	if atomic.AddUint32(&na.stop, 1) != 1 {
		return errors.New("net adapter stopped more than once")
	}
	return na.server.Stop()
}

func (na *NetAdapter) newOnConnectedHandler() server.OnConnectedHandler {
	return func(connection server.Connection) error {
		router, err := na.routerInitializer()
		if err != nil {
			return err
		}
		connection.SetOnDisconnectedHandler(func() error {
			na.unregisterConnection(connection)
			return router.Close()
		})
		router.SetOnIDReceivedHandler(func(id *ID) {
			na.registerConnection(connection, router, id)
		})

		spawn(func() { na.startReceiveLoop(connection, router) })
		spawn(func() { na.startSendLoop(connection, router) })
		return nil
	}
}

func (na *NetAdapter) registerConnection(connection server.Connection, router *Router, id *ID) {
	na.connectionIDs[connection] = id
	na.idsToConnections[id] = connection
	na.idsToRouters[id] = router
}

func (na *NetAdapter) unregisterConnection(connection server.Connection) {
	id, ok := na.connectionIDs[connection]
	if !ok {
		return
	}

	delete(na.connectionIDs, connection)
	delete(na.idsToConnections, id)
	delete(na.idsToRouters, id)
}

func (na *NetAdapter) startReceiveLoop(connection server.Connection, router *Router) {
	for atomic.LoadUint32(&na.stop) != 0 {
		message, err := connection.Receive()
		if err != nil {
			log.Warnf("Failed to receive from %s: %s", connection, err)
			break
		}
		err = router.RouteInputMessage(message)
		if err != nil {
			log.Warnf("Failed to receive from %s: %s", connection, err)
			break
		}
	}

	err := connection.Disconnect()
	if err != nil {
		log.Warnf("Failed to disconnect from %s: %s", connection, err)
	}
}

func (na *NetAdapter) startSendLoop(connection server.Connection, router *Router) {
	for atomic.LoadUint32(&na.stop) != 0 {
		message := router.TakeOutputMessage()
		err := connection.Send(message)
		if err != nil {
			log.Warnf("Failed to send to %s: %s", connection, err)
			break
		}
	}

	err := connection.Disconnect()
	if err != nil {
		log.Warnf("Failed to disconnect from %s: %s", connection, err)
	}
}

// SetRouterInitializer sets the routerInitializer function
// for the net adapter
func (na *NetAdapter) SetRouterInitializer(routerInitializer RouterInitializer) {
	na.routerInitializer = routerInitializer
}

// ID returns this netAdapter's ID in the network
func (na *NetAdapter) ID() *ID {
	return na.id
}

// Broadcast sends the given `message` to every peer corresponding
// to each ID in `ids`
func (na *NetAdapter) Broadcast(ids []*ID, message wire.Message) error {
	for _, id := range ids {
		router, ok := na.idsToRouters[id]
		if !ok {
			return errors.Errorf("id %s is not registered", id)
		}
		router.RouteInputMessage(message)
	}
	return nil
}
