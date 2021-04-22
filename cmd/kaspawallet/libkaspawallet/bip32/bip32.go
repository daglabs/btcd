package bip32

import "crypto/rand"

func GenerateSeed() ([]byte, error) {
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, err
	}

	return randBytes, nil
}

func NewMasterWithPath(seed []byte, version [4]byte, pathString string) (*ExtendedKey, error) {
	masterKey, err := NewMaster(seed, version)
	if err != nil {
		return nil, err
	}

	return masterKey.DeriveFromPath(pathString)
}

func NewPublicMasterWithPath(seed []byte, version [4]byte, pathString string) (*ExtendedKey, error) {
	masterKey, err := NewMaster(seed, version)
	if err != nil {
		return nil, err
	}

	path, err := parsePath(pathString)
	if err != nil {
		return nil, err
	}

	descendantKey, err := masterKey.path(path)
	if err != nil {
		return nil, err
	}

	return descendantKey.Public()
}
