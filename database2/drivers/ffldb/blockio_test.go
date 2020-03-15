package ffldb

import (
	"os"
	"testing"

	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/wire"
)

func TestDeleteFile(t *testing.T) {
	testBlock := util.NewBlock(wire.NewMsgBlock(
		wire.NewBlockHeader(1, []*daghash.Hash{}, &daghash.Hash{}, &daghash.Hash{}, &daghash.Hash{}, 0, 0)))

	tests := []struct {
		fileNum     uint32
		expectedErr bool
	}{
		{0, false},
		{1, true},
	}

	for _, test := range tests {
		func() {
			pdb := newTestDb("TestDeleteFile", t)
			defer func() {
				if !pdb.closed {
					pdb.Close()
				}
			}()

			err := pdb.Update(func(dbTx *Transaction) error {
				dbTx.StoreBlock(testBlock)
				return nil
			})
			if err != nil {
				t.Fatalf("TestDeleteFile: Error storing block: %s", err)
			}

			err = pdb.Close()
			if err != nil {
				t.Fatalf("TestDeleteFile: Error closing file before deletion: %s", err)
			}

			err = pdb.store.deleteFile(test.fileNum)
			if (err != nil) != test.expectedErr {
				t.Errorf("TestDeleteFile: %d: Expected error status: %t, but got: %t",
					test.fileNum, test.expectedErr, (err != nil))
			}
			if err == nil {
				filePath := blockFilePath(pdb.store.basePath, test.fileNum)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("TestDeleteFile: %d: File %s still exists", test.fileNum, filePath)
				}
			}
		}()
	}
}

// TestHandleRollbackErrors tests all error-cases in *blockStore.handleRollback().
// The non-error-cases are tested in the more general tests.
// Since handleRollback just logs errors, this test simply causes all error-cases to be hit,
// and makes sure no panic occurs, as well as ensures the writeCursor was updated correctly.
func TestHandleRollbackErrors(t *testing.T) {
	testBlock := util.NewBlock(wire.NewMsgBlock(
		wire.NewBlockHeader(1, []*daghash.Hash{}, &daghash.Hash{}, &daghash.Hash{}, &daghash.Hash{}, 0, 0)))

	testBlockSize := uint32(testBlock.MsgBlock().SerializeSize())
	tests := []struct {
		name    string
		fileNum uint32
		offset  uint32
	}{
		// offset should be size of block + 12 bytes for block network, size and checksum
		{"Nothing to rollback", 1, testBlockSize + 12},
	}

	for _, test := range tests {
		func() {
			pdb := newTestDb("TestHandleRollbackErrors", t)
			defer pdb.Close()

			// Set maxBlockFileSize to testBlockSize so that writeCursor.curFileNum increments
			pdb.store.maxBlockFileSize = testBlockSize

			err := pdb.Update(func(dbTx *Transaction) error {
				return dbTx.StoreBlock(testBlock)
			})
			if err != nil {
				t.Fatalf("TestHandleRollbackErrors: %s: Error adding test block to database: %s", test.name, err)
			}

			pdb.store.handleRollback(test.fileNum, test.offset)

			if pdb.store.writeCursor.curFileNum != test.fileNum {
				t.Errorf("TestHandleRollbackErrors: %s: Expected fileNum: %d, but got: %d",
					test.name, test.fileNum, pdb.store.writeCursor.curFileNum)
			}

			if pdb.store.writeCursor.curOffset != test.offset {
				t.Errorf("TestHandleRollbackErrors: %s: offset fileNum: %d, but got: %d",
					test.name, test.offset, pdb.store.writeCursor.curOffset)
			}
		}()
	}
}
