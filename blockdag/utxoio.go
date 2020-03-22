package blockdag

import (
	"bytes"
	"encoding/binary"
	"github.com/kaspanet/kaspad/util/binaryserializer"
	"github.com/pkg/errors"
	"io"
	"math"
	"math/big"

	"github.com/kaspanet/kaspad/ecc"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/wire"
)

// serializeBlockUTXODiffData serializes diff data in the following format:
// 	Name         | Data type | Description
//  ------------ | --------- | -----------
// 	hasDiffChild | Boolean   | Indicates if a diff child exist
//  diffChild    | Hash      | The diffChild's hash. Empty if hasDiffChild is true.
//  diff		 | UTXODiff  | The diff data's diff
func serializeBlockUTXODiffData(w io.Writer, diffData *blockUTXODiffData) error {
	hasDiffChild := diffData.diffChild != nil
	err := wire.WriteElement(w, hasDiffChild)
	if err != nil {
		return err
	}
	if hasDiffChild {
		err := wire.WriteElement(w, diffData.diffChild.hash)
		if err != nil {
			return err
		}
	}

	err = serializeUTXODiff(w, diffData.diff)
	if err != nil {
		return err
	}

	return nil
}

// utxoEntryHeaderCode returns the calculated header code to be used when
// serializing the provided utxo entry.
func utxoEntryHeaderCode(entry *UTXOEntry) uint64 {
	// As described in the serialization format comments, the header code
	// encodes the blue score shifted over one bit and the block reward flag
	// in the lowest bit.
	headerCode := uint64(entry.BlockBlueScore()) << 1
	if entry.IsCoinbase() {
		headerCode |= 0x01
	}

	return headerCode
}

func (diffStore *utxoDiffStore) deserializeBlockUTXODiffData(serializedDiffData []byte) (*blockUTXODiffData, error) {
	diffData := &blockUTXODiffData{}
	r := bytes.NewBuffer(serializedDiffData)

	var hasDiffChild bool
	err := wire.ReadElement(r, &hasDiffChild)
	if err != nil {
		return nil, err
	}

	if hasDiffChild {
		hash := &daghash.Hash{}
		err := wire.ReadElement(r, hash)
		if err != nil {
			return nil, err
		}
		diffData.diffChild = diffStore.dag.index.LookupNode(hash)
	}

	diffData.diff, err = deserializeUTXODiff(r)
	if err != nil {
		return nil, err
	}

	return diffData, nil
}

func deserializeUTXODiff(r io.Reader) (*UTXODiff, error) {
	diff := &UTXODiff{
		useMultiset: true,
	}

	var err error
	diff.toAdd, err = deserializeUTXOCollection(r)
	if err != nil {
		return nil, err
	}

	diff.toRemove, err = deserializeUTXOCollection(r)
	if err != nil {
		return nil, err
	}

	diff.diffMultiset, err = deserializeMultiset(r)
	if err != nil {
		return nil, err
	}

	return diff, nil
}

func deserializeUTXOCollection(r io.Reader) (utxoCollection, error) {
	count, err := wire.ReadVarIntLittleEndian(r)
	if err != nil {
		return nil, err
	}
	collection := utxoCollection{}
	for i := uint64(0); i < count; i++ {
		utxoEntry, outpoint, err := deserializeUTXO(r, true)
		if err != nil {
			return nil, err
		}
		collection.add(*outpoint, utxoEntry)
	}
	return collection, nil
}

func deserializeUTXO(r io.Reader, useCompression bool) (*UTXOEntry, *wire.Outpoint, error) {
	outpoint, err := deserializeOutpoint(r)
	if err != nil {
		return nil, nil, err
	}

	utxoEntry, err := deserializeUTXOEntry(r, useCompression)
	if err != nil {
		return nil, nil, err
	}
	return utxoEntry, outpoint, nil
}

// deserializeMultiset deserializes an EMCH multiset.
// See serializeMultiset for more details.
func deserializeMultiset(r io.Reader) (*ecc.Multiset, error) {
	xBytes := make([]byte, multisetPointSize)
	yBytes := make([]byte, multisetPointSize)
	err := binary.Read(r, byteOrder, xBytes)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, byteOrder, yBytes)
	if err != nil {
		return nil, err
	}
	var x, y big.Int
	x.SetBytes(xBytes)
	y.SetBytes(yBytes)
	return ecc.NewMultisetFromPoint(ecc.S256(), &x, &y), nil
}

// serializeUTXODiff serializes UTXODiff by serializing
// UTXODiff.toAdd, UTXODiff.toRemove and UTXODiff.Multiset one after the other.
func serializeUTXODiff(w io.Writer, diff *UTXODiff) error {
	if !diff.useMultiset {
		return errors.New("Cannot serialize a UTXO diff without a multiset")
	}
	err := serializeUTXOCollection(w, diff.toAdd)
	if err != nil {
		return err
	}

	err = serializeUTXOCollection(w, diff.toRemove)
	if err != nil {
		return err
	}
	err = serializeMultiset(w, diff.diffMultiset)
	if err != nil {
		return err
	}
	return nil
}

// serializeUTXOCollection serializes utxoCollection by iterating over
// the utxo entries and serializing them and their corresponding outpoint
// prefixed by a varint that indicates their size.
func serializeUTXOCollection(w io.Writer, collection utxoCollection) error {
	err := wire.WriteVarIntLittleEndian(w, uint64(len(collection)))
	if err != nil {
		return err
	}
	for outpoint, utxoEntry := range collection {
		err := serializeUTXO(w, utxoEntry, &outpoint, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// serializeMultiset serializes an ECMH multiset. The serialization
// is done by taking the (x,y) coordinnates of the multiset point and
// padding each one of them with 32 byte (it'll be 32 byte in most
// cases anyway except one of the coordinates is zero) and writing
// them one after the other.
func serializeMultiset(w io.Writer, ms *ecc.Multiset) error {
	x, y := ms.Point()
	xBytes := make([]byte, multisetPointSize)
	copy(xBytes, x.Bytes())
	yBytes := make([]byte, multisetPointSize)
	copy(yBytes, y.Bytes())

	err := binary.Write(w, byteOrder, xBytes)
	if err != nil {
		return err
	}
	err = binary.Write(w, byteOrder, yBytes)
	if err != nil {
		return err
	}
	return nil
}

// serializeUTXO serializes a utxo entry-outpoint pair
func serializeUTXO(w io.Writer, entry *UTXOEntry, outpoint *wire.Outpoint, compressEntries bool) error {
	err := serializeOutpoint(w, outpoint)
	if err != nil {
		return err
	}

	err = serializeUTXOEntry(w, entry, compressEntries)
	if err != nil {
		return err
	}
	return nil
}

// p2pkhUTXOEntryMaxSerializeSize is the maximum serialized size for a P2PKH UTXO entry.
// Varint (header code) + 8 bytes (amount) + compressed P2PKH script size.
var p2pkhUTXOEntryMaxSerializeSize = wire.VarIntSerializeSize(math.MaxUint64) + 8 + cstPayToPubKeyHashLen

// serializeUTXOEntry returns the entry serialized to a format that is suitable
// for long-term storage. The format is described in detail above.
func serializeUTXOEntry(w io.Writer, entry *UTXOEntry, useCompression bool) error {
	// Encode the header code.
	headerCode := utxoEntryHeaderCode(entry)

	err := wire.WriteVarIntLittleEndian(w, headerCode)
	if err != nil {
		return err
	}

	if useCompression {
		return putCompressedTxOut(w, entry.Amount(), entry.ScriptPubKey())
	}

	err = binaryserializer.PutUint64(w, byteOrder, entry.Amount())
	if err != nil {
		return err
	}

	err = wire.WriteVarIntLittleEndian(w, uint64(len(entry.ScriptPubKey())))
	if err != nil {
		return err
	}

	_, err = w.Write(entry.ScriptPubKey())
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// deserializeUTXOEntry decodes a UTXO entry from the passed serialized byte
// slice into a new UTXOEntry using a format that is suitable for long-term
// storage. The format is described in detail above.
func deserializeUTXOEntry(r io.Reader, useCompression bool) (*UTXOEntry, error) {
	// Deserialize the header code.
	headerCode, err := wire.ReadVarIntLittleEndian(r)
	if err != nil {
		return nil, err
	}

	// Decode the header code.
	//
	// Bit 0 indicates whether the containing transaction is a coinbase.
	// Bits 1-x encode blue score of the containing transaction.
	isCoinbase := headerCode&0x01 != 0
	blockBlueScore := headerCode >> 1

	entry := &UTXOEntry{
		blockBlueScore: blockBlueScore,
		packedFlags:    0,
	}

	if isCoinbase {
		entry.packedFlags |= tfCoinbase
	}

	if useCompression {
		entry.amount, entry.scriptPubKey, err = decodeCompressedTxOut(r)
		if err != nil {
			return nil, err
		}

		return entry, nil
	}

	entry.amount, err = binaryserializer.Uint64(r, byteOrder)
	if err != nil {
		return nil, err
	}

	scriptPubKeyLen, err := wire.ReadVarIntLittleEndian(r)
	if err != nil {
		return nil, err
	}

	entry.scriptPubKey = make([]byte, scriptPubKeyLen)
	_, err = r.Read(entry.scriptPubKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return entry, nil
}
