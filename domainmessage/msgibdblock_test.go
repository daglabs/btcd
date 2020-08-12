// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domainmessage

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// TestIBDBlock tests the MsgIBDBlock API.
func TestIBDBlock(t *testing.T) {
	pver := ProtocolVersion

	// Block 1 header.
	parentHashes := blockOne.Header.ParentHashes
	hashMerkleRoot := blockOne.Header.HashMerkleRoot
	acceptedIDMerkleRoot := blockOne.Header.AcceptedIDMerkleRoot
	utxoCommitment := blockOne.Header.UTXOCommitment
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	bh := NewBlockHeader(1, parentHashes, hashMerkleRoot, acceptedIDMerkleRoot, utxoCommitment, bits, nonce)

	// Ensure the command is expected value.
	wantCmd := MessageCommand(17)
	msg := NewMsgIBDBlock(NewMsgBlock(bh))
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgIBDBlock: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	wantPayload := uint32(1024 * 1024 * 32)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure we get the same block header data back out.
	if !reflect.DeepEqual(&msg.Header, bh) {
		t.Errorf("NewMsgIBDBlock: wrong block header - got %v, want %v",
			spew.Sdump(&msg.Header), spew.Sdump(bh))
	}

	// Ensure transactions are added properly.
	tx := blockOne.Transactions[0].Copy()
	msg.AddTransaction(tx)
	if !reflect.DeepEqual(msg.Transactions, blockOne.Transactions) {
		t.Errorf("AddTransaction: wrong transactions - got %v, want %v",
			spew.Sdump(msg.Transactions),
			spew.Sdump(blockOne.Transactions))
	}

	// Ensure transactions are properly cleared.
	msg.ClearTransactions()
	if len(msg.Transactions) != 0 {
		t.Errorf("ClearTransactions: wrong transactions - got %v, want %v",
			len(msg.Transactions), 0)
	}
}

// TestIBDBlockEncoding tests the MsgIBDBlock domainmessage encode and decode for various numbers
// of transaction inputs and outputs and protocol versions.
func TestIBDBlockEncoding(t *testing.T) {
	tests := []struct {
		in     *MsgIBDBlock // Message to encode
		out    *MsgIBDBlock // Expected decoded message
		buf    []byte       // Encoded value
		txLocs []TxLoc      // Expected transaction locations
		pver   uint32       // Protocol version for domainmessage encoding
	}{
		// Latest protocol version.
		{
			&MsgIBDBlock{MsgBlock: &blockOne},
			&MsgIBDBlock{MsgBlock: &blockOne},
			blockOneBytes,
			blockOneTxLocs,
			ProtocolVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to domainmessage format.
		var buf bytes.Buffer
		err := test.in.KaspaEncode(&buf, test.pver)
		if err != nil {
			t.Errorf("KaspaEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("KaspaEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from domainmessage format.
		var msg MsgIBDBlock
		msg.MsgBlock = new(MsgBlock)
		rbuf := bytes.NewReader(test.buf)
		err = msg.KaspaDecode(rbuf, test.pver)
		if err != nil {
			t.Errorf("KaspaDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("KaspaDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&msg), spew.Sdump(test.out))
			continue
		}
	}
}
