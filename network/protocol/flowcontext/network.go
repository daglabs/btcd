package flowcontext

import (
	"github.com/kaspanet/kaspad/network/connmanager"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/network/netadapter"
	"github.com/kaspanet/kaspad/network/protocol/common"
	peerpkg "github.com/kaspanet/kaspad/network/protocol/peer"
	"github.com/pkg/errors"
)

// NetAdapter returns the net adapter that is associated to the flow context.
func (f *FlowContext) NetAdapter() *netadapter.NetAdapter {
	return f.netAdapter
}

// ConnectionManager returns the connection manager that is associated to the flow context.
func (f *FlowContext) ConnectionManager() *connmanager.ConnectionManager {
	return f.connectionManager
}

// AddToPeers marks this peer as ready and adds it to the ready peers list.
func (f *FlowContext) AddToPeers(peer *peerpkg.Peer) error {
	f.peersMutex.Lock()
	defer f.peersMutex.Unlock()

	if _, ok := f.peers[peer.ID()]; ok {
		return errors.Wrapf(common.ErrPeerWithSameIDExists, "peer with ID %s already exists", peer.ID())
	}

	f.peers[peer.ID()] = peer

	return nil
}

// readyPeerConnections returns the NetConnections of all the ready peers.
func (f *FlowContext) readyPeerConnections() []*netadapter.NetConnection {
	f.peersMutex.RLock()
	defer f.peersMutex.RUnlock()
	peerConnections := make([]*netadapter.NetConnection, len(f.peers))
	i := 0
	for _, peer := range f.peers {
		peerConnections[i] = peer.Connection()
		i++
	}
	return peerConnections
}

// Broadcast broadcast the given message to all the ready peers.
func (f *FlowContext) Broadcast(message domainmessage.Message) error {
	return f.netAdapter.Broadcast(f.readyPeerConnections(), message)
}

// Peers returns the currently active peers
func (f *FlowContext) Peers() []*peerpkg.Peer {
	f.peersMutex.RLock()
	defer f.peersMutex.RUnlock()

	peers := make([]*peerpkg.Peer, len(f.peers))
	i := 0
	for _, peer := range f.peers {
		peers[i] = peer
		i++
	}
	return peers
}
