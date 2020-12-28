// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dagconfig

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionhelper"
)

var genesisTxOuts = []*externalapi.DomainTransactionOutput{}

var genesisTxPayload = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Blue score
	0x00, 0x00, //script version
	0x17,                                           // Varint
	0xa9, 0x14, 0xda, 0x17, 0x45, 0xe9, 0xb5, 0x49, // OP-TRUE p2sh
	0xbd, 0x0b, 0xfa, 0x1a, 0x56, 0x99, 0x71, 0xc7,
	0x7e, 0xba, 0x30, 0xcd, 0x5a, 0x4b, 0x87,
}

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network.
var genesisCoinbaseTx = transactionhelper.NewSubnetworkTransaction(0, []*externalapi.DomainTransactionInput{}, genesisTxOuts,
	&subnetworks.SubnetworkIDCoinbase, 0, genesisTxPayload)

// genesisHash is the hash of the first block in the block DAG for the main
// network (genesis block).
var genesisHash = externalapi.DomainHash{
	0x2a, 0xf7, 0x9a, 0xfb, 0x2c, 0xf7, 0xde, 0xe0,
	0xdf, 0xb3, 0x52, 0x4d, 0xbb, 0x3a, 0x83, 0x57,
	0xa6, 0xd2, 0x3e, 0x63, 0x51, 0x48, 0xb1, 0xf8,
	0xe7, 0x8b, 0xc7, 0x30, 0xed, 0x24, 0xe5, 0x80,
}

// genesisMerkleRoot is the hash of the first transaction in the genesis block
// for the main network.
var genesisMerkleRoot = externalapi.DomainHash{
	0xbd, 0x8b, 0x6a, 0x2a, 0xdc, 0x2e, 0x34, 0x5d,
	0x68, 0x54, 0x87, 0xc1, 0x4f, 0xfa, 0xbf, 0x55,
	0xec, 0xb1, 0x49, 0x25, 0xd2, 0x22, 0x98, 0x34,
	0x90, 0x5e, 0xc9, 0xf6, 0xa0, 0x76, 0x37, 0xd0,
}

// genesisBlock defines the genesis block of the block DAG which serves as the
// public transaction ledger for the main network.
var genesisBlock = externalapi.DomainBlock{
	Header: &externalapi.DomainBlockHeader{
		Version:              0,
		ParentHashes:         []*externalapi.DomainHash{},
		HashMerkleRoot:       genesisMerkleRoot,
		AcceptedIDMerkleRoot: externalapi.DomainHash{},
		UTXOCommitment:       externalapi.DomainHash{},
		TimeInMilliseconds:   0x176a95cef33,
		Bits:                 0x207fffff,
		Nonce:                0x0,
	},
	Transactions: []*externalapi.DomainTransaction{genesisCoinbaseTx},
}

var devnetGenesisTxOuts = []*externalapi.DomainTransactionOutput{}

var devnetGenesisTxPayload = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Blue score
	0x17,       // Varint
	0x00, 0x00, // Script version
	0xa9, 0x14, 0xda, 0x17, 0x45, 0xe9, 0xb5, 0x49, // OP-TRUE p2sh
	0xbd, 0x0b, 0xfa, 0x1a, 0x56, 0x99, 0x71, 0xc7,
	0x7e, 0xba, 0x30, 0xcd, 0x5a, 0x4b, 0x87,
	0x6b, 0x61, 0x73, 0x70, 0x61, 0x2d, 0x64, 0x65, 0x76, 0x6e, 0x65, 0x74, // kaspa-devnet
}

// devnetGenesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the development network.
var devnetGenesisCoinbaseTx = transactionhelper.NewSubnetworkTransaction(0,
	[]*externalapi.DomainTransactionInput{}, devnetGenesisTxOuts,
	&subnetworks.SubnetworkIDCoinbase, 0, devnetGenesisTxPayload)

// devGenesisHash is the hash of the first block in the block DAG for the development
// network (genesis block).
var devnetGenesisHash = externalapi.DomainHash{
	0x48, 0x9b, 0x5c, 0x34, 0x66, 0x9a, 0x40, 0xfe,
	0x28, 0xac, 0x21, 0x8b, 0x6c, 0x5c, 0x6f, 0x04,
	0x93, 0x31, 0x06, 0x32, 0xf0, 0xba, 0x35, 0x76,
	0xb8, 0x7a, 0x0c, 0xd7, 0x15, 0x4f, 0x4b, 0x47,
}

// devnetGenesisMerkleRoot is the hash of the first transaction in the genesis block
// for the devopment network.
var devnetGenesisMerkleRoot = externalapi.DomainHash{
	0x8e, 0x8a, 0x7e, 0x66, 0x6d, 0x21, 0x0f, 0x23,
	0xb0, 0xba, 0xcb, 0x0e, 0xef, 0x2c, 0xc4, 0xaf,
	0x07, 0xe3, 0xe8, 0x05, 0xc3, 0xd4, 0x85, 0xa7,
	0x7f, 0xef, 0x6d, 0x4b, 0x73, 0x30, 0xe4, 0xd6,
}

// devnetGenesisBlock defines the genesis block of the block DAG which serves as the
// public transaction ledger for the development network.
var devnetGenesisBlock = externalapi.DomainBlock{
	Header: &externalapi.DomainBlockHeader{
		Version:              0,
		ParentHashes:         []*externalapi.DomainHash{},
		HashMerkleRoot:       devnetGenesisMerkleRoot,
		AcceptedIDMerkleRoot: externalapi.DomainHash{},
		UTXOCommitment:       externalapi.DomainHash{},
		TimeInMilliseconds:   0x176a95cef33,
		Bits:                 0x1e7fffff,
		Nonce:                0x164cc,
	},
	Transactions: []*externalapi.DomainTransaction{devnetGenesisCoinbaseTx},
}

var simnetGenesisTxOuts = []*externalapi.DomainTransactionOutput{}

var simnetGenesisTxPayload = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Blue score
	0x00, 0x00, // Script version
	0x17,                                           // Varint
	0xa9, 0x14, 0xda, 0x17, 0x45, 0xe9, 0xb5, 0x49, // OP-TRUE p2sh
	0xbd, 0x0b, 0xfa, 0x1a, 0x56, 0x99, 0x71, 0xc7,
	0x7e, 0xba, 0x30, 0xcd, 0x5a, 0x4b, 0x87,
	0x6b, 0x61, 0x73, 0x70, 0x61, 0x2d, 0x73, 0x69, 0x6d, 0x6e, 0x65, 0x74, // kaspa-simnet
}

// simnetGenesisCoinbaseTx is the coinbase transaction for the simnet genesis block.
var simnetGenesisCoinbaseTx = transactionhelper.NewSubnetworkTransaction(0,
	[]*externalapi.DomainTransactionInput{}, simnetGenesisTxOuts,
	&subnetworks.SubnetworkIDCoinbase, 0, simnetGenesisTxPayload)

// simnetGenesisHash is the hash of the first block in the block DAG for
// the simnet (genesis block).
var simnetGenesisHash = externalapi.DomainHash{
	0xd3, 0x7d, 0xd2, 0xb9, 0x25, 0xad, 0x16, 0xa3,
	0xe6, 0x3b, 0xc1, 0x8c, 0xac, 0xf6, 0x87, 0x00,
	0xbd, 0x3c, 0x49, 0xd5, 0xac, 0xd7, 0x94, 0xcb,
	0x19, 0xb1, 0x9a, 0xf9, 0xb8, 0x39, 0xb0, 0xc1,
}

// simnetGenesisMerkleRoot is the hash of the first transaction in the genesis block
// for the devopment network.
var simnetGenesisMerkleRoot = externalapi.DomainHash{
	0xd0, 0x94, 0x84, 0xa1, 0x37, 0x01, 0xfc, 0xda,
	0xa7, 0x71, 0x3a, 0x4d, 0x7c, 0x0b, 0xc5, 0x6c,
	0x23, 0x34, 0xb4, 0x93, 0xb8, 0xaf, 0xb3, 0x63,
	0x78, 0x34, 0x25, 0x17, 0xbe, 0x00, 0xf1, 0x80,
}

// simnetGenesisBlock defines the genesis block of the block DAG which serves as the
// public transaction ledger for the development network.
var simnetGenesisBlock = externalapi.DomainBlock{
	Header: &externalapi.DomainBlockHeader{
		Version:              0,
		ParentHashes:         []*externalapi.DomainHash{},
		HashMerkleRoot:       simnetGenesisMerkleRoot,
		AcceptedIDMerkleRoot: externalapi.DomainHash{},
		UTXOCommitment:       externalapi.DomainHash{},
		TimeInMilliseconds:   0x176a95cf016,
		Bits:                 0x207fffff,
		Nonce:                0x0,
	},
	Transactions: []*externalapi.DomainTransaction{simnetGenesisCoinbaseTx},
}

var testnetGenesisTxOuts = []*externalapi.DomainTransactionOutput{}

var testnetGenesisTxPayload = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Blue score
	0x00, 0x00, // Script version
	0x01,                                                                         // Varint
	0x00,                                                                         // OP-FALSE
	0x6b, 0x61, 0x73, 0x70, 0x61, 0x2d, 0x74, 0x65, 0x73, 0x74, 0x6e, 0x65, 0x74, // kaspa-testnet
}

// testnetGenesisCoinbaseTx is the coinbase transaction for the testnet genesis block.
var testnetGenesisCoinbaseTx = transactionhelper.NewSubnetworkTransaction(0,
	[]*externalapi.DomainTransactionInput{}, testnetGenesisTxOuts,
	&subnetworks.SubnetworkIDCoinbase, 0, testnetGenesisTxPayload)

// testnetGenesisHash is the hash of the first block in the block DAG for the test
// network (genesis block).
var testnetGenesisHash = externalapi.DomainHash{
	0xc2, 0x89, 0xab, 0xf0, 0xdf, 0x19, 0x4e, 0x42,
	0x50, 0xa0, 0xce, 0x3b, 0x50, 0x06, 0x3a, 0x35,
	0xd8, 0xeb, 0xa6, 0x9b, 0x3b, 0xd8, 0xda, 0xb3,
	0xa7, 0xce, 0x94, 0x96, 0xb5, 0x26, 0xdf, 0x80,
}

// testnetGenesisMerkleRoot is the hash of the first transaction in the genesis block
// for testnet.
var testnetGenesisMerkleRoot = externalapi.DomainHash{
	0x88, 0xd4, 0x13, 0xd5, 0xdd, 0x4a, 0x70, 0x13,
	0x11, 0x49, 0xfd, 0x89, 0x20, 0x4a, 0x78, 0xb6,
	0x8e, 0x09, 0xc7, 0x4a, 0xac, 0x34, 0x45, 0x08,
	0xd5, 0x99, 0x2d, 0x2f, 0x04, 0x5d, 0x82, 0xad,
}

// testnetGenesisBlock defines the genesis block of the block DAG which serves as the
// public transaction ledger for testnet.
var testnetGenesisBlock = externalapi.DomainBlock{
	Header: &externalapi.DomainBlockHeader{
		Version:              0,
		ParentHashes:         []*externalapi.DomainHash{},
		HashMerkleRoot:       testnetGenesisMerkleRoot,
		AcceptedIDMerkleRoot: externalapi.DomainHash{},
		UTXOCommitment:       externalapi.DomainHash{},
		TimeInMilliseconds:   0x176a95cf016,
		Bits:                 0x1e7fffff,
		Nonce:                0x2c30,
	},
	Transactions: []*externalapi.DomainTransaction{testnetGenesisCoinbaseTx},
}
