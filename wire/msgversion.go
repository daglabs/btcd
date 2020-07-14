// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"fmt"
	"github.com/kaspanet/kaspad/netadapter/id"
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/kaspanet/kaspad/version"
	"github.com/pkg/errors"
	"io"
	"strings"

	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
)

// MaxUserAgentLen is the maximum allowed length for the user agent field in a
// version message (MsgVersion).
const MaxUserAgentLen = 256

// DefaultUserAgent for wire in the stack
var DefaultUserAgent = fmt.Sprintf("/kaspad:%s/", version.Version())

// MsgVersion implements the Message interface and represents a kaspa version
// message. It is used for a peer to advertise itself as soon as an outbound
// connection is made. The remote peer then uses this information along with
// its own to negotiate. The remote peer must then respond with a version
// message of its own containing the negotiated values followed by a verack
// message (MsgVerAck). This exchange must take place before any further
// communication is allowed to proceed.
type MsgVersion struct {
	// Version of the protocol the node is using.
	ProtocolVersion uint32

	// Bitfield which identifies the enabled services.
	Services ServiceFlag

	// Time the message was generated. This is encoded as an int64 on the wire.
	Timestamp mstime.Time

	// Address of the local peer.
	Address *NetAddress

	// The peer unique ID
	ID *id.ID

	// The user agent that generated messsage. This is a encoded as a varString
	// on the wire. This has a max length of MaxUserAgentLen.
	UserAgent string

	// The selected tip hash of the generator of the version message.
	SelectedTipHash *daghash.Hash

	// Don't announce transactions to peer.
	DisableRelayTx bool

	// The subnetwork of the generator of the version message. Should be nil in full nodes
	SubnetworkID *subnetworkid.SubnetworkID
}

// HasService returns whether the specified service is supported by the peer
// that generated the message.
func (msg *MsgVersion) HasService(service ServiceFlag) bool {
	return msg.Services&service == service
}

// AddService adds service as a supported service by the peer generating the
// message.
func (msg *MsgVersion) AddService(service ServiceFlag) {
	msg.Services |= service
}

// KaspaDecode decodes r using the kaspa protocol encoding into the receiver.
// The version message is special in that the protocol version hasn't been
// negotiated yet. As a result, the pver field is ignored and any fields which
// are added in new versions are optional. This also mean that r must be a
// *bytes.Buffer so the number of remaining bytes can be ascertained.
//
// This is part of the Message interface implementation.
func (msg *MsgVersion) KaspaDecode(r io.Reader, pver uint32) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return errors.Errorf("MsgVersion.KaspaDecode reader is not a " +
			"*bytes.Buffer")
	}

	err := readElements(buf, &msg.ProtocolVersion, &msg.Services,
		(*int64Time)(&msg.Timestamp))
	if err != nil {
		return err
	}

	// Read subnetwork ID
	var isFullNode bool
	err = ReadElement(r, &isFullNode)
	if err != nil {
		return err
	}
	if isFullNode {
		msg.SubnetworkID = nil
	} else {
		var subnetworkID subnetworkid.SubnetworkID
		err = ReadElement(r, &subnetworkID)
		if err != nil {
			return err
		}
		msg.SubnetworkID = &subnetworkID
	}

	var hasAddress bool
	err = ReadElement(r, &hasAddress)
	if err != nil {
		return err
	}

	if hasAddress {
		msg.Address = new(NetAddress)
		err = readNetAddress(buf, pver, msg.Address, false)
		if err != nil {
			return err
		}
	}

	msg.ID = new(id.ID)
	err = ReadElement(buf, msg.ID)
	if err != nil {
		return err
	}
	userAgent, err := ReadVarString(buf, pver)
	if err != nil {
		return err
	}
	err = validateUserAgent(userAgent)
	if err != nil {
		return err
	}
	msg.UserAgent = userAgent

	msg.SelectedTipHash = &daghash.Hash{}
	err = ReadElement(buf, msg.SelectedTipHash)
	if err != nil {
		return err
	}

	var relayTx bool
	err = ReadElement(r, &relayTx)
	if err != nil {
		return err
	}
	msg.DisableRelayTx = !relayTx

	return nil
}

// KaspaEncode encodes the receiver to w using the kaspa protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgVersion) KaspaEncode(w io.Writer, pver uint32) error {
	err := validateUserAgent(msg.UserAgent)
	if err != nil {
		return err
	}

	err = writeElements(w, msg.ProtocolVersion, msg.Services,
		msg.Timestamp.UnixMilliseconds())
	if err != nil {
		return err
	}

	// Write subnetwork ID
	isFullNode := msg.SubnetworkID == nil
	err = WriteElement(w, isFullNode)
	if err != nil {
		return err
	}
	if !isFullNode {
		err = WriteElement(w, msg.SubnetworkID)
		if err != nil {
			return err
		}
	}

	if msg.Address != nil {
		err = WriteElement(w, true)
		if err != nil {
			return err
		}

		err = writeNetAddress(w, pver, msg.Address, false)
		if err != nil {
			return err
		}
	}

	err = WriteElement(w, msg.ID)
	if err != nil {
		return err
	}

	err = WriteVarString(w, msg.UserAgent)
	if err != nil {
		return err
	}

	err = WriteElement(w, msg.SelectedTipHash)
	if err != nil {
		return err
	}

	// The wire encoding for the field is true when transactions should be
	// relayed, so reverse it from the DisableRelayTx field.
	err = WriteElement(w, !msg.DisableRelayTx)
	if err != nil {
		return err
	}
	return nil
}

// Command returns the protocol command string for the message. This is part
// of the Message interface implementation.
func (msg *MsgVersion) Command() string {
	return CmdVersion
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver. This is part of the Message interface implementation.
func (msg *MsgVersion) MaxPayloadLength(pver uint32) uint32 {
	// Protocol version 4 bytes + services 8 bytes + timestamp 16 bytes +
	// remote and local net addresses + nonce 8 bytes + length of user
	// agent (varInt) + max allowed useragent length + selected tip hash length +
	// relay transactions flag 1 byte.
	return 29 + (maxNetAddressPayload(pver) * 2) + MaxVarIntPayload +
		MaxUserAgentLen + daghash.HashSize
}

// NewMsgVersion returns a new kaspa version message that conforms to the
// Message interface using the passed parameters and defaults for the remaining
// fields.
func NewMsgVersion(addr *NetAddress, id *id.ID,
	selectedTipHash *daghash.Hash, subnetworkID *subnetworkid.SubnetworkID) *MsgVersion {

	// Limit the timestamp to one millisecond precision since the protocol
	// doesn't support better.
	return &MsgVersion{
		ProtocolVersion: ProtocolVersion,
		Services:        0,
		Timestamp:       mstime.Now(),
		Address:         addr,
		ID:              id,
		UserAgent:       DefaultUserAgent,
		SelectedTipHash: selectedTipHash,
		DisableRelayTx:  false,
		SubnetworkID:    subnetworkID,
	}
}

// validateUserAgent checks userAgent length against MaxUserAgentLen
func validateUserAgent(userAgent string) error {
	if len(userAgent) > MaxUserAgentLen {
		str := fmt.Sprintf("user agent too long [len %d, max %d]",
			len(userAgent), MaxUserAgentLen)
		return messageError("MsgVersion", str)
	}
	return nil
}

// AddUserAgent adds a user agent to the user agent string for the version
// message. The version string is not defined to any strict format, although
// it is recommended to use the form "major.minor.revision" e.g. "2.6.41".
func (msg *MsgVersion) AddUserAgent(name string, version string,
	comments ...string) {

	newUserAgent := fmt.Sprintf("%s:%s", name, version)
	if len(comments) != 0 {
		newUserAgent = fmt.Sprintf("%s(%s)", newUserAgent,
			strings.Join(comments, "; "))
	}
	newUserAgent = fmt.Sprintf("%s%s/", msg.UserAgent, newUserAgent)
	msg.UserAgent = newUserAgent
}
