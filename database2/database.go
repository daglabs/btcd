package database2

// Database defines the interface of a database that can begin
// transactions, open cursors, and close itself.
//
// Important: This is not part of the DataAccessor interface
// because the Transaction interface includes it. Were we to
// merge Database with DataAccessor, implementors of the
// Transaction interface would be forced to implement methods
// such as Begin and Close, which is undesirable.
type Database interface {
	// A handle to the database needs to be able to do
	// anything that the underlying database can do.
	DataAccessor

	// Begin begins a new database transaction.
	Begin() (Transaction, error)

	// Cursor begins a new cursor over the given bucket.
	Cursor(bucket []byte) (Cursor, error)

	// Close closes the database.
	Close() error
}
