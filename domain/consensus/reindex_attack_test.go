package consensus_test

import (
	"fmt"
	"github.com/kaspanet/kaspad/domain/consensus"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/logger"
	"os"
	"testing"
)

// Test configuration
const(
	numBlocksExponent = 10
	logLevel = "warn"
	validateMining = false
)

func TestNoAttack(t *testing.T) {
	logger.SetLogLevels(logLevel)
	params := dagconfig.SimnetParams
	params.SkipProofOfWork = true
	tc, teardown, err := consensus.NewFactory().NewTestConsensus(&params, "TestNoAttack")
	if err != nil {
		t.Fatalf("Error setting up consensus: %+v", err)
	}
	defer teardown(false)

	tc.TestParams().ValidateMining = validateMining

	fileName := fmt.Sprintf(
		"../../testdata/dags/noattack-dag-blocks--2^%d-delay-factor--1-k--18.json",
		numBlocksExponent)
	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	//now := time.Now()
	err = tc.MineJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Printf("passed %ds\n", time.Since(now).Seconds())

	// Do whatever you want with TestConsensus.
}


func TestAttack(t *testing.T) {
	logger.SetLogLevels(logLevel)
	params := dagconfig.SimnetParams
	params.SkipProofOfWork = true
	tc, teardown, err := consensus.NewFactory().NewTestConsensus(&params, "TestAttack")
	if err != nil {
		t.Fatalf("Error setting up consensus: %+v", err)
	}
	defer teardown(false)

	tc.TestParams().ValidateMining = validateMining

	fileName := fmt.Sprintf(
		"../../testdata/dags/attack-dag-blocks--2^%d-delay-factor--1-k--18.json",
		numBlocksExponent)
	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	//now := time.Now()
	err = tc.MineJSON(f)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//1. consensus.NewFactory().NewConsensus(parar)
	//2. tc.DAGTopologyManager().Children(params.GenesisHash)
	//3. heap := tc.DAGTraversalManager().NewDownHeap()
	//4. blocks := make([]*externalapi.DomainBlock)
	//5. con.ValidateAndInsertBlock()



	//fmt.Printf("passed %ds\n", time.Since(now).Seconds())

	// Do whatever you want with TestConsensus.
}