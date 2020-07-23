package netadapter

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/kaspanet/kaspad/config"
	"github.com/kaspanet/kaspad/netadapter/id"
	routerpkg "github.com/kaspanet/kaspad/netadapter/router"
	"github.com/kaspanet/kaspad/netadapter/server"
	"github.com/kaspanet/kaspad/netadapter/server/grpcserver"
	"github.com/kaspanet/kaspad/wire"
	"github.com/pkg/errors"
)

// RouterInitializer is a function that initializes a new
// router to be used with a new connection
type RouterInitializer func(netConnection *NetConnection) (*routerpkg.Router, error)

// NetAdapter is an abstraction layer over networking.
// This type expects a RouteInitializer function. This
// function weaves together the various "routes" (messages
// and message handlers) without exposing anything related
// to networking internals.
type NetAdapter struct {
	cfg               *config.Config
	id                *id.ID
	server            server.Server
	routerInitializer RouterInitializer
	stop              uint32

	connectionsToRouters map[*NetConnection]*routerpkg.Router
	sync.RWMutex
}

// NewNetAdapter creates and starts a new NetAdapter on the
// given listeningPort
func NewNetAdapter(cfg *config.Config) (*NetAdapter, error) {
	netAdapterID, err := id.GenerateID()
	if err != nil {
		return nil, err
	}
	s, err := grpcserver.NewGRPCServer(cfg.Listeners)
	if err != nil {
		return nil, err
	}
	adapter := NetAdapter{
		cfg:    cfg,
		id:     netAdapterID,
		server: s,

		connectionsToRouters: make(map[*NetConnection]*routerpkg.Router),
	}

	adapter.server.SetOnConnectedHandler(adapter.onConnectedHandler)

	return &adapter, nil
}

// Start begins the operation of the NetAdapter
func (na *NetAdapter) Start() error {
	err := na.server.Start()
	if err != nil {
		return err
	}

	return nil
}

// Stop safely closes the NetAdapter
func (na *NetAdapter) Stop() error {
	if atomic.AddUint32(&na.stop, 1) != 1 {
		return errors.New("net adapter stopped more than once")
	}
	return na.server.Stop()
}

// Connect tells the NetAdapter's underlying server to initiate a connection
// to the given address
func (na *NetAdapter) Connect(address string) error {
	_, err := na.server.Connect(address)
	return err
}

// Connections returns a list of connections currently connected and active
func (na *NetAdapter) Connections() []*NetConnection {
	netConnections := make([]*NetConnection, 0, len(na.connectionsToRouters))

	for netConnection := range na.connectionsToRouters {
		netConnections = append(netConnections, netConnection)
	}

	return netConnections
}

// ConnectionCount returns the count of the connected connections
func (na *NetAdapter) ConnectionCount() int {
	return len(na.connectionsToRouters)
}

func (na *NetAdapter) onConnectedHandler(connection server.Connection) error {
	netConnection := newNetConnection(connection, nil)
	router, err := na.routerInitializer(netConnection)
	if err != nil {
		return err
	}
	connection.Start(router)

	na.connectionsToRouters[netConnection] = router

	router.SetOnRouteCapacityReachedHandler(func() {
		err := connection.Disconnect()
		if err != nil {
			if !errors.Is(err, server.ErrNetwork) {
				panic(err)
			}
			log.Warnf("Failed to disconnect from %s", connection)
		}
	})
	connection.SetOnDisconnectedHandler(func() error {
		delete(na.connectionsToRouters, netConnection)
		return router.Close()
	})
	return nil
}

// SetRouterInitializer sets the routerInitializer function
// for the net adapter
func (na *NetAdapter) SetRouterInitializer(routerInitializer RouterInitializer) {
	na.routerInitializer = routerInitializer
}

// ID returns this netAdapter's ID in the network
func (na *NetAdapter) ID() *id.ID {
	return na.id
}

// Broadcast sends the given `message` to every peer corresponding
// to each NetConnection in the given netConnections
func (na *NetAdapter) Broadcast(netConnections []*NetConnection, message wire.Message) error {
	na.RLock()
	defer na.RUnlock()
	for _, netConnection := range netConnections {
		router := na.connectionsToRouters[netConnection]
		err := router.EnqueueIncomingMessage(message)
		if err != nil {
			if errors.Is(err, routerpkg.ErrRouteClosed) {
				log.Debugf("Cannot enqueue message to %s: router is closed", netConnection)
				continue
			}
			return err
		}
	}
	return nil
}

// GetBestLocalAddress returns the most appropriate local address to use
// for the given remote address.
func (na *NetAdapter) GetBestLocalAddress() (*wire.NetAddress, error) {
	//TODO(libp2p) Reimplement this, and check reachability to the other node
	if len(na.cfg.ExternalIPs) > 0 {
		host, portString, err := net.SplitHostPort(na.cfg.ExternalIPs[0])
		if err != nil {
			portString = na.cfg.NetParams().DefaultPort
		}
		portInt, err := strconv.Atoi(portString)
		if err != nil {
			return nil, err
		}

		ip := net.ParseIP(host)
		if ip == nil {
			hostAddrs, err := net.LookupHost(host)
			if err != nil {
				return nil, err
			}
			ip = net.ParseIP(hostAddrs[0])
			if ip == nil {
				return nil, errors.Errorf("Cannot resolve IP address for host '%s'", host)
			}
		}
		return wire.NewNetAddressIPPort(ip, uint16(portInt), wire.SFNodeNetwork), nil

	}
	listenAddress := na.cfg.Listeners[0]
	_, portString, err := net.SplitHostPort(listenAddress)
	if err != nil {
		portString = na.cfg.NetParams().DefaultPort
	}

	portInt, err := strconv.Atoi(portString)
	if err != nil {
		return nil, err
	}

	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err != nil {
			continue
		}

		return wire.NewNetAddressIPPort(ip, uint16(portInt), wire.SFNodeNetwork), nil
	}
	return nil, errors.New("no address was found")
}

// Disconnect disconnects the given connection
func (na *NetAdapter) Disconnect(netConnection *NetConnection) error {
	err := netConnection.connection.Disconnect()
	if err != nil {
		if !errors.Is(err, server.ErrNetwork) {
			return err
		}
		log.Warnf("Error disconnecting from %s: %s", netConnection, err)
	}
	return nil
}

// IsBanned checks whether the given address had previously
// been banned
func (na *NetAdapter) IsBanned(address *net.TCPAddr) bool {
	return na.server.IsBanned(address)
}

// Ban prevents the given netConnection from connecting again
func (na *NetAdapter) Ban(netConnection *NetConnection) {
	na.server.Ban(netConnection.connection.Address())
}
