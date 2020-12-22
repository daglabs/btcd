package model_test

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"reflect"
	"testing"
)

func initTestTransactionAcceptanceDataForClone() []*externalapi.TransactionAcceptanceData {

	tests := []*externalapi.TransactionAcceptanceData{
		{
			&externalapi.DomainTransaction{
				Version: 1,
				Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
					externalapi.DomainTransactionID{0x01}, 0xFFFF},
					[]byte{1, 2, 3},
					uint64(0xFFFFFFFF),
					utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
				Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
					[]byte{1, 2}},
					{uint64(0xFFFF),
						[]byte{1, 3}}},
				LockTime:     1,
				SubnetworkID: externalapi.DomainSubnetworkID{0x01},
				Gas:          1,
				PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				Payload: []byte{0x01},
				Fee:     0,
				Mass:    1,
				ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			},
			1,
			true,
		},
	}
	return tests
}

type testTransactionAcceptanceDataToCompare struct {
	transactionAcceptanceData *externalapi.TransactionAcceptanceData
	expectedResult            bool
}

type testTransactionAcceptanceDataStruct struct {
	baseTransactionAcceptanceData        *externalapi.TransactionAcceptanceData
	transactionAcceptanceDataToCompareTo []testTransactionAcceptanceDataToCompare
}

func initTransactionAcceptanceDataForEqual() []testTransactionAcceptanceDataStruct {
	var testTransactionAcceptanceDataBase = externalapi.TransactionAcceptanceData{

		&externalapi.DomainTransaction{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		1,
		true,
	}

	var testTransactionAcceptanceData1 = externalapi.TransactionAcceptanceData{
		&externalapi.DomainTransaction{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		1,
		true,
	}
	// test 2: different transactions
	var testTransactionAcceptanceData2 = externalapi.TransactionAcceptanceData{
		&externalapi.DomainTransaction{
			Version: 2,
			Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		1,
		true,
	}
	//test 3: different Fee
	var testTransactionAcceptanceData3 = externalapi.TransactionAcceptanceData{
		&externalapi.DomainTransaction{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		2,
		true,
	}
	//test 4: different isAccepted
	var testTransactionAcceptanceData4 = externalapi.TransactionAcceptanceData{
		&externalapi.DomainTransaction{
			Version: 1,
			Inputs: []*externalapi.DomainTransactionInput{{externalapi.DomainOutpoint{
				externalapi.DomainTransactionID{0x01}, 0xFFFF},
				[]byte{1, 2, 3},
				uint64(0xFFFFFFFF),
				utxo.NewUTXOEntry(1, []byte{0, 1, 2, 3}, true, 2)}},
			Outputs: []*externalapi.DomainTransactionOutput{{uint64(0xFFFF),
				[]byte{1, 2}},
				{uint64(0xFFFF),
					[]byte{1, 3}}},
			LockTime:     1,
			SubnetworkID: externalapi.DomainSubnetworkID{0x01},
			Gas:          1,
			PayloadHash: externalapi.DomainHash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			Payload: []byte{0x01},
			Fee:     0,
			Mass:    1,
			ID: &externalapi.DomainTransactionID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		},
		1,
		false,
	}

	tests := []testTransactionAcceptanceDataStruct{
		{
			baseTransactionAcceptanceData: &testTransactionAcceptanceDataBase,
			transactionAcceptanceDataToCompareTo: []testTransactionAcceptanceDataToCompare{
				{
					transactionAcceptanceData: &testTransactionAcceptanceData1,
					expectedResult:            true,
				}, {
					transactionAcceptanceData: &testTransactionAcceptanceData2,
					expectedResult:            false,
				}, {
					transactionAcceptanceData: &testTransactionAcceptanceData3,
					expectedResult:            false,
				}, {
					transactionAcceptanceData: &testTransactionAcceptanceData4,
					expectedResult:            false,
				}, {
					transactionAcceptanceData: nil,
					expectedResult:            false,
				},
			},
		}, {
			baseTransactionAcceptanceData: nil,
			transactionAcceptanceDataToCompareTo: []testTransactionAcceptanceDataToCompare{
				{
					transactionAcceptanceData: &testTransactionAcceptanceData1,
					expectedResult:            false,
				}, {
					transactionAcceptanceData: nil,
					expectedResult:            true,
				},
			},
		},
	}
	return tests
}

func TestTransactionAcceptanceData_Equal(t *testing.T) {
	acceptanceData := initTransactionAcceptanceDataForEqual()
	for i, test := range acceptanceData {
		for j, subTest := range test.transactionAcceptanceDataToCompareTo {
			result1 := test.baseTransactionAcceptanceData.Equal(subTest.transactionAcceptanceData)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.transactionAcceptanceData.Equal(test.baseTransactionAcceptanceData)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestTransactionAcceptanceData_Clone(t *testing.T) {

	testTransactionAcceptanceData := initTestTransactionAcceptanceDataForClone()
	for i, transactionAcceptanceData := range testTransactionAcceptanceData {
		transactionAcceptanceDataClone := transactionAcceptanceData.Clone()
		if !transactionAcceptanceDataClone.Equal(transactionAcceptanceData) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(transactionAcceptanceData, transactionAcceptanceDataClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func initTestBlockAcceptanceDataForClone() []*externalapi.BlockAcceptanceData {

	tests := []*externalapi.BlockAcceptanceData{{&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{
			{
				&externalapi.DomainTransaction{
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
				},
				1,
				true,
			}},
	},
	}
	return tests
}

type testBlockAcceptanceDataToCompare struct {
	blockAcceptanceData *externalapi.BlockAcceptanceData
	expectedResult      bool
}

type testBlockAcceptanceDataStruct struct {
	baseBlockAcceptanceData        *externalapi.BlockAcceptanceData
	blockAcceptanceDataToCompareTo []testBlockAcceptanceDataToCompare
}

func iniBlockAcceptanceDataForEqual() []testBlockAcceptanceDataStruct {
	var testBlockAcceptanceDataBase = externalapi.BlockAcceptanceData{
		&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{{
			&externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{
					externalapi.DomainOutpoint{
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
			},
			1,
			true,
		}}}
	//test 1: structs are equal
	var testBlockAcceptanceData1 = externalapi.BlockAcceptanceData{
		&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{{
			&externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{
					externalapi.DomainOutpoint{
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
			},
			1,
			true,
		}}}
	// test 2: different size
	var testBlockAcceptanceData2 = externalapi.BlockAcceptanceData{
		&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{{
			&externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{
					externalapi.DomainOutpoint{
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
			},
			1,
			true,
		}, {}}}
	//test 3: different transactions, same size
	var testBlockAcceptanceData3 = externalapi.BlockAcceptanceData{
		&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{{
			&externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{
					externalapi.DomainOutpoint{
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
			},
			1,
			false,
		}}}

	// test 4 - different block hash
	var testBlockAcceptanceData4 = externalapi.BlockAcceptanceData{
		&externalapi.DomainHash{2},
		[]*externalapi.TransactionAcceptanceData{{
			&externalapi.DomainTransaction{
				1,
				[]*externalapi.DomainTransactionInput{{
					externalapi.DomainOutpoint{
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
			},
			1,
			true,
		}}}

	tests := []testBlockAcceptanceDataStruct{
		{
			baseBlockAcceptanceData: &testBlockAcceptanceDataBase,
			blockAcceptanceDataToCompareTo: []testBlockAcceptanceDataToCompare{
				{
					blockAcceptanceData: &testBlockAcceptanceData1,
					expectedResult:      true,
				}, {
					blockAcceptanceData: &testBlockAcceptanceData2,
					expectedResult:      false,
				}, {
					blockAcceptanceData: &testBlockAcceptanceData3,
					expectedResult:      false,
				}, {
					blockAcceptanceData: nil,
					expectedResult:      false,
				},
				{
					blockAcceptanceData: &testBlockAcceptanceData4,
					expectedResult:      false,
				},
			},
		}, {
			baseBlockAcceptanceData: nil,
			blockAcceptanceDataToCompareTo: []testBlockAcceptanceDataToCompare{
				{
					blockAcceptanceData: &testBlockAcceptanceData1,
					expectedResult:      false,
				}, {
					blockAcceptanceData: nil,
					expectedResult:      true,
				},
			},
		},
	}
	return tests
}

func TestBlockAcceptanceData_Equal(t *testing.T) {

	blockAcceptances := iniBlockAcceptanceDataForEqual()
	for i, test := range blockAcceptances {
		for j, subTest := range test.blockAcceptanceDataToCompareTo {
			result1 := test.baseBlockAcceptanceData.Equal(subTest.blockAcceptanceData)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.blockAcceptanceData.Equal(test.baseBlockAcceptanceData)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestBlockAcceptanceData_Clone(t *testing.T) {

	testBlockAcceptanceData := initTestBlockAcceptanceDataForClone()
	for i, blockAcceptanceData := range testBlockAcceptanceData {
		blockAcceptanceDataClone := blockAcceptanceData.Clone()
		if !blockAcceptanceDataClone.Equal(blockAcceptanceData) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(blockAcceptanceData, blockAcceptanceDataClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}

func initTestAcceptanceDataForClone() []externalapi.AcceptanceData {

	test1 := []*externalapi.BlockAcceptanceData{{
		&externalapi.DomainHash{1},
		[]*externalapi.TransactionAcceptanceData{
			{
				&externalapi.DomainTransaction{
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
				},
				1,
				true,
			}},
	},
	}
	tests := []externalapi.AcceptanceData{test1, test1}
	return tests
}

type testAcceptanceDataToCompare struct {
	acceptanceData externalapi.AcceptanceData
	expectedResult bool
}

type testAcceptanceDataStruct struct {
	baseAcceptanceData        externalapi.AcceptanceData
	acceptanceDataToCompareTo []testAcceptanceDataToCompare
}

func initAcceptanceDataForEqual() []testAcceptanceDataStruct {
	var testAcceptanceDataBase = []*externalapi.BlockAcceptanceData{
		{
			&externalapi.DomainHash{1},
			[]*externalapi.TransactionAcceptanceData{{
				&externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{{
						externalapi.DomainOutpoint{
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
				},
				1,
				true,
			}}}}
	//test 1: structs are equal
	var testAcceptanceData1 = []*externalapi.BlockAcceptanceData{
		{&externalapi.DomainHash{1},
			[]*externalapi.TransactionAcceptanceData{{
				&externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{{
						externalapi.DomainOutpoint{
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
				},
				1,
				true,
			}}}}
	// test 2: different size
	var testAcceptanceData2 = []*externalapi.BlockAcceptanceData{
		{&externalapi.DomainHash{1},
			[]*externalapi.TransactionAcceptanceData{{
				&externalapi.DomainTransaction{
					1,
					[]*externalapi.DomainTransactionInput{{
						externalapi.DomainOutpoint{
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
				},
				1,
				true,
			}}}, {}}
	//test 3: different transactions, same size
	var testAcceptanceData3 = []*externalapi.BlockAcceptanceData{
		{&externalapi.DomainHash{1},
			[]*externalapi.TransactionAcceptanceData{{
				&externalapi.DomainTransaction{
					2,
					[]*externalapi.DomainTransactionInput{{
						externalapi.DomainOutpoint{
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
				},
				1,
				true,
			}}}}

	tests := []testAcceptanceDataStruct{
		{
			baseAcceptanceData: testAcceptanceDataBase,
			acceptanceDataToCompareTo: []testAcceptanceDataToCompare{
				{
					acceptanceData: testAcceptanceData1,
					expectedResult: true,
				}, {
					acceptanceData: testAcceptanceData2,
					expectedResult: false,
				}, {
					acceptanceData: testAcceptanceData3,
					expectedResult: false,
				},
			},
		},
	}
	return tests
}

func TestAcceptanceData_Equal(t *testing.T) {

	acceptances := initAcceptanceDataForEqual()
	for i, test := range acceptances {
		for j, subTest := range test.acceptanceDataToCompareTo {
			result1 := test.baseAcceptanceData.Equal(subTest.acceptanceData)
			if result1 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result1)
			}
			result2 := subTest.acceptanceData.Equal(test.baseAcceptanceData)
			if result2 != subTest.expectedResult {
				t.Fatalf("Test #%d:%d: Expected %t but got %t", i, j, subTest.expectedResult, result2)
			}
		}
	}
}

func TestAcceptanceData_Clone(t *testing.T) {

	testAcceptanceData := initTestAcceptanceDataForClone()
	for i, acceptanceData := range testAcceptanceData {
		acceptanceDataClone := acceptanceData.Clone()
		if !acceptanceDataClone.Equal(acceptanceData) {
			t.Fatalf("Test #%d:[Equal] clone should be equal to the original", i)
		}
		if !reflect.DeepEqual(acceptanceData, acceptanceDataClone) {
			t.Fatalf("Test #%d:[DeepEqual] clone should be equal to the original", i)
		}
	}
}
