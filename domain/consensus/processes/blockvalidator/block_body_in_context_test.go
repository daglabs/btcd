package blockvalidator_test

import (
	"testing"

	"github.com/kaspanet/kaspad/domain/consensus/model"

	"github.com/kaspanet/kaspad/domain/consensus"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/consensus/utils/testutils"
	"github.com/pkg/errors"
)

func TestCheckBlockIsNotPruned(t *testing.T) {
	testutils.ForAllNets(t, true, func(t *testing.T, consensusConfig *consensus.Config) {
		// This is done to reduce the pruning depth to 6 blocks
		consensusConfig.FinalityDuration = 2 * consensusConfig.TargetTimePerBlock
		consensusConfig.K = 0

		factory := consensus.NewFactory()

		tc, teardown, err := factory.NewTestConsensus(consensusConfig, "TestCheckBlockIsNotPruned")
		if err != nil {
			t.Fatalf("Error setting up consensus: %+v", err)
		}
		defer teardown(false)

		// Add blocks until the pruning point changes
		tipHash := consensusConfig.GenesisHash
		tipHash, _, err = tc.AddBlock([]*externalapi.DomainHash{tipHash}, nil, nil)
		if err != nil {
			t.Fatalf("AddBlock: %+v", err)
		}

		beforePruningBlock, err := tc.GetBlock(tipHash)
		if err != nil {
			t.Fatalf("beforePruningBlock: %+v", err)
		}

		for {
			tipHash, _, err = tc.AddBlock([]*externalapi.DomainHash{tipHash}, nil, nil)
			if err != nil {
				t.Fatalf("AddBlock: %+v", err)
			}

			pruningPoint, err := tc.PruningPoint()
			if err != nil {
				t.Fatalf("PruningPoint: %+v", err)
			}

			if !pruningPoint.Equal(consensusConfig.GenesisHash) {
				break
			}
		}

		_, err = tc.ValidateAndInsertBlock(beforePruningBlock)
		if !errors.Is(err, ruleerrors.ErrPrunedBlock) {
			t.Fatalf("Unexpected error: %+v", err)
		}

		beforePruningBlockBlockStatus, err := tc.BlockStatusStore().Get(tc.DatabaseContext(), model.NewStagingArea(),
			consensushashing.BlockHash(beforePruningBlock))
		if err != nil {
			t.Fatalf("BlockStatusStore().Get: %+v", err)
		}

		// Check that the block still has header only status although it got rejected.
		if beforePruningBlockBlockStatus != externalapi.StatusHeaderOnly {
			t.Fatalf("Unexpected status %s", beforePruningBlockBlockStatus)
		}
	})
}

func TestCheckParentBlockBodiesExist(t *testing.T) {
	testutils.ForAllNets(t, true, func(t *testing.T, consensusConfig *consensus.Config) {
		// This is done to reduce the pruning depth to 6 blocks
		consensusConfig.FinalityDuration = 2 * consensusConfig.TargetTimePerBlock
		consensusConfig.K = 0

		factory := consensus.NewFactory()

		tc, teardown, err := factory.NewTestConsensus(consensusConfig, "TestCheckParentBlockBodiesExist")
		if err != nil {
			t.Fatalf("Error setting up consensus: %+v", err)
		}
		defer teardown(false)

		headerHash, _, err := tc.AddUTXOInvalidHeader([]*externalapi.DomainHash{consensusConfig.GenesisHash})
		if err != nil {
			t.Fatalf("AddUTXOInvalidHeader: %+v", err)
		}

		_, _, err = tc.AddUTXOInvalidBlock([]*externalapi.DomainHash{headerHash})
		errMissingParents := &ruleerrors.ErrMissingParents{}
		if !errors.As(err, errMissingParents) {
			t.Fatalf("Unexpected error: %+v", err)
		}

		if !externalapi.HashesEqual(errMissingParents.MissingParentHashes, []*externalapi.DomainHash{headerHash}) {
			t.Fatalf("unexpected missing parents %s", errMissingParents.MissingParentHashes)
		}

		// Add blocks until the pruning point changes
		tipHash := consensusConfig.GenesisHash
		anticonePruningBlock, err := tc.BuildUTXOInvalidBlock([]*externalapi.DomainHash{tipHash})
		if err != nil {
			t.Fatalf("BuildUTXOInvalidBlock: %+v", err)
		}

		// Add only the header of anticonePruningBlock
		_, err = tc.ValidateAndInsertBlock(&externalapi.DomainBlock{
			Header:       anticonePruningBlock.Header,
			Transactions: nil,
		})
		if err != nil {
			t.Fatalf("ValidateAndInsertBlock: %+v", err)
		}

		for {
			tipHash, _, err = tc.AddBlock([]*externalapi.DomainHash{tipHash}, nil, nil)
			if err != nil {
				t.Fatalf("AddUTXOInvalidHeader: %+v", err)
			}

			pruningPoint, err := tc.PruningPoint()
			if err != nil {
				t.Fatalf("PruningPoint: %+v", err)
			}

			if !pruningPoint.Equal(consensusConfig.GenesisHash) {
				break
			}
		}

		// Add anticonePruningBlock's body and Check that it's valid to point to
		// a header only block in the past of the pruning point.
		_, err = tc.ValidateAndInsertBlock(anticonePruningBlock)
		if err != nil {
			t.Fatalf("ValidateAndInsertBlock: %+v", err)
		}
	})
}

func TestIsFinalizedTransaction(t *testing.T) {
	testutils.ForAllNets(t, true, func(t *testing.T, consensusConfig *consensus.Config) {
		stagingArea := model.NewStagingArea()

		consensusConfig.BlockCoinbaseMaturity = 0
		factory := consensus.NewFactory()

		tc, teardown, err := factory.NewTestConsensus(consensusConfig, "TestIsFinalizedTransaction")
		if err != nil {
			t.Fatalf("Error setting up consensus: %+v", err)
		}
		defer teardown(false)

		// Build a small DAG
		outerParents := []*externalapi.DomainHash{consensusConfig.GenesisHash}
		for i := 0; i < 5; i++ {
			var innerParents []*externalapi.DomainHash
			for i := 0; i < 4; i++ {
				blockHash, _, err := tc.AddBlock(outerParents, nil, nil)
				if err != nil {
					t.Fatalf("AddBlock: %+v", err)
				}
				innerParents = append(innerParents, blockHash)
			}
			outerParents = []*externalapi.DomainHash{}
			for i := 0; i < 3; i++ {
				blockHash, _, err := tc.AddBlock(innerParents, nil, nil)
				if err != nil {
					t.Fatalf("AddBlock: %+v", err)
				}
				outerParents = append(outerParents, blockHash)
			}
		}

		block, err := tc.BuildBlock(
			&externalapi.DomainCoinbaseData{&externalapi.ScriptPublicKey{}, nil}, nil)
		if err != nil {
			t.Fatalf("Error getting block: %+v", err)
		}
		_, err = tc.ValidateAndInsertBlock(block)
		if err != nil {
			t.Fatalf("Error Inserting block: %+v", err)
		}
		blockGhostDAG, err := tc.GHOSTDAGDataStore().Get(tc.DatabaseContext(), stagingArea, consensushashing.BlockHash(block))
		if err != nil {
			t.Fatalf("Error getting GhostDAG Data: %+v", err)
		}
		blockParents := block.Header.ParentHashes()
		parentToSpend, err := tc.GetBlock(blockParents[0])
		if err != nil {
			t.Fatalf("Error getting block1: %+v", err)
		}

		checkForLockTimeAndSequence := func(lockTime, sequence uint64, shouldPass bool) {
			tx, err := testutils.CreateTransaction(parentToSpend.Transactions[0], 1)
			if err != nil {
				t.Fatalf("Error creating tx: %+v", err)
			}

			tx.LockTime = lockTime
			tx.Inputs[0].Sequence = sequence

			_, _, err = tc.AddBlock(blockParents, nil, []*externalapi.DomainTransaction{tx})
			if (shouldPass && err != nil) || (!shouldPass && !errors.Is(err, ruleerrors.ErrUnfinalizedTx)) {
				t.Fatalf("shouldPass: %t Unexpected error: %+v", shouldPass, err)
			}
		}

		// Check that the same blueScore or higher fails, but lower passes.
		checkForLockTimeAndSequence(blockGhostDAG.BlueScore()+1, 0, false)
		checkForLockTimeAndSequence(blockGhostDAG.BlueScore(), 0, false)
		checkForLockTimeAndSequence(blockGhostDAG.BlueScore()-1, 0, true)

		pastMedianTime, err := tc.PastMedianTimeManager().PastMedianTime(stagingArea, consensushashing.BlockHash(block))
		if err != nil {
			t.Fatalf("PastMedianTime: %+v", err)
		}
		// Check that the same pastMedianTime or higher fails, but lower passes.
		checkForLockTimeAndSequence(uint64(pastMedianTime)+1, 0, false)
		checkForLockTimeAndSequence(uint64(pastMedianTime), 0, false)
		checkForLockTimeAndSequence(uint64(pastMedianTime)-1, 0, true)

		// We check that if the transaction is marked as finalized it'll pass for any lock time.
		checkForLockTimeAndSequence(uint64(pastMedianTime), constants.MaxTxInSequenceNum, true)
		checkForLockTimeAndSequence(2, constants.MaxTxInSequenceNum, true)
	})
}
