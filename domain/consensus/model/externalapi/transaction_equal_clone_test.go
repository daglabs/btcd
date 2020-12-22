package externalapi_test

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"reflect"
	"testing"
)

// Changed fields of a test struct compared to a base test struct marked as "changed" and
// pointing in some cases name changed struct field

type transactionToCompare struct {
	tx             *externalapi.DomainTransaction
	expectedResult bool
	expectsPanic   bool
}

type testDomainTransactionStruct struct {
	baseTx                 *externalapi.DomainTransaction
	transactionToCompareTo []*transactionToCompare
}

type transactionInputToCompare struct {
	tx             *externalapi.DomainTransactionInput
	expectedResult bool
}

type testDomainTransactionInputStruct struct {
	baseTx                      *externalapi.DomainTransactionInput
	transactionInputToCompareTo []*transactionInputToCompare
}

type transactionOutputToCompare struct {
	tx             *externalapi.DomainTransactionOutput
	expectedResult bool
}

type testDomainTransactionOutputStruct struct {
	baseTx                       *externalapi.DomainTransactionOutput
	transactionOutputToCompareTo []*transactionOutputToCompare
}

type domainOutpointToCompare struct {
	domainOutpoint *externalapi.DomainOutpoint
	expectedResult bool
}

type testDomainOutpointStruct struct {
	baseDomainOutpoint        *externalapi.DomainOutpoint
	domainOutpointToCompareTo []*domainOutpointToCompare
}

type domainTransactionIDToCompare struct {
	domainTransactionID *externalapi.DomainTransactionID
	expectedResult      bool
}

type testDomainTransactionIDStruct struct {
	baseDomainTransactionID        *externalapi.DomainTransactionID
	domainTransactionIDToCompareTo []*domainTransactionIDToCompare
}

func initTestBaseTransaction() *externalapi.DomainTransaction {

	testTx := &externalapi.DomainTransaction{
		1,
		[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
			externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
		[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
			[]byte{1, 2}},
			{uint64(0xFFFF),
				[]byte{1, 3}}},
		1,
		externalapi.DomainSubnetworkID{0x01},
		1,
		externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		[]byte{0x01},
		0,
		1,
		&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
	}
	return testTx
}

func initTestTransactionToCompare() []*transactionToCompare {

	testTx := []*transactionToCompare{{
		tx: &externalapi.DomainTransaction{
			1,
			[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			1,
			externalapi.DomainSubnetworkID{0x01},
			1,
			externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, //Changed
			[]byte{0x01},
			0,
			1,
			&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransaction{
			1,
			[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 3}}, //Changed
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			1,
			externalapi.DomainSubnetworkID{0x01},
			1,
			externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]byte{0x01},
			0,
			1,
			&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransaction{
			1,
			[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			1,
			externalapi.DomainSubnetworkID{0x01, 0x02}, //Changed
			1,
			externalapi.DomainHash{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]byte{0x01},
			0,
			1,
			&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransaction{
			1,
			[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			1,
			externalapi.DomainSubnetworkID{0x01},
			1,
			externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]byte{0x01, 0x02}, //Changed
			0,
			1,
			&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransaction{
			1,
			[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}, {uint64(0xFFFF),
				[]byte{1, 3}}},
			1,
			externalapi.DomainSubnetworkID{0x01},
			1,
			externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]byte{0x01},
			0,
			1,
			&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		expectedResult: true,
	},
		{
			// ID changed
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
			},
			expectsPanic: true,
		},
		{
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}},
					{uint64(0xFFFF),
						[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				1000000000, //Changed
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				2, //Changed
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			}, //6
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				2, //Changed
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			}, //7
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				2, //Changed
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)},
					{externalapi.DomainOutpoint{
						externalapi.DomainTransactionID{0x01}, 0xFFFF},
						[]byte{1, 2, 3},
						uint64(0xFFFFFFFF),
						utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}, {uint64(0xFFFFF),
					[]byte{1, 2, 3}}}, //changed Outputs
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			expectedResult: false,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				nil, //changed
			},
			expectedResult: true,
		}, {
			tx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3, 4}, true, 2)}},
				[]*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}}, {uint64(0xFFFF),
					[]byte{1, 3}}},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			expectedResult: false,
		},
	}
	return testTx
}

func initTestDomainTransactionForClone() []*externalapi.DomainTransaction {

	tests := []*externalapi.DomainTransaction{
		{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{
				{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)},
			},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			Payload: []byte{0x01},
			Fee:     5555555555,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		}, {
			Version:      1,
			Inputs:       []*externalapi.DomainTransactionInput{},
			Outputs:      []*externalapi.DomainTransactionOutput{},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID:      &externalapi.DomainTransactionID{},
		},
	}
	return tests
}

func initTestDomainTransactionForEqual() []testDomainTransactionStruct {

	tests := []testDomainTransactionStruct{
		{
			baseTx:                 initTestBaseTransaction(),
			transactionToCompareTo: initTestTransactionToCompare(),
		},
		{
			baseTx: nil,
			transactionToCompareTo: []*transactionToCompare{{
				tx: &externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{},
					[]*externalapi.DomainTransactionOutput{},
					1,
					externalapi.DomainSubnetworkID{0x01},
					1,
					externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					[]byte{0x01},
					0,
					1,
					&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				},
				expectedResult: false,
			}, {
				tx:             nil,
				expectedResult: true}},
		}, {
			baseTx: &externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{},
				[]*externalapi.DomainTransactionOutput{},
				1,
				externalapi.DomainSubnetworkID{0x01},
				1,
				externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				[]byte{0x01},
				0,
				1,
				&externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			transactionToCompareTo: []*transactionToCompare{{
				tx:             nil,
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{},
					[]*externalapi.DomainTransactionOutput{},
					1,
					externalapi.DomainSubnetworkID{0x01},
					0,
					externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					[]byte{0x01},
					0,
					1,
					nil,
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{},
					[]*externalapi.DomainTransactionOutput{},
					1,
					externalapi.DomainSubnetworkID{0x01},
					1,
					externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					[]byte{0x01},
					0,
					1,
					nil,
				},
				expectedResult: true,
			}},
		},
	}
	return tests
}

func initTestBaseDomainTransactionInput() *externalapi.DomainTransactionInput {
	basetxInput := &externalapi.DomainTransactionInput{
		externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
		[]byte{1, 2, 3},
		uint64(0xFFFFFFFF),
		utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
	}
	return basetxInput
}

func initTestDomainTxInputToCompare() []*transactionInputToCompare {
	txInput := []*transactionInputToCompare{{
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		},
		expectedResult: true,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, false, 2), // Changed
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0), // Changed
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3, 4}, // Changed
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01, 0x02}, 0xFFFF}, // Changed
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0), // Changed
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01, 0x02}, 0xFFFF}, // Changed
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0), // Changed
			utxo.NewUTXOEntry(2 /* Changed */, []byte{0, 1, 2, 3}, true, 2), // Changed
		},
		expectedResult: false,
	}, {
		tx: &externalapi.DomainTransactionInput{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01, 0x02}, 0xFFFF}, // Changed
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0), // Changed
			utxo.NewUTXOEntry(3 /* Changed */, []byte{0, 1, 2, 3}, true, 3), // Changed
		},
		expectedResult: false,
	}, {
		tx:             nil,
		expectedResult: false,
	}}
	return txInput

}

func initTestDomainTransactionInputForClone() []*externalapi.DomainTransactionInput {
	txInput := []*externalapi.DomainTransactionInput{
		{
			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		}, {

			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFFF),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		}, {

			externalapi.DomainOutpoint{externalapi.DomainTransactionID{0x01}, 0xFFFF},
			[]byte{1, 2, 3},
			uint64(0xFFFFFFF0),
			utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2),
		}}
	return txInput
}

func initTestBaseDomainTransactionOutput() *externalapi.DomainTransactionOutput {
	basetxOutput := &externalapi.DomainTransactionOutput{
		0xFFFFFFFF,
		[]byte{0xFF, 0xFF},
	}
	return basetxOutput
}

func initTestDomainTransactionOutputForClone() []*externalapi.DomainTransactionOutput {
	txInput := []*externalapi.DomainTransactionOutput{
		{
			0xFFFFFFFF,
			[]byte{0xF0, 0xFF},
		}, {
			0xFFFFFFF1,
			[]byte{0xFF, 0xFF},
		}}
	return txInput
}

func initTestDomainTransactionOutputForEqual() []testDomainTransactionOutputStruct {
	tests := []testDomainTransactionOutputStruct{
		{
			baseTx: initTestBaseDomainTransactionOutput(),
			transactionOutputToCompareTo: []*transactionOutputToCompare{{
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFFF,
					[]byte{0xFF, 0xFF}},
				expectedResult: true,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFFF,
					[]byte{0xF0, 0xFF}, // Changed
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0, // Changed
					[]byte{0xFF, 0xFF},
				},
				expectedResult: false,
			}, {
				tx:             nil,
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0,               // Changed
					[]byte{0xFF, 0xFF, 0x01}, // Changed
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0, // Changed
					[]byte{},   // Changed
				},
				expectedResult: false,
			}},
		},
		{
			baseTx: nil,
			transactionOutputToCompareTo: []*transactionOutputToCompare{{
				tx:             nil,
				expectedResult: true,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFFF,
					[]byte{0xFF, 0xFF}},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFFF,
					[]byte{0xF0, 0xFF}, // Changed
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0, // Changed
					[]byte{0xFF, 0xFF},
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0,
					[]byte{0xFF, 0xFF, 0x01}, // Changed
				},
				expectedResult: false,
			}, {
				tx: &externalapi.DomainTransactionOutput{
					0xFFFFFFF0,
					[]byte{}, // Changed
				},
				expectedResult: false,
			}},
		},
	}
	return tests
}

func initTestDomainTransactionInputForEqual() []testDomainTransactionInputStruct {

	tests := []testDomainTransactionInputStruct{
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
			func() {
				defer func() {
					r := recover()
					panicked := r != nil
					if panicked != subTest.expectsPanic {
						t.Fatalf("panicked expected to be %t but got %t", subTest.expectsPanic, panicked)
					}
				}()
				result1 := test.baseTx.Equal(subTest.tx)
				if result1 != subTest.expectedResult {
					t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
				}
			}()
			func() {
				defer func() {
					r := recover()
					panicked := r != nil
					if panicked != subTest.expectsPanic {
						t.Fatalf("panicked expected to be %t but got %t", subTest.expectsPanic, panicked)
					}
				}()
				result2 := subTest.tx.Equal(test.baseTx)
				if result2 != subTest.expectedResult {
					t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
				}
			}()
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

func initTestDomainOutpointForClone() []*externalapi.DomainOutpoint {
	outpoint := []*externalapi.DomainOutpoint{{
		externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		1},
	}
	return outpoint
}

func initTestDomainOutpointForEqual() []testDomainOutpointStruct {

	var outpoint = []*domainOutpointToCompare{{
		domainOutpoint: &externalapi.DomainOutpoint{
			externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			1},
		expectedResult: true,
	}, {
		domainOutpoint: &externalapi.DomainOutpoint{
			externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
			1},
		expectedResult: false,
	}, {
		domainOutpoint: &externalapi.DomainOutpoint{
			externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0},
			2},
		expectedResult: false,
	}}
	tests := []testDomainOutpointStruct{
		{
			baseDomainOutpoint: &externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				1},
			domainOutpointToCompareTo: outpoint,
		}, {baseDomainOutpoint: &externalapi.DomainOutpoint{
			externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			1},
			domainOutpointToCompareTo: []*domainOutpointToCompare{{domainOutpoint: nil, expectedResult: false}},
		}, {baseDomainOutpoint: nil,
			domainOutpointToCompareTo: []*domainOutpointToCompare{{domainOutpoint: nil, expectedResult: true}},
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

func initTestDomainTransactionIDForClone() []*externalapi.DomainTransactionID {
	outpoint := []*externalapi.DomainTransactionID{
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

func initTestDomainTransactionIDForEqual() []testDomainTransactionIDStruct {

	var outpoint = []*domainTransactionIDToCompare{{
		domainTransactionID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		expectedResult: true,
	}, {
		domainTransactionID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
		expectedResult: false,
	}, {
		domainTransactionID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0},
		expectedResult: false,
	}}
	tests := []testDomainTransactionIDStruct{
		{
			baseDomainTransactionID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			domainTransactionIDToCompareTo: outpoint,
		}, {
			baseDomainTransactionID: nil,
			domainTransactionIDToCompareTo: []*domainTransactionIDToCompare{{
				domainTransactionID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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
