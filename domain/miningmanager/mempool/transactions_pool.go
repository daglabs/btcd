package mempool

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/miningmanager/mempool/model"
)

type transactionsPool struct {
	mempool                               *mempool
	allTransactions                       model.IDToTransaction
	highPriorityTransactions              model.IDToTransaction
	chainedTransactionsByPreviousOutpoint model.OutpointToTransaction
	transactionsOrderedByFeeRate          model.TransactionsOrderedByFeeRate
	lastExpireScan                        uint64
}

func newTransactionsPool(mp *mempool) *transactionsPool {
	return &transactionsPool{
		mempool:                               mp,
		allTransactions:                       model.IDToTransaction{},
		highPriorityTransactions:              model.IDToTransaction{},
		chainedTransactionsByPreviousOutpoint: model.OutpointToTransaction{},
		transactionsOrderedByFeeRate:          model.TransactionsOrderedByFeeRate{},
		lastExpireScan:                        0,
	}
}

// this function MUST be called with the mempool mutex locked for writes
func (tp *transactionsPool) addTransaction(transaction *externalapi.DomainTransaction,
	parentTransactionsInPool model.OutpointToTransaction, isHighPriority bool) (*model.MempoolTransaction, error) {

	virtualDAAScore, err := tp.mempool.virtualDAAScore()
	if err != nil {
		return nil, err
	}

	mempoolTransaction := model.NewMempoolTransaction(
		transaction, parentTransactionsInPool, isHighPriority, virtualDAAScore)

	err = tp.addMempoolTransaction(mempoolTransaction)
	if err != nil {
		return nil, err
	}

	return mempoolTransaction, nil
}

// this function MUST be called with the mempool mutex locked for writes
func (tp *transactionsPool) addMempoolTransaction(transaction *model.MempoolTransaction) error {
	tp.allTransactions[*transaction.TransactionID()] = transaction

	for outpoint, parentTransactionInPool := range transaction.ParentTransactionsInPool() {
		tp.chainedTransactionsByPreviousOutpoint[outpoint] = parentTransactionInPool
	}

	tp.mempool.mempoolUTXOSet.addTransaction(transaction)

	err := tp.transactionsOrderedByFeeRate.Push(transaction)
	if err != nil {
		return err
	}

	if transaction.IsHighPriority() {
		tp.highPriorityTransactions[*transaction.TransactionID()] = transaction
	}

	return nil
}

// this function MUST be called with the mempool mutex locked for writes
func (tp *transactionsPool) removeTransaction(transaction *model.MempoolTransaction) error {
	delete(tp.allTransactions, *transaction.TransactionID())

	err := tp.transactionsOrderedByFeeRate.Remove(transaction)
	if err != nil {
		return err
	}

	delete(tp.highPriorityTransactions, *transaction.TransactionID())

	for outpoint := range transaction.ParentTransactionsInPool() {
		delete(tp.chainedTransactionsByPreviousOutpoint, outpoint)
	}

	return nil
}

// this function MUST be called with the mempool mutex locked for writes
func (tp *transactionsPool) expireOldTransactions() error {
	virtualDAAScore, err := tp.mempool.virtualDAAScore()
	if err != nil {
		return err
	}

	if virtualDAAScore-tp.lastExpireScan < tp.mempool.config.transactionExpireScanIntervalDAAScore {
		return nil
	}

	for _, mempoolTransaction := range tp.allTransactions {
		// Never expire high priority transactions
		if mempoolTransaction.IsHighPriority() {
			continue
		}

		// Remove all transactions whose addedAtDAAScore is older then transactionExpireIntervalDAAScore
		if virtualDAAScore-mempoolTransaction.AddedAtDAAScore() > tp.mempool.config.transactionExpireIntervalDAAScore {
			err = tp.mempool.RemoveTransaction(mempoolTransaction.TransactionID(), true)
			if err != nil {
				return err
			}
		}
	}

	tp.lastExpireScan = virtualDAAScore
	return nil
}

// this function MUST be called with the mempool mutex locked for reads
func (tp *transactionsPool) allReadyTransactions() []*externalapi.DomainTransaction {
	result := []*externalapi.DomainTransaction{}

	for _, mempoolTransaction := range tp.allTransactions {
		if len(mempoolTransaction.ParentTransactionsInPool()) == 0 {
			result = append(result, mempoolTransaction.Transaction())
		}
	}

	return result
}

// this function MUST be called with the mempool mutex locked for reads
func (tp *transactionsPool) getParentTransactionsInPool(
	transaction *externalapi.DomainTransaction) model.OutpointToTransaction {

	parentsTransactionsInPool := model.OutpointToTransaction{}

	for _, input := range transaction.Inputs {
		if transaction, ok := tp.allTransactions[input.PreviousOutpoint.TransactionID]; ok {
			parentsTransactionsInPool[input.PreviousOutpoint] = transaction
		}
	}

	return parentsTransactionsInPool
}

// this function MUST be called with the mempool mutex locked for reads
func (tp *transactionsPool) getRedeemers(transaction *model.MempoolTransaction) []*model.MempoolTransaction {
	queue := []*model.MempoolTransaction{transaction}
	redeemers := []*model.MempoolTransaction{}
	for len(queue) > 0 {
		var current *model.MempoolTransaction
		current, queue = queue[0], queue[1:]

		outpoint := externalapi.DomainOutpoint{TransactionID: *current.TransactionID()}
		for i := range current.Transaction().Outputs {
			outpoint.Index = uint32(i)
			if redeemerTransaction, ok := tp.chainedTransactionsByPreviousOutpoint[outpoint]; ok {
				queue = append(queue, redeemerTransaction)
				redeemers = append(redeemers, redeemerTransaction)
			}
		}
	}
	return redeemers
}

// this function MUST be called with the mempool mutex locked for writes
func (tp *transactionsPool) limitTransactionCount() {
	for len(tp.allTransactions) > tp.mempool.config.maximumOrphanTransactionCount {
		err := tp.removeTransaction(tp.transactionsOrderedByFeeRate.GetByIndex(0))
		if err != nil {
			return
		}
	}
}
