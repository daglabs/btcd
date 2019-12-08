package dagconfig_test

import (
	"bytes"
	"github.com/daglabs/kaspad/util/hdkeychain"
	"reflect"
	"testing"

	. "github.com/daglabs/kaspad/dagconfig"
)

// Define some of the required parameters for a user-registered
// network.  This is necessary to test the registration of and
// lookup of encoding magics from the network.
var mockNetParams = Params{
	Name: "mocknet",
	Net:  1<<32 - 1,
	HDKeyIDPair: hdkeychain.HDKeyIDPair{
		PrivateKeyID: [4]byte{0x01, 0x02, 0x03, 0x04},
		PublicKeyID:  [4]byte{0x05, 0x06, 0x07, 0x08},
	},
}

func TestRegister(t *testing.T) {
	type registerTest struct {
		name   string
		params *Params
		err    error
	}
	type hdTest struct {
		priv []byte
		want []byte
		err  error
	}

	tests := []struct {
		name     string
		register []registerTest
		hdMagics []hdTest
	}{
		{
			name: "default networks",
			register: []registerTest{
				{
					name:   "duplicate mainnet",
					params: &MainNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate regtest",
					params: &RegressionNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate testnet",
					params: &TestNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate simnet",
					params: &SimNetParams,
					err:    ErrDuplicateNet,
				},
			},
			hdMagics: []hdTest{
				{
					priv: MainNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: MainNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: TestNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: TestNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: RegressionNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: RegressionNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: SimNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: SimNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: mockNetParams.HDKeyIDPair.PrivateKeyID[:],
					err:  hdkeychain.ErrUnknownHDKeyID,
				},
				{
					priv: []byte{0xff, 0xff, 0xff, 0xff},
					err:  hdkeychain.ErrUnknownHDKeyID,
				},
				{
					priv: []byte{0xff},
					err:  hdkeychain.ErrUnknownHDKeyID,
				},
			},
		},
		{
			name: "register mocknet",
			register: []registerTest{
				{
					name:   "mocknet",
					params: &mockNetParams,
					err:    nil,
				},
			},
			hdMagics: []hdTest{
				{
					priv: mockNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: mockNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
			},
		},
		{
			name: "more duplicates",
			register: []registerTest{
				{
					name:   "duplicate mainnet",
					params: &MainNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate regtest",
					params: &RegressionNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate testnet",
					params: &TestNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate simnet",
					params: &SimNetParams,
					err:    ErrDuplicateNet,
				},
				{
					name:   "duplicate mocknet",
					params: &mockNetParams,
					err:    ErrDuplicateNet,
				},
			},
			hdMagics: []hdTest{
				{
					priv: MainNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: MainNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: TestNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: TestNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: RegressionNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: RegressionNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: SimNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: SimNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: mockNetParams.HDKeyIDPair.PrivateKeyID[:],
					want: mockNetParams.HDKeyIDPair.PublicKeyID[:],
					err:  nil,
				},
				{
					priv: []byte{0xff, 0xff, 0xff, 0xff},
					err:  hdkeychain.ErrUnknownHDKeyID,
				},
				{
					priv: []byte{0xff},
					err:  hdkeychain.ErrUnknownHDKeyID,
				},
			},
		},
	}

	for _, test := range tests {
		for _, regTest := range test.register {
			err := Register(regTest.params)

			// HDKeyIDPairs must be registered separately
			hdkeychain.RegisterHDKeyIDPair(regTest.params.HDKeyIDPair)

			if err != regTest.err {
				t.Errorf("%s:%s: Registered network with unexpected error: got %v expected %v",
					test.name, regTest.name, err, regTest.err)
			}
		}
		for i, magTest := range test.hdMagics {
			pubKey, err := hdkeychain.HDPrivateKeyToPublicKeyID(magTest.priv[:])
			if !reflect.DeepEqual(err, magTest.err) {
				t.Errorf("%s: HD magic %d mismatched error: got %v expected %v ",
					test.name, i, err, magTest.err)
				continue
			}
			if magTest.err == nil && !bytes.Equal(pubKey, magTest.want[:]) {
				t.Errorf("%s: HD magic %d private and public mismatch: got %v expected %v ",
					test.name, i, pubKey, magTest.want[:])
			}
		}
	}
}
