// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domainmessage

import (
	"testing"

	"github.com/kaspanet/kaspad/util/daghash"
)

// TestRequstIBDBlocks tests the MsgRequestIBDBlocks API.
func TestRequstIBDBlocks(t *testing.T) {
	hashStr := "000000000002e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	lowHash, err := daghash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	hashStr = "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	highHash, err := daghash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Ensure we get the same data back out.
	msg := NewMsgRequstIBDBlocks(lowHash, highHash)
	if !msg.HighHash.IsEqual(highHash) {
		t.Errorf("NewMsgRequstIBDBlocks: wrong high hash - got %v, want %v",
			msg.HighHash, highHash)
	}

	// Ensure the command is expected value.
	wantCmd := MessageCommand(4)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgRequstIBDBlocks: wrong command - got %v want %v",
			cmd, wantCmd)
	}
}
