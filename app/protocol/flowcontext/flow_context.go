package flowcontext

import (
	"sync"
	"time"

	"github.com/kaspanet/kaspad/app/protocol/flows/blockrelay"
	"github.com/kaspanet/kaspad/app/protocol/flows/relaytransactions"
	peerpkg "github.com/kaspanet/kaspad/app/protocol/peer"
	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/mempool"
	"github.com/kaspanet/kaspad/infrastructure/config"
	"github.com/kaspanet/kaspad/infrastructure/network/addressmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/connmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter"
	"github.com/kaspanet/kaspad/infrastructure/network/netadapter/id"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
)

// OnBlockAddedToDAGHandler is a handler function that's triggered
// when a block is added to the DAG
type OnBlockAddedToDAGHandler func(block *util.Block)

// OnTransactionAddedToMempoolHandler is a handler function that's triggered
// when a transaction is added to the mempool
type OnTransactionAddedToMempoolHandler func()

// FlowContext holds state that is relevant to more than one flow or one peer, and allows communication between
// different flows that can be associated to different peers.
type FlowContext struct {
	cfg               *config.Config
	netAdapter        *netadapter.NetAdapter
	txPool            *mempool.TxPool
	dag               *blockdag.BlockDAG
	addressManager    *addressmanager.AddressManager
	connectionManager *connmanager.ConnectionManager

	onBlockAddedToDAGHandler           OnBlockAddedToDAGHandler
	onTransactionAddedToMempoolHandler OnTransactionAddedToMempoolHandler

	transactionsToRebroadcastLock sync.Mutex
	transactionsToRebroadcast     map[daghash.TxID]*util.Tx
	lastRebroadcastTime           time.Time
	sharedRequestedTransactions   *relaytransactions.SharedRequestedTransactions

	sharedRequestedBlocks *blockrelay.SharedRequestedBlocks

	isInIBD       uint32
	startIBDMutex sync.Mutex
	ibdPeer       *peerpkg.Peer

	peers      map[id.ID]*peerpkg.Peer
	peersMutex sync.RWMutex
}

// New returns a new instance of FlowContext.
func New(cfg *config.Config, dag *blockdag.BlockDAG, addressManager *addressmanager.AddressManager,
	txPool *mempool.TxPool, netAdapter *netadapter.NetAdapter,
	connectionManager *connmanager.ConnectionManager) *FlowContext {

	return &FlowContext{
		cfg:                         cfg,
		netAdapter:                  netAdapter,
		dag:                         dag,
		addressManager:              addressManager,
		connectionManager:           connectionManager,
		txPool:                      txPool,
		sharedRequestedTransactions: relaytransactions.NewSharedRequestedTransactions(),
		sharedRequestedBlocks:       blockrelay.NewSharedRequestedBlocks(),
		peers:                       make(map[id.ID]*peerpkg.Peer),
		transactionsToRebroadcast:   make(map[daghash.TxID]*util.Tx),
	}
}

// SetOnBlockAddedToDAGHandler sets the onBlockAddedToDAG handler
func (f *FlowContext) SetOnBlockAddedToDAGHandler(onBlockAddedToDAGHandler OnBlockAddedToDAGHandler) {
	f.onBlockAddedToDAGHandler = onBlockAddedToDAGHandler
}

// SetOnTransactionAddedToMempoolHandler sets the onTransactionAddedToMempool handler
func (f *FlowContext) SetOnTransactionAddedToMempoolHandler(onTransactionAddedToMempoolHandler OnTransactionAddedToMempoolHandler) {
	f.onTransactionAddedToMempoolHandler = onTransactionAddedToMempoolHandler
}
