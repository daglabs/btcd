package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/kaspanet/kaspad/network/rpc/model"
	"github.com/kaspanet/kaspad/util/daghash"
	"strconv"
)

// handleGetBlockHeader implements the getBlockHeader command.
func handleGetBlockHeader(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*model.GetBlockHeaderCmd)

	// Fetch the header from DAG.
	hash, err := daghash.NewHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	blockHeader, err := s.dag.HeaderByHash(hash)
	if err != nil {
		return nil, &model.RPCError{
			Code:    model.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

	// When the verbose flag isn't set, simply return the serialized block
	// header as a hex-encoded string.
	if c.Verbose != nil && !*c.Verbose {
		var headerBuf bytes.Buffer
		err := blockHeader.Serialize(&headerBuf)
		if err != nil {
			context := "Failed to serialize block header"
			return nil, internalRPCError(err.Error(), context)
		}
		return hex.EncodeToString(headerBuf.Bytes()), nil
	}

	// The verbose flag is set, so generate the JSON object and return it.

	// Get the hashes for the next blocks unless there are none.
	childHashes, err := s.dag.ChildHashesByHash(hash)
	if err != nil {
		context := "No next block"
		return nil, internalRPCError(err.Error(), context)
	}
	childHashStrings := daghash.Strings(childHashes)

	blockConfirmations, err := s.dag.BlockConfirmationsByHash(hash)
	if err != nil {
		context := "Could not get block confirmations"
		return nil, internalRPCError(err.Error(), context)
	}

	selectedParentHash, err := s.dag.SelectedParentHash(hash)
	if err != nil {
		context := "Could not get block selected parent"
		return nil, internalRPCError(err.Error(), context)
	}

	params := s.dag.Params
	blockHeaderReply := model.GetBlockHeaderVerboseResult{
		Hash:                 c.Hash,
		Confirmations:        blockConfirmations,
		Version:              blockHeader.Version,
		VersionHex:           fmt.Sprintf("%08x", blockHeader.Version),
		HashMerkleRoot:       blockHeader.HashMerkleRoot.String(),
		AcceptedIDMerkleRoot: blockHeader.AcceptedIDMerkleRoot.String(),
		ChildHashes:          childHashStrings,
		ParentHashes:         daghash.Strings(blockHeader.ParentHashes),
		SelectedParentHash:   selectedParentHash.String(),
		Nonce:                blockHeader.Nonce,
		Time:                 blockHeader.Timestamp.UnixMilliseconds(),
		Bits:                 strconv.FormatInt(int64(blockHeader.Bits), 16),
		Difficulty:           getDifficultyRatio(blockHeader.Bits, params),
	}
	return blockHeaderReply, nil
}
