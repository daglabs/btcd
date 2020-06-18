package database

// Database defines the interface of a database that can begin
// transactions and close itself.
//
// Important: This is not part of the DataAccessor interface
// because the Transaction interface includes it. Were we to
// merge Database with DataAccessor, implementors of the
// Transaction interface would be forced to implement methods
// such as Begin and Close, which is undesirable.
type Database interface {
	DataAccessor

	// DeleteUpToLocation deletes as much data as it can from the given store, while guaranteeing
	// that the data belongs to dbLocation, its following locations and dbPreservedLocations will
	// be kept.
	DeleteUpToLocation(storeName string, dbLocation StoreLocation,
		dbPreservedLocations []StoreLocation) error

	// Begin begins a new database transaction.
	Begin() (Transaction, error)

	// Close closes the database.
	Close() error
}
