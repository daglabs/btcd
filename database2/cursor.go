package database2

// Cursor iterates over database entries given some bucket.
type Cursor interface {
	// Next moves the iterator to the next key/value pair. It returns whether the
	// iterator is exhausted. Returns false if the cursor is closed.
	Next() bool

	// Error returns any accumulated error. Exhausting all the key/value pairs
	// is not considered to be an error.
	Error() error

	// First moves the iterator to the first key/value pair. It returns whether
	// such pair exist.
	First() (bool, error)

	// Seek moves the iterator to the first key/value pair whose key is greater
	// than or equal to the given key. It returns whether such pair exist.
	Seek(key []byte) (bool, error)

	// Key returns the key of the current key/value pair, or nil if done. The caller
	// should not modify the contents of the returned slice, and its contents may
	// change on the next call to Next.
	Key() ([]byte, error)

	// Value returns the value of the current key/value pair, or nil if done. The
	// caller should not modify the contents of the returned slice, and its contents
	// may change on the next call to Next.
	Value() ([]byte, error)

	// Close releases associated resources.
	Close() error
}
