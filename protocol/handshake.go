package protocol

import (
	"sync"
	"sync/atomic"

	routerpkg "github.com/kaspanet/kaspad/netadapter/router"
	"github.com/kaspanet/kaspad/protocol/flows/receiveversion"
	"github.com/kaspanet/kaspad/protocol/flows/sendversion"
	peerpkg "github.com/kaspanet/kaspad/protocol/peer"
	"github.com/kaspanet/kaspad/util/locks"
	"github.com/kaspanet/kaspad/wire"
	"github.com/pkg/errors"
)

func (m *Manager) handshake(router *routerpkg.Router, peer *peerpkg.Peer) (closed bool, err error) {

	receiveVersionRoute, err := router.AddIncomingRoute([]wire.MessageCommand{wire.CmdVersion})
	if err != nil {
		panic(err)
	}

	sendVersionRoute, err := router.AddIncomingRoute([]wire.MessageCommand{wire.CmdVerAck})
	if err != nil {
		panic(err)
	}

	// For the handshake to finish, we need to get from the other node
	// a version and verack messages, so we increase the wait group by 2
	// and block the handshake with wg.Wait().
	wg := sync.WaitGroup{}
	wg.Add(2)

	errChanUsed := uint32(0)
	errChan := make(chan error)

	var peerAddress *wire.NetAddress
	spawn(func() {
		defer wg.Done()
		address, closed, err := receiveversion.ReceiveVersion(receiveVersionRoute, router.OutgoingRoute(),
			m.netAdapter, peer, m.dag)
		if err != nil {
			log.Errorf("error from ReceiveVersion: %s", err)
		}
		if err != nil || closed {
			if atomic.AddUint32(&errChanUsed, 1) != 1 {
				errChan <- err
			}
			return
		}
		peerAddress = address
	})

	spawn(func() {
		defer wg.Done()
		closed, err := sendversion.SendVersion(sendVersionRoute, router.OutgoingRoute(), m.netAdapter, m.dag)
		if err != nil {
			log.Errorf("error from SendVersion: %s", err)
		}
		if err != nil || closed {
			if atomic.AddUint32(&errChanUsed, 1) != 1 {
				errChan <- err
			}
			return
		}
	})

	select {
	case err := <-errChan:
		if err != nil {
			return false, err
		}
		return true, nil
	case <-locks.ReceiveFromChanWhenDone(func() { wg.Wait() }):
	}

	err = peerpkg.AddToReadyPeers(peer)
	if err != nil {
		if errors.Is(err, peerpkg.ErrPeerWithSameIDExists) {
			return false, err
		}
		panic(err)
	}

	peerID, err := peer.ID()
	if err != nil {
		panic(err)
	}

	err = m.netAdapter.AssociateRouterID(router, peerID)
	if err != nil {
		panic(err)
	}

	if peerAddress != nil {
		subnetworkID, err := peer.SubnetworkID()
		if err != nil {
			panic(err)
		}
		m.addressManager.AddAddress(peerAddress, peerAddress, subnetworkID)
		m.addressManager.Good(peerAddress, subnetworkID)
	}

	err = router.RemoveRoute([]wire.MessageCommand{wire.CmdVersion, wire.CmdVerAck})
	if err != nil {
		panic(err)
	}
	return false, nil
}
