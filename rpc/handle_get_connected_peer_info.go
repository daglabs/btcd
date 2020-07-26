package rpc

import (
	"github.com/kaspanet/kaspad/rpc/model"
)

// handleGetConnectedPeerInfo implements the getConnectedPeerInfo command.
func handleGetConnectedPeerInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	peers := s.protocolManager.Peers()
	infos := make([]*model.GetConnectedPeerInfoResult, 0, len(peers))
	for _, peer := range peers {
		info := &model.GetConnectedPeerInfoResult{
			ID:                        peer.ID().String(),
			Address:                   peer.Address(),
			LastPingDuration:          peer.LastPingDuration().Milliseconds(),
			SelectedTipHash:           peer.SelectedTipHash().String(),
			IsSyncNode:                peer == s.protocolManager.IBDPeer(),
			IsOutbound:                peer.IsOutbound(),
			TimeOffset:                peer.TimeOffset().Milliseconds(),
			UserAgent:                 peer.UserAgent(),
			AdvertisedProtocolVersion: peer.AdvertisedProtocolVersion(),
			TimeConnected:             peer.TimeConnected().Milliseconds(),
		}
		infos = append(infos, info)
	}
	return infos, nil
}
