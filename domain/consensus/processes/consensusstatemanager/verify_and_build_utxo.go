package consensusstatemanager

import (
	"sort"

	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionid"

	"github.com/kaspanet/kaspad/domain/consensus/utils/consensusserialization"

	"github.com/kaspanet/kaspad/domain/consensus/utils/merkle"

	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/pkg/errors"
)

func (csm *consensusStateManager) verifyAndBuildUTXO(block *externalapi.DomainBlock, blockHash *externalapi.DomainHash,
	pastUTXODiff *model.UTXODiff, acceptanceData model.AcceptanceData, multiset model.Multiset) error {

	err := csm.validateUTXOCommitment(block, blockHash, multiset)
	if err != nil {
		return err
	}

	err = csm.validateAcceptedIDMerkleRoot(block, blockHash, acceptanceData)
	if err != nil {
		return err
	}

	coinbaseTransaction := block.Transactions[0]
	err = csm.validateCoinbaseTransaction(blockHash, coinbaseTransaction)

	err = csm.validateBlockTransactionsAgainstPastUTXO(block, blockHash, pastUTXODiff, err)
	if err != nil {
		return err
	}

	return nil
}

func (csm *consensusStateManager) validateBlockTransactionsAgainstPastUTXO(block *externalapi.DomainBlock,
	blockHash *externalapi.DomainHash, pastUTXODiff *model.UTXODiff, err error) error {

	ghostdagData, err := csm.ghostdagDataStore.Get(csm.databaseContext, blockHash)
	if err != nil {
		return err
	}
	selectedParentMedianTime, err := csm.pastMedianTimeManager.PastMedianTime(ghostdagData.SelectedParent)
	if err != nil {
		return err
	}

	for _, transaction := range block.Transactions {
		err = csm.populateTransactionWithUTXOEntriesFromVirtualOrDiff(transaction, pastUTXODiff)
		if err != nil {
			return err
		}

		err = csm.transactionValidator.ValidateTransactionInContextAndPopulateMassAndFee(
			transaction, blockHash, selectedParentMedianTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func (csm *consensusStateManager) validateAcceptedIDMerkleRoot(block *externalapi.DomainBlock,
	blockHash *externalapi.DomainHash, acceptanceData model.AcceptanceData) error {

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

	multisetHash := multiset.Hash()
	if block.Header.UTXOCommitment != *multisetHash {
		return errors.Wrapf(ruleerrors.ErrBadUTXOCommitment, "block %s UTXO commitment is invalid - block "+
			"header indicates %s, but calculated value is %s", blockHash, &block.Header.UTXOCommitment, multisetHash)
	}

	return nil
}

func calculateAcceptedIDMerkleRoot(multiblockAcceptanceData model.AcceptanceData) *externalapi.DomainHash {
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
	_, coinbaseData, err := csm.coinbaseManager.ExtractCoinbaseDataAndBlueScore(coinbaseTransaction)
	if err != nil {
		return err
	}

	expectedCoinbaseTransaction, err := csm.coinbaseManager.ExpectedCoinbaseTransaction(blockHash, coinbaseData)
	if err != nil {
		return err
	}

	coinbaseTransactionHash := consensusserialization.TransactionHash(coinbaseTransaction)
	expectedCoinbaseTransactionHash := consensusserialization.TransactionHash(expectedCoinbaseTransaction)
	if *coinbaseTransactionHash != *expectedCoinbaseTransactionHash {
		return errors.Wrap(ruleerrors.ErrBadCoinbaseTransaction, "coinbase transaction is not built as expected")
	}

	return nil
}
