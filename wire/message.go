// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
)

// MaxMessagePayload is the maximum bytes a message can be regardless of other
// individual limits imposed by messages themselves.
const MaxMessagePayload = (1024 * 1024 * 32) // 32MB

// MessageCommand is a number in the header of a message that represents its type.
type MessageCommand uint32

func (cmd MessageCommand) String() string {
	cmdString, ok := messageCommandToString[cmd]
	if !ok {
		cmdString = "unknown command"
	}
	return fmt.Sprintf("%s [code %d]", cmdString, uint8(cmd))
}

// Commands used in kaspa message headers which describe the type of message.
const (
	CmdVersion             MessageCommand = 0
	CmdVerAck              MessageCommand = 1
	CmdRequestAddresses    MessageCommand = 2
	CmdAddresses           MessageCommand = 3
	CmdRequestIBDBlocks    MessageCommand = 4
	CmdBlock               MessageCommand = 8
	CmdTx                  MessageCommand = 9
	CmdPing                MessageCommand = 10
	CmdPong                MessageCommand = 11
	CmdRequestBlockLocator MessageCommand = 18
	CmdBlockLocator        MessageCommand = 19
	CmdSelectedTip         MessageCommand = 20
	CmdRequestSelectedTip  MessageCommand = 21
	CmdInvRelayBlock       MessageCommand = 22
	CmdRequestRelayBlocks  MessageCommand = 23
	CmdInvTransaction      MessageCommand = 25
	CmdRequestTransactions MessageCommand = 26
	CmdIBDBlock            MessageCommand = 27
)

var messageCommandToString = map[MessageCommand]string{
	CmdVersion:             "Version",
	CmdVerAck:              "VerAck",
	CmdRequestAddresses:    "RequestAddresses",
	CmdAddresses:           "Addresses",
	CmdRequestIBDBlocks:    "RequestBlocks",
	CmdBlock:               "Block",
	CmdTx:                  "Tx",
	CmdPing:                "Ping",
	CmdPong:                "Pong",
	CmdRequestBlockLocator: "RequestBlockLocator",
	CmdBlockLocator:        "BlockLocator",
	CmdSelectedTip:         "SelectedTip",
	CmdRequestSelectedTip:  "RequestSelectedTip",
	CmdInvRelayBlock:       "InvRelayBlock",
	CmdRequestRelayBlocks:  "RequestRelayBlocks",
	CmdInvTransaction:      "InvTransaction",
	CmdRequestTransactions: "RequestTransactions",
	CmdIBDBlock:            "IBDBlock",
}

// Message is an interface that describes a kaspa message. A type that
// implements Message has complete control over the representation of its data
// and may therefore contain additional or fewer fields than those which
// are used directly in the protocol encoded message.
type Message interface {
	Command() MessageCommand
}
