package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// ConsensusStateManager manages the node's consensus state
type ConsensusStateManager interface {
	AddBlock(blockHash *externalapi.DomainHash) (*externalapi.SelectedParentChainChanges, error)
	PopulateTransactionWithUTXOEntries(transaction *externalapi.DomainTransaction) error
	UpdatePruningPoint(newPruningPoint *externalapi.DomainBlock, serializedUTXOSet []byte) error
	RestorePastUTXOSetIterator(blockHash *externalapi.DomainHash) (ReadOnlyUTXOSetIterator, error)
	CalculatePastUTXOAndAcceptanceData(blockHash *externalapi.DomainHash) (UTXODiff, AcceptanceData, Multiset, error)
}
