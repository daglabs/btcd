package transactionvalidator

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
)

const sigCacheSize = 10_000

// transactionValidator exposes a set of validation classes, after which
// it's possible to determine whether either a transaction is valid
type transactionValidator struct {
	blockCoinbaseMaturity      uint64
	databaseContext            model.DBReader
	pastMedianTimeManager      model.PastMedianTimeManager
	ghostdagDataStore          model.GHOSTDAGDataStore
	daaBlocksStore             model.DAABlocksStore
	enableNonNativeSubnetworks bool
	massPerTxByte              uint64
	massPerScriptPubKeyByte    uint64
	massPerSigOp               uint64
	maxCoinbasePayloadLength   uint64
	sigCache                   *txscript.SigCache
	sigCacheECDSA              *txscript.SigCacheECDSA
}

// New instantiates a new TransactionValidator
func New(blockCoinbaseMaturity uint64,
	enableNonNativeSubnetworks bool,
	massPerTxByte uint64,
	massPerScriptPubKeyByte uint64,
	massPerSigOp uint64,
	maxCoinbasePayloadLength uint64,
	databaseContext model.DBReader,
	pastMedianTimeManager model.PastMedianTimeManager,
	ghostdagDataStore model.GHOSTDAGDataStore,
	daaBlocksStore model.DAABlocksStore) model.TransactionValidator {

	return &transactionValidator{
		blockCoinbaseMaturity:      blockCoinbaseMaturity,
		enableNonNativeSubnetworks: enableNonNativeSubnetworks,
		massPerTxByte:              massPerTxByte,
		massPerScriptPubKeyByte:    massPerScriptPubKeyByte,
		massPerSigOp:               massPerSigOp,
		maxCoinbasePayloadLength:   maxCoinbasePayloadLength,
		databaseContext:            databaseContext,
		pastMedianTimeManager:      pastMedianTimeManager,
		ghostdagDataStore:          ghostdagDataStore,
		daaBlocksStore:             daaBlocksStore,
		sigCache:                   txscript.NewSigCache(sigCacheSize),
		sigCacheECDSA:              txscript.NewSigCacheECDSA(sigCacheSize),
	}
}
