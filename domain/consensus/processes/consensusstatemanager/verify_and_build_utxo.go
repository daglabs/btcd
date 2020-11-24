package consensusstatemanager

import (
	"sort"

	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionhelper"

	"github.com/kaspanet/kaspad/domain/consensus/utils/coinbase"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionid"

	"github.com/kaspanet/kaspad/domain/consensus/utils/consensusserialization"

	"github.com/kaspanet/kaspad/domain/consensus/utils/merkle"

	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/pkg/errors"
)

func (csm *consensusStateManager) verifyUTXO(block *externalapi.DomainBlock, blockHash *externalapi.DomainHash,
	pastUTXODiff *model.UTXODiff, acceptanceData model.AcceptanceData, multiset model.Multiset) error {

	log.Tracef("verifyUTXO start for block %s", blockHash)
	defer log.Tracef("verifyUTXO end for block %s", blockHash)

	log.Tracef("Validating UTXO commitment for block %s", blockHash)
	err := csm.validateUTXOCommitment(block, blockHash, multiset)
	if err != nil {
		return err
	}
	log.Tracef("UTXO commitment validation passed for block %s", blockHash)

	log.Tracef("Validating acceptedIDMerkleRoot for block %s", blockHash)
	err = csm.validateAcceptedIDMerkleRoot(block, blockHash, acceptanceData)
	if err != nil {
		return err
	}
	log.Tracef("AcceptedIDMerkleRoot validation passed for block %s", blockHash)

	coinbaseTransaction := block.Transactions[0]
	log.Tracef("Validating coinbase transaction %s for block %s",
		consensusserialization.TransactionID(coinbaseTransaction), blockHash)
	err = csm.validateCoinbaseTransaction(blockHash, coinbaseTransaction)
	if err != nil {
		return err
	}
	log.Tracef("Coinbase transaction validation passed for block %s", blockHash)

	log.Tracef("Validating transactions against past UTXO for block %s", blockHash)
	err = csm.validateBlockTransactionsAgainstPastUTXO(block, blockHash, pastUTXODiff)
	if err != nil {
		return err
	}
	log.Tracef("Transactions against past UTXO validation passed for block %s", blockHash)

	return nil
}

func (csm *consensusStateManager) validateBlockTransactionsAgainstPastUTXO(block *externalapi.DomainBlock,
	blockHash *externalapi.DomainHash, pastUTXODiff *model.UTXODiff) error {

	log.Tracef("validateBlockTransactionsAgainstPastUTXO start for block %s", blockHash)
	defer log.Tracef("validateBlockTransactionsAgainstPastUTXO end for block %s", blockHash)

	selectedParentMedianTime, err := csm.pastMedianTimeManager.PastMedianTime(blockHash)
	if err != nil {
		return err
	}
	log.Tracef("The past median time of %s is %d", blockHash, selectedParentMedianTime)

	for i, transaction := range block.Transactions {
		transactionID := consensusserialization.TransactionID(transaction)
		log.Tracef("Validating transaction %s in block %s against "+
			"the block's past UTXO", transactionID, blockHash)
		if i == transactionhelper.CoinbaseTransactionIndex {
			log.Tracef("Skipping transaction %s because it is the coinbase", transactionID)
			continue
		}

		log.Tracef("Populating transaction %s with UTXO entries", transactionID)
		err = csm.populateTransactionWithUTXOEntriesFromVirtualOrDiff(transaction, pastUTXODiff)
		if err != nil {
			return err
		}

		log.Tracef("Validating transaction %s and populating it with mass and fee", transactionID)
		err = csm.transactionValidator.ValidateTransactionInContextAndPopulateMassAndFee(
			transaction, blockHash, selectedParentMedianTime)
		if err != nil {
			return err
		}
		log.Tracef("Validation against the block's past UTXO "+
			"passed for transaction %s in block %s", transactionID, blockHash)
	}
	return nil
}

func (csm *consensusStateManager) validateAcceptedIDMerkleRoot(block *externalapi.DomainBlock,
	blockHash *externalapi.DomainHash, acceptanceData model.AcceptanceData) error {

	log.Tracef("validateAcceptedIDMerkleRoot start for block %s", blockHash)
	defer log.Tracef("validateAcceptedIDMerkleRoot end for block %s", blockHash)

	calculatedAcceptedIDMerkleRoot := calculateAcceptedIDMerkleRoot(acceptanceData)
	if block.Header.AcceptedIDMerkleRoot != *calculatedAcceptedIDMerkleRoot {
		return errors.Wrapf(ruleerrors.ErrBadMerkleRoot, "block %s accepted ID merkle root is invalid - block "+
			"header indicates %s, but calculated value is %s",
			blockHash, &block.Header.UTXOCommitment, calculatedAcceptedIDMerkleRoot)
	}

	return nil
}

func (csm *consensusStateManager) validateUTXOCommitment(
	block *externalapi.DomainBlock, blockHash *externalapi.DomainHash, multiset model.Multiset) error {

	log.Tracef("validateUTXOCommitment start for block %s", blockHash)
	defer log.Tracef("validateUTXOCommitment end for block %s", blockHash)

	multisetHash := multiset.Hash()
	if block.Header.UTXOCommitment != *multisetHash {
		return errors.Wrapf(ruleerrors.ErrBadUTXOCommitment, "block %s UTXO commitment is invalid - block "+
			"header indicates %s, but calculated value is %s", blockHash, &block.Header.UTXOCommitment, multisetHash)
	}

	return nil
}

func calculateAcceptedIDMerkleRoot(multiblockAcceptanceData model.AcceptanceData) *externalapi.DomainHash {
	log.Tracef("calculateAcceptedIDMerkleRoot start")
	defer log.Tracef("calculateAcceptedIDMerkleRoot end")

	var acceptedTransactions []*externalapi.DomainTransaction

	for _, blockAcceptanceData := range multiblockAcceptanceData {
		for _, transactionAcceptance := range blockAcceptanceData.TransactionAcceptanceData {
			if !transactionAcceptance.IsAccepted {
				continue
			}
			acceptedTransactions = append(acceptedTransactions, transactionAcceptance.Transaction)
		}
	}
	sort.Slice(acceptedTransactions, func(i, j int) bool {
		return transactionid.Less(
			consensusserialization.TransactionID(acceptedTransactions[i]),
			consensusserialization.TransactionID(acceptedTransactions[j]))
	})

	return merkle.CalculateIDMerkleRoot(acceptedTransactions)
}
func (csm *consensusStateManager) validateCoinbaseTransaction(blockHash *externalapi.DomainHash,
	coinbaseTransaction *externalapi.DomainTransaction) error {

	log.Tracef("validateCoinbaseTransaction start for block %s", blockHash)
	defer log.Tracef("validateCoinbaseTransaction end for block %s", blockHash)

	log.Tracef("Extracting coinbase data for coinbase transaction %s in block %s",
		consensusserialization.TransactionID(coinbaseTransaction), blockHash)
	_, coinbaseData, err := coinbase.ExtractCoinbaseDataAndBlueScore(coinbaseTransaction)
	if err != nil {
		return err
	}

	log.Tracef("Calculating the expected coinbase transaction for the given coinbase data and block %s", blockHash)
	expectedCoinbaseTransaction, err := csm.coinbaseManager.ExpectedCoinbaseTransaction(blockHash, coinbaseData)
	if err != nil {
		return err
	}

	coinbaseTransactionHash := consensusserialization.TransactionHash(coinbaseTransaction)
	expectedCoinbaseTransactionHash := consensusserialization.TransactionHash(expectedCoinbaseTransaction)
	log.Tracef("given coinbase hash: %s, expected coinbase hash: %s",
		coinbaseTransactionHash, expectedCoinbaseTransactionHash)

	if *coinbaseTransactionHash != *expectedCoinbaseTransactionHash {
		return errors.Wrap(ruleerrors.ErrBadCoinbaseTransaction, "coinbase transaction is not built as expected")
	}

	return nil
}
