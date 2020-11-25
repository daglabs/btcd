package externalapi

// UTXOEntry houses details about an individual transaction output in a utxo
// set such as whether or not it was contained in a coinbase tx, the blue
// score of the block that accepts the tx, its public key script, and how
// much it pays.
type UTXOEntry struct {
	Amount          uint64
	ScriptPublicKey []byte // The public key script for the output.
	BlockBlueScore  uint64 // Blue score of the block accepting the tx.
	IsCoinbase      bool
}

// Clone returns a clone of UTXOEntry
func (entry *UTXOEntry) Clone() *UTXOEntry {
	if entry == nil {
		return nil
	}

	scriptPublicKeyClone := make([]byte, len(entry.ScriptPublicKey))
	copy(scriptPublicKeyClone, entry.ScriptPublicKey)

	return &UTXOEntry{
		Amount:          entry.Amount,
		ScriptPublicKey: scriptPublicKeyClone,
		BlockBlueScore:  entry.BlockBlueScore,
		IsCoinbase:      entry.IsCoinbase,
	}
}

// NewUTXOEntry creates a new utxoEntry representing the given txOut
func NewUTXOEntry(amount uint64, scriptPubKey []byte, isCoinbase bool, blockBlueScore uint64) *UTXOEntry {
	return &UTXOEntry{
		Amount:          amount,
		ScriptPublicKey: scriptPubKey,
		BlockBlueScore:  blockBlueScore,
		IsCoinbase:      isCoinbase,
	}
}
