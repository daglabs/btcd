package validator

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/hashes"
	"github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionhelper"
)

const (
	// sompiPerKaspa is the number of sompi in one kaspa (1 KAS).
	sompiPerKaspa = 100000000

	// maxSompi is the maximum transaction amount allowed in sompi.
	maxSompi = 21000000 * sompiPerKaspa
)

func (bv *Validator) checkTransactionInIsolation(tx *model.DomainTransaction) error {
	err := bv.checkTransactionInputCount(tx)
	if err != nil {
		return err
	}
	err = bv.checkTransactionAmountRanges(tx)
	if err != nil {
		return err
	}
	err = bv.checkDuplicateTransactionInputs(tx)
	if err != nil {
		return err
	}
	err = bv.checkCoinbaseLength(tx)
	if err != nil {
		return err
	}
	err = bv.checkTransactionPayloadHash(tx)
	if err != nil {
		return err
	}
	err = bv.checkGasInBuiltInOrNativeTransactions(tx)
	if err != nil {
		return err
	}
	err = bv.checkSubnetworkRegistryTransaction(tx)
	if err != nil {
		return err
	}
	err = bv.checkNativeTransactionPayload(tx)
	if err != nil {
		return err
	}

	// TODO: fill it with the right subnetwork id.
	err = bv.checkTransactionSubnetwork(tx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (bv *Validator) checkTransactionInputCount(tx *model.DomainTransaction) error {
	// A non-coinbase transaction must have at least one input.
	if !transactionhelper.IsCoinBase(tx) && len(tx.Inputs) == 0 {
		return ruleerrors.Errorf(ruleerrors.ErrNoTxInputs, "transaction has no inputs")
	}
	return nil
}

func (bv *Validator) checkTransactionAmountRanges(tx *model.DomainTransaction) error {
	// Ensure the transaction amounts are in range. Each transaction
	// output must not be negative or more than the max allowed per
	// transaction. Also, the total of all outputs must abide by the same
	// restrictions. All amounts in a transaction are in a unit value known
	// as a sompi. One kaspa is a quantity of sompi as defined by the
	// sompiPerKaspa constant.
	var totalSompi uint64
	for _, txOut := range tx.Outputs {
		sompi := txOut.Value
		if sompi > maxSompi {
			return ruleerrors.Errorf(ruleerrors.ErrBadTxOutValue, "transaction output value of %d is "+
				"higher than max allowed value of %d", sompi, maxSompi)
		}

		// Binary arithmetic guarantees that any overflow is detected and reported.
		// This is impossible for Kaspa, but perhaps possible if an alt increases
		// the total money supply.
		newTotalSompi := totalSompi + sompi
		if newTotalSompi < totalSompi {
			return ruleerrors.Errorf(ruleerrors.ErrBadTxOutValue, "total value of all transaction "+
				"outputs exceeds max allowed value of %d",
				maxSompi)
		}
		totalSompi = newTotalSompi
		if totalSompi > maxSompi {
			return ruleerrors.Errorf(ruleerrors.ErrBadTxOutValue, "total value of all transaction "+
				"outputs is %d which is higher than max "+
				"allowed value of %d", totalSompi,
				maxSompi)
		}
	}

	return nil
}

func (bv *Validator) checkDuplicateTransactionInputs(tx *model.DomainTransaction) error {
	existingTxOut := make(map[model.DomainOutpoint]struct{})
	for _, txIn := range tx.Inputs {
		if _, exists := existingTxOut[txIn.PreviousOutpoint]; exists {
			return ruleerrors.Errorf(ruleerrors.ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutpoint] = struct{}{}
	}
	return nil
}

func (bv *Validator) checkCoinbaseLength(tx *model.DomainTransaction) error {
	if !transactionhelper.IsCoinBase(tx) {
		return nil
	}

	// Coinbase payload length must not exceed the max length.
	payloadLen := len(tx.Payload)
	const maxCoinbasePayloadLen = 150
	if payloadLen > maxCoinbasePayloadLen {
		return ruleerrors.Errorf(ruleerrors.ErrBadCoinbasePayloadLen, "coinbase transaction payload length "+
			"of %d is out of range (max: %d)",
			payloadLen, maxCoinbasePayloadLen)
	}

	return nil
}

func (bv *Validator) checkTransactionPayloadHash(tx *model.DomainTransaction) error {
	if tx.SubnetworkID != subnetworks.SubnetworkIDNative {
		payloadHash := hashes.HashData(tx.Payload)
		if tx.PayloadHash != payloadHash {
			return ruleerrors.Errorf(ruleerrors.ErrInvalidPayloadHash, "invalid payload hash")
		}
	} else if tx.PayloadHash != (model.DomainHash{}) {
		return ruleerrors.Errorf(ruleerrors.ErrInvalidPayloadHash, "unexpected non-empty payload hash in native subnetwork")
	}
	return nil
}

func (bv *Validator) checkGasInBuiltInOrNativeTransactions(tx *model.DomainTransaction) error {
	// Transactions in native, registry and coinbase subnetworks must have Gas = 0
	if subnetworks.IsBuiltInOrNative(tx.SubnetworkID) && tx.Gas > 0 {
		return ruleerrors.Errorf(ruleerrors.ErrInvalidGas, "transaction in the native or "+
			"registry subnetworks has gas > 0 ")
	}
	return nil
}

func (bv *Validator) checkSubnetworkRegistryTransaction(tx *model.DomainTransaction) error {
	if tx.SubnetworkID != subnetworks.SubnetworkIDRegistry {
		return nil
	}

	if len(tx.Payload) != 8 {
		return ruleerrors.Errorf(ruleerrors.ErrSubnetworkRegistry, "validation failed: subnetwork registry "+
			"tx has an invalid payload")
	}
	return nil
}

func (bv *Validator) checkNativeTransactionPayload(tx *model.DomainTransaction) error {
	if tx.SubnetworkID == subnetworks.SubnetworkIDNative && len(tx.Payload) > 0 {
		return ruleerrors.Errorf(ruleerrors.ErrInvalidPayload, "transaction in the native subnetwork "+
			"includes a payload")
	}
	return nil
}

func (bv *Validator) checkTransactionSubnetwork(tx *model.DomainTransaction, subnetworkID *model.DomainSubnetworkID) error {
	// If we are a partial node, only transactions on built in subnetworks
	// or our own subnetwork may have a payload
	isLocalNodeFull := subnetworkID == nil
	shouldTxBeFull := subnetworks.IsBuiltIn(tx.SubnetworkID) || tx.SubnetworkID == *subnetworkID
	if !isLocalNodeFull && !shouldTxBeFull && len(tx.Payload) > 0 {
		return ruleerrors.Errorf(ruleerrors.ErrInvalidPayload,
			"transaction that was expected to be partial has a payload "+
				"with length > 0")
	}
	return nil
}
