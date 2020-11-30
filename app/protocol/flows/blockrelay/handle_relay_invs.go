package blockrelay

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/protocol/common"
	peerpkg "github.com/kaspanet/kaspad/app/protocol/peer"
	"github.com/kaspanet/kaspad/app/protocol/protocolerrors"
	"github.com/kaspanet/kaspad/domain"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/blocks"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensusserialization"
	"github.com/kaspanet/kaspad/infrastructure/config"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/pkg/errors"
)

// RelayInvsContext is the interface for the context needed for the HandleRelayInvs flow.
type RelayInvsContext interface {
	Domain() domain.Domain
	Config() *config.Config
	NetAdapter() *netadapter.NetAdapter
	OnNewBlock(block *externalapi.DomainBlock) error
	SharedRequestedBlocks() *SharedRequestedBlocks
	Broadcast(message appmessage.Message) error
	AddOrphan(orphanBlock *externalapi.DomainBlock)
	IsOrphan(blockHash *externalapi.DomainHash) bool
	IsInIBD() bool
}

type handleRelayInvsFlow struct {
	RelayInvsContext
	incomingRoute, outgoingRoute *router.Route
	peer                         *peerpkg.Peer
	invsQueue                    []*appmessage.MsgInvRelayBlock
}

// HandleRelayInvs listens to appmessage.MsgInvRelayBlock messages, requests their corresponding blocks if they
// are missing, adds them to the DAG and propagates them to the rest of the network.
func HandleRelayInvs(context RelayInvsContext, incomingRoute *router.Route, outgoingRoute *router.Route,
	peer *peerpkg.Peer) error {

	flow := &handleRelayInvsFlow{
		RelayInvsContext: context,
		incomingRoute:    incomingRoute,
		outgoingRoute:    outgoingRoute,
		peer:             peer,
		invsQueue:        make([]*appmessage.MsgInvRelayBlock, 0),
	}
	return flow.start()
}

func (flow *handleRelayInvsFlow) start() error {
	for {
		inv, err := flow.readInv()
		if err != nil {
			return err
		}

		log.Debugf("Got relay inv for block %s", inv.Hash)

		blockInfo, err := flow.Domain().Consensus().GetBlockInfo(inv.Hash)
		if err != nil {
			return err
		}
		if blockInfo.Exists {
			if blockInfo.BlockStatus == externalapi.StatusInvalid {
				return protocolerrors.Errorf(true, "sent inv of an invalid block %s",
					inv.Hash)
			}
			continue
		}

		if flow.IsOrphan(inv.Hash) {
			continue
		}

		// Block relay is disabled during IBD
		if flow.IsInIBD() {
			continue
		}

		block, err := flow.requestBlock(inv.Hash)
		if err != nil {
			return err
		}
		if block == nil {
			continue
		}

		missingParents, err := flow.processBlock(block)
		if err != nil {
			return err
		}
		if len(missingParents) > 0 {
			err := flow.processOrphan(block, missingParents)
			if err != nil {
				return err
			}
			continue
		}

		err = flow.relayBlock(block)
		if err != nil {
			return err
		}
	}
}

func (flow *handleRelayInvsFlow) readInv() (*appmessage.MsgInvRelayBlock, error) {
	if len(flow.invsQueue) > 0 {
		var inv *appmessage.MsgInvRelayBlock
		inv, flow.invsQueue = flow.invsQueue[0], flow.invsQueue[1:]
		return inv, nil
	}

	msg, err := flow.incomingRoute.Dequeue()
	if err != nil {
		return nil, err
	}

	inv, ok := msg.(*appmessage.MsgInvRelayBlock)
	if !ok {
		return nil, protocolerrors.Errorf(true, "unexpected %s message in the block relay handleRelayInvsFlow while "+
			"expecting an inv message", msg.Command())
	}
	return inv, nil
}

func (flow *handleRelayInvsFlow) requestBlock(requestHash *externalapi.DomainHash) (*externalapi.DomainBlock, error) {
	exists := flow.SharedRequestedBlocks().addIfNotExists(requestHash)
	if exists {
		return nil, nil
	}

	// In case the function returns earlier than expected, we want to make sure requestedBlocks is
	// clean from any pending blocks.
	defer flow.SharedRequestedBlocks().remove(requestHash)

	// The block can become known from another peer in the process of orphan resolution
	blockInfo, err := flow.Domain().Consensus().GetBlockInfo(requestHash)
	if err != nil {
		return nil, err
	}
	if blockInfo.Exists {
		return nil, nil
	}

	getRelayBlocksMsg := appmessage.NewMsgRequestRelayBlocks([]*externalapi.DomainHash{requestHash})
	err = flow.outgoingRoute.Enqueue(getRelayBlocksMsg)
	if err != nil {
		return nil, err
	}

	msgBlock, err := flow.readMsgBlock()
	if err != nil {
		return nil, err
	}

	block := appmessage.MsgBlockToDomainBlock(msgBlock)
	blockHash := consensusserialization.BlockHash(block)

	if *blockHash != *requestHash {
		return nil, protocolerrors.Errorf(true, "got unrequested block %s", blockHash)
	}

	return block, nil
}

// readMsgBlock returns the next msgBlock in msgChan, and populates invsQueue with any inv messages that meanwhile arrive.
//
// Note: this function assumes msgChan can contain only appmessage.MsgInvRelayBlock and appmessage.MsgBlock messages.
func (flow *handleRelayInvsFlow) readMsgBlock() (msgBlock *appmessage.MsgBlock, err error) {
	for {
		message, err := flow.incomingRoute.DequeueWithTimeout(common.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		switch message := message.(type) {
		case *appmessage.MsgInvRelayBlock:
			flow.invsQueue = append(flow.invsQueue, message)
		case *appmessage.MsgBlock:
			return message, nil
		default:
			return nil, errors.Errorf("unexpected message %s", message.Command())
		}
	}
}

func (flow *handleRelayInvsFlow) processBlock(block *externalapi.DomainBlock) ([]*externalapi.DomainHash, error) {
	blockHash := consensusserialization.BlockHash(block)
	err := flow.Domain().Consensus().ValidateAndInsertBlock(block)
	if err != nil {
		if !errors.As(err, &ruleerrors.RuleError{}) {
			return nil, errors.Wrapf(err, "failed to process block %s", blockHash)
		}

		missingParentsError := &ruleerrors.ErrMissingParents{}
		if errors.As(err, missingParentsError) {
			blueScore, err := blocks.ExtractBlueScore(block)
			if err != nil {
				return nil, protocolerrors.Errorf(true, "received an orphan "+
					"block %s with malformed blue score", blockHash)
			}

			const maxOrphanBlueScoreDiff = 10000
			virtualSelectedParent, err := flow.Domain().Consensus().GetVirtualSelectedParent()
			if err != nil {
				return nil, err
			}
			selectedTipBlueScore, err := blocks.ExtractBlueScore(virtualSelectedParent)
			if err != nil {
				return nil, err
			}

			if blueScore > selectedTipBlueScore+maxOrphanBlueScoreDiff {
				log.Infof("Orphan block %s has blue score %d and the selected tip blue score is "+
					"%d. Ignoring orphans with a blue score difference from the selected tip greater than %d",
					blockHash, blueScore, selectedTipBlueScore, maxOrphanBlueScoreDiff)
				return nil, nil
			}

			return missingParentsError.MissingParentHashes, nil
		}
		log.Infof("Rejected block %s from %s: %s", blockHash, flow.peer, err)

		return nil, protocolerrors.Wrapf(true, err, "got invalid block %s from relay", blockHash)
	}
	return nil, nil
}

func (flow *handleRelayInvsFlow) relayBlock(block *externalapi.DomainBlock) error {
	blockHash := consensusserialization.BlockHash(block)
	err := flow.Broadcast(appmessage.NewMsgInvBlock(blockHash))
	if err != nil {
		return err
	}

	log.Infof("Accepted block %s via relay", blockHash)

	return flow.OnNewBlock(block)
}

func (flow *handleRelayInvsFlow) processOrphan(block *externalapi.DomainBlock, missingParents []*externalapi.DomainHash) error {
	blockHash := consensusserialization.BlockHash(block)

	// Return if the block has been orphaned from elsewhere already
	if flow.IsOrphan(blockHash) {
		return nil
	}

	// Add the block to the orphan set if it's within orphan resolution range
	isBlockInOrphanResolutionRange, err := flow.isBlockInOrphanResolutionRange(blockHash)
	if err != nil {
		return err
	}
	if isBlockInOrphanResolutionRange {
		flow.addToOrphanSetAndRequestMissingParents(block, missingParents)
		return nil
	}

	// Start IBD unless we already are in IBD
	return flow.runIBDIfNotRunning(blockHash)
}

func (flow *handleRelayInvsFlow) isBlockInOrphanResolutionRange(blockHash *externalapi.DomainHash) (bool, error) {
	return true, nil
}

func (flow *handleRelayInvsFlow) addToOrphanSetAndRequestMissingParents(
	block *externalapi.DomainBlock, missingParents []*externalapi.DomainHash) {

	flow.AddOrphan(block)
	invMessages := make([]*appmessage.MsgInvRelayBlock, len(missingParents))
	for i, missingParent := range missingParents {
		invMessages[i] = appmessage.NewMsgInvBlock(missingParent)
	}
	flow.invsQueue = append(invMessages, flow.invsQueue...)
}
