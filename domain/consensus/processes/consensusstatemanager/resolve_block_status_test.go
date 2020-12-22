package consensusstatemanager_test

import (
	"errors"
	"testing"

	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"

	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"

	"github.com/kaspanet/kaspad/domain/consensus"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/testutils"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionhelper"
	"github.com/kaspanet/kaspad/domain/dagconfig"
)

func TestDoubleSpends(t *testing.T) {
	testutils.ForAllNets(t, true, func(t *testing.T, params *dagconfig.Params) {
		params.BlockCoinbaseMaturity = 0

		factory := consensus.NewFactory()

		consensus, teardown, err := factory.NewTestConsensus(params, "TestUTXOCommitment")
		if err != nil {
			t.Fatalf("Error setting up consensus: %+v", err)
		}
		defer teardown(false)

		// Mine chain of two blocks to fund our double spend
		firstBlockHash, _, err := consensus.AddBlock([]*externalapi.DomainHash{params.GenesisHash}, nil, nil)
		if err != nil {
			t.Fatalf("Error creating firstBlock: %+v", err)
		}
		fundingBlockHash, _, err := consensus.AddBlock([]*externalapi.DomainHash{firstBlockHash}, nil, nil)
		if err != nil {
			t.Fatalf("Error creating fundingBlock: %+v", err)
		}
		fundingBlock, err := consensus.GetBlock(fundingBlockHash)
		if err != nil {
			t.Fatalf("Error getting fundingBlock: %+v", err)
		}

		// Get funding transaction
		fundingTransaction := fundingBlock.Transactions[transactionhelper.CoinbaseTransactionIndex]

		// Create two transactions that spends the same output, but with different IDs
		spendingTransaction1, err := testutils.CreateTransaction(fundingTransaction)
		if err != nil {
			t.Fatalf("Error creating spendingTransaction1: %+v", err)
		}
		spendingTransaction2, err := testutils.CreateTransaction(fundingTransaction)
		if err != nil {
			t.Fatalf("Error creating spendingTransaction2: %+v", err)
		}
		spendingTransaction2.Outputs[0].Value-- // tweak the value to create a different ID
		spendingTransaction1ID := consensushashing.TransactionID(spendingTransaction1)
		spendingTransaction2ID := consensushashing.TransactionID(spendingTransaction2)
		if spendingTransaction1ID.Equal(spendingTransaction2ID) {
			t.Fatalf("spendingTransaction1 and spendingTransaction2 ids are equal")
		}

		// Mine a block with spendingTransaction1 and make sure that it's valid
		goodBlock1Hash, _, err := consensus.AddBlock([]*externalapi.DomainHash{fundingBlockHash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction1})
		if err != nil {
			t.Fatalf("Error adding goodBlock1: %+v", err)
		}
		goodBlock1Status, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), goodBlock1Hash)
		if err != nil {
			t.Fatalf("Error getting status of goodBlock1: %+v", err)
		}
		if goodBlock1Status != externalapi.StatusUTXOValid {
			t.Fatalf("GoodBlock1 status expected to be '%s', but is '%s'", externalapi.StatusUTXOValid, goodBlock1Status)
		}

		// To check that a block containing the same transaction already in it's past is disqualified:
		// Add a block on top of goodBlock, containing spendingTransaction1, and make sure it's disqualified
		doubleSpendingBlock1Hash, _, err := consensus.AddBlock([]*externalapi.DomainHash{goodBlock1Hash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction1})
		if err != nil {
			t.Fatalf("Error adding doubleSpendingBlock1: %+v", err)
		}
		doubleSpendingBlock1Status, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), doubleSpendingBlock1Hash)
		if err != nil {
			t.Fatalf("Error getting status of goodBlock: %+v", err)
		}
		if doubleSpendingBlock1Status != externalapi.StatusDisqualifiedFromChain {
			t.Fatalf("doubleSpendingBlock1 status expected to be '%s', but is '%s'",
				externalapi.StatusDisqualifiedFromChain, doubleSpendingBlock1Status)
		}

		// To check that a block containing a transaction that double-spends a transaction that
		// is in it's past is disqualified:
		// Add a block on top of goodBlock, containing spendingTransaction2, and make sure it's disqualified
		doubleSpendingBlock2Hash, _, err := consensus.AddBlock([]*externalapi.DomainHash{goodBlock1Hash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction2})
		if err != nil {
			t.Fatalf("Error adding doubleSpendingBlock2: %+v", err)
		}
		doubleSpendingBlock2Status, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), doubleSpendingBlock2Hash)
		if err != nil {
			t.Fatalf("Error getting status of goodBlock: %+v", err)
		}
		if doubleSpendingBlock2Status != externalapi.StatusDisqualifiedFromChain {
			t.Fatalf("doubleSpendingBlock2 status expected to be '%s', but is '%s'",
				externalapi.StatusDisqualifiedFromChain, doubleSpendingBlock2Status)
		}

		// To make sure that a block double-spending itself is rejected:
		// Add a block on top of goodBlock, containing both spendingTransaction1 and spendingTransaction2, and make
		// sure AddBlock returns a RuleError
		_, _, err = consensus.AddBlock([]*externalapi.DomainHash{goodBlock1Hash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction1, spendingTransaction2})
		if err == nil {
			t.Fatalf("No error when adding a self-double-spending block")
		}
		if !errors.Is(err, ruleerrors.ErrDoubleSpendInSameBlock) {
			t.Fatalf("Adding self-double-spending block should have "+
				"returned ruleerrors.ErrDoubleSpendInSameBlock, but instead got: %+v", err)
		}

		// To make sure that a block containing the same transaction twice is rejected:
		// Add a block on top of goodBlock, containing spendingTransaction1 twice, and make
		// sure AddBlock returns a RuleError
		_, _, err = consensus.AddBlock([]*externalapi.DomainHash{goodBlock1Hash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction1, spendingTransaction1})
		if err == nil {
			t.Fatalf("No error when adding a block containing the same transactin twice")
		}
		if !errors.Is(err, ruleerrors.ErrDuplicateTx) {
			t.Fatalf("Adding block that contains the same transaction twice should have "+
				"returned ruleerrors.ErrDuplicateTx, but instead got: %+v", err)
		}

		// Check that a block will not get disqualified if it has a transaction that double spends
		// a transaction from its anticone.
		goodBlock2Hash, _, err := consensus.AddBlock([]*externalapi.DomainHash{fundingBlockHash}, nil,
			[]*externalapi.DomainTransaction{spendingTransaction2})
		if err != nil {
			t.Fatalf("Error adding goodBlock: %+v", err)
		}
		//use ResolveBlockStatus, since goodBlock2 might not be the selectedTip
		goodBlock2Status, err := consensus.ConsensusStateManager().ResolveBlockStatus(goodBlock2Hash)
		if err != nil {
			t.Fatalf("Error getting status of goodBlock: %+v", err)
		}
		if goodBlock2Status != externalapi.StatusUTXOValid {
			t.Fatalf("GoodBlock2 status expected to be '%s', but is '%s'", externalapi.StatusUTXOValid, goodBlock2Status)
		}
	})
}

func TestResolveBlockStatusSanity(t *testing.T) {
	testutils.ForAllNets(t, true, func(t *testing.T, params *dagconfig.Params) {
		consensus, teardown, err := consensus.NewFactory().NewTestConsensus(params, "TestResolveBlockStatusSanity")
		if err != nil {
			t.Fatalf("Error setting up consensus: %+v", err)
		}
		defer teardown(false)

		genesisHash := params.GenesisHash
		allHashes := []*externalapi.DomainHash{genesisHash}

		// Make sure that the status of genesisHash is valid
		genesisStatus, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), genesisHash)
		if err != nil {
			t.Fatalf("error getting genesis status: %s", err)
		}
		if genesisStatus != externalapi.StatusUTXOValid {
			t.Fatalf("genesis is unexpectedly non-valid. Its status is: %s", genesisStatus)
		}

		chainLength := int(params.K) + 1

		// Add a chain of blocks over the genesis and make sure all their
		// statuses are valid
		currentHash := genesisHash
		for i := 0; i < chainLength; i++ {
			addedBlockHash, _, err := consensus.AddBlock([]*externalapi.DomainHash{currentHash}, nil, nil)
			if err != nil {
				t.Fatalf("error adding block %d: %s", i, err)
			}
			blockStatus, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), addedBlockHash)
			if err != nil {
				t.Fatalf("error getting block %d (%s) status: %s", i, addedBlockHash, err)
			}
			if blockStatus != externalapi.StatusUTXOValid {
				t.Fatalf("block %d (%s) is unexpectedly non-valid. Its status is: %s", i, addedBlockHash, blockStatus)
			}
			currentHash = addedBlockHash
			allHashes = append(allHashes, addedBlockHash)
		}

		// Add another chain of blocks over the genesis that's shorter than
		// the original chain by 1. Here we expect all the statuses to be
		// StatusUTXOPendingVerification
		currentHash = genesisHash
		for i := 0; i < chainLength-1; i++ {
			addedBlockHash, _, err := consensus.AddBlock([]*externalapi.DomainHash{currentHash}, nil, nil)
			if err != nil {
				t.Fatalf("error adding block %d: %s", i, err)
			}
			blockStatus, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), addedBlockHash)
			if err != nil {
				t.Fatalf("error getting block %d (%s) status: %s", i, addedBlockHash, err)
			}
			if blockStatus != externalapi.StatusUTXOPendingVerification {
				t.Fatalf("block %d (%s) has unexpected status. "+
					"Want: %s, got: %s", i, addedBlockHash, externalapi.StatusUTXOPendingVerification, blockStatus)
			}
			currentHash = addedBlockHash
			allHashes = append(allHashes, addedBlockHash)
		}

		// Add another two blocks to the second chain. This should trigger
		// resolving the entire chain
		for i := 0; i < 2; i++ {
			addedBlockHash, _, err := consensus.AddBlock([]*externalapi.DomainHash{currentHash}, nil, nil)
			if err != nil {
				t.Fatalf("error adding block %d: %s", i, err)
			}
			currentHash = addedBlockHash
			allHashes = append(allHashes, addedBlockHash)
		}

		// Make sure that all the blocks in the DAG now have StatusUTXOValid
		for _, hash := range allHashes {
			blockStatus, err := consensus.BlockStatusStore().Get(consensus.DatabaseContext(), hash)
			if err != nil {
				t.Fatalf("error getting block %s status: %s", hash, err)
			}
			if blockStatus != externalapi.StatusUTXOValid {
				t.Fatalf("block %s is unexpectedly non-valid. Its status is: %s", hash, blockStatus)
			}
		}
	})
}
