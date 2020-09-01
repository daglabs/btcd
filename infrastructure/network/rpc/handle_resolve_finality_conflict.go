package rpc

import (
	"github.com/kaspanet/kaspad/infrastructure/network/rpc/model"
	"github.com/kaspanet/kaspad/util/daghash"
)

// handleResolveFinalityConflict implements the resolveFinalityConflict command.
func handleResolveFinalityConflict(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*model.ResolveFinalityConflictCmd)

	finalityBlockHash, err := daghash.NewHashFromStr(c.FinalityBlockHash)
	if err != nil {
		return nil, err
	}

	return nil, s.dag.ResolveFinalityConflict(finalityBlockHash)
}
