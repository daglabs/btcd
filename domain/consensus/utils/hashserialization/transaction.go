package hashserialization

import (
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/utils/hashes"
	"github.com/kaspanet/kaspad/domain/consensus/utils/transactionhelper"
	"github.com/kaspanet/kaspad/util/binaryserializer"
	"github.com/pkg/errors"
	"io"
)

// txEncoding is a bitmask defining which transaction fields we
// want to encode and which to ignore.
type txEncoding uint8

const (
	txEncodingFull txEncoding = 0

	// TODO: Consider if we need to ever exclude the payload, or use a different approach to partial nodes
	// where we'll get rid of PayloadHash field, never exclude the payload when hashing, and provide
	// partial nodes with their relevant block region with a merkle proof.
	txEncodingExcludePayload txEncoding = 1 << iota

	txEncodingExcludeSignatureScript
)

func TransactionHash(tx *model.DomainTransaction) *model.DomainHash {
	// Encode the header and double sha256 everything prior to the number of
	// transactions.
	writer := hashes.NewDoubleHashWriter()
	err := serializeTransaction(writer, tx, txEncodingExcludePayload)
	if err != nil {
		// It seems like this could only happen if the writer returned an error.
		// and this writer should never return an error (no allocations or possible failures)
		// the only non-writer error path here is unknown types in `WriteElement`
		panic(errors.Wrap(err, "TransactionHash() failed. this should never fail for structurally-valid transactions"))
	}

	res := writer.Finalize()
	return &res
}

// TxID generates the Hash for the transaction without the signature script, gas and payload fields.
func TransactionID(tx *model.DomainTransaction) *model.DomainTransactionID {
	// Encode the transaction, replace signature script with zeroes, cut off
	// payload and calculate double sha256 on the result.
	var encodingFlags txEncoding
	if !transactionhelper.IsCoinBase(tx) {
		encodingFlags = txEncodingExcludeSignatureScript | txEncodingExcludePayload
	}
	writer := hashes.NewDoubleHashWriter()
	err := serializeTransaction(writer, tx, encodingFlags)
	if err != nil {
		// this writer never return errors (no allocations or possible failures) so errors can only come from validity checks,
		// and we assume we never construct malformed transactions.
		panic(errors.Wrap(err, "TransactionID() failed. this should never fail for structurally-valid transactions"))
	}
	txID := model.DomainTransactionID(writer.Finalize())
	return &txID
}

func serializeTransaction(w io.Writer, tx *model.DomainTransaction, encodingFlags txEncoding) error {
	err := binaryserializer.PutUint32(w, littleEndian, uint32(tx.Version))
	if err != nil {
		return err
	}

	count := uint64(len(tx.Inputs))
	err = WriteElement(w, count)
	if err != nil {
		return err
	}

	for _, ti := range tx.Inputs {
		err = writeTransactionInput(w, ti, encodingFlags)
		if err != nil {
			return err
		}
	}

	count = uint64(len(tx.Outputs))
	err = WriteElement(w, count)
	if err != nil {
		return err
	}

	for _, output := range tx.Outputs {
		err = writeTxOut(w, output)
		if err != nil {
			return err
		}
	}

	err = binaryserializer.PutUint64(w, littleEndian, tx.LockTime)
	if err != nil {
		return err
	}

	_, err = w.Write(tx.SubnetworkID[:])
	if err != nil {
		return err
	}

	// TODO: Move to transaction validation
	//if subnetworks.IsBuiltIn(tx.SubnetworkID) && tx.Gas != 0 {
	//	str := "Transactions from built-in should have 0 gas"
	//	return messageError("MsgTx.KaspaEncode", str)
	//}

	err = binaryserializer.PutUint64(w, littleEndian, tx.Gas)
	if err != nil {
		return err
	}

	err = WriteElement(w, tx.PayloadHash)
	if err != nil {
		return err
	}

	if encodingFlags&txEncodingExcludePayload != txEncodingExcludePayload {
		err = writeVarBytes(w, tx.Payload)
		if err != nil {
			return err
		}
	} else {
		err = writeVarBytes(w, []byte{})
		if err != nil {
			return err
		}
	}

	// TODO: Move to transaction validation
	//else if tx.Payload != nil {
	//	str := "Transactions from native subnetwork should have <nil> payload"
	//	return messageError("MsgTx.KaspaEncode", str)
	//} else if tx.PayloadHash != nil {
	//	str := "Transactions from native subnetwork should have <nil> payload hash"
	//	return messageError("MsgTx.KaspaEncode", str)
	//} else if tx.Gas != 0 {
	//	str := "Transactions from native subnetwork should have 0 gas"
	//	return messageError("MsgTx.KaspaEncode", str)
	//}

	return nil
}

// writeTransactionInput encodes ti to the kaspa protocol encoding for a transaction
// input to w.
func writeTransactionInput(w io.Writer, ti *model.DomainTransactionInput, encodingFlags txEncoding) error {
	err := writeOutpoint(w, &ti.PreviousOutpoint)
	if err != nil {
		return err
	}

	if encodingFlags&txEncodingExcludeSignatureScript != txEncodingExcludeSignatureScript {
		err = writeVarBytes(w, ti.SignatureScript)
	} else {
		err = writeVarBytes(w, []byte{})
	}
	if err != nil {
		return err
	}

	return binaryserializer.PutUint64(w, littleEndian, ti.Sequence)
}

func writeOutpoint(w io.Writer, outpoint *model.DomainOutpoint) error {
	_, err := w.Write(outpoint.ID[:])
	if err != nil {
		return err
	}

	return binaryserializer.PutUint32(w, littleEndian, outpoint.Index)
}

func writeVarBytes(w io.Writer, data []byte) error {
	dataLength := uint64(len(data))
	err := WriteElement(w, dataLength)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func writeTxOut(w io.Writer, to *model.DomainTransactionOutput) error {
	err := binaryserializer.PutUint64(w, littleEndian, to.Value)
	if err != nil {
		return err
	}

	return writeVarBytes(w, to.ScriptPublicKey)
}