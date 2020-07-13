package protocol

import (
	"github.com/kaspanet/kaspad/addrmgr"
	"github.com/kaspanet/kaspad/blockdag"
	"github.com/kaspanet/kaspad/netadapter"
	"github.com/kaspanet/kaspad/protocol/handlerelayblockrequests"
	"github.com/kaspanet/kaspad/protocol/handlerelayinvs"
	peerpkg "github.com/kaspanet/kaspad/protocol/peer"
	"github.com/kaspanet/kaspad/wire"
	"sync/atomic"
)

// Manager manages the p2p protocol
type Manager struct {
	netAdapter *netadapter.NetAdapter
}

// NewManager creates a new instance of the p2p protocol manager
func NewManager(listeningAddrs []string, dag *blockdag.BlockDAG,
	addressManager *addrmgr.AddrManager) (*Manager, error) {

	netAdapter, err := netadapter.NewNetAdapter(listeningAddrs)
	if err != nil {
		return nil, err
	}

	routerInitializer := newRouterInitializer(netAdapter, dag, addressManager)
	netAdapter.SetRouterInitializer(routerInitializer)

	manager := Manager{
		netAdapter: netAdapter,
	}
	return &manager, nil
}

// Start starts the p2p protocol
func (p *Manager) Start() error {
	return p.netAdapter.Start()
}

// Stop stops the p2p protocol
func (p *Manager) Stop() error {
	return p.netAdapter.Stop()
}

func newRouterInitializer(netAdapter *netadapter.NetAdapter, dag *blockdag.BlockDAG,
	addressManager *addrmgr.AddrManager) netadapter.RouterInitializer {
	return func() (*netadapter.Router, error) {
		router := netadapter.NewRouter()
		spawn(func() {
			err := startFlows(netAdapter, router, dag, addressManager)
			if err != nil {
				// TODO(libp2p) Ban peer
			}
		})
		return router, nil
	}
}

func startFlows(netAdapter *netadapter.NetAdapter, router *netadapter.Router, dag *blockdag.BlockDAG,
	addressManager *addrmgr.AddrManager) error {
	stop := make(chan error)
	stopped := uint32(0)

	peer := new(peerpkg.Peer)

	closed, err := handshake(router, netAdapter, peer, dag, addressManager)
	if err != nil {
		return err
	}
	if closed {
		return nil
	}

	addFlow("HandleRelayInvs", router, []string{wire.CmdInvRelayBlock, wire.CmdBlock}, &stopped, stop,
		func(ch chan wire.Message) error {
			return handlerelayinvs.HandleRelayInvs(ch, peer, netAdapter, router, dag)
		},
	)

	addFlow("HandleRelayBlockRequests", router, []string{wire.CmdGetRelayBlocks}, &stopped, stop,
		func(ch chan wire.Message) error {
			return handlerelayblockrequests.HandleRelayBlockRequests(ch, peer, router, dag)
		},
	)

	// TODO(libp2p): Remove this and change it with a real Ping-Pong flow.
	addFlow("PingPong", router, []string{wire.CmdPing, wire.CmdPong}, &stopped, stop,
		func(ch chan wire.Message) error {
			router.WriteOutgoingMessage(wire.NewMsgPing(666))
			for message := range ch {
				log.Infof("Got message: %+v", message.Command())
				if message.Command() == "ping" {
					router.WriteOutgoingMessage(wire.NewMsgPong(666))
				}
			}
			return nil
		},
	)

	err = <-stop
	return err
}

func addFlow(name string, router *netadapter.Router, messageTypes []string, stopped *uint32,
	stopChan chan error, flow func(ch chan wire.Message) error) {

	ch := make(chan wire.Message)
	err := router.AddRoute(messageTypes, ch)
	if err != nil {
		panic(err)
	}

	spawn(func() {
		err := flow(ch)
		if err != nil {
			log.Errorf("error from %s flow: %s", name, err)
		}
		if atomic.AddUint32(stopped, 1) == 1 {
			stopChan <- err
		}
	})
}
