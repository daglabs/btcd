package protocol

import (
	"github.com/kaspanet/kaspad/app/protocol/flows/rejects"
	"sync/atomic"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/protocol/flows/addressexchange"
	"github.com/kaspanet/kaspad/app/protocol/flows/blockrelay"
	"github.com/kaspanet/kaspad/app/protocol/flows/handshake"
	"github.com/kaspanet/kaspad/app/protocol/flows/ibd"
	"github.com/kaspanet/kaspad/app/protocol/flows/ibd/selectedtip"
	"github.com/kaspanet/kaspad/app/protocol/flows/ping"
	"github.com/kaspanet/kaspad/app/protocol/flows/relaytransactions"
	peerpkg "github.com/kaspanet/kaspad/app/protocol/peer"
	"github.com/kaspanet/kaspad/app/protocol/protocolerrors"
	"github.com/kaspanet/kaspad/infrastructure/network/addressmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter"
	routerpkg "github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

type flowInitializeFunc func(route *routerpkg.Route, peer *peerpkg.Peer) error
type flowExecuteFunc func(peer *peerpkg.Peer)

type flow struct {
	name        string
	executeFunc flowExecuteFunc
}

func (m *Manager) routerInitializer(router *routerpkg.Router, netConnection *netadapter.NetConnection) {
	// isStopping flag is raised the moment that the connection associated with this router is disconnected
	// errChan is used by the flow goroutines to return to runFlows when an error occurs.
	// They are both initialized here and passed to register flows.
	isStopping := uint32(0)
	errChan := make(chan error)

	flows := m.registerFlows(router, errChan, &isStopping)
	receiveVersionRoute, sendVersionRoute := registerHandshakeRoutes(router)

	// After flows were registered - spawn a new thread that will wait for connection to finish initializing
	// and start receiving messages
	spawn("routerInitializer-runFlows", func() {
		isBanned, err := m.context.ConnectionManager().IsBanned(netConnection)
		if err != nil && !errors.Is(err, addressmanager.ErrAddressNotFound) {
			panic(err)
		}
		if isBanned {
			netConnection.Disconnect()
			return
		}

		netConnection.SetOnInvalidMessageHandler(func(err error) {
			if atomic.AddUint32(&isStopping, 1) == 1 {
				errChan <- protocolerrors.Wrap(true, err, "received bad message")
			}
		})

		peer, err := handshake.HandleHandshake(m.context, netConnection, receiveVersionRoute,
			sendVersionRoute, router.OutgoingRoute())
		if err != nil {
			m.handleError(err, netConnection, router.OutgoingRoute())
			return
		}
		defer m.context.RemoveFromPeers(peer)

		removeHandshakeRoutes(router)

		err = m.runFlows(flows, peer, errChan)
		if err != nil {
			m.handleError(err, netConnection, router.OutgoingRoute())
			return
		}
	})
}

func (m *Manager) handleError(err error, netConnection *netadapter.NetConnection, outgoingRoute *routerpkg.Route) {
	if protocolErr := &(protocolerrors.ProtocolError{}); errors.As(err, &protocolErr) {
		if !m.context.Config().DisableBanning && protocolErr.ShouldBan {
			log.Warnf("Banning %s (reason: %s)", netConnection, protocolErr.Cause)

			err := m.context.ConnectionManager().Ban(netConnection)
			if err != nil && !errors.Is(err, addressmanager.ErrAddressNotFound) {
				panic(err)
			}

			err = outgoingRoute.Enqueue(appmessage.NewMsgReject(protocolErr.Error()))
			if err != nil && !errors.Is(err, routerpkg.ErrRouteClosed) {
				panic(err)
			}
		}
		netConnection.Disconnect()
		return
	}
	if errors.Is(err, routerpkg.ErrTimeout) {
		log.Warnf("Got timeout from %s. Disconnecting...", netConnection)
		netConnection.Disconnect()
		return
	}
	if errors.Is(err, routerpkg.ErrRouteClosed) {
		return
	}
	panic(err)
}

func (m *Manager) registerFlows(router *routerpkg.Router, errChan chan error, isStopping *uint32) (flows []*flow) {
	flows = m.registerAddressFlows(router, isStopping, errChan)
	flows = append(flows, m.registerBlockRelayFlows(router, isStopping, errChan)...)
	flows = append(flows, m.registerPingFlows(router, isStopping, errChan)...)
	flows = append(flows, m.registerIBDFlows(router, isStopping, errChan)...)
	flows = append(flows, m.registerTransactionRelayFlow(router, isStopping, errChan)...)
	flows = append(flows, m.registerRejectsFlow(router, isStopping, errChan)...)

	return flows
}

func (m *Manager) registerAddressFlows(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("SendAddresses", router, []appmessage.MessageCommand{appmessage.CmdRequestAddresses}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return addressexchange.SendAddresses(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerOneTimeFlow("ReceiveAddresses", router, []appmessage.MessageCommand{appmessage.CmdAddresses}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return addressexchange.ReceiveAddresses(m.context, incomingRoute, outgoingRoute, peer)
			},
		),
	}
}

func (m *Manager) registerBlockRelayFlows(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("HandleRelayInvs", router, []appmessage.MessageCommand{appmessage.CmdInvRelayBlock, appmessage.CmdBlock}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return blockrelay.HandleRelayInvs(m.context, incomingRoute,
					outgoingRoute, peer)
			},
		),

		m.registerFlow("HandleRelayBlockRequests", router, []appmessage.MessageCommand{appmessage.CmdRequestRelayBlocks}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return blockrelay.HandleRelayBlockRequests(m.context, incomingRoute, outgoingRoute, peer)
			},
		),
	}
}

func (m *Manager) registerPingFlows(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("ReceivePings", router, []appmessage.MessageCommand{appmessage.CmdPing}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ping.ReceivePings(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerFlow("SendPings", router, []appmessage.MessageCommand{appmessage.CmdPong}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ping.SendPings(m.context, incomingRoute, outgoingRoute, peer)
			},
		),
	}
}

func (m *Manager) registerIBDFlows(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("HandleIBD", router, []appmessage.MessageCommand{appmessage.CmdBlockLocator, appmessage.CmdIBDBlock,
			appmessage.CmdDoneHeaders, appmessage.CmdIBDRootNotFound, appmessage.CmdIBDRootUTXOSetAndBlock, appmessage.CmdHeader},
			isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ibd.HandleIBD(m.context, incomingRoute, outgoingRoute, peer)
			},
		),

		m.registerFlow("RequestSelectedTip", router,
			[]appmessage.MessageCommand{appmessage.CmdSelectedTip}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return selectedtip.RequestSelectedTip(m.context, incomingRoute, outgoingRoute, peer)
			},
		),

		m.registerFlow("HandleRequestSelectedTip", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestSelectedTip}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return selectedtip.HandleRequestSelectedTip(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerFlow("HandleRequestBlockLocator", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestBlockLocator}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ibd.HandleRequestBlockLocator(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerFlow("HandleRequestHeaders", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestHeaders, appmessage.CmdRequestNextHeaders}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ibd.HandleRequestHeaders(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerFlow("HandleRequestIBDRootUTXOSetAndBlock", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestIBDRootUTXOSetAndBlock}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ibd.HandleRequestIBDRootUTXOSetAndBlock(m.context, incomingRoute, outgoingRoute)
			},
		),

		m.registerFlow("HandleIBDBlockRequests", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestIBDBlocks}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return ibd.HandleIBDBlockRequests(m.context, incomingRoute, outgoingRoute)
			},
		),
	}
}

func (m *Manager) registerTransactionRelayFlow(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("HandleRelayedTransactions", router,
			[]appmessage.MessageCommand{appmessage.CmdInvTransaction, appmessage.CmdTx, appmessage.CmdTransactionNotFound}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return relaytransactions.HandleRelayedTransactions(m.context, incomingRoute, outgoingRoute)
			},
		),
		m.registerFlow("HandleRequestTransactions", router,
			[]appmessage.MessageCommand{appmessage.CmdRequestTransactions}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return relaytransactions.HandleRequestedTransactions(m.context, incomingRoute, outgoingRoute)
			},
		),
	}
}

func (m *Manager) registerRejectsFlow(router *routerpkg.Router, isStopping *uint32, errChan chan error) []*flow {
	outgoingRoute := router.OutgoingRoute()

	return []*flow{
		m.registerFlow("HandleRejects", router,
			[]appmessage.MessageCommand{appmessage.CmdReject}, isStopping, errChan,
			func(incomingRoute *routerpkg.Route, peer *peerpkg.Peer) error {
				return rejects.HandleRejects(m.context, incomingRoute, outgoingRoute)
			},
		),
	}
}

func (m *Manager) registerFlow(name string, router *routerpkg.Router, messageTypes []appmessage.MessageCommand, isStopping *uint32,
	errChan chan error, initializeFunc flowInitializeFunc) *flow {

	route, err := router.AddIncomingRoute(messageTypes)
	if err != nil {
		panic(err)
	}

	return &flow{
		name: name,
		executeFunc: func(peer *peerpkg.Peer) {
			err := initializeFunc(route, peer)
			if err != nil {
				m.context.HandleError(err, name, isStopping, errChan)
				return
			}
		},
	}
}

func (m *Manager) registerOneTimeFlow(name string, router *routerpkg.Router, messageTypes []appmessage.MessageCommand,
	isStopping *uint32, stopChan chan error, initializeFunc flowInitializeFunc) *flow {

	route, err := router.AddIncomingRoute(messageTypes)
	if err != nil {
		panic(err)
	}

	return &flow{
		name: name,
		executeFunc: func(peer *peerpkg.Peer) {
			defer func() {
				err := router.RemoveRoute(messageTypes)
				if err != nil {
					panic(err)
				}
			}()

			err := initializeFunc(route, peer)
			if err != nil {
				m.context.HandleError(err, name, isStopping, stopChan)
				return
			}
		},
	}
}

func registerHandshakeRoutes(router *routerpkg.Router) (
	receiveVersionRoute *routerpkg.Route, sendVersionRoute *routerpkg.Route) {
	receiveVersionRoute, err := router.AddIncomingRoute([]appmessage.MessageCommand{appmessage.CmdVersion})
	if err != nil {
		panic(err)
	}

	sendVersionRoute, err = router.AddIncomingRoute([]appmessage.MessageCommand{appmessage.CmdVerAck})
	if err != nil {
		panic(err)
	}

	return receiveVersionRoute, sendVersionRoute
}

func removeHandshakeRoutes(router *routerpkg.Router) {
	err := router.RemoveRoute([]appmessage.MessageCommand{appmessage.CmdVersion, appmessage.CmdVerAck})
	if err != nil {
		panic(err)
	}
}
