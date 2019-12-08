// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/daglabs/kaspad/blockdag"
	"github.com/daglabs/kaspad/dagconfig"
	"github.com/daglabs/kaspad/util"
	"github.com/daglabs/kaspad/util/daghash"
	"github.com/daglabs/kaspad/wire"
)

// solveGenesisBlock attempts to find some combination of a nonce and
// current timestamp which makes the passed block hash to a value less than the
// target difficulty.
func solveGenesisBlock(block *wire.MsgBlock, powBits uint32, netName string) {
	// Create some convenience variables.
	header := &block.Header
	targetDifficulty := util.CompactToBig(header.Bits)
	header.Bits = powBits

	// Search through the entire nonce range for a solution while
	// periodically checking for early quit and stale block
	// conditions along with updates to the speed monitor.
	maxNonce := ^uint64(0) // 2^64 - 1
	for {
		header.Timestamp = time.Unix(time.Now().Unix(), 0)
		for i := uint64(0); i <= maxNonce; i++ {
			// Update the nonce and hash the block header.  Each
			// hash is actually a double sha256 (two hashes), so
			// increment the number of hashes completed for each
			// attempt accordingly.
			header.Nonce = i
			hash := header.BlockHash()

			// The block is solved when the new block hash is less
			// than the target difficulty.  Yay!
			if daghash.HashToBig(hash).Cmp(targetDifficulty) <= 0 {
				fmt.Printf("\n\nGenesis block of %s is solved:\n", netName)
				fmt.Printf("timestamp: 0x%x\n", header.Timestamp.Unix())
				fmt.Printf("bits (difficulty): 0x%x\n", header.Bits)
				fmt.Printf("nonce: 0x%x\n", header.Nonce)
				fmt.Printf("hash: %v\n\n\n", hex.EncodeToString(hash[:]))
				return
			}
		}
	}
}

func validateGenesisBlock(genesisBlock *wire.MsgBlock, netName string) bool {
	block := util.NewBlock(genesisBlock)
	hashMerkleTree := blockdag.BuildHashMerkleTreeStore(block.Transactions())
	calculatedHashMerkleRoot := hashMerkleTree.Root()
	header := genesisBlock.Header
	if !header.HashMerkleRoot.IsEqual(calculatedHashMerkleRoot) {
		fmt.Printf("%s: genesis block hash merkle root is invalid - block "+
			"header indicates %s, but calculated value is %s\n\n",
			netName, hex.EncodeToString(header.HashMerkleRoot[:]),
			hex.EncodeToString(calculatedHashMerkleRoot[:]))
		return false
	}
	return true
}

func validateAndSolve(genesisBlock *wire.MsgBlock, powBits uint32, netName string) {
	// Validate merkle root
	if validateGenesisBlock(genesisBlock, netName) {
		// Solve genesis block
		solveGenesisBlock(genesisBlock, powBits, netName)
	}
}

// main
func main() {
	bigOne := big.NewInt(1)

	validateAndSolve(dagconfig.MainNetParams.GenesisBlock,
		util.BigToCompact(new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)),
		"MainNet")

	validateAndSolve(dagconfig.DevNetParams.GenesisBlock,
		util.BigToCompact(new(big.Int).Sub(new(big.Int).Lsh(bigOne, 239), bigOne)),
		"DevNet")
}
