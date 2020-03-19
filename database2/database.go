package database2

// Database defines the interface of a generic kaspad database.
type Database interface {
	// Put sets the value for the given key. It overwrites
	// any previous value for that key.
	Put(key []byte, value []byte) error

	// Get gets the value for the given key. It returns an
	// error if the given key does not exist.
	Get(key []byte) ([]byte, error)

	// Has returns true if the database does contains the
	// given key.
	Has(key []byte) (bool, error)

	// AppendFlatData appends the given data to the flat
	// file store defined by storeName. This function
	// returns a serialized location handle that's meant
	// to be stored and later used when querying the data
	// that has just now been inserted.
	AppendFlatData(storeName string, data []byte) ([]byte, error)

	// RetrieveFlatData retrieves data from the flat file
	// stored defined by storeName using the given serialized
	// location handle. See AppendFlatData for further details.
	RetrieveFlatData(storeName string, location []byte) ([]byte, error)

	// CurrentFlatDataLocation returns the serialized
	// location handle to the current location within
	// the flat file store defined storeName. It is mainly
	// to be used to rollback flat file stores in case
	// of data incongruency.
	CurrentFlatDataLocation(storeName string) []byte

	// RollbackFlatData truncates the flat file store defined
	// by the given storeName to the location defined by the
	// given serialized location handle.
	RollbackFlatData(storeName string, location []byte) error
}
