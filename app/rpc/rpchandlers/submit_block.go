package rpchandlers

import (
	"encoding/hex"
	"fmt"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/app/rpc/rpccontext"
	"github.com/kaspanet/kaspad/app/rpc/rpcerrors"
	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/router"
	"github.com/kaspanet/kaspad/util"
)

// HandleSubmitBlock handles the respectively named RPC command
func HandleSubmitBlock(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	submitBlockRequest := request.(*appmessage.SubmitBlockRequestMessage)

	// Deserialize the submitted block.
	serializedBlock, err := hex.DecodeString(submitBlockRequest.BlockHex)
	if err != nil {
		return nil, &rpcerrors.RPCError{
			Message: fmt.Sprintf("Block hex could not be parsed: %s", err),
		}
	}
	block, err := util.NewBlockFromBytes(serializedBlock)
	if err != nil {
		return nil, &rpcerrors.RPCError{
			Message: "Block decode failed: " + err.Error(),
		}
	}

	err = context.ProtocolManager.AddBlock(block, blockdag.BFDisallowDelay|blockdag.BFDisallowOrphans)
	if err != nil {
		return nil, &rpcerrors.RPCError{
			Message: fmt.Sprintf("Block rejected. Reason: %s", err),
		}
	}

	log.Infof("Accepted block %s via submitBlock", block.Hash())

	response := appmessage.NewSubmitBlockResponseMessage()
	return response, nil
}
