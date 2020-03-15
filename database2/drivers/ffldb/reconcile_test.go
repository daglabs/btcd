package ffldb

import (
	"reflect"
	"testing"
)

func TestSerializeWriteRow(t *testing.T) {
	tests := []struct {
		curBlockFileNum  uint32
		curFileOffset    uint32
		expectedWriteRow []byte
	}{
		// WriteRow format:
		// 		First 4 bytes: curBlockFileNum
		//		Next  4 bytes: curFileOffset
		//		Next  4 bytes: Castagnoli CRC-32 checksum
		// One can easily calculate checksums using the following code:
		// https://play.golang.org/p/zoMKT-ORyF9
		{0, 0, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x8A, 0xB2, 0x28, 0x8C}},
		{10, 11, []byte{0x0A, 0x00, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00, 0xC1, 0xA6, 0x0D, 0xC8}},
	}

	for i, test := range tests {
		actualWriteRow := serializeWriteRow(test.curBlockFileNum, test.curFileOffset)

		if !reflect.DeepEqual(test.expectedWriteRow, actualWriteRow) {
			t.Errorf("TestSerializeWriteRow: %d: Expected: %v, but got: %v",
				i, test.expectedWriteRow, actualWriteRow)
		}
	}
}

func TestDeserializeWriteRow(t *testing.T) {
	tests := []struct {
		writeRow                []byte
		expectedCurBlockFileNum uint32
		expectedCurFileOffset   uint32
		expectedError           bool
	}{
		// WriteRow format:
		// 		First 4 bytes: curBlockFileNum
		//		Next  4 bytes: curFileOffset
		//		Next  4 bytes: Castagnoli CRC-32 checksum
		// One can easily calculate checksums using the following code:
		// https://play.golang.org/p/zoMKT-ORyF9
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x8A, 0xB2, 0x28, 0x8C}, 0, 0, false},
		{[]byte{0x0A, 0x00, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00, 0xC1, 0xA6, 0x0D, 0xC8}, 10, 11, false},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x8A, 0xB2, 0x28, 0x8D}, 0, 0, true},
		{[]byte{0x0A, 0x00, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0, 0, true},
	}

	for i, test := range tests {
		actualCurBlockFileNum, actualCurFileOffset, err := deserializeWriteRow(test.writeRow)

		if (err != nil) != test.expectedError {
			t.Errorf("TestDeserializeWriteRow: %d: Expected error status: %t, but got: %t",
				i, test.expectedError, err != nil)
		}

		if test.expectedCurBlockFileNum != actualCurBlockFileNum {
			t.Errorf("TestDeserializeWriteRow: %d: Expected curBlockFileNum: %d, but got: %d",
				i, test.expectedCurBlockFileNum, actualCurBlockFileNum)
		}

		if test.expectedCurFileOffset != actualCurFileOffset {
			t.Errorf("TestDeserializeWriteRow: %d: Expected curFileOffset: %d, but got: %d",
				i, test.expectedCurFileOffset, actualCurFileOffset)
		}
	}
}

// setWriteRow is a low-level helper method to update the write row in the
// metadata bucket to enable certain test-cases in TestReconcileErrors
// if writeRow = nil deletes the write row altogether
func setWriteRow(pdb *DB, writeRow []byte, t *testing.T) {
	tx, err := pdb.begin(true)
	if err != nil {
		t.Fatalf("TestReconcileErrors: Error getting tx for setting "+
			"writeLoc in metadata: %s", err)
	}

	if writeRow == nil {
		tx.metaBucket.Delete(writeLocKeyName)
		if err != nil {
			t.Fatalf("TestReconcileErrors: Error deleting writeLoc from metadata: %s",
				err)
		}
	} else {
		tx.metaBucket.Put(writeLocKeyName, writeRow)
		if err != nil {
			t.Fatalf("TestReconcileErrors: Error updating writeLoc in metadata: %s",
				err)
		}
	}

	err = pdb.cache.commitTx(tx)
	if err != nil {
		t.Fatalf("TestReconcileErrors: Error commiting the update of "+
			"writeLoc in metadata: %s", err)
	}

	pdb.writeLock.Unlock()
	pdb.closeLock.RUnlock()
}

// TestReconcileErrors tests all error-cases in reconclieDB.
// The non-error-cases are tested in the more general tests.
func TestReconcileErrors(t *testing.T) {
	// Set-up tests
	pdb := newTestDb("TestReconcileErrors", t)

	// Test without writeLoc
	setWriteRow(pdb, nil, t)
	_, err := reconcileDB(pdb, false)
	if err == nil {
		t.Errorf("TestReconcileErrors: ReconcileDB() didn't error out when " +
			"running without a writeRowLoc")
	}

	// Test with writeLoc in metadata after the actual cursor position
	setWriteRow(pdb, serializeWriteRow(1, 0), t)
	_, err = reconcileDB(pdb, false)
	if err == nil {
		t.Errorf("TestReconcileErrors: ReconcileDB() didn't error out when " +
			"curBlockFileNum after the actual cursor position")
	}
	setWriteRow(pdb, serializeWriteRow(0, 1), t)
	_, err = reconcileDB(pdb, false)
	if err == nil {
		t.Errorf("TestReconcileErrors: ReconcileDB() didn't error out when " +
			"curFileOffset after the actual cursor position")
	}

	// Restore previous writeRow
	setWriteRow(pdb, serializeWriteRow(0, 0), t)

	// Test create with closed DB to force initDB to fail
	pdb.Close()
	_, err = reconcileDB(pdb, true)
	if err == nil {
		t.Errorf("ReconcileDB didn't error out when running with closed DB and create = true")
	}
}
