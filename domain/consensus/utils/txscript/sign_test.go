// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"
	"testing"

	"github.com/kaspanet/go-secp256k1"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/util"
)

func mkGetKey(keys map[string]*secp256k1.SchnorrKeyPair) KeyDB {
	if keys == nil {
		return KeyClosure(func(addr util.Address) (*secp256k1.SchnorrKeyPair, error) {
			return nil, errors.New("nope")
		})
	}
	return KeyClosure(func(addr util.Address) (*secp256k1.SchnorrKeyPair, error) {
		key, ok := keys[addr.EncodeAddress()]
		if !ok {
			return nil, errors.New("nope")
		}
		return key, nil
	})
}

func mkGetScript(scripts map[string][]byte) ScriptDB {
	if scripts == nil {
		return ScriptClosure(func(addr util.Address) ([]byte, error) {
			return nil, errors.New("nope")
		})
	}
	return ScriptClosure(func(addr util.Address) ([]byte, error) {
		script, ok := scripts[addr.EncodeAddress()]
		if !ok {
			return nil, errors.New("nope")
		}
		return script, nil
	})
}

func checkScripts(msg string, tx *externalapi.DomainTransaction, idx int, sigScript, scriptPubKey []byte) error {
	tx.Inputs[idx].SignatureScript = sigScript
	var flags ScriptFlags
	vm, err := NewEngine(scriptPubKey, tx, idx,
		flags, nil)
	if err != nil {
		return errors.Errorf("failed to make script engine for %s: %v",
			msg, err)
	}

	err = vm.Execute()
	if err != nil {
		return errors.Errorf("invalid script signature for %s: %v", msg,
			err)
	}

	return nil
}

func signAndCheck(msg string, tx *externalapi.DomainTransaction, idx int, scriptPubKey []byte,
	hashType SigHashType, kdb KeyDB, sdb ScriptDB,
	previousScript []byte) error {

	sigScript, err := SignTxOutput(&dagconfig.TestnetParams, tx, idx,
		scriptPubKey, hashType, kdb, sdb, nil)
	if err != nil {
		return errors.Errorf("failed to sign output %s: %v", msg, err)
	}

	return checkScripts(msg, tx, idx, sigScript, scriptPubKey)
}

func TestSignTxOutput(t *testing.T) {
	t.Parallel()

	// make key
	// make script based on key.
	// sign with magic pixie dust.
	hashTypes := []SigHashType{
		SigHashAll,
		SigHashNone,
		SigHashSingle,
		SigHashAll | SigHashAnyOneCanPay,
		SigHashNone | SigHashAnyOneCanPay,
		SigHashSingle | SigHashAnyOneCanPay,
	}
	inputs := []*externalapi.DomainTransactionInput{
		{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: externalapi.DomainTransactionID{},
				Index:         0,
			},
			Sequence: 4294967295,
		},
		{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: externalapi.DomainTransactionID{},
				Index:         1,
			},
			Sequence: 4294967295,
		},
		{
			PreviousOutpoint: externalapi.DomainOutpoint{
				TransactionID: externalapi.DomainTransactionID{},
				Index:         2,
			},
			Sequence: 4294967295,
		},
	}
	outputs := []*externalapi.DomainTransactionOutput{
		{
			Value: 1,
		},
		{
			Value: 2,
		},
		{
			Value: 3,
		},
	}
	tx := &externalapi.DomainTransaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Pay to Pubkey Hash (merging with correct)
	for _, hashType := range hashTypes {
		for i := range tx.Inputs {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			key, err := secp256k1.GeneratePrivateKey()
			if err != nil {
				t.Errorf("failed to make privKey for %s: %s",
					msg, err)
				break
			}

			pubKey, err := key.SchnorrPublicKey()
			if err != nil {
				t.Errorf("failed to make a publickey for %s: %s",
					key, err)
				break
			}

			serializedPubKey, err := pubKey.Serialize()
			if err != nil {
				t.Errorf("failed to make a pubkey for %s: %s",
					key, err)
				break
			}
			address, err := util.NewAddressPubKeyHash(
				util.Hash160(serializedPubKey[:]), util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			scriptPubKey, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make scriptPubKey "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

			// by the above loop, this should be valid, now sign
			// again and merge.
			sigScript, err = SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, sigScript, scriptPubKey)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

	// Pay to Pubkey Hash (compressed)
	for _, hashType := range hashTypes {
		for i := range tx.Inputs {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := secp256k1.GeneratePrivateKey()
			if err != nil {
				t.Errorf("failed to make privKey for %s: %s",
					msg, err)
				break
			}

			pubKey, err := key.SchnorrPublicKey()
			if err != nil {
				t.Errorf("failed to make a publickey for %s: %s",
					key, err)
				break
			}

			serializedPubKey, err := pubKey.Serialize()
			if err != nil {
				t.Errorf("failed to make a pubkey for %s: %s",
					key, err)
				break
			}

			address, err := util.NewAddressPubKeyHash(
				util.Hash160(serializedPubKey[:]), util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			scriptPubKey, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make scriptPubKey "+
					"for %s: %v", msg, err)
			}

			if err := signAndCheck(msg, tx, i, scriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(nil), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

	// Pay to Pubkey Hash with duplicate merge
	for _, hashType := range hashTypes {
		for i := range tx.Inputs {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := secp256k1.GeneratePrivateKey()
			if err != nil {
				t.Errorf("failed to make privKey for %s: %s",
					msg, err)
				break
			}

			pubKey, err := key.SchnorrPublicKey()
			if err != nil {
				t.Errorf("failed to make a publickey for %s: %s",
					key, err)
				break
			}

			serializedPubKey, err := pubKey.Serialize()
			if err != nil {
				t.Errorf("failed to make a pubkey for %s: %s",
					key, err)
				break
			}

			address, err := util.NewAddressPubKeyHash(
				util.Hash160(serializedPubKey[:]), util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			scriptPubKey, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make scriptPubKey "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

			// by the above loop, this should be valid, now sign
			// again and merge.
			sigScript, err = SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, sigScript, scriptPubKey)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

	// As before, but with p2sh now.

	// Pay to Pubkey Hash
	for _, hashType := range hashTypes {
		for i := range tx.Inputs {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := secp256k1.GeneratePrivateKey()
			if err != nil {
				t.Errorf("failed to make privKey for %s: %s",
					msg, err)
				break
			}

			pubKey, err := key.SchnorrPublicKey()
			if err != nil {
				t.Errorf("failed to make a publickey for %s: %s",
					key, err)
				break
			}

			serializedPubKey, err := pubKey.Serialize()
			if err != nil {
				t.Errorf("failed to make a pubkey for %s: %s",
					key, err)
				break
			}

			address, err := util.NewAddressPubKeyHash(
				util.Hash160(serializedPubKey[:]), util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			scriptPubKey, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make scriptPubKey "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := util.NewAddressScriptHash(
				scriptPubKey, util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptScriptPubKey, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script scriptPubKey for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, scriptScriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): scriptPubKey,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

	// Pay to Pubkey Hash (compressed) with duplicate merge
	for _, hashType := range hashTypes {
		for i := range tx.Inputs {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := secp256k1.GeneratePrivateKey()
			if err != nil {
				t.Errorf("failed to make privKey for %s: %s",
					msg, err)
				break
			}

			pubKey, err := key.SchnorrPublicKey()
			if err != nil {
				t.Errorf("failed to make a publickey for %s: %s",
					key, err)
				break
			}

			serializedPubKey, err := pubKey.Serialize()
			if err != nil {
				t.Errorf("failed to make a pubkey for %s: %s",
					key, err)
				break
			}

			address, err := util.NewAddressPubKeyHash(
				util.Hash160(serializedPubKey[:]), util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			scriptPubKey, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make scriptPubKey "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := util.NewAddressScriptHash(
				scriptPubKey, util.Bech32PrefixKaspaTest)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptScriptPubKey, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script scriptPubKey for "+
					"%s: %v", msg, err)
				break
			}

			_, err = SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptScriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): scriptPubKey,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

			// by the above loop, this should be valid, now sign
			// again and merge.
			sigScript, err := SignTxOutput(&dagconfig.TestnetParams,
				tx, i, scriptScriptPubKey, hashType,
				mkGetKey(map[string]*secp256k1.SchnorrKeyPair{
					address.EncodeAddress(): key,
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): scriptPubKey,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, sigScript, scriptScriptPubKey)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}
}

type tstInput struct {
	txout              *externalapi.DomainTransactionOutput
	sigscriptGenerates bool
	inputValidates     bool
	indexOutOfRange    bool
}

type tstSigScript struct {
	name               string
	inputs             []tstInput
	hashType           SigHashType
	scriptAtWrongIndex bool
}

var coinbaseOutpoint = &externalapi.DomainOutpoint{
	Index: (1 << 32) - 1,
}

// Pregenerated private key, with associated public key and scriptPubKeys
// for the uncompressed and compressed hash160.
var (
	privKeyD = secp256k1.SerializedPrivateKey{0x6b, 0x0f, 0xd8, 0xda, 0x54, 0x22, 0xd0, 0xb7,
		0xb4, 0xfc, 0x4e, 0x55, 0xd4, 0x88, 0x42, 0xb3, 0xa1, 0x65,
		0xac, 0x70, 0x7f, 0x3d, 0xa4, 0x39, 0x5e, 0xcb, 0x3b, 0xb0,
		0xd6, 0x0e, 0x06, 0x92}
	oldUncompressedScriptPubKey = []byte{0x76, 0xa9, 0x14, 0xd1, 0x7c, 0xb5,
		0xeb, 0xa4, 0x02, 0xcb, 0x68, 0xe0, 0x69, 0x56, 0xbf, 0x32,
		0x53, 0x90, 0x0e, 0x0a, 0x86, 0xc9, 0xfa, 0x88, 0xac}
	oldCompressedScriptPubKey = []byte{0x76, 0xa9, 0x14, 0x27, 0x4d, 0x9f, 0x7f,
		0x61, 0x7e, 0x7c, 0x7a, 0x1c, 0x1f, 0xb2, 0x75, 0x79, 0x10,
		0x43, 0x65, 0x68, 0x27, 0x9d, 0x86, 0x88, 0xac}
	p2pkhScriptPubKey = []byte{0x76, 0xa9, 0x14, 0x7e, 0x01, 0x76, 0xb6,
		0x72, 0x08, 0xc0, 0x08, 0x98, 0x85, 0x97, 0x00, 0x4e, 0x1a, 0x8d,
		0x60, 0x89, 0xfe, 0x42, 0x6f, 0x88, 0xac}
	shortScriptPubKey = []byte{0x76, 0xa9, 0x14, 0xd1, 0x7c, 0xb5,
		0xeb, 0xa4, 0x02, 0xcb, 0x68, 0xe0, 0x69, 0x56, 0xbf, 0x32,
		0x53, 0x90, 0x0e, 0x0a, 0x88, 0xac}
)

// Pretend output amounts.
const coinbaseVal = 2500000000
const fee = 5000000

var sigScriptTests = []tstSigScript{
	{
		name: "one input old uncompressed",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: oldUncompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "two inputs old uncompressed",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: oldUncompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal + fee,
					ScriptPublicKey: oldUncompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "one input old compressed",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: oldCompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "two inputs old compressed",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: oldCompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal + fee,
					ScriptPublicKey: oldCompressedScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "one input 32byte pubkey",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "two inputs 32byte pubkey",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal + fee,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashNone",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashNone,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashSingle",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashSingle,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashAll | SigHashAnyoneCanPay",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll | SigHashAnyOneCanPay,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashAnyoneCanPay",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAnyOneCanPay,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType non-exist",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           0x04,
		scriptAtWrongIndex: false,
	},
	{
		name: "short ScriptPubKey",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: shortScriptPubKey,
				},
				sigscriptGenerates: false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: false,
	},
	{
		name: "valid script at wrong index",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal + fee,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: true,
	},
	{
		name: "index out of range",
		inputs: []tstInput{
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout: &externalapi.DomainTransactionOutput{
					Value:           coinbaseVal + fee,
					ScriptPublicKey: p2pkhScriptPubKey,
				},
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		scriptAtWrongIndex: true,
	},
}

// Test the sigscript generation for valid and invalid inputs, all
// hashTypes, and with and without compression. This test creates
// sigscripts to spend fake coinbase inputs, as sigscripts cannot be
// created for the DomainTransactions in txTests, since they come from the blockDAG
// and we don't have the private keys.
func TestSignatureScript(t *testing.T) {
	t.Parallel()

	privKey, _ := secp256k1.DeserializePrivateKey(&privKeyD)

nexttest:
	for i := range sigScriptTests {
		outputs := []*externalapi.DomainTransactionOutput{
			{Value: 500, ScriptPublicKey: []byte{OpReturn}},
		}

		inputs := []*externalapi.DomainTransactionInput{}
		for range sigScriptTests[i].inputs {
			inputs = append(inputs, &externalapi.DomainTransactionInput{
				PreviousOutpoint: *coinbaseOutpoint,
			})
		}
		tx := &externalapi.DomainTransaction{
			Version: 1,
			Inputs:  inputs,
			Outputs: outputs,
		}

		var script []byte
		var err error
		for j := range tx.Inputs {
			var idx int
			if sigScriptTests[i].inputs[j].indexOutOfRange {
				t.Errorf("at test %v", sigScriptTests[i].name)
				idx = len(sigScriptTests[i].inputs)
			} else {
				idx = j
			}
			script, err = SignatureScript(tx, idx,
				sigScriptTests[i].inputs[j].txout.ScriptPublicKey,
				sigScriptTests[i].hashType, privKey)

			if (err == nil) != sigScriptTests[i].inputs[j].sigscriptGenerates {
				if err == nil {
					t.Errorf("passed test '%v' incorrectly",
						sigScriptTests[i].name)
				} else {
					t.Errorf("failed test '%v': %v",
						sigScriptTests[i].name, err)
				}
				continue nexttest
			}
			if !sigScriptTests[i].inputs[j].sigscriptGenerates {
				// done with this test
				continue nexttest
			}

			tx.Inputs[j].SignatureScript = script
		}

		// If testing using a correct sigscript but for an incorrect
		// index, use last input script for first input. Requires > 0
		// inputs for test.
		if sigScriptTests[i].scriptAtWrongIndex {
			tx.Inputs[0].SignatureScript = script
			sigScriptTests[i].inputs[0].inputValidates = false
		}

		// Validate tx input scripts
		var scriptFlags ScriptFlags
		for j := range tx.Inputs {
			vm, err := NewEngine(sigScriptTests[i].
				inputs[j].txout.ScriptPublicKey, tx, j, scriptFlags, nil)
			if err != nil {
				t.Errorf("cannot create script vm for test %v: %v",
					sigScriptTests[i].name, err)
				continue nexttest
			}
			err = vm.Execute()
			if (err == nil) != sigScriptTests[i].inputs[j].inputValidates {
				if err == nil {
					t.Errorf("passed test '%v' validation incorrectly: %v",
						sigScriptTests[i].name, err)
				} else {
					t.Errorf("failed test '%v' validation: %v",
						sigScriptTests[i].name, err)
				}
				continue nexttest
			}
		}
	}
}
