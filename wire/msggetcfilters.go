// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/daglabs/kaspad/util/daghash"
)

// MaxGetCFiltersReqRange the maximum number of filters that may be requested in
// a getcfheaders message.
const MaxGetCFiltersReqRange = 1000

// MsgGetCFilters implements the Message interface and represents a bitcoin
// getcfilters message. It is used to request committed filters for a range of
// blocks.
type MsgGetCFilters struct {
	FilterType  FilterType
	StartHeight uint64
	StopHash    *daghash.Hash
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetCFilters) BtcDecode(r io.Reader, pver uint32) error {
	err := ReadElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

	err = ReadElement(r, &msg.StartHeight)
	if err != nil {
		return err
	}

	msg.StopHash = &daghash.Hash{}
	return ReadElement(r, msg.StopHash)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetCFilters) BtcEncode(w io.Writer, pver uint32) error {
	err := WriteElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	err = WriteElement(w, &msg.StartHeight)
	if err != nil {
		return err
	}

	return WriteElement(w, msg.StopHash)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetCFilters) Command() string {
	return CmdGetCFilters
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetCFilters) MaxPayloadLength(pver uint32) uint32 {
	// Filter type + uint64 + block hash
	return 1 + 8 + daghash.HashSize
}

// NewMsgGetCFilters returns a new bitcoin getcfilters message that conforms to
// the Message interface using the passed parameters and defaults for the
// remaining fields.
func NewMsgGetCFilters(filterType FilterType, startHeight uint64,
	stopHash *daghash.Hash) *MsgGetCFilters {
	return &MsgGetCFilters{
		FilterType:  filterType,
		StartHeight: startHeight,
		StopHash:    stopHash,
	}
}
