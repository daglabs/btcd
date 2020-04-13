// All tests within this file should call testForAllDatabaseTypes
// over the actual test. This is to make sure that all supported
// database types adhere to the interfaces defined in this package.

package database_test

import (
	"bytes"
	"github.com/kaspanet/kaspad/database"
	"testing"
)

func TestTransactionPut(t *testing.T) {
	testForAllDatabaseTypes(t, "TestTransactionPut", testTransactionPut)
}

func testTransactionPut(t *testing.T, db database.Database, testName string) {
	// Begin a new transaction
	dbTx, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	defer func() {
		err := dbTx.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
	}()

	// Put value1 into the transaction
	key := database.MakeBucket().Key([]byte("key"))
	value1 := []byte("value1")
	err = dbTx.Put(key, value1)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Put value2 into the transaction with the same key
	value2 := []byte("value2")
	err = dbTx.Put(key, value2)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Commit the transaction
	err = dbTx.Commit()
	if err != nil {
		t.Fatalf("%s: Commit "+
			"unexpectedly failed: %s", testName, err)
	}

	// Make sure that the returned value is value2
	returnedValue, err := db.Get(key)
	if err != nil {
		t.Fatalf("%s: Get "+
			"unexpectedly failed: %s", testName, err)
	}
	if !bytes.Equal(returnedValue, value2) {
		t.Fatalf("%s: Get "+
			"returned wrong value. Want: %s, got: %s",
			testName, string(value2), string(returnedValue))
	}
}

func TestTransactionGet(t *testing.T) {
	testForAllDatabaseTypes(t, "TestTransactionGet", testTransactionGet)
}

func testTransactionGet(t *testing.T, db database.Database, testName string) {
	// Put a value into the database
	key1 := database.MakeBucket().Key([]byte("key1"))
	value1 := []byte("value1")
	err := db.Put(key1, value1)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Begin a new transaction
	dbTx, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	defer func() {
		err := dbTx.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
	}()

	// Get the value back and make sure it's the same one
	returnedValue, err := dbTx.Get(key1)
	if err != nil {
		t.Fatalf("%s: Get "+
			"unexpectedly failed: %s", testName, err)
	}
	if !bytes.Equal(returnedValue, value1) {
		t.Fatalf("%s: Get "+
			"returned wrong value. Want: %s, got: %s",
			testName, string(value1), string(returnedValue))
	}

	// Try getting a non-existent value and make sure
	// the returned error is ErrNotFound
	_, err = dbTx.Get(database.MakeBucket().Key([]byte("doesn't exist")))
	if err == nil {
		t.Fatalf("%s: Get "+
			"unexpectedly succeeded", testName)
	}
	if !database.IsNotFoundError(err) {
		t.Fatalf("%s: Get "+
			"returned wrong error: %s", testName, err)
	}

	// Put a new value into the database outside of the transaction
	key2 := database.MakeBucket().Key([]byte("key2"))
	value2 := []byte("value2")
	err = db.Put(key2, value2)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Make sure that the new value doesn't exist inside the transaction
	_, err = dbTx.Get(key2)
	if err == nil {
		t.Fatalf("%s: Get "+
			"unexpectedly succeeded", testName)
	}
	if !database.IsNotFoundError(err) {
		t.Fatalf("%s: Get "+
			"returned wrong error: %s", testName, err)
	}
}

func TestTransactionHas(t *testing.T) {
	testForAllDatabaseTypes(t, "TestTransactionHas", testTransactionHas)
}

func testTransactionHas(t *testing.T, db database.Database, testName string) {
	// Put a value into the database
	key1 := database.MakeBucket().Key([]byte("key1"))
	value1 := []byte("value1")
	err := db.Put(key1, value1)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Begin a new transaction
	dbTx, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	defer func() {
		err := dbTx.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
	}()

	// Make sure that Has returns true for the value we just put
	exists, err := dbTx.Has(key1)
	if err != nil {
		t.Fatalf("%s: Has "+
			"unexpectedly failed: %s", testName, err)
	}
	if !exists {
		t.Fatalf("%s: Has "+
			"unexpectedly returned that the value does not exist", testName)
	}

	// Make sure that Has returns false for a non-existent value
	exists, err = dbTx.Has(database.MakeBucket().Key([]byte("doesn't exist")))
	if err != nil {
		t.Fatalf("%s: Has "+
			"unexpectedly failed: %s", testName, err)
	}
	if exists {
		t.Fatalf("%s: Has "+
			"unexpectedly returned that the value exists", testName)
	}

	// Put a new value into the database outside of the transaction
	key2 := database.MakeBucket().Key([]byte("key2"))
	value2 := []byte("value2")
	err = db.Put(key2, value2)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Make sure that the new value doesn't exist inside the transaction
	exists, err = dbTx.Has(key2)
	if err != nil {
		t.Fatalf("%s: Has "+
			"unexpectedly failed: %s", testName, err)
	}
	if exists {
		t.Fatalf("%s: Has "+
			"unexpectedly returned that the value exists", testName)
	}
}

func TestTransactionDelete(t *testing.T) {
	testForAllDatabaseTypes(t, "TestTransactionDelete", testTransactionDelete)
}

func testTransactionDelete(t *testing.T, db database.Database, testName string) {
	// Put a value into the database
	key := database.MakeBucket().Key([]byte("key"))
	value := []byte("value")
	err := db.Put(key, value)
	if err != nil {
		t.Fatalf("%s: Put "+
			"unexpectedly failed: %s", testName, err)
	}

	// Begin two new transactions
	dbTx1, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	dbTx2, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	defer func() {
		err := dbTx1.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
		err = dbTx2.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
	}()

	// Delete the value in the first transaction
	err = dbTx1.Delete(key)
	if err != nil {
		t.Fatalf("%s: Delete "+
			"unexpectedly failed: %s", testName, err)
	}

	// Commit the first transaction
	err = dbTx1.Commit()
	if err != nil {
		t.Fatalf("%s: Commit "+
			"unexpectedly failed: %s", testName, err)
	}

	// Make sure that Has returns false for the deleted value
	exists, err := db.Has(key)
	if err != nil {
		t.Fatalf("%s: Has "+
			"unexpectedly failed: %s", testName, err)
	}
	if exists {
		t.Fatalf("%s: Has "+
			"unexpectedly returned that the value exists", testName)
	}

	// Make sure that the second transaction was no affected
	exists, err = dbTx2.Has(key)
	if err != nil {
		t.Fatalf("%s: Has "+
			"unexpectedly failed: %s", testName, err)
	}
	if !exists {
		t.Fatalf("%s: Has "+
			"unexpectedly returned that the value does not exist", testName)
	}
}

func TestTransactionAppendToStoreAndRetrieveFromStore(t *testing.T) {
	testForAllDatabaseTypes(t, "TestTransactionAppendToStoreAndRetrieveFromStore", testTransactionAppendToStoreAndRetrieveFromStore)
}

func testTransactionAppendToStoreAndRetrieveFromStore(t *testing.T, db database.Database, testName string) {
	// Begin a new transaction
	dbTx, err := db.Begin()
	if err != nil {
		t.Fatalf("%s: Begin "+
			"unexpectedly failed: %s", testName, err)
	}
	defer func() {
		err := dbTx.RollbackUnlessClosed()
		if err != nil {
			t.Fatalf("%s: RollbackUnlessClosed "+
				"unexpectedly failed: %s", testName, err)
		}
	}()

	// Append some data into the store
	storeName := "store"
	data := []byte("data")
	location, err := dbTx.AppendToStore(storeName, data)
	if err != nil {
		t.Fatalf("%s: AppendToStore "+
			"unexpectedly failed: %s", testName, err)
	}

	// Retrieve the data and make sure it's equal to what was appended
	retrievedData, err := dbTx.RetrieveFromStore(storeName, location)
	if err != nil {
		t.Fatalf("%s: RetrieveFromStore "+
			"unexpectedly failed: %s", testName, err)
	}
	if !bytes.Equal(retrievedData, data) {
		t.Fatalf("%s: RetrieveFromStore "+
			"returned unexpected data. Want: %s, got: %s",
			testName, string(data), string(retrievedData))
	}

	// Make sure that an invalid location returns ErrNotFound
	fakeLocation := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	_, err = dbTx.RetrieveFromStore(storeName, fakeLocation)
	if err == nil {
		t.Fatalf("%s: RetrieveFromStore "+
			"unexpectedly succeeded", testName)
	}
	if !database.IsNotFoundError(err) {
		t.Fatalf("%s: RetrieveFromStore "+
			"returned wrong error: %s", testName, err)
	}
}
