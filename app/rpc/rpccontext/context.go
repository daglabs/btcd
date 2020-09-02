package rpccontext

import (
	"github.com/kaspanet/kaspad/app/protocol"
	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/blockdag/indexers"
	"github.com/kaspanet/kaspad/domain/mempool"
	"github.com/kaspanet/kaspad/domain/mining"
	"github.com/kaspanet/kaspad/infrastructure/network/addressmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/connmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter"
)

type Context struct {
	NetAdapter             *netadapter.NetAdapter
	DAG                    *blockdag.BlockDAG
	ProtocolManager        *protocol.Manager
	ConnectionManager      *connmanager.ConnectionManager
	BlockTemplateGenerator *mining.BlkTmplGenerator
	Mempool                *mempool.TxPool
	AddressManager         *addressmanager.AddressManager
	AcceptanceIndex        *indexers.AcceptanceIndex

	BlockTemplateState  *BlockTemplateState
	NotificationManager *NotificationManager
}

func NewContext(
	netAdapter *netadapter.NetAdapter,
	dag *blockdag.BlockDAG,
	protocolManager *protocol.Manager,
	connectionManager *connmanager.ConnectionManager,
	blockTemplateGenerator *mining.BlkTmplGenerator,
	mempool *mempool.TxPool,
	addressManager *addressmanager.AddressManager,
	acceptanceIndex *indexers.AcceptanceIndex) *Context {
	context := &Context{
		NetAdapter:             netAdapter,
		DAG:                    dag,
		ProtocolManager:        protocolManager,
		ConnectionManager:      connectionManager,
		BlockTemplateGenerator: blockTemplateGenerator,
		Mempool:                mempool,
		AddressManager:         addressManager,
		AcceptanceIndex:        acceptanceIndex,
	}
	context.BlockTemplateState = NewBlockTemplateState(context)
	context.NotificationManager = NewNotificationManager()
	return context
}
