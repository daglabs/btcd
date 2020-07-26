package rpc

import (
	"github.com/kaspanet/kaspad/addressmanager"
	"github.com/kaspanet/kaspad/rpcmodel"
)

// handleGetPeerAddresses handles getPeerAddresses commands.
func handleGetPeerAddresses(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	peersState, err := s.cfg.addressManager.PeersStateForSerialization()
	if err != nil {
		return nil, err
	}

	rpcPeersState := rpcmodel.GetPeerAddressesResult{
		Version:              peersState.Version,
		Key:                  peersState.Key,
		Addresses:            make([]*rpcmodel.GetPeerAddressesKnownAddressResult, len(peersState.Addresses)),
		NewBuckets:           make(map[string]*rpcmodel.GetPeerAddressesNewBucketResult),
		NewBucketFullNodes:   rpcmodel.GetPeerAddressesNewBucketResult{},
		TriedBuckets:         make(map[string]*rpcmodel.GetPeerAddressesTriedBucketResult),
		TriedBucketFullNodes: rpcmodel.GetPeerAddressesTriedBucketResult{},
	}

	for i, addr := range peersState.Addresses {
		rpcPeersState.Addresses[i] = &rpcmodel.GetPeerAddressesKnownAddressResult{
			Addr:         string(addr.Address),
			Src:          string(addr.SourceAddress),
			SubnetworkID: addr.SubnetworkID,
			Attempts:     addr.Attempts,
			TimeStamp:    addr.TimeStamp,
			LastAttempt:  addr.LastAttempt,
			LastSuccess:  addr.LastSuccess,
		}
	}

	for subnetworkID, bucket := range peersState.SubnetworkNewAddressBucketArrays {
		rpcPeersState.NewBuckets[subnetworkID] = &rpcmodel.GetPeerAddressesNewBucketResult{}
		for i, addr := range bucket {
			rpcPeersState.NewBuckets[subnetworkID][i] = convertAddressKeySliceToString(addr)
		}
	}

	for i, addr := range peersState.FullNodeNewAddressBucketArray {
		rpcPeersState.NewBucketFullNodes[i] = convertAddressKeySliceToString(addr)
	}

	for subnetworkID, bucket := range peersState.SubnetworkTriedAddressBucketArrays {
		rpcPeersState.TriedBuckets[subnetworkID] = &rpcmodel.GetPeerAddressesTriedBucketResult{}
		for i, addr := range bucket {
			rpcPeersState.TriedBuckets[subnetworkID][i] = convertAddressKeySliceToString(addr)
		}
	}

	for i, addr := range peersState.FullNodeTriedAddressBucketArray {
		rpcPeersState.TriedBucketFullNodes[i] = convertAddressKeySliceToString(addr)
	}

	return rpcPeersState, nil
}

func convertAddressKeySliceToString(addressKeys []addressmanager.AddressKey) []string {
	strings := make([]string, len(addressKeys))
	for j, addr := range addressKeys {
		strings[j] = string(addr)
	}
	return strings
}
