// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dagconfig

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/util/network"
	"math/big"
	"time"

	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
)

// These variables are the DAG proof-of-work limit parameters for each default
// network.
var (
	// bigOne is 1 represented as a big.Int. It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// mainPowMax is the highest proof of work value a Kaspa block can
	// have for the main network. It is the value 2^255 - 1.
	mainPowMax = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// testnetPowMax is the highest proof of work value a Kaspa block
	// can have for the test network. It is the value 2^239 - 1.
	testnetPowMax = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 239), bigOne)

	// simnetPowMax is the highest proof of work value a Kaspa block
	// can have for the simulation test network. It is the value 2^255 - 1.
	simnetPowMax = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// devnetPowMax is the highest proof of work value a Kaspa block
	// can have for the development network. It is the value
	// 2^239 - 1.
	devnetPowMax = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 239), bigOne)
)

const (
	ghostdagK                      = 18
	difficultyAdjustmentWindowSize = 2640
	timestampDeviationTolerance    = 132
	finalityDuration               = 24 * time.Hour
	targetTimePerBlock             = 1 * time.Second
)

// KType defines the size of GHOSTDAG consensus algorithm K parameter.
type KType uint8

// Params defines a Kaspa network by its parameters. These parameters may be
// used by Kaspa applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// K defines the K parameter for GHOSTDAG consensus algorithm.
	// See ghostdag.go for further details.
	K KType

	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net appmessage.KaspaNet

	// RPCPort defines the rpc server port
	RPCPort string

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []string

	// GenesisBlock defines the first block of the DAG.
	GenesisBlock *appmessage.MsgBlock

	// GenesisHash is the starting block hash.
	GenesisHash *daghash.Hash

	// PowMax defines the highest allowed proof of work value for a block
	// as a uint256.
	PowMax *big.Int

	// BlockCoinbaseMaturity is the number of blocks required before newly mined
	// coins can be spent.
	BlockCoinbaseMaturity uint64

	// SubsidyReductionInterval is the interval of blocks before the subsidy
	// is reduced.
	SubsidyReductionInterval uint64

	// TargetTimePerBlock is the desired amount of time to generate each
	// block.
	TargetTimePerBlock time.Duration

	// FinalityDuration is the duration of the finality window.
	FinalityDuration time.Duration

	// TimestampDeviationTolerance is the maximum offset a block timestamp
	// is allowed to be in the future before it gets delayed
	TimestampDeviationTolerance uint64

	// DifficultyAdjustmentWindowSize is the size of window that is inspected
	// to calculate the required difficulty of each block.
	DifficultyAdjustmentWindowSize uint64

	// These fields are related to voting on consensus rule changes as
	// defined by BIP0009.
	//
	// RuleChangeActivationThreshold is the number of blocks in a threshold
	// state retarget window for which a positive vote for a rule change
	// must be cast in order to lock in a rule change. It should typically
	// be 95% for the main network and 75% for test networks.
	//
	// MinerConfirmationWindow is the number of blocks in each threshold
	// state retarget window.
	//
	// Deployments define the specific consensus rule changes to be voted
	// on.
	RuleChangeActivationThreshold uint64
	MinerConfirmationWindow       uint64

	// Mempool parameters
	RelayNonStdTxs bool

	// AcceptUnroutable specifies whether this network accepts unroutable
	// IP addresses, such as 10.0.0.0/8
	AcceptUnroutable bool

	// Human-readable prefix for Bech32 encoded addresses
	Prefix util.Bech32Prefix

	// Address encoding magics
	PrivateKeyID byte // First byte of a WIF private key

	// EnableNonNativeSubnetworks enables non-native/coinbase transactions
	EnableNonNativeSubnetworks bool

	// DisableDifficultyAdjustment determine whether to use difficulty
	DisableDifficultyAdjustment bool
}

// NormalizeRPCServerAddress returns addr with the current network default
// port appended if there is not already a port specified.
func (p *Params) NormalizeRPCServerAddress(addr string) (string, error) {
	return network.NormalizeAddress(addr, p.RPCPort)
}

// MainnetParams defines the network parameters for the main Kaspa network.
var MainnetParams = Params{
	K:           ghostdagK,
	Name:        "kaspa-mainnet",
	Net:         appmessage.Mainnet,
	RPCPort:     "16110",
	DefaultPort: "16111",
	DNSSeeds:    []string{"dnsseed.kas.pa"},

	// DAG parameters
	GenesisBlock:                   &genesisBlock,
	GenesisHash:                    &genesisHash,
	PowMax:                         mainPowMax,
	BlockCoinbaseMaturity:          100,
	SubsidyReductionInterval:       210000,
	TargetTimePerBlock:             targetTimePerBlock,
	FinalityDuration:               finalityDuration,
	DifficultyAdjustmentWindowSize: difficultyAdjustmentWindowSize,
	TimestampDeviationTolerance:    timestampDeviationTolerance,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1916, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016, //

	// Mempool parameters
	RelayNonStdTxs: false,

	// AcceptUnroutable specifies whether this network accepts unroutable
	// IP addresses, such as 10.0.0.0/8
	AcceptUnroutable: false,

	// Human-readable part for Bech32 encoded addresses
	Prefix: util.Bech32PrefixKaspa,

	// Address encoding magics
	PrivateKeyID: 0x80, // starts with 5 (uncompressed) or K (compressed)

	// EnableNonNativeSubnetworks enables non-native/coinbase transactions
	EnableNonNativeSubnetworks: false,

	DisableDifficultyAdjustment: false,
}

// TestnetParams defines the network parameters for the test Kaspa network.
var TestnetParams = Params{
	K:           ghostdagK,
	Name:        "kaspa-testnet",
	Net:         appmessage.Testnet,
	RPCPort:     "16210",
	DefaultPort: "16211",
	DNSSeeds:    []string{"testnet-dnsseed.kas.pa"},

	// DAG parameters
	GenesisBlock:                   &testnetGenesisBlock,
	GenesisHash:                    &testnetGenesisHash,
	PowMax:                         testnetPowMax,
	BlockCoinbaseMaturity:          100,
	SubsidyReductionInterval:       210000,
	TargetTimePerBlock:             targetTimePerBlock,
	FinalityDuration:               finalityDuration,
	DifficultyAdjustmentWindowSize: difficultyAdjustmentWindowSize,
	TimestampDeviationTolerance:    timestampDeviationTolerance,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,

	// Mempool parameters
	RelayNonStdTxs: true,

	// AcceptUnroutable specifies whether this network accepts unroutable
	// IP addresses, such as 10.0.0.0/8
	AcceptUnroutable: false,

	// Human-readable part for Bech32 encoded addresses
	Prefix: util.Bech32PrefixKaspaTest,

	// Address encoding magics
	PrivateKeyID: 0xef, // starts with 9 (uncompressed) or c (compressed)

	// EnableNonNativeSubnetworks enables non-native/coinbase transactions
	EnableNonNativeSubnetworks: false,

	DisableDifficultyAdjustment: false,
}

// SimnetParams defines the network parameters for the simulation test Kaspa
// network. This network is similar to the normal test network except it is
// intended for private use within a group of individuals doing simulation
// testing. The functionality is intended to differ in that the only nodes
// which are specifically specified are used to create the network rather than
// following normal discovery rules. This is important as otherwise it would
// just turn into another public testnet.
var SimnetParams = Params{
	K:           ghostdagK,
	Name:        "kaspa-simnet",
	Net:         appmessage.Simnet,
	RPCPort:     "16510",
	DefaultPort: "16511",
	DNSSeeds:    []string{}, // NOTE: There must NOT be any seeds.

	// DAG parameters
	GenesisBlock:                   &simnetGenesisBlock,
	GenesisHash:                    &simnetGenesisHash,
	PowMax:                         simnetPowMax,
	BlockCoinbaseMaturity:          100,
	SubsidyReductionInterval:       210000,
	TargetTimePerBlock:             time.Millisecond,
	FinalityDuration:               time.Minute,
	DifficultyAdjustmentWindowSize: difficultyAdjustmentWindowSize,
	TimestampDeviationTolerance:    timestampDeviationTolerance,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 75, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       100,

	// Mempool parameters
	RelayNonStdTxs: true,

	// AcceptUnroutable specifies whether this network accepts unroutable
	// IP addresses, such as 10.0.0.0/8
	AcceptUnroutable: false,

	PrivateKeyID: 0x64, // starts with 4 (uncompressed) or F (compressed)
	// Human-readable part for Bech32 encoded addresses
	Prefix: util.Bech32PrefixKaspaSim,

	// EnableNonNativeSubnetworks enables non-native/coinbase transactions
	EnableNonNativeSubnetworks: false,

	DisableDifficultyAdjustment: true,
}

// DevnetParams defines the network parameters for the development Kaspa network.
var DevnetParams = Params{
	K:           ghostdagK,
	Name:        "kaspa-devnet",
	Net:         appmessage.Devnet,
	RPCPort:     "16610",
	DefaultPort: "16611",
	DNSSeeds:    []string{}, // NOTE: There must NOT be any seeds.

	// DAG parameters
	GenesisBlock:                   &devnetGenesisBlock,
	GenesisHash:                    &devnetGenesisHash,
	PowMax:                         devnetPowMax,
	BlockCoinbaseMaturity:          100,
	SubsidyReductionInterval:       210000,
	TargetTimePerBlock:             targetTimePerBlock,
	FinalityDuration:               finalityDuration,
	DifficultyAdjustmentWindowSize: difficultyAdjustmentWindowSize,
	TimestampDeviationTolerance:    timestampDeviationTolerance,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,

	// Mempool parameters
	RelayNonStdTxs: true,

	// AcceptUnroutable specifies whether this network accepts unroutable
	// IP addresses, such as 10.0.0.0/8
	AcceptUnroutable: true,

	// Human-readable part for Bech32 encoded addresses
	Prefix: util.Bech32PrefixKaspaDev,

	// Address encoding magics
	PrivateKeyID: 0xef, // starts with 9 (uncompressed) or c (compressed)

	// EnableNonNativeSubnetworks enables non-native/coinbase transactions
	EnableNonNativeSubnetworks: false,

	DisableDifficultyAdjustment: false,
}

var (
	// ErrDuplicateNet describes an error where the parameters for a Kaspa
	// network could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate Kaspa network")
)

var (
	registeredNets = make(map[appmessage.KaspaNet]struct{})
)

// Register registers the network parameters for a Kaspa network. This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible. Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}

	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error. This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainnetParams)
	mustRegister(&TestnetParams)
	mustRegister(&SimnetParams)
}
