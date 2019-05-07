// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bloom_test

import (
	"bytes"
	"testing"

	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/util/bloom"
	"github.com/daglabs/btcd/wire"
	"github.com/davecgh/go-spew/spew"
)

func TestMerkleBlock3(t *testing.T) {
	blockBytes := []byte{
		0x01, 0x00, 0x00, 0x00, // Version
		0x01,                                                             // NumParentBlocks
		0x79, 0xCD, 0xA8, 0x56, 0xB1, 0x43, 0xD9, 0xDB, 0x2C, 0x1C, 0xAF, // ParentHashes
		0xF0, 0x1D, 0x1A, 0xEC, 0xC8, 0x63, 0x0D, 0x30, 0x62, 0x5D, 0x10,
		0xE8, 0xB4, 0xB8, 0xB0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xB5, 0x0C, 0xC0, 0x69, 0xD6, 0xA3, 0xE3, 0x3E, 0x3F, 0xF8, 0x4A, // HashMerkleRoot
		0x5C, 0x41, 0xD9, 0xD3, 0xFE, 0xBE, 0x7C, 0x77, 0x0F, 0xDC, 0xC9,
		0x6B, 0x2C, 0x3F, 0xF6, 0x0A, 0xBE, 0x18, 0x4F, 0x19, 0x63,
		0x7F, 0x16, 0xC5, 0x96, 0x2E, 0x8B, 0xD9, 0x63, 0x65, 0x9C, 0x79, // Fake IDMerkleRoot. TODO: (Ori) Replace to a real IDMerkleRoot
		0x3C, 0xE3, 0x70, 0xD9, 0x5F, 0x09, 0x3B, 0xC7, 0xE3, 0x67, 0x11,
		0x7B, 0x3C, 0x30, 0xC1, 0xF8, 0xFD, 0xD0, 0xD9, 0x72, 0x87,
		0x3C, 0xE3, 0x70, 0xD9, 0x5F, 0x09, 0x3B, 0xC7, 0xE3, 0x67, 0x11, // AcceptedIDMerkleRoot
		0x7F, 0x16, 0xC5, 0x96, 0x2E, 0x8B, 0xD9, 0x63, 0x65, 0x9C, 0x79,
		0x7B, 0x3C, 0x30, 0xC1, 0xF8, 0xFD, 0xD0, 0xD9, 0x72, 0x87,
		0x67, 0x29, 0x1B, 0x4D, 0x00, 0x00, 0x00, 0x00, //Time
		0x4C, 0x86, 0x04, 0x1B, // Bits
		0x8F, 0xA4, 0x5D, 0x63, 0x00, 0x00, 0x00, 0x00, // Fake Nonce. TODO: (Ori) Replace to a real nonce
		0x01,                                                             // NumTxs
		0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Tx
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x08, 0x04, 0x4C,
		0x86, 0x04, 0x1B, 0x02, 0x0A, 0x02, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01,
		0x00, 0xF2, 0x05, 0x2A, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04,
		0xEC, 0xD3, 0x22, 0x9B, 0x05, 0x71, 0xC3, 0xBE, 0x87, 0x6F, 0xEA,
		0xAC, 0x04, 0x42, 0xA9, 0xF1, 0x3C, 0x5A, 0x57, 0x27, 0x42, 0x92,
		0x7A, 0xF1, 0xDC, 0x62, 0x33, 0x53, 0xEC, 0xF8, 0xC2, 0x02, 0x22,
		0x5F, 0x64, 0x86, 0x81, 0x37, 0xA1, 0x8C, 0xDD, 0x85, 0xCB, 0xBB,
		0x4C, 0x74, 0xFB, 0xCC, 0xFD, 0x4F, 0x49, 0x63, 0x9C, 0xF1, 0xBD,
		0xC9, 0x4A, 0x56, 0x72, 0xBB, 0x15, 0xAD, 0x5D, 0x4C, 0xAC, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
	}

	blk, err := util.NewBlockFromBytes(blockBytes)
	if err != nil {
		t.Errorf("TestMerkleBlock3 NewBlockFromBytes failed: %v", err)
		return
	}

	f := bloom.NewFilter(10, 0, 0.000001, wire.BloomUpdateAll)

	inputStr := "4ee77df1e2c3126a4a3469e7b1ee3c73093f7f79fef726690fde230c47a02dc6"
	hash, err := daghash.NewHashFromStr(inputStr)
	if err != nil {
		t.Errorf("TestMerkleBlock3 NewHashFromStr failed: %v", err)
		return
	}

	f.AddHash(hash)

	mBlock, _ := bloom.NewMerkleBlock(blk, f)

	want := []byte{
		0x01, 0x00, 0x00, 0x00, 0x01, 0x79, 0xcd, 0xa8,
		0x56, 0xb1, 0x43, 0xd9, 0xdb, 0x2c, 0x1c, 0xaf,
		0xf0, 0x1d, 0x1a, 0xec, 0xc8, 0x63, 0x0d, 0x30,
		0x62, 0x5d, 0x10, 0xe8, 0xb4, 0xb8, 0xb0, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xb5, 0x0c, 0xc0,
		0x69, 0xd6, 0xa3, 0xe3, 0x3e, 0x3f, 0xf8, 0x4a,
		0x5c, 0x41, 0xd9, 0xd3, 0xfe, 0xbe, 0x7c, 0x77,
		0x0f, 0xdc, 0xc9, 0x6b, 0x2c, 0x3f, 0xf6, 0x0a,
		0xbe, 0x18, 0x4f, 0x19, 0x63, 0x7f, 0x16, 0xc5,
		0x96, 0x2e, 0x8b, 0xd9, 0x63, 0x65, 0x9c, 0x79,
		0x3c, 0xe3, 0x70, 0xd9, 0x5f, 0x09, 0x3b, 0xc7,
		0xe3, 0x67, 0x11, 0x7b, 0x3c, 0x30, 0xc1, 0xf8,
		0xfd, 0xd0, 0xd9, 0x72, 0x87, 0x3c, 0xe3, 0x70,
		0xd9, 0x5f, 0x09, 0x3b, 0xc7, 0xe3, 0x67, 0x11,
		0x7f, 0x16, 0xc5, 0x96, 0x2e, 0x8b, 0xd9, 0x63,
		0x65, 0x9c, 0x79, 0x7b, 0x3c, 0x30, 0xc1, 0xf8,
		0xfd, 0xd0, 0xd9, 0x72, 0x87, 0x67, 0x29, 0x1b,
		0x4d, 0x00, 0x00, 0x00, 0x00, 0x4c, 0x86, 0x04,
		0x1b, 0x8f, 0xa4, 0x5d, 0x63, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0xe2, 0x03,
		0xc3, 0xca, 0x45, 0x44, 0x5e, 0xcd, 0xbc, 0x90,
		0xdf, 0x49, 0xa2, 0x91, 0x41, 0xdf, 0x31, 0x92,
		0xee, 0xdb, 0x9d, 0x42, 0x33, 0xf5, 0x22, 0xff,
		0x26, 0xca, 0x44, 0xde, 0x3e, 0x9e, 0x01, 0x00,
	}
	t.Log(spew.Sdump(want))
	if err != nil {
		t.Errorf("TestMerkleBlock3 DecodeString failed: %v", err)
		return
	}

	got := bytes.NewBuffer(nil)
	err = mBlock.BtcEncode(got, wire.ProtocolVersion)
	if err != nil {
		t.Errorf("TestMerkleBlock3 BtcEncode failed: %v", err)
		return
	}

	if !bytes.Equal(want, got.Bytes()) {
		t.Errorf("TestMerkleBlock3 failed merkle block comparison: "+
			"got:\n %v want:\n %v", spew.Sdump(got.Bytes()), spew.Sdump(want))
		return
	}
}
