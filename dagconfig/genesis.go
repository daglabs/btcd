// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dagconfig

import (
	"math"
	"time"

	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/wire"
)

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network, regression test network, and test network (version 3).
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				TxID:  daghash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x00, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
				0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
			},
			Sequence: math.MaxUint64,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
				0x51,
			},
		},
	},
	LockTime:     0,
	SubNetworkID: wire.SubNetworkDAGCoin,
}

// genesisHash is the hash of the first block in the block chain for the main
// network (genesis block).
var genesisHash = daghash.Hash([daghash.HashSize]byte{ // Make go vet happy.
	0x53, 0xb8, 0xf9, 0x4b, 0xec, 0x3f, 0xae, 0x0a,
	0x7c, 0x79, 0x7a, 0x8c, 0x87, 0xfb, 0x4c, 0x37,
	0xff, 0x68, 0xed, 0xdb, 0x4a, 0x96, 0xd6, 0xbd,
	0x36, 0xf0, 0x28, 0x93, 0xe7, 0x09, 0xc3, 0xcc,
})

// genesisMerkleRoot is the hash of the first transaction in the genesis block
// for the main network.
var genesisMerkleRoot = daghash.Hash([daghash.HashSize]byte{ // Make go vet happy.
	0x76, 0x2b, 0x33, 0xa9, 0x4c, 0xd4, 0x36, 0x13,
	0x29, 0x5e, 0x9b, 0x68, 0xb7, 0xad, 0x2b, 0x16,
	0x7c, 0x63, 0x89, 0xc3, 0x54, 0xc9, 0xa7, 0x06,
	0x8c, 0x23, 0x24, 0x3c, 0x53, 0x6d, 0x56, 0x23,
})

// genesisBlock defines the genesis block of the block chain which serves as the
// public transaction ledger for the main network.
var genesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:        1,
		ParentHashes:   []daghash.Hash{},
		HashMerkleRoot: genesisMerkleRoot,
		IDMerkleRoot:   genesisMerkleRoot,
		Timestamp:      time.Unix(0x5c3cafec, 0),
		Bits:           0x207fffff,
		Nonce:          0xbffffffffffffffa,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// regTestGenesisHash is the hash of the first block in the block chain for the
// regression test network (genesis block).
var regTestGenesisHash = genesisHash

// regTestGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the regression test network.  It is the same as the merkle root for
// the main network.
var regTestGenesisMerkleRoot = genesisMerkleRoot

// regTestGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the regression test network.
var regTestGenesisBlock = genesisBlock

// testNet3GenesisHash is the hash of the first block in the block chain for the
// test network (version 3).
var testNet3GenesisHash = genesisHash

// testNet3GenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the test network (version 3).  It is the same as the merkle root
// for the main network.
var testNet3GenesisMerkleRoot = genesisMerkleRoot

// testNet3GenesisBlock defines the genesis block of the block chain which
// serves as the public transaction ledger for the test network (version 3).
var testNet3GenesisBlock = genesisBlock

// simNetGenesisHash is the hash of the first block in the block chain for the
// simulation test network.
var simNetGenesisHash = genesisHash

// simNetGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the simulation test network.  It is the same as the merkle root for
// the main network.
var simNetGenesisMerkleRoot = genesisMerkleRoot

// simNetGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the simulation test network.
var simNetGenesisBlock = genesisBlock
