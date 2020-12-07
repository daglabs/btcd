package externalapi

import (
	"reflect"
	"testing"
)

type TransactionToCompare struct {
	tx             *DomainTransaction
	expectedResult bool
}

type TestDomainTransactionStruct struct {
	baseTx                 *DomainTransaction
	transactionToCompareTo []*TransactionToCompare
}

type TransactionInputToCompare struct {
	tx             *DomainTransactionInput
	expectedResult bool
}

type TestDomainTransactionInputStruct struct {
	baseTx                      *DomainTransactionInput
	transactionInputToCompareTo []*TransactionInputToCompare
}

type TransactionOutputToCompare struct {
	tx             *DomainTransactionOutput
	expectedResult bool
}

type TestDomainTransactionOutputStruct struct {
	baseTx                       *DomainTransactionOutput
	transactionOutputToCompareTo []*TransactionOutputToCompare
}

type DomainOutpointToCompare struct {
	domainOutpoint *DomainOutpoint
	expectedResult bool
}

type TestDomainOutpointStruct struct {
	baseDomainOutpoint        *DomainOutpoint
	domainOutpointToCompareTo []*DomainOutpointToCompare
}

type DomainTransactionIDToCompare struct {
	domainTransactionID *DomainTransactionID
	expectedResult      bool
}

type TestDomainTransactionIDStruct struct {
	baseDomainTransactionID        *DomainTransactionID
	domainTransactionIDToCompareTo []*DomainTransactionIDToCompare
}

func initTestBaseTransaction() *DomainTransaction {

	testTx := &DomainTransaction{
		Version: 1,
		Inputs: []*DomainTransactionInput{{DomainOutpoint{
			DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true}}},
		Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
			[]byte{1, 2}},
			{uint64(0xFFFF),
				[]byte{1, 3}}},
		LockTime:     1,
		SubnetworkID: DomainSubnetworkID{0x01},
		Gas:          1,
		PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Payload: []byte{0x01},
		Fee:     0,
		Mass:    1,
		ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
	}
	return testTx
}

func initTestTransactionToCompare() []*TransactionToCompare {

	testTx := []*TransactionToCompare{{
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, //1
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //0
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 3}}, //
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //1
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01, 0x02}, //4
			Gas:          1,
			PayloadHash: DomainHash{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //2
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01, 0x02}, //
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //3
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //4
		expectedResult: true,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     1000000000, //6
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //5
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 2, //
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //6
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    2, //
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		}, //7
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     2, //
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //8
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}},
				{DomainOutpoint{
					DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					&UTXOEntry{1,
						[]byte{0, 2, 2, 3},
						1,
						false}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //9
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}, {uint64(0xFFFFF),
				[]byte{1, 2, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //10
		expectedResult: false,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID:      nil,
		}, //11
		expectedResult: true,
	}, {
		tx: &DomainTransaction{
			Version: 1,
			Inputs: []*DomainTransactionInput{{DomainOutpoint{
				DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				&UTXOEntry{1,
					[]byte{0, 1, 2, 3, 4},
					2,
					true}}},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, //12
		expectedResult: false,
	},
	}
	return testTx

}

func initTestDomainTransactionForClone() []*DomainTransaction {

	tests := []*DomainTransaction{
		{
			Version: 1,
			Inputs: []*DomainTransactionInput{
				{DomainOutpoint{
					DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					&UTXOEntry{1,
						[]byte{0, 1, 2, 3},
						2,
						true}},
			},
			Outputs: []*DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			Payload: []byte{0x01},
			Fee:     5555555555,
			Mass:    1,
			ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, {
			Version:      1,
			Inputs:       []*DomainTransactionInput{},
			Outputs:      []*DomainTransactionOutput{},
			LockTime:     1,
			SubnetworkID: DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID:      &DomainTransactionID{},
		},
	}
	return tests
}

func initTestDomainTransactionForEqual() []TestDomainTransactionStruct {

	tests := []TestDomainTransactionStruct{
		{
			baseTx:                 initTestBaseTransaction(),
			transactionToCompareTo: initTestTransactionToCompare(),
		},
		{
			baseTx: nil,
			transactionToCompareTo: []*TransactionToCompare{{
				tx: &DomainTransaction{
					Version:      1,
					Inputs:       []*DomainTransactionInput{},
					Outputs:      []*DomainTransactionOutput{},
					LockTime:     1,
					SubnetworkID: DomainSubnetworkID{0x01},
					Gas:          1,
					PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					Payload: []byte{0x01},
					Fee:     0,
					Mass:    1,
					ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				},
				expectedResult: false,
			}, {
				tx:             nil,
				expectedResult: true}},
		}, {
			baseTx: &DomainTransaction{
				Version:      1,
				Inputs:       []*DomainTransactionInput{},
				Outputs:      []*DomainTransactionOutput{},
				LockTime:     1,
				SubnetworkID: DomainSubnetworkID{0x01},
				Gas:          1,
				PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				Payload: []byte{0x01},
				Fee:     0,
				Mass:    1,
				ID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			transactionToCompareTo: []*TransactionToCompare{{
				tx:             nil,
				expectedResult: false,
			}, {
				tx: &DomainTransaction{
					Version:      1,
					Inputs:       []*DomainTransactionInput{},
					Outputs:      []*DomainTransactionOutput{},
					LockTime:     1,
					SubnetworkID: DomainSubnetworkID{0x01},
					Gas:          1,
					PayloadHash: DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					Payload: []byte{0x01},
					Fee:     0,
					Mass:    1,
					ID:      nil,
				},
				expectedResult: true,
			}},
		},
	}
	return tests
}

func initTestBaseDomainTransactionInput() *DomainTransactionInput {
	basetxInput := &DomainTransactionInput{
		DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
		[]byte{1, 2, 3},
		uint64(0xFFFFFFFF),
		&UTXOEntry{1,
			[]byte{0, 1, 2, 3},
			2,
			true},
	}
	return basetxInput
}

func initTestDomainTxInputToCompare() []*TransactionInputToCompare {
	txInput := []*TransactionInputToCompare{{
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		},
		expectedResult: true,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				false},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3, 4},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01, 0x02}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01, 0x02}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			&UTXOEntry{2,
				[]byte{0, 1, 2, 3},
				2,
				true},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionInput{
			DomainOutpoint{DomainTransactionID{0x01, 0x02}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				3,
				true},
		},
		expectedResult: false,
	}, {
		tx:             nil,
		expectedResult: false,
	}}
	return txInput

}

func initTestDomainTransactionInputForClone() []*DomainTransactionInput {
	txInput := []*DomainTransactionInput{
		{
			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		}, {

			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				false},
		}, {

			DomainOutpoint{DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			&UTXOEntry{1,
				[]byte{0, 1, 2, 3},
				2,
				true},
		}}
	return txInput
}

func initTestBaseDomainTransactionOutput() *DomainTransactionOutput {
	basetxOutput := &DomainTransactionOutput{
		0xFFFFFFFF,
		[]byte{0xFF, 0xFF},
	}
	return basetxOutput
}

func initTestDomainTxOutputToCompare() []*TransactionOutputToCompare {
	txInput := []*TransactionOutputToCompare{{
		tx: &DomainTransactionOutput{
			0xFFFFFFFF,
			[]byte{0xFF, 0xFF}},
		expectedResult: true,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFFF,
			[]byte{0xF0, 0xFF},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{0xFF, 0xFF},
		},
		expectedResult: false,
	}, {
		tx:             nil,
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{0xFF, 0xFF, 0x01},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{},
		},
		expectedResult: false,
	}}
	return txInput
}

func initTestAnotherDomainTxOutputToCompare() []*TransactionOutputToCompare {
	txInput := []*TransactionOutputToCompare{{
		tx: &DomainTransactionOutput{
			0xFFFFFFFF,
			[]byte{0xFF, 0xFF}},
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFFF,
			[]byte{0xF0, 0xFF},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{0xFF, 0xFF},
		},
		expectedResult: false,
	}, {
		tx:             nil,
		expectedResult: true,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{0xFF, 0xFF, 0x01},
		},
		expectedResult: false,
	}, {
		tx: &DomainTransactionOutput{
			0xFFFFFFF0,
			[]byte{},
		},
		expectedResult: false,
	}}
	return txInput
}

func initTestDomainTransactionOutputForClone() []*DomainTransactionOutput {
	txInput := []*DomainTransactionOutput{
		{
			0xFFFFFFFF,
			[]byte{0xF0, 0xFF},
		}, {
			0xFFFFFFF1,
			[]byte{0xFF, 0xFF},
		}}
	return txInput
}

func initTestDomainTransactionOutputForEqual() []TestDomainTransactionOutputStruct {

	tests := []TestDomainTransactionOutputStruct{
		{
			baseTx:                       initTestBaseDomainTransactionOutput(),
			transactionOutputToCompareTo: initTestDomainTxOutputToCompare(),
		},
		{
			baseTx:                       nil,
			transactionOutputToCompareTo: initTestAnotherDomainTxOutputToCompare(),
		},
	}
	return tests
}

func initTestDomainTransactionInputForEqual() []TestDomainTransactionInputStruct {

	tests := []TestDomainTransactionInputStruct{
		{
			baseTx:                      initTestBaseDomainTransactionInput(),
			transactionInputToCompareTo: initTestDomainTxInputToCompare(),
		},
	}
	return tests
}

func TestDomainTransaction_Equal(t *testing.T) {

	txTests := initTestDomainTransactionForEqual()
	for i, test := range txTests {
		for j, subTest := range test.transactionToCompareTo {
			result1 := test.baseTx.Equal(subTest.tx)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.tx.Equal(test.baseTx)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestDomainTransaction_Clone(t *testing.T) {

	txs := initTestDomainTransactionForClone()
	for i, tx := range txs {
		txClone := tx.Clone()
		if !txClone.Equal(tx) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(tx, txClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func TestDomainTransactionInput_Equal(t *testing.T) {

	txTests := initTestDomainTransactionInputForEqual()
	for i, test := range txTests {
		for j, subTest := range test.transactionInputToCompareTo {
			result1 := test.baseTx.Equal(subTest.tx)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.tx.Equal(test.baseTx)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestDomainTransactionInput_Clone(t *testing.T) {

	txInputs := initTestDomainTransactionInputForClone()
	for i, txInput := range txInputs {
		txInputClone := txInput.Clone()
		if !txInputClone.Equal(txInput) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(txInput, txInputClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func TestDomainTransactionOutput_Equal(t *testing.T) {

	txTests := initTestDomainTransactionOutputForEqual()
	for i, test := range txTests {
		for j, subTest := range test.transactionOutputToCompareTo {
			result1 := test.baseTx.Equal(subTest.tx)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.tx.Equal(test.baseTx)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestDomainTransactionOutput_Clone(t *testing.T) {

	txInputs := initTestDomainTransactionOutputForClone()
	for i, txOutput := range txInputs {
		txOutputClone := txOutput.Clone()
		if !txOutputClone.Equal(txOutput) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(txOutput, txOutputClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func initTestDomainOutpointForClone() []*DomainOutpoint {
	outpoint := []*DomainOutpoint{{
		DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		1},
	}
	return outpoint
}

func initTestDomainOutpointForEqual() []TestDomainOutpointStruct {

	var outpoint = []*DomainOutpointToCompare{{
		domainOutpoint: &DomainOutpoint{
			DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			1},
		expectedResult: true,
	}, {
		domainOutpoint: &DomainOutpoint{
			DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
			1},
		expectedResult: false,
	}, {
		domainOutpoint: &DomainOutpoint{
			DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0},
			2},
		expectedResult: false,
	}}
	tests := []TestDomainOutpointStruct{
		{
			baseDomainOutpoint: &DomainOutpoint{
				DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				1},
			domainOutpointToCompareTo: outpoint,
		}, {baseDomainOutpoint: &DomainOutpoint{
			DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			1},
			domainOutpointToCompareTo: []*DomainOutpointToCompare{{domainOutpoint: nil, expectedResult: false}},
		}, {baseDomainOutpoint: nil,
			domainOutpointToCompareTo: []*DomainOutpointToCompare{{domainOutpoint: nil, expectedResult: true}},
		},
	}
	return tests
}

func TestDomainOutpoint_Equal(t *testing.T) {

	domainOutpoints := initTestDomainOutpointForEqual()
	for i, test := range domainOutpoints {
		for j, subTest := range test.domainOutpointToCompareTo {
			result1 := test.baseDomainOutpoint.Equal(subTest.domainOutpoint)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.domainOutpoint.Equal(test.baseDomainOutpoint)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestDomainOutpoint_Clone(t *testing.T) {

	domainOutpoints := initTestDomainOutpointForClone()
	for i, outpoint := range domainOutpoints {
		outpointClone := outpoint.Clone()
		if !outpointClone.Equal(outpoint) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(outpoint, outpointClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func initTestDomainTransactionIDForClone() []*DomainTransactionID {
	outpoint := []*DomainTransactionID{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}

	return outpoint
}

func initTestDomainTransactionIDForEqual() []TestDomainTransactionIDStruct {

	var outpoint = []*DomainTransactionIDToCompare{{
		domainTransactionID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		expectedResult: true,
	}, {
		domainTransactionID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		expectedResult: false,
	}, {
		domainTransactionID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0},
		expectedResult: false,
	}}
	tests := []TestDomainTransactionIDStruct{
		{
			baseDomainTransactionID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			domainTransactionIDToCompareTo: outpoint,
		}, {
			baseDomainTransactionID: nil,
			domainTransactionIDToCompareTo: []*DomainTransactionIDToCompare{{
				domainTransactionID: &DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
				expectedResult: false,
			}},
		},
	}
	return tests
}

func TestDomainTransactionID_Equal(t *testing.T) {

	domainDomainTransactionIDs := initTestDomainTransactionIDForEqual()
	for i, test := range domainDomainTransactionIDs {
		for j, subTest := range test.domainTransactionIDToCompareTo {
			result1 := test.baseDomainTransactionID.Equal(subTest.domainTransactionID)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.domainTransactionID.Equal(test.baseDomainTransactionID)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestDomainTransactionID_Clone(t *testing.T) {

	domainDomainTransactionIDs := initTestDomainTransactionIDForClone()
	for i, domainTransactionID := range domainDomainTransactionIDs {
		domainTransactionIDClone := domainTransactionID.Clone()
		if !domainTransactionIDClone.Equal(domainTransactionID) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(domainTransactionID, domainTransactionIDClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}
