package consensus

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/model/testapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/db/database"
)

type testConsensus struct {
	*consensus
	dagParams *dagconfig.Params
	database  database.Database

	testBlockBuilder          testapi.TestBlockBuilder
	testReachabilityManager   testapi.TestReachabilityManager
	testConsensusStateManager testapi.TestConsensusStateManager
	testTransactionValidator  testapi.TestTransactionValidator
}

func (tc *testConsensus) DAGParams() *dagconfig.Params {
	return tc.dagParams
}

func (tc *testConsensus) BuildBlockWithParents(parentHashes []*externalapi.DomainHash,
	coinbaseData *externalapi.DomainCoinbaseData, transactions []*externalapi.DomainTransaction) (
	*externalapi.DomainBlock, model.UTXODiff, error) {

	// Require write lock because BuildBlockWithParents stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	block, diff, err := tc.testBlockBuilder.BuildBlockWithParents(parentHashes, coinbaseData, transactions)
	if err != nil {
		return nil, nil, err
	}

	return block, diff, nil
}

func (tc *testConsensus) AddBlock(parentHashes []*externalapi.DomainHash, coinbaseData *externalapi.DomainCoinbaseData,
	transactions []*externalapi.DomainTransaction) (*externalapi.DomainHash, *externalapi.BlockInsertionResult, error) {

	// Require write lock because BuildBlockWithParents stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	block, _, err := tc.testBlockBuilder.BuildBlockWithParents(parentHashes, coinbaseData, transactions)
	if err != nil {
		return nil, nil, err
	}

	blockInsertionResult, err := tc.blockProcessor.ValidateAndInsertBlock(block)
	if err != nil {
		return nil, nil, err
	}

	return consensushashing.BlockHash(block), blockInsertionResult, nil
}

func (tc *testConsensus) AddUTXOInvalidHeader(parentHashes []*externalapi.DomainHash) (*externalapi.DomainHash,
	*externalapi.BlockInsertionResult, error) {

	// Require write lock because BuildBlockWithParents stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	header, err := tc.testBlockBuilder.BuildUTXOInvalidHeader(parentHashes)
	if err != nil {
		return nil, nil, err
	}

	blockInsertionResult, err := tc.blockProcessor.ValidateAndInsertBlock(&externalapi.DomainBlock{
		Header:       header,
		Transactions: nil,
	})
	if err != nil {
		return nil, nil, err
	}

	return consensushashing.HeaderHash(header), blockInsertionResult, nil
}

func (tc *testConsensus) AddUTXOInvalidBlock(parentHashes []*externalapi.DomainHash) (*externalapi.DomainHash,
	*externalapi.BlockInsertionResult, error) {

	// Require write lock because BuildBlockWithParents stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	block, err := tc.testBlockBuilder.BuildUTXOInvalidBlock(parentHashes)
	if err != nil {
		return nil, nil, err
	}

	blockInsertionResult, err := tc.blockProcessor.ValidateAndInsertBlock(block)
	if err != nil {
		return nil, nil, err
	}

	return consensushashing.BlockHash(block), blockInsertionResult, nil
}

func (tc *testConsensus) BuildUTXOInvalidBlock(parentHashes []*externalapi.DomainHash) (*externalapi.DomainBlock, error) {

	// Require write lock because BuildBlockWithParents stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	return tc.testBlockBuilder.BuildUTXOInvalidBlock(parentHashes)
}

func (tc *testConsensus) BuildHeaderWithParents(parentHashes []*externalapi.DomainHash) (externalapi.BlockHeader, error) {
	// Require write lock because BuildUTXOInvalidHeader stages temporary data
	tc.lock.Lock()
	defer tc.lock.Unlock()

	return tc.testBlockBuilder.BuildUTXOInvalidHeader(parentHashes)
}

func (tc *testConsensus) DiscardAllStores() {
	tc.AcceptanceDataStore().Discard()
	tc.BlockHeaderStore().Discard()
	tc.BlockRelationStore().Discard()
	tc.BlockStatusStore().Discard()
	tc.BlockStore().Discard()
	tc.ConsensusStateStore().Discard()
	tc.GHOSTDAGDataStore().Discard()
	tc.HeaderTipsStore().Discard()
	tc.MultisetStore().Discard()
	tc.PruningStore().Discard()
	tc.ReachabilityDataStore().Discard()
	tc.UTXODiffStore().Discard()
	tc.HeadersSelectedChainStore().Discard()
}
