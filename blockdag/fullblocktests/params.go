// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fullblocktests

import (
	"encoding/hex"
	"math"
	"math/big"
	"time"

	"github.com/daglabs/btcd/util/hdkeychain"
	"github.com/daglabs/btcd/util/subnetworkid"

	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/util/daghash"
	"github.com/daglabs/btcd/wire"
)

// newHashFromStr converts the passed big-endian hex string into a
// wire.Hash.  It only differs from the one available in daghash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newHashFromStr(hexStr string) *daghash.Hash {
	hash, err := daghash.NewHashFromStr(hexStr)
	if err != nil {
		panic(err)
	}
	return hash
}

// newTxIDFromStr converts the passed big-endian hex string into a
// wire.TxID.  It only differs from the one available in daghash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newTxIDFromStr(hexStr string) *daghash.TxID {
	txID, err := daghash.NewTxIDFromStr(hexStr)
	if err != nil {
		panic(err)
	}
	return txID
}

// fromHex converts the passed hex string into a byte slice and will panic if
// there is an error.  This is only provided for the hard-coded constants so
// errors in the source code can be detected. It will only (and must only) be
// called for initialization purposes.
func fromHex(s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return r
}

var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// regressionPowLimit is the highest proof of work value a Bitcoin block
	// can have for the regression test network.  It is the value 2^255 - 1.
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// regTestGenesisBlock defines the genesis block of the block chain which serves
	// as the public transaction ledger for the regression test network.
	regTestGenesisBlock = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:        1,
			ParentHashes:   []*daghash.Hash{},
			HashMerkleRoot: newHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
			Timestamp:      time.Unix(0x5b28c636, 0), // 2018-06-19 09:00:38 +0000 UTC
			Bits:           0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
			Nonce:          1,
		},
		Transactions: []*wire.MsgTx{{
			Version: 1,
			TxIn: []*wire.TxIn{{
				PreviousOutpoint: wire.Outpoint{
					TxID:  daghash.TxID{},
					Index: 0xffffffff,
				},
				SignatureScript: fromHex("04ffff001d010445" +
					"5468652054696d65732030332f4a616e2f" +
					"32303039204368616e63656c6c6f72206f" +
					"6e206272696e6b206f66207365636f6e64" +
					"206261696c6f757420666f72206261686b73"),
				Sequence: math.MaxUint64,
			}},
			TxOut: []*wire.TxOut{{
				Value: 0,
				PkScript: fromHex("4104678afdb0fe5548271967f1" +
					"a67130b7105cd6a828e03909a67962e0ea1f" +
					"61deb649f6bc3f4cef38c4f35504e51ec138" +
					"c4f35504e51ec112de5c384df7ba0b8d578a" +
					"4c702b6bf11d5fac"),
			}},
			LockTime:     0,
			SubnetworkID: *subnetworkid.SubnetworkIDNative,
		}},
	}
)

// regressionNetParams defines the network parameters for the regression test
// network.
//
// NOTE: The test generator intentionally does not use the existing definitions
// in the dagconfig package since the intent is to be able to generate known
// good tests which exercise that code.  Using the dagconfig parameters would
// allow them to change out from under the tests potentially invalidating them.
var regressionNetParams = &dagconfig.Params{
	Name:        "regtest",
	Net:         wire.TestNet,
	DefaultPort: "18444",

	// DAG parameters
	GenesisBlock:                   &regTestGenesisBlock,
	GenesisHash:                    newHashFromStr("5bec7567af40504e0994db3b573c186fffcc4edefe096ff2e58d00523bd7e8a6"),
	PowMax:                         regressionPowLimit,
	PowMaxBits:                     0x207fffff,
	BlockCoinbaseMaturity:          100,
	SubsidyReductionInterval:       150,
	TargetTimePerBlock:             time.Second * 10, // 10 seconds
	DifficultyAdjustmentWindowSize: 2640,
	TimestampDeviationTolerance:    132,
	GenerateSupported:              true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Mempool parameters
	RelayNonStdTxs: true,

	// Address encoding magics
	PrivateKeyID: 0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDKeyIDPair: hdkeychain.HDKeyPairRegressionNet,

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1,
}
