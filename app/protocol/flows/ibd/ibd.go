package ibd

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/protocol/blocklogger"
	"github.com/kaspanet/kaspad/app/protocol/common"
	peerpkg "github.com/kaspanet/kaspad/app/protocol/peer"
	"github.com/kaspanet/kaspad/app/protocol/protocolerrors"
	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/pkg/errors"
)

// HandleIBDContext is the interface for the context needed for the HandleIBD flow.
type HandleIBDContext interface {
	DAG() *blockdag.BlockDAG
	OnNewBlock(block *util.Block) error
	StartIBDIfRequired()
	FinishIBD()
}

type handleIBDFlow struct {
	HandleIBDContext
	incomingRoute, outgoingRoute *router.Route
	peer                         *peerpkg.Peer
}

// HandleIBD waits for IBD start and handles it when IBD is triggered for this peer
func HandleIBD(context HandleIBDContext, incomingRoute *router.Route, outgoingRoute *router.Route,
	peer *peerpkg.Peer) error {

	flow := &handleIBDFlow{
		HandleIBDContext: context,
		incomingRoute:    incomingRoute,
		outgoingRoute:    outgoingRoute,
		peer:             peer,
	}
	return flow.start()
}

func (flow *handleIBDFlow) start() error {
	for {
		err := flow.runIBD()
		if err != nil {
			return err
		}
	}
}

func (flow *handleIBDFlow) runIBD() error {
	flow.peer.WaitForIBDStart()
	defer flow.FinishIBD()

	peerSelectedTipHash := flow.peer.SelectedTipHash()
	log.Debugf("Trying to find highest shared chain block with peer %s with selected tip %s", flow.peer, peerSelectedTipHash)
	highestSharedBlockHash, err := flow.findHighestSharedBlockHash(peerSelectedTipHash)
	if err != nil {
		return err
	}

	log.Debugf("Found highest shared chain block %s with peer %s", highestSharedBlockHash, flow.peer)

	return flow.downloadBlocks(highestSharedBlockHash, peerSelectedTipHash)
}

func (flow *handleIBDFlow) findHighestSharedBlockHash(peerSelectedTipHash *daghash.Hash) (lowHash *daghash.Hash,
	err error) {

	lowHash = flow.DAG().Params.GenesisHash
	highHash := peerSelectedTipHash

	for {
		err := flow.sendGetBlockLocator(lowHash, highHash)
		if err != nil {
			return nil, err
		}

		blockLocatorHashes, err := flow.receiveBlockLocator()
		if err != nil {
			return nil, err
		}

		// We check whether the locator's highest hash is in the local DAG.
		// If it is, return it. If it isn't, we need to narrow our
		// getBlockLocator request and try again.
		locatorHighHash := blockLocatorHashes[0]
		if flow.DAG().IsInDAG(locatorHighHash) {
			return locatorHighHash, nil
		}

		highHash, lowHash = flow.DAG().FindNextLocatorBoundaries(blockLocatorHashes)
	}
}

func (flow *handleIBDFlow) sendGetBlockLocator(lowHash *daghash.Hash, highHash *daghash.Hash) error {

	msgGetBlockLocator := appmessage.NewMsgRequestBlockLocator(highHash, lowHash)
	return flow.outgoingRoute.Enqueue(msgGetBlockLocator)
}

func (flow *handleIBDFlow) receiveBlockLocator() (blockLocatorHashes []*daghash.Hash, err error) {
	message, err := flow.incomingRoute.DequeueWithTimeout(common.DefaultTimeout)
	if err != nil {
		return nil, err
	}
	msgBlockLocator, ok := message.(*appmessage.MsgBlockLocator)
	if !ok {
		return nil,
			protocolerrors.Errorf(true, "received unexpected message type. "+
				"expected: %s, got: %s", appmessage.CmdBlockLocator, message.Command())
	}
	return msgBlockLocator.BlockLocatorHashes, nil
}

func (flow *handleIBDFlow) downloadBlocks(highestSharedBlockHash *daghash.Hash,
	peerSelectedTipHash *daghash.Hash) error {

	err := flow.sendGetBlocks(highestSharedBlockHash, peerSelectedTipHash)
	if err != nil {
		return err
	}

	blocksReceived := 0
	for {
		msgIBDBlock, doneIBD, err := flow.receiveIBDBlock()
		if err != nil {
			return err
		}

		if doneIBD {
			return nil
		}

		err = flow.processIBDBlock(msgIBDBlock)
		if err != nil {
			return err
		}

		blocksReceived++
		if blocksReceived%ibdBatchSize == 0 {
			err = flow.outgoingRoute.Enqueue(appmessage.NewMsgRequestNextIBDBlocks())
			if err != nil {
				return err
			}
		}
	}
}

func (flow *handleIBDFlow) sendGetBlocks(highestSharedBlockHash *daghash.Hash,
	peerSelectedTipHash *daghash.Hash) error {

	msgGetBlockInvs := appmessage.NewMsgRequstIBDBlocks(highestSharedBlockHash, peerSelectedTipHash)
	return flow.outgoingRoute.Enqueue(msgGetBlockInvs)
}

func (flow *handleIBDFlow) receiveIBDBlock() (msgIBDBlock *appmessage.MsgIBDBlock, doneIBD bool, err error) {
	message, err := flow.incomingRoute.DequeueWithTimeout(common.DefaultTimeout)
	if err != nil {
		return nil, false, err
	}
	switch message := message.(type) {
	case *appmessage.MsgIBDBlock:
		return message, false, nil
	case *appmessage.MsgDoneIBDBlocks:
		return nil, true, nil
	default:
		return nil, false,
			protocolerrors.Errorf(true, "received unexpected message type. "+
				"expected: %s, got: %s", appmessage.CmdIBDBlock, message.Command())
	}
}

func (flow *handleIBDFlow) processIBDBlock(msgIBDBlock *appmessage.MsgIBDBlock) error {
	block := util.NewBlock(msgIBDBlock.MsgBlock)
	if flow.DAG().IsInDAG(block.Hash()) {
		log.Debugf("IBD block %s is already in the DAG. Skipping...", block.Hash())
		return nil
	}
	isOrphan, isDelayed, err := flow.DAG().ProcessBlock(block, blockdag.BFNone)
	if err != nil {
		if !errors.As(err, &blockdag.RuleError{}) {
			return errors.Wrapf(err, "failed to process block %s during IBD", block.Hash())
		}
		log.Infof("Rejected block %s from %s during IBD: %s", block.Hash(), flow.peer, err)

		return protocolerrors.Wrapf(true, err, "got invalid block %s during IBD", block.Hash())
	}
	if isOrphan {
		return protocolerrors.Errorf(true, "received orphan block %s "+
			"during IBD", block.Hash())
	}
	if isDelayed {
		return protocolerrors.Errorf(false, "received delayed block %s "+
			"during IBD", block.Hash())
	}
	err = flow.OnNewBlock(block)
	if err != nil {
		return err
	}
	err = blocklogger.LogBlock(block)
	if err != nil {
		return err
	}
	return nil
}
