package flowcontext

import (
	"github.com/kaspanet/kaspad/mempool"
	"github.com/kaspanet/kaspad/protocol/flows/relaytransactions"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/wire"
	"github.com/pkg/errors"
	"time"
)

// AddTransaction adds transaction to the mempool and propagates it.
func (f *FlowContext) AddTransaction(tx *util.Tx) error {
	f.transactionsToRebroadcastLock.Lock()
	defer f.transactionsToRebroadcastLock.Unlock()

	transactionsAcceptedToMempool, err := f.txPool.ProcessTransaction(tx, false, 0)
	if err != nil {
		return err
	}

	if len(transactionsAcceptedToMempool) > 1 {
		panic(errors.New("got more than one accepted transactions when no orphans were allowed"))
	}

	f.transactionsToRebroadcast[*tx.ID()] = tx
	inv := wire.NewMsgTxInv([]*daghash.TxID{tx.ID()})
	return f.Broadcast(inv)
}

func (f *FlowContext) updateTransactionsToRebroadcast(block *util.Block) {
	f.transactionsToRebroadcastLock.Lock()
	defer f.transactionsToRebroadcastLock.Unlock()
	// Note: if the block is red, its transactions won't be rebroadcasted
	// anymore, although they are not included in the UTXO set.
	// This is probably ok, since red blocks are quite rare.
	for _, tx := range block.Transactions() {
		delete(f.transactionsToRebroadcast, *tx.ID())
	}
}

func (f *FlowContext) shouldRebroadcastTransactions() bool {
	const rebroadcastInterval = 30 * time.Second
	return time.Since(f.lastRebroadcastTime) > rebroadcastInterval
}

func (f *FlowContext) txIDsToRebroadcast() []*daghash.TxID {
	f.transactionsToRebroadcastLock.Lock()
	defer f.transactionsToRebroadcastLock.Unlock()

	txIDs := make([]*daghash.TxID, len(f.transactionsToRebroadcast))
	i := 0
	for _, tx := range f.transactionsToRebroadcast {
		txIDs[i] = tx.ID()
		i++
	}
	return txIDs
}

// SharedRequestedTransactions returns a *relaytransactions.SharedRequestedTransactions for sharing
// data about requested transactions between different peers.
func (f *FlowContext) SharedRequestedTransactions() *relaytransactions.SharedRequestedTransactions {
	return f.sharedRequestedTransactions
}

// TxPool returns the transaction pool associated to the manager.
func (f *FlowContext) TxPool() *mempool.TxPool {
	return f.txPool
}
