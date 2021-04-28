package bip32

import "github.com/pkg/errors"

// BitcoinMainnetPrivate is the version that is used for
// bitcoin mainnet bip32 private extended keys.
var BitcoinMainnetPrivate = [4]byte{
	0x04,
	0x88,
	0xad,
	0xe4,
}

// BitcoinMainnetPublic is the version that is used for
// bitcoin mainnet bip32 public extended keys.
var BitcoinMainnetPublic = [4]byte{
	0x04,
	0x88,
	0xb2,
	0x1e,
}

// KaspaMainnetPrivate is the version that is used for
// kaspa mainnet bip32 private extended keys.
var KaspaMainnetPrivate = [4]byte{
	0x03,
	0x8f,
	0x2e,
	0xf4,
}

// KaspaMainnetPublic is the version that is used for
// kaspa mainnet bip32 public extended keys.
var KaspaMainnetPublic = [4]byte{
	0x03,
	0x8f,
	0x33,
	0x2e,
}

// KaspaTestnetPrivate is the version that is used for
// kaspa testnet bip32 public extended keys.
var KaspaTestnetPrivate = [4]byte{
	0x03,
	144,
	158,
	7,
}

// KaspaTestnetPublic is the version that is used for
// kaspa testnet bip32 public extended keys.
var KaspaTestnetPublic = [4]byte{
	3,
	144,
	162,
	65,
}

// KaspaDevnetPrivate is the version that is used for
// kaspa devnet bip32 public extended keys.
var KaspaDevnetPrivate = [4]byte{
	3,
	139,
	61,
	128,
}

// KaspaDevnetPublic is the version that is used for
// kaspa devnet bip32 public extended keys.
var KaspaDevnetPublic = [4]byte{
	3,
	139,
	65,
	186,
}

// KaspaSimnetPrivate is the version that is used for
// kaspa simnet bip32 public extended keys.
var KaspaSimnetPrivate = [4]byte{
	3,
	144,
	66,
	66,
}

// KaspaSimnetPublic is the version that is used for
// kaspa simnet bip32 public extended keys.
var KaspaSimnetPublic = [4]byte{
	3,
	144,
	70,
	125,
}

func toPublicVersion(version [4]byte) ([4]byte, error) {
	switch version {
	case BitcoinMainnetPrivate:
		return BitcoinMainnetPublic, nil
	case KaspaMainnetPrivate:
		return KaspaMainnetPublic, nil
	case KaspaTestnetPrivate:
		return KaspaTestnetPublic, nil
	case KaspaDevnetPrivate:
		return KaspaDevnetPublic, nil
	case KaspaSimnetPrivate:
		return KaspaSimnetPublic, nil
	}

	return [4]byte{}, errors.Errorf("unknown version %x", version)
}

func isPrivateVersion(version [4]byte) bool {
	switch version {
	case BitcoinMainnetPrivate:
		return true
	case KaspaMainnetPrivate:
		return true
	case KaspaTestnetPrivate:
		return true
	case KaspaDevnetPrivate:
		return true
	case KaspaSimnetPrivate:
		return true
	}

	return false
}
