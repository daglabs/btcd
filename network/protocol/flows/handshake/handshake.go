package handshake

import (
	"sync"
	"sync/atomic"

	"github.com/kaspanet/kaspad/network/addressmanager"
	"github.com/kaspanet/kaspad/network/protocol/common"
	"github.com/kaspanet/kaspad/network/protocol/protocolerrors"

	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/infrastructure/config"
	"github.com/kaspanet/kaspad/network/netadapter"

	"github.com/kaspanet/kaspad/network/domainmessage"
	routerpkg "github.com/kaspanet/kaspad/network/netadapter/router"
	peerpkg "github.com/kaspanet/kaspad/network/protocol/peer"
	"github.com/kaspanet/kaspad/util/locks"
	"github.com/pkg/errors"
)

// HandleHandshakeContext is the interface for the context needed for the HandleHandshake flow.
type HandleHandshakeContext interface {
	Config() *config.Config
	NetAdapter() *netadapter.NetAdapter
	DAG() *blockdag.BlockDAG
	AddressManager() *addressmanager.AddressManager
	StartIBDIfRequired()
	AddToPeers(peer *peerpkg.Peer) error
	HandleError(err error, flowName string, isStopping *uint32, errChan chan<- error)
}

// HandleHandshake sets up the handshake protocol - It sends a version message and waits for an incoming
// version message, as well as a verack for the sent version
func HandleHandshake(context HandleHandshakeContext, netConnection *netadapter.NetConnection,
	receiveVersionRoute *routerpkg.Route, sendVersionRoute *routerpkg.Route, outgoingRoute *routerpkg.Route,
) (peer *peerpkg.Peer, err error) {

	// For HandleHandshake to finish, we need to get from the other node
	// a version and verack messages, so we increase the wait group by 2
	// and block HandleHandshake with wg.Wait().
	wg := sync.WaitGroup{}
	wg.Add(2)

	isStopping := uint32(0)
	errChan := make(chan error)

	peer = peerpkg.New(netConnection)

	var peerAddress *domainmessage.NetAddress
	spawn("HandleHandshake-ReceiveVersion", func() {
		defer wg.Done()
		address, err := ReceiveVersion(context, receiveVersionRoute, outgoingRoute, peer)
		if err != nil {
			handleError(err, "ReceiveVersion", &isStopping, errChan)
			return
		}
		peerAddress = address
	})

	spawn("HandleHandshake-SendVersion", func() {
		defer wg.Done()
		err := SendVersion(context, sendVersionRoute, outgoingRoute)
		if err != nil {
			handleError(err, "SendVersion", &isStopping, errChan)
			return
		}
	})

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
		return nil, nil
	case <-locks.ReceiveFromChanWhenDone(func() { wg.Wait() }):
	}

	err = context.AddToPeers(peer)
	if err != nil {
		if errors.As(err, &common.ErrPeerWithSameIDExists) {
			return nil, protocolerrors.Wrap(false, err, "peer already exists")
		}
		return nil, err
	}

	if peerAddress != nil {
		subnetworkID := peer.SubnetworkID()
		context.AddressManager().AddAddress(peerAddress, peerAddress, subnetworkID)
		context.AddressManager().Good(peerAddress, subnetworkID)
	}

	context.StartIBDIfRequired()

	return peer, nil
}

// Handshake is different from other flows, since in it should forward router.ErrRouteClosed to errChan
// Therefore we implement a separate handleError for handshake
func handleError(err error, flowName string, isStopping *uint32, errChan chan error) {
	if errors.Is(err, routerpkg.ErrRouteClosed) {
		if atomic.AddUint32(isStopping, 1) == 1 {
			errChan <- err
		}
		return
	}

	if protocolErr := &(protocolerrors.ProtocolError{}); errors.As(err, &protocolErr) {
		log.Errorf("Handshake protocol error from %s: %s", flowName, err)
		if atomic.AddUint32(isStopping, 1) == 1 {
			errChan <- err
		}
		return
	}
	panic(err)
}
