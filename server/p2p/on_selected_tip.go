package p2p

import (
	"github.com/kaspanet/kaspad/peer"
	"github.com/kaspanet/kaspad/wire"
)

func (sp *Peer) OnSelectedTip(peer *peer.Peer, msg *wire.MsgSelectedTip) {
	done := make(chan struct{})
	sp.server.SyncManager.QueueSelectedTipMsg(msg, peer, done)
	<-done
}
