package dagtraversalmanager_test

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/testapi"
	"reflect"
	"sort"
	"testing"

	"github.com/kaspanet/kaspad/domain/consensus"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/hashset"
	"github.com/kaspanet/kaspad/domain/consensus/utils/testutils"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/pkg/errors"
)

func TestBlueBlockWindow(t *testing.T) {
	tests := map[string][]*struct {
		parents                          []string
		id                               string //id is a virtual entity that is used only for tests so we can define relations between blocks without knowing their hash
		expectedWindowWithGenesisPadding []string
	}{
		"kaspa-mainnet": {
			{
				parents:                          []string{"A"},
				id:                               "B",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "C",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "D",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "E",
				expectedWindowWithGenesisPadding: []string{"C", "D", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "F",
				expectedWindowWithGenesisPadding: []string{"C", "D", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"A"},
				id:                               "G",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"G"},
				id:                               "H",
				expectedWindowWithGenesisPadding: []string{"G", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"H", "F"},
				id:                               "I",
				expectedWindowWithGenesisPadding: []string{"F", "C", "H", "D", "B", "G", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"I"},
				id:                               "J",
				expectedWindowWithGenesisPadding: []string{"I", "F", "C", "H", "D", "B", "G", "A", "A", "A"},
			},
			{
				parents:                          []string{"J"},
				id:                               "K",
				expectedWindowWithGenesisPadding: []string{"J", "I", "F", "C", "H", "D", "B", "G", "A", "A"},
			},
			{
				parents:                          []string{"K"},
				id:                               "L",
				expectedWindowWithGenesisPadding: []string{"K", "J", "I", "F", "C", "H", "D", "B", "G", "A"},
			},
			{
				parents:                          []string{"L"},
				id:                               "M",
				expectedWindowWithGenesisPadding: []string{"L", "K", "J", "I", "F", "C", "H", "D", "B", "G"},
			},
			{
				parents:                          []string{"M"},
				id:                               "N",
				expectedWindowWithGenesisPadding: []string{"M", "L", "K", "J", "I", "F", "C", "H", "D", "B"},
			},
			{
				parents:                          []string{"N"},
				id:                               "O",
				expectedWindowWithGenesisPadding: []string{"N", "M", "L", "K", "J", "I", "F", "C", "H", "D"},
			},
		},
		"kaspa-testnet": {
			{
				parents:                          []string{"A"},
				id:                               "B",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "C",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "D",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "E",
				expectedWindowWithGenesisPadding: []string{"D", "C", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "F",
				expectedWindowWithGenesisPadding: []string{"D", "C", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"A"},
				id:                               "G",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"G"},
				id:                               "H",
				expectedWindowWithGenesisPadding: []string{"G", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"H", "F"},
				id:                               "I",
				expectedWindowWithGenesisPadding: []string{"F", "D", "H", "C", "B", "G", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"I"},
				id:                               "J",
				expectedWindowWithGenesisPadding: []string{"I", "F", "D", "H", "C", "B", "G", "A", "A", "A"},
			},
			{
				parents:                          []string{"J"},
				id:                               "K",
				expectedWindowWithGenesisPadding: []string{"J", "I", "F", "D", "H", "C", "B", "G", "A", "A"},
			},
			{
				parents:                          []string{"K"},
				id:                               "L",
				expectedWindowWithGenesisPadding: []string{"K", "J", "I", "F", "D", "H", "C", "B", "G", "A"},
			},
			{
				parents:                          []string{"L"},
				id:                               "M",
				expectedWindowWithGenesisPadding: []string{"L", "K", "J", "I", "F", "D", "H", "C", "B", "G"},
			},
			{
				parents:                          []string{"M"},
				id:                               "N",
				expectedWindowWithGenesisPadding: []string{"M", "L", "K", "J", "I", "F", "D", "H", "C", "B"},
			},
			{
				parents:                          []string{"N"},
				id:                               "O",
				expectedWindowWithGenesisPadding: []string{"N", "M", "L", "K", "J", "I", "F", "D", "H", "C"},
			},
		},
		"kaspa-devnet": {
			{
				parents:                          []string{"A"},
				id:                               "B",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "C",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "D",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"C", "D"},
				id:                               "E",
				expectedWindowWithGenesisPadding: []string{"C", "D", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"C", "D"},
				id:                               "F",
				expectedWindowWithGenesisPadding: []string{"C", "D", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"A"},
				id:                               "G",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"G"},
				id:                               "H",
				expectedWindowWithGenesisPadding: []string{"G", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"H", "F"},
				id:                               "I",
				expectedWindowWithGenesisPadding: []string{"F", "C", "D", "H", "G", "B", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"I"},
				id:                               "J",
				expectedWindowWithGenesisPadding: []string{"I", "F", "C", "D", "H", "G", "B", "A", "A", "A"},
			},
			{
				parents:                          []string{"J"},
				id:                               "K",
				expectedWindowWithGenesisPadding: []string{"J", "I", "F", "C", "D", "H", "G", "B", "A", "A"},
			},
			{
				parents:                          []string{"K"},
				id:                               "L",
				expectedWindowWithGenesisPadding: []string{"K", "J", "I", "F", "C", "D", "H", "G", "B", "A"},
			},
			{
				parents:                          []string{"L"},
				id:                               "M",
				expectedWindowWithGenesisPadding: []string{"L", "K", "J", "I", "F", "C", "D", "H", "G", "B"},
			},
			{
				parents:                          []string{"M"},
				id:                               "N",
				expectedWindowWithGenesisPadding: []string{"M", "L", "K", "J", "I", "F", "C", "D", "H", "G"},
			},
			{
				parents:                          []string{"N"},
				id:                               "O",
				expectedWindowWithGenesisPadding: []string{"N", "M", "L", "K", "J", "I", "F", "C", "D", "H"},
			},
		},
		"kaspa-simnet": {
			{
				parents:                          []string{"A"},
				id:                               "B",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "C",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"B"},
				id:                               "D",
				expectedWindowWithGenesisPadding: []string{"B", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "E",
				expectedWindowWithGenesisPadding: []string{"D", "C", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"D", "C"},
				id:                               "F",
				expectedWindowWithGenesisPadding: []string{"D", "C", "B", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"A"},
				id:                               "G",
				expectedWindowWithGenesisPadding: []string{"A", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"G"},
				id:                               "H",
				expectedWindowWithGenesisPadding: []string{"G", "A", "A", "A", "A", "A", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"H", "F"},
				id:                               "I",
				expectedWindowWithGenesisPadding: []string{"F", "D", "C", "H", "B", "G", "A", "A", "A", "A"},
			},
			{
				parents:                          []string{"I"},
				id:                               "J",
				expectedWindowWithGenesisPadding: []string{"I", "F", "D", "C", "H", "B", "G", "A", "A", "A"},
			},
			{
				parents:                          []string{"J"},
				id:                               "K",
				expectedWindowWithGenesisPadding: []string{"J", "I", "F", "D", "C", "H", "B", "G", "A", "A"},
			},
			{
				parents:                          []string{"K"},
				id:                               "L",
				expectedWindowWithGenesisPadding: []string{"K", "J", "I", "F", "D", "C", "H", "B", "G", "A"},
			},
			{
				parents:                          []string{"L"},
				id:                               "M",
				expectedWindowWithGenesisPadding: []string{"L", "K", "J", "I", "F", "D", "C", "H", "B", "G"},
			},
			{
				parents:                          []string{"M"},
				id:                               "N",
				expectedWindowWithGenesisPadding: []string{"M", "L", "K", "J", "I", "F", "D", "C", "H", "B"},
			},
			{
				parents:                          []string{"N"},
				id:                               "O",
				expectedWindowWithGenesisPadding: []string{"N", "M", "L", "K", "J", "I", "F", "D", "C", "H"},
			},
		},
	}
	testutils.ForAllNets(t, true, func(t *testing.T, params *dagconfig.Params) {
		params.K = 1
		factory := consensus.NewFactory()
		tc, tearDown, err := factory.NewTestConsensus(params, "TestBlueBlockWindow")
		if err != nil {
			t.Fatalf("NewTestConsensus: %s", err)
		}
		defer tearDown(false)

		windowSize := 10
		blockByIDMap := make(map[string]*externalapi.DomainHash)
		idByBlockMap := make(map[externalapi.DomainHash]string)
		blockByIDMap["A"] = params.GenesisHash
		idByBlockMap[*params.GenesisHash] = "A"

		blocksData := tests[params.Name]

		for _, blockData := range blocksData {
			parents := hashset.New()
			for _, parentID := range blockData.parents {
				parent := blockByIDMap[parentID]
				parents.Add(parent)
			}

			block, _, err := tc.AddBlock(parents.ToSlice(), nil, nil)
			if err != nil {
				t.Fatalf("AddBlock: %+v", err)
			}

			blockByIDMap[blockData.id] = block
			idByBlockMap[*block] = blockData.id

			window, err := tc.DAGTraversalManager().BlueWindow(block, windowSize)
			if err != nil {
				t.Fatalf("BlueWindow: %s", err)
			}
			sortWindow(t, tc, window)
			if err := checkWindowIDs(window, blockData.expectedWindowWithGenesisPadding, idByBlockMap); err != nil {
				t.Errorf("Unexpected values for window for block %s: %s", blockData.id, err)
			}
		}
	})
}

func sortWindow(t *testing.T, tc testapi.TestConsensus, window []*externalapi.DomainHash) {
	sort.Slice(window, func(i, j int) bool {
		ghostdagDataI, err := tc.GHOSTDAGDataStore().Get(tc.DatabaseContext(), window[i])
		if err != nil {
			t.Fatalf("Failed getting ghostdag data for %s", err)
		}
		ghostdagDataJ, err := tc.GHOSTDAGDataStore().Get(tc.DatabaseContext(), window[j])
		if err != nil {
			t.Fatalf("Failed getting ghostdag data for %s", err)
		}
		return !tc.GHOSTDAGManager().Less(window[i], ghostdagDataI, window[j], ghostdagDataJ)
	})
}

func checkWindowIDs(window []*externalapi.DomainHash, expectedIDs []string, idByBlockMap map[externalapi.DomainHash]string) error {
	ids := make([]string, len(window))
	for i, node := range window {
		ids[i] = idByBlockMap[*node]
	}
	if !reflect.DeepEqual(ids, expectedIDs) {
		return errors.Errorf("window expected to have blocks %s but got %s", expectedIDs, ids)
	}
	return nil
}
