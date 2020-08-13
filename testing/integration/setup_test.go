package integration

import (
	"path/filepath"
	"testing"

	"github.com/kaspanet/kaspad/app"
	"github.com/kaspanet/kaspad/infrastructure/config"
	"github.com/kaspanet/kaspad/infrastructure/dbaccess"
)

type appHarness struct {
	app                     *app.App
	rpcClient               *rpcClient
	p2pAddress              string
	rpcAddress              string
	miningAddress           string
	miningAddressPrivateKey string
	config                  *config.Config
	databaseContext         *dbaccess.DatabaseContext
}

type harnessParams struct {
	p2pAddress              string
	rpcAddress              string
	miningAddress           string
	miningAddressPrivateKey string
}

// setupHarness creates a single appHarness with given parameters
func setupHarness(t *testing.T, params *harnessParams) (harness *appHarness, teardownFunc func()) {
	harness = &appHarness{
		p2pAddress:              params.p2pAddress,
		rpcAddress:              params.rpcAddress,
		miningAddress:           params.miningAddress,
		miningAddressPrivateKey: params.miningAddressPrivateKey,
	}

	setConfig(t, harness)
	setDatabaseContext(t, harness)
	setApp(t, harness)
	harness.app.Start()
	setRPCClient(t, harness)

	return harness, func() {
		teardownHarness(t, harness)
	}
}

// setupHarnesses creates multiple appHarnesses, according to number of parameters passed
func setupHarnesses(t *testing.T, harnessesParams []*harnessParams) (harnesses []*appHarness, teardownFunc func()) {
	var teardowns []func()
	for _, params := range harnessesParams {
		harness, teardownFunc := setupHarness(t, params)
		harnesses = append(harnesses, harness)
		teardowns = append(teardowns, teardownFunc)
	}

	return harnesses, func() {
		for _, teardownFunc := range teardowns {
			teardownFunc()
		}
	}
}

// standardSetup creates a standard setup of 3 appHarnesses that should work for most tests
func standardSetup(t *testing.T) (appHarness1, appHarness2, appHarness3 *appHarness, teardownFunc func()) {
	harnesses, teardown := setupHarnesses(t, []*harnessParams{
		{
			p2pAddress:              p2pAddress1,
			rpcAddress:              rpcAddress1,
			miningAddress:           miningAddress1,
			miningAddressPrivateKey: miningAddress1PrivateKey,
		},
		{
			p2pAddress:              p2pAddress2,
			rpcAddress:              rpcAddress2,
			miningAddress:           miningAddress2,
			miningAddressPrivateKey: miningAddress2PrivateKey,
		}, {
			p2pAddress:              p2pAddress3,
			rpcAddress:              rpcAddress3,
			miningAddress:           miningAddress3,
			miningAddressPrivateKey: miningAddress3PrivateKey,
		},
	})

	return harnesses[0], harnesses[1], harnesses[2], teardown
}

func setRPCClient(t *testing.T, harness *appHarness) {
	var err error
	harness.rpcClient, err = newRPCClient(harness.rpcAddress)
	if err != nil {
		t.Fatalf("Error getting RPC client %+v", err)
	}
}

func teardownHarness(t *testing.T, harness *appHarness) {
	harness.app.Stop()

	err := harness.databaseContext.Close()
	if err != nil {
		t.Errorf("Error closing database context: %+v", err)
	}
}

func setApp(t *testing.T, harness *appHarness) {
	var err error
	harness.app, err = app.New(harness.config, harness.databaseContext, make(chan struct{}))
	if err != nil {
		t.Fatalf("Error creating app: %+v", err)
	}
}

func setDatabaseContext(t *testing.T, harness *appHarness) {
	var err error
	harness.databaseContext, err = openDB(harness.config)
	if err != nil {
		t.Fatalf("Error openning database: %+v", err)
	}
}

func openDB(cfg *config.Config) (*dbaccess.DatabaseContext, error) {
	dbPath := filepath.Join(cfg.DataDir, "db")
	return dbaccess.New(dbPath)
}
