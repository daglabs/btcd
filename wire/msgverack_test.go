// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// TestVerAck tests the MsgVerAck API.
func TestVerAck(t *testing.T) {
	pver := ProtocolVersion

	// Ensure the command is expected value.
	wantCmd := "verack"
	msg := NewMsgVerAck()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgVerAck: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value.
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

// TestVerAckWire tests the MsgVerAck wire encode and decode for various
// protocol versions.
func TestVerAckWire(t *testing.T) {
	msgVerAck := NewMsgVerAck()
	msgVerAckEncoded := []byte{}

	tests := []struct {
		in   *MsgVerAck // Message to encode
		out  *MsgVerAck // Expected decoded message
		buf  []byte     // Wire encoding
		pver uint32     // Protocol version for wire encoding
	}{
		// Latest protocol version.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			ProtocolVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
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

		// Decode the message from wire format.
		var msg MsgVerAck
		rbuf := bytes.NewReader(test.buf)
		err = msg.KaspaDecode(rbuf, test.pver)
		if err != nil {
			t.Errorf("KaspaDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("KaspaDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}
