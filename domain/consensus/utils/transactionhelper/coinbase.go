package transactionhelper

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
)

const CoinbaseTransactionIndex = 0

// IsCoinBase determines whether or not a transaction is a coinbase transaction. A coinbase
// transaction is a special transaction created by miners that distributes fees and block subsidy
// to the previous blocks' miners, and to specify the scriptPubKey that will be used to pay the current
// miner in future blocks. Each input of the coinbase transaction should set index to maximum
// value and reference the relevant block id, instead of previous transaction id.
func IsCoinBase(tx *model.DomainTransaction) bool {
	// A coinbase transaction must have subnetwork id SubnetworkIDCoinbase
	return *tx.SubnetworkID == *subnetworks.SubnetworkIDCoinbase
}