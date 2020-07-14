package peer

import (
	"github.com/kaspanet/kaspad/netadapter/id"
	"github.com/kaspanet/kaspad/util/daghash"
	mathUtil "github.com/kaspanet/kaspad/util/math"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/kaspanet/kaspad/wire"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

// Peer holds data about a peer.
type Peer struct {
	ready uint32

	selectedTipHashMtx sync.RWMutex
	selectedTipHash    *daghash.Hash

	id                    *id.ID
	userAgent             string
	services              wire.ServiceFlag
	advertisedProtocolVer uint32 // protocol version advertised by remote
	protocolVersion       uint32 // negotiated protocol version
	disableRelayTx        bool
	subnetworkID          *subnetworkid.SubnetworkID
}

// SelectedTipHash returns the selected tip of the peer.
func (p *Peer) SelectedTipHash() (*daghash.Hash, error) {
	if atomic.LoadUint32(&p.ready) == 0 {
		return nil, errors.New("peer is not ready yet")
	}
	p.selectedTipHashMtx.RLock()
	defer p.selectedTipHashMtx.RUnlock()
	return p.selectedTipHash, nil
}

// SetSelectedTipHash sets the selected tip of the peer.
func (p *Peer) SetSelectedTipHash(hash *daghash.Hash) error {
	if atomic.LoadUint32(&p.ready) == 0 {
		return errors.New("peer is not ready yet")
	}
	p.selectedTipHashMtx.Lock()
	defer p.selectedTipHashMtx.Unlock()
	p.selectedTipHash = hash
	return nil
}

// SubnetworkID returns the subnetwork the peer is associated with.
// It is nil in full nodes.
func (p *Peer) SubnetworkID() (*subnetworkid.SubnetworkID, error) {
	if atomic.LoadUint32(&p.ready) == 0 {
		return nil, errors.New("peer is not ready yet")
	}
	return p.subnetworkID, nil
}

// ID returns the peer ID.
func (p *Peer) ID() (*id.ID, error) {
	if atomic.LoadUint32(&p.ready) == 0 {
		return nil, errors.New("peer is not ready yet")
	}
	return p.id, nil
}

// MarkAsReady marks the peer as ready.
func (p *Peer) MarkAsReady() error {
	if atomic.AddUint32(&p.ready, 1) != 1 {
		return errors.New("peer is already ready")
	}
	return nil
}

// UpdateFieldsFromMsgVersion updates the peer with the data from the version message.
func (p *Peer) UpdateFieldsFromMsgVersion(msg *wire.MsgVersion) {
	// Negotiate the protocol version.
	p.advertisedProtocolVer = msg.ProtocolVersion
	p.protocolVersion = mathUtil.MinUint32(p.protocolVersion, p.advertisedProtocolVer)
	log.Debugf("Negotiated protocol version %d for peer %s",
		p.protocolVersion, p)

	// Set the peer's ID.
	p.id = msg.ID

	// Set the supported services for the peer to what the remote peer
	// advertised.
	p.services = msg.Services

	// Set the remote peer's user agent.
	p.userAgent = msg.UserAgent

	p.disableRelayTx = msg.DisableRelayTx
	p.selectedTipHash = msg.SelectedTipHash
	p.subnetworkID = msg.SubnetworkID
}

func (p *Peer) String() string {
	//TODO(libp2p)
	panic("unimplemented")
}

var (
	readyPeers    = make(map[*id.ID]*Peer, 0)
	readyPeersMtx sync.RWMutex
)

// ErrPeerWithSameIDExists signifies that a peer with the same ID already exist.
var ErrPeerWithSameIDExists = errors.New("ready with the same ID already exists")

// AddToReadyPeers marks this peer as ready and adds it to the ready peers list.
func AddToReadyPeers(peer *Peer) error {
	readyPeersMtx.RLock()
	defer readyPeersMtx.RUnlock()

	if _, ok := readyPeers[peer.id]; ok {
		return errors.Wrapf(ErrPeerWithSameIDExists, "peer with ID %s already exists", peer.id)
	}

	err := peer.MarkAsReady()
	if err != nil {
		return err
	}

	readyPeers[peer.id] = peer
	return nil
}

// GetReadyPeerIDs returns the peer IDs of all the ready peers.
func GetReadyPeerIDs() []*id.ID {
	readyPeersMtx.RLock()
	defer readyPeersMtx.RUnlock()
	peerIDs := make([]*id.ID, len(readyPeers))
	i := 0
	for peerID := range readyPeers {
		peerIDs[i] = peerID
	}
	return peerIDs
}

// IDExists returns whether there's a peer with the given ID.
func IDExists(peerID *id.ID) bool {
	_, ok := readyPeers[peerID]
	return ok
}
