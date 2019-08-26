// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"errors"
	"fmt"

	"github.com/daglabs/btcd/util/bech32"
	"golang.org/x/crypto/ripemd160"
)

var (
	// ErrChecksumMismatch describes an error where decoding failed due
	// to a bad checksum.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrUnknownAddressType describes an error where an address can not
	// decoded as a specific address type due to the string encoding
	// begining with an identifier byte unknown to any standard or
	// registered (via chaincfg.Register) network.
	ErrUnknownAddressType = errors.New("unknown address type")
)

const (
	// PubKeyHash addresses always have the version byte set to 0.
	pubKeyHashAddrID = 0x00

	// ScriptHash addresses always have the version byte set to 8.
	scriptHashAddrID = 0x08
)

// Bech32Prefix is the human-readable prefix for a Bech32 address.
type Bech32Prefix int

// Constants that define Bech32 address prefixes. Every network is assigned
// a unique prefix.
const (
	// Unknown/Erroneous prefix
	Bech32PrefixUnknown Bech32Prefix = iota

	// Prefix for the main network.
	Bech32PrefixDAGCoin

	// Prefix for the regression test network.
	Bech32PrefixDAGReg

	// Prefix for the test network.
	Bech32PrefixDAGTest

	// Prefix for the simulation network.
	Bech32PrefixDAGSim
)

// Map from strings to Bech32 address prefix constants for parsing purposes.
var stringsToBech32Prefixes = map[string]Bech32Prefix{
	"dagcoin": Bech32PrefixDAGCoin,
	"dagreg":  Bech32PrefixDAGReg,
	"dagtest": Bech32PrefixDAGTest,
	"dagsim":  Bech32PrefixDAGSim,
}

// ParsePrefix attempts to parse a Bech32 address prefix.
func ParsePrefix(prefixString string) (Bech32Prefix, error) {
	prefix, ok := stringsToBech32Prefixes[prefixString]
	if !ok {
		return Bech32PrefixUnknown, fmt.Errorf("could not parse prefix %s", prefixString)
	}

	return prefix, nil
}

// Converts from Bech32 address prefixes to their string values
func (prefix Bech32Prefix) String() string {
	for key, value := range stringsToBech32Prefixes {
		if prefix == value {
			return key
		}
	}

	return ""
}

// encodeAddress returns a human-readable payment address given a network prefix
// and a ripemd160 hash which encodes the bitcoin network and address type.  It is used
// in both pay-to-pubkey-hash (P2PKH) and pay-to-script-hash (P2SH) address
// encoding.
func encodeAddress(prefix Bech32Prefix, hash160 []byte, version byte) string {
	return bech32.Encode(prefix.String(), hash160[:ripemd160.Size], version)
}

// Address is an interface type for any type of destination a transaction
// output may spend to.  This includes pay-to-pubkey (P2PK), pay-to-pubkey-hash
// (P2PKH), and pay-to-script-hash (P2SH).  Address is designed to be generic
// enough that other kinds of addresses may be added in the future without
// changing the decoding and encoding API.
type Address interface {
	// String returns the string encoding of the transaction output
	// destination.
	//
	// Please note that String differs subtly from EncodeAddress: String
	// will return the value as a string without any conversion, while
	// EncodeAddress may convert destination types (for example,
	// converting pubkeys to P2PKH addresses) before encoding as a
	// payment address string.
	String() string

	// EncodeAddress returns the string encoding of the payment address
	// associated with the Address value.  See the comment on String
	// for how this method differs from String.
	EncodeAddress() string

	// ScriptAddress returns the raw bytes of the address to be used
	// when inserting the address into a txout's script.
	ScriptAddress() []byte

	// IsForPrefix returns whether or not the address is associated with the
	// passed bitcoin network.
	IsForPrefix(prefix Bech32Prefix) bool
}

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The bitcoin network address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string, defaultPrefix Bech32Prefix) (Address, error) {
	prefixString, decoded, version, err := bech32.Decode(addr)
	if err != nil {
		return nil, fmt.Errorf("decoded address is of unknown format: %s", err)
	}

	prefix, err := ParsePrefix(prefixString)
	if err != nil {
		return nil, fmt.Errorf("decoded address's prefix could not be parsed: %s", err)
	}
	if defaultPrefix != prefix {
		return nil, fmt.Errorf("decoded address is of wrong network: %s", err)
	}

	// Switch on decoded length to determine the type.
	switch len(decoded) {
	case ripemd160.Size: // P2PKH or P2SH
		switch version {
		case pubKeyHashAddrID:
			return newAddressPubKeyHash(defaultPrefix, decoded)
		case scriptHashAddrID:
			return newAddressScriptHashFromHash(defaultPrefix, decoded)
		default:
			return nil, ErrUnknownAddressType
		}
	default:
		return nil, errors.New("decoded address is of unknown size")
	}
}

// AddressPubKeyHash is an Address for a pay-to-pubkey-hash (P2PKH)
// transaction.
type AddressPubKeyHash struct {
	prefix Bech32Prefix
	hash   [ripemd160.Size]byte
}

// NewAddressPubKeyHashFromPublicKey return a new AddressPubKeyHash from given public key
func NewAddressPubKeyHashFromPublicKey(publicKey []byte, prefix Bech32Prefix) (*AddressPubKeyHash, error) {
	pkHash := Hash160(publicKey)
	return newAddressPubKeyHash(prefix, pkHash)
}

// NewAddressPubKeyHash returns a new AddressPubKeyHash.  pkHash mustbe 20
// bytes.
func NewAddressPubKeyHash(pkHash []byte, prefix Bech32Prefix) (*AddressPubKeyHash, error) {
	return newAddressPubKeyHash(prefix, pkHash)
}

// newAddressPubKeyHash is the internal API to create a pubkey hash address
// with a known leading identifier byte for a network, rather than looking
// it up through its parameters.  This is useful when creating a new address
// structure from a string encoding where the identifer byte is already
// known.
func newAddressPubKeyHash(prefix Bech32Prefix, pkHash []byte) (*AddressPubKeyHash, error) {
	// Check for a valid pubkey hash length.
	if len(pkHash) != ripemd160.Size {
		return nil, errors.New("pkHash must be 20 bytes")
	}

	addr := &AddressPubKeyHash{prefix: prefix}
	copy(addr.hash[:], pkHash)
	return addr, nil
}

// EncodeAddress returns the string encoding of a pay-to-pubkey-hash
// address.  Part of the Address interface.
func (a *AddressPubKeyHash) EncodeAddress() string {
	return encodeAddress(a.prefix, a.hash[:], pubKeyHashAddrID)
}

// ScriptAddress returns the bytes to be included in a txout script to pay
// to a pubkey hash.  Part of the Address interface.
func (a *AddressPubKeyHash) ScriptAddress() []byte {
	return a.hash[:]
}

// IsForPrefix returns whether or not the pay-to-pubkey-hash address is associated
// with the passed bitcoin network.
func (a *AddressPubKeyHash) IsForPrefix(prefix Bech32Prefix) bool {
	return a.prefix == prefix
}

// String returns a human-readable string for the pay-to-pubkey-hash address.
// This is equivalent to calling EncodeAddress, but is provided so the type can
// be used as a fmt.Stringer.
func (a *AddressPubKeyHash) String() string {
	return a.EncodeAddress()
}

// Hash160 returns the underlying array of the pubkey hash.  This can be useful
// when an array is more appropiate than a slice (for example, when used as map
// keys).
func (a *AddressPubKeyHash) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}

// AddressScriptHash is an Address for a pay-to-script-hash (P2SH)
// transaction.
type AddressScriptHash struct {
	prefix Bech32Prefix
	hash   [ripemd160.Size]byte
}

// NewAddressScriptHash returns a new AddressScriptHash.
func NewAddressScriptHash(serializedScript []byte, prefix Bech32Prefix) (*AddressScriptHash, error) {
	scriptHash := Hash160(serializedScript)
	return newAddressScriptHashFromHash(prefix, scriptHash)
}

// NewAddressScriptHashFromHash returns a new AddressScriptHash.  scriptHash
// must be 20 bytes.
func NewAddressScriptHashFromHash(scriptHash []byte, prefix Bech32Prefix) (*AddressScriptHash, error) {
	return newAddressScriptHashFromHash(prefix, scriptHash)
}

// newAddressScriptHashFromHash is the internal API to create a script hash
// address with a known leading identifier byte for a network, rather than
// looking it up through its parameters.  This is useful when creating a new
// address structure from a string encoding where the identifer byte is already
// known.
func newAddressScriptHashFromHash(prefix Bech32Prefix, scriptHash []byte) (*AddressScriptHash, error) {
	// Check for a valid script hash length.
	if len(scriptHash) != ripemd160.Size {
		return nil, errors.New("scriptHash must be 20 bytes")
	}

	addr := &AddressScriptHash{prefix: prefix}
	copy(addr.hash[:], scriptHash)
	return addr, nil
}

// EncodeAddress returns the string encoding of a pay-to-script-hash
// address.  Part of the Address interface.
func (a *AddressScriptHash) EncodeAddress() string {
	return encodeAddress(a.prefix, a.hash[:], scriptHashAddrID)
}

// ScriptAddress returns the bytes to be included in a txout script to pay
// to a script hash.  Part of the Address interface.
func (a *AddressScriptHash) ScriptAddress() []byte {
	return a.hash[:]
}

// IsForPrefix returns whether or not the pay-to-script-hash address is associated
// with the passed bitcoin network.
func (a *AddressScriptHash) IsForPrefix(prefix Bech32Prefix) bool {
	return a.prefix == prefix
}

// String returns a human-readable string for the pay-to-script-hash address.
// This is equivalent to calling EncodeAddress, but is provided so the type can
// be used as a fmt.Stringer.
func (a *AddressScriptHash) String() string {
	return a.EncodeAddress()
}

// Hash160 returns the underlying array of the script hash.  This can be useful
// when an array is more appropiate than a slice (for example, when used as map
// keys).
func (a *AddressScriptHash) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}
