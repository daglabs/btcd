package flowcontext

import (
	"github.com/kaspanet/kaspad/blockdag"
	peerpkg "github.com/kaspanet/kaspad/protocol/peer"
	"sync/atomic"
	"time"
)

// StartIBDIfRequired selects a peer and starts IBD against it
// if required
func (f *FlowContext) StartIBDIfRequired() {
	f.startIBDMutex.Lock()
	defer f.startIBDMutex.Unlock()

	if f.IsInIBD() {
		return
	}

	peer := f.selectPeerForIBD(f.dag)
	if peer == nil {
		f.requestSelectedTipsIfRequired()
		return
	}

	atomic.StoreUint32(&f.isInIBD, 1)
	peer.StartIBD()
}

// IsInIBD is true if IBD is currently running
func (f *FlowContext) IsInIBD() bool {
	return atomic.LoadUint32(&f.isInIBD) != 0
}

// selectPeerForIBD returns the first peer whose selected tip
// hash is not in our DAG
func (f *FlowContext) selectPeerForIBD(dag *blockdag.BlockDAG) *peerpkg.Peer {
	for _, peer := range f.peers {
		peerSelectedTipHash := peer.SelectedTipHash()
		if !dag.IsInDAG(peerSelectedTipHash) {
			return peer
		}
	}
	return nil
}

func (f *FlowContext) requestSelectedTipsIfRequired() {
	if f.isDAGTimeCurrent() {
		return
	}
	f.requestSelectedTips()
}

func (f *FlowContext) isDAGTimeCurrent() bool {
	const minDurationToRequestSelectedTips = time.Minute
	return f.dag.Now().Sub(f.dag.SelectedTipHeader().Timestamp) > minDurationToRequestSelectedTips
}

func (f *FlowContext) requestSelectedTips() {
	for _, peer := range f.peers {
		peer.RequestSelectedTipIfRequired()
	}
}

// FinishIBD finishes the current IBD flow and starts a new one if required.
func (f *FlowContext) FinishIBD() {
	atomic.StoreUint32(&f.isInIBD, 0)

	f.StartIBDIfRequired()
}
