// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/kaspanet/kaspad/util/binaryserializer"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/subnetworkid"
)

// MaxVarIntPayload is the maximum payload size for a variable length integer.
const MaxVarIntPayload = 9

var (
	// littleEndian is a convenience variable since binary.LittleEndian is
	// quite long.
	littleEndian = binary.LittleEndian

	// bigEndian is a convenience variable since binary.BigEndian is quite
	// long.
	bigEndian = binary.BigEndian
)

// errNonCanonicalVarInt is the common format string used for non-canonically
// encoded variable length integer errors.
var errNonCanonicalVarInt = "non-canonical varint %x - discriminant %x must " +
	"encode a value greater than %x"

// int64Time represents a unix timestamp encoded with an int64. It is used as
// a way to signal the readElement function how to decode a timestamp into a Go
// time.Time since it is otherwise ambiguous.
type int64Time time.Time

// ReadElement reads the next sequence of bytes from r using little endian
// depending on the concrete type of element pointed to.
func ReadElement(r io.Reader, element interface{}) error {
	// Attempt to read the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case *int32:
		rv, err := binaryserializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil

	case *uint32:
		rv, err := binaryserializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int64:
		rv, err := binaryserializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil

	case *uint64:
		rv, err := binaryserializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *bool:
		rv, err := binaryserializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil

	// Unix timestamp encoded as an int64.
	case *int64Time:
		rv, err := binaryserializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64Time(time.Unix(int64(rv), 0))
		return nil

	// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case *[CommandSize]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case *[16]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *daghash.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *subnetworkid.SubnetworkID:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *ServiceFlag:
		rv, err := binaryserializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = ServiceFlag(rv)
		return nil

	case *InvType:
		rv, err := binaryserializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = InvType(rv)
		return nil

	case *KaspaNet:
		rv, err := binaryserializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = KaspaNet(rv)
		return nil

	case *BloomUpdateType:
		rv, err := binaryserializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = BloomUpdateType(rv)
		return nil

	case *RejectCode:
		rv, err := binaryserializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = RejectCode(rv)
		return nil
	}

	// Fall back to the slower binary.Read if a fast path was not available
	// above.
	return binary.Read(r, littleEndian, element)
}

// readElements reads multiple items from r. It is equivalent to multiple
// calls to readElement.
func readElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := ReadElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteElement writes the little endian representation of element to w.
func WriteElement(w io.Writer, element interface{}) error {
	// Attempt to write the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case int32:
		err := binaryserializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case uint32:
		err := binaryserializer.PutUint32(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := binaryserializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case uint64:
		err := binaryserializer.PutUint64(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case bool:
		var err error
		if e {
			err = binaryserializer.PutUint8(w, 0x01)
		} else {
			err = binaryserializer.PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil

	// Message header checksum.
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case [CommandSize]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *daghash.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *subnetworkid.SubnetworkID:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case ServiceFlag:
		err := binaryserializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case InvType:
		err := binaryserializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case KaspaNet:
		err := binaryserializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case BloomUpdateType:
		err := binaryserializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil

	case RejectCode:
		err := binaryserializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil
	}

	// Fall back to the slower binary.Write if a fast path was not available
	// above.
	return binary.Write(w, littleEndian, element)
}

// writeElements writes multiple items to w. It is equivalent to multiple
// calls to writeElement.
func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := WriteElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadVarInt reads a variable length integer from r and returns it as a uint64.
func ReadVarInt(r io.Reader) (uint64, error) {
	discriminant, err := binaryserializer.Uint8(r)
	if err != nil {
		return 0, err
	}

	var rv uint64
	switch discriminant {
	case 0xff:
		sv, err := binaryserializer.Uint64(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = sv

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x100000000)
		if rv < min {
			return 0, messageError("readVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	case 0xfe:
		sv, err := binaryserializer.Uint32(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x10000)
		if rv < min {
			return 0, messageError("readVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	case 0xfd:
		sv, err := binaryserializer.Uint16(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0xfd)
		if rv < min {
			return 0, messageError("readVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	default:
		rv = uint64(discriminant)
	}

	return rv, nil
}

// WriteVarInt serializes val to w using a variable number of bytes depending
// on its value.
func WriteVarInt(w io.Writer, val uint64) error {
	if val < 0xfd {
		return binaryserializer.PutUint8(w, uint8(val))
	}

	if val <= math.MaxUint16 {
		err := binaryserializer.PutUint8(w, 0xfd)
		if err != nil {
			return err
		}
		return binaryserializer.PutUint16(w, littleEndian, uint16(val))
	}

	if val <= math.MaxUint32 {
		err := binaryserializer.PutUint8(w, 0xfe)
		if err != nil {
			return err
		}
		return binaryserializer.PutUint32(w, littleEndian, uint32(val))
	}

	err := binaryserializer.PutUint8(w, 0xff)
	if err != nil {
		return err
	}
	return binaryserializer.PutUint64(w, littleEndian, val)
}

// VarIntSerializeSize returns the number of bytes it would take to serialize
// val as a variable length integer.
func VarIntSerializeSize(val uint64) int {
	// The value is small enough to be represented by itself, so it's
	// just 1 byte.
	if val < 0xfd {
		return 1
	}

	// Discriminant 1 byte plus 2 bytes for the uint16.
	if val <= math.MaxUint16 {
		return 3
	}

	// Discriminant 1 byte plus 4 bytes for the uint32.
	if val <= math.MaxUint32 {
		return 5
	}

	// Discriminant 1 byte plus 8 bytes for the uint64.
	return 9
}

// ReadVarString reads a variable length string from r and returns it as a Go
// string. A variable length string is encoded as a variable length integer
// containing the length of the string followed by the bytes that represent the
// string itself. An error is returned if the length is greater than the
// maximum block payload size since it helps protect against memory exhaustion
// attacks and forced panics through malformed messages.
func ReadVarString(r io.Reader, pver uint32) (string, error) {
	count, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}

	// Prevent variable length strings that are larger than the maximum
	// message size. It would be possible to cause memory exhaustion and
	// panics without a sane upper bound on this count.
	if count > MaxMessagePayload {
		str := fmt.Sprintf("variable length string is too long "+
			"[count %d, max %d]", count, MaxMessagePayload)
		return "", messageError("ReadVarString", str)
	}

	buf := make([]byte, count)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// WriteVarString serializes str to w as a variable length integer containing
// the length of the string followed by the bytes that represent the string
// itself.
func WriteVarString(w io.Writer, str string) error {
	err := WriteVarInt(w, uint64(len(str)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	return err
}

// ReadVarBytes reads a variable length byte array. A byte array is encoded
// as a varInt containing the length of the array followed by the bytes
// themselves. An error is returned if the length is greater than the
// passed maxAllowed parameter which helps protect against memory exhaustion
// attacks and forced panics through malformed messages. The fieldName
// parameter is only used for the error message so it provides more context in
// the error.
func ReadVarBytes(r io.Reader, pver uint32, maxAllowed uint32,
	fieldName string) ([]byte, error) {

	count, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	// Prevent byte array larger than the max message size. It would
	// be possible to cause memory exhaustion and panics without a sane
	// upper bound on this count.
	if count > uint64(maxAllowed) {
		str := fmt.Sprintf("%s is larger than the max allowed size "+
			"[count %d, max %d]", fieldName, count, maxAllowed)
		return nil, messageError("ReadVarBytes", str)
	}

	b := make([]byte, count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// WriteVarBytes serializes a variable length byte array to w as a varInt
// containing the number of bytes, followed by the bytes themselves.
func WriteVarBytes(w io.Writer, pver uint32, bytes []byte) error {
	slen := uint64(len(bytes))
	err := WriteVarInt(w, slen)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}
