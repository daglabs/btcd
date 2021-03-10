package utxo

import (
	"bytes"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

type utxoEntry struct {
	amount          uint64
	scriptPublicKey *externalapi.ScriptPublicKey
	blockBlueScore  uint64
	isCoinbase      bool
}

// NewUTXOEntry creates a new utxoEntry representing the given txOut
func NewUTXOEntry(amount uint64, scriptPubKey *externalapi.ScriptPublicKey, isCoinbase bool, blockBlueScore uint64) externalapi.UTXOEntry {
	scriptPubKeyClone := externalapi.ScriptPublicKey{Script: make([]byte, len(scriptPubKey.Script)), Version: scriptPubKey.Version}
	copy(scriptPubKeyClone.Script, scriptPubKey.Script)
	return &utxoEntry{
		amount:          amount,
		scriptPublicKey: &scriptPubKeyClone,
		blockBlueScore:  blockBlueScore,
		isCoinbase:      isCoinbase,
	}
}

func (u *utxoEntry) Amount() uint64 {
	return u.amount
}

func (u *utxoEntry) ScriptPublicKey() *externalapi.ScriptPublicKey {
	clone := externalapi.ScriptPublicKey{Script: make([]byte, len(u.scriptPublicKey.Script)), Version: u.scriptPublicKey.Version}
	copy(clone.Script, u.scriptPublicKey.Script)
	return &clone
}

func (u *utxoEntry) BlockBlueScore() uint64 {
	return u.blockBlueScore
}

func (u *utxoEntry) IsCoinbase() bool {
	return u.isCoinbase
}

// Equal returns whether entry equals to other
func (u *utxoEntry) Equal(other externalapi.UTXOEntry) bool {
	if u == nil || other == nil {
		return u == other
	}

	// If only the underlying value of other is nil it'll
	// make `other == nil` return false, so we check it
	// explicitly.
	downcastedOther := other.(*utxoEntry)
	if u == nil || downcastedOther == nil {
		return u == downcastedOther
	}

	if u.Amount() != other.Amount() {
		return false
	}

	if !bytes.Equal(u.ScriptPublicKey().Script, other.ScriptPublicKey().Script) {
		return false
	}

	if u.ScriptPublicKey().Version != other.ScriptPublicKey().Version {
		return false
	}

	if u.BlockBlueScore() != other.BlockBlueScore() {
		return false
	}

	if u.IsCoinbase() != other.IsCoinbase() {
		return false
	}

	return true
}
