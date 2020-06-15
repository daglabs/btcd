package p2p

import (
	"github.com/kaspanet/kaspad/peer"
	"github.com/kaspanet/kaspad/wire"
)

// OnFilterAdd is invoked when a peer receives a filteradd kaspa
// message and is used by remote peers to add data to an already loaded bloom
// filter. The peer will be disconnected if a filter is not loaded when this
// message is received or the server is not configured to allow bloom filters.
func (sp *Peer) OnFilterAdd(_ *peer.Peer, msg *wire.MsgFilterAdd) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	if sp.filter.IsLoaded() {
		sp.AddBanScoreAndPushRejectMsg(wire.CmdFilterAdd, wire.RejectInvalid, nil,
			peer.BanScoreNoFilterLoaded, 0, "sent a filteradd request with no filter loaded")
		return
	}

	sp.filter.Add(msg.Data)
}
