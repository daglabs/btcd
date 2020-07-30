package wire

import (
	"testing"

	"github.com/kaspanet/kaspad/util/daghash"
)

// TestGetBlockLocator tests the MsgRequestBlockLocator API.
func TestGetBlockLocator(t *testing.T) {
	hashStr := "000000000002e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	highHash, err := daghash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Ensure the command is expected value.
	wantCmd := MessageCommand(9)
	msg := NewMsgGetBlockLocator(highHash, &daghash.ZeroHash)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetBlockLocator: wrong command - got %v want %v",
			cmd, wantCmd)
	}
}
