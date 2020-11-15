// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package addressmanager

import (
	"encoding/binary"
	"sync"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/pkg/errors"
)

// AddressRandomizer is the interface for the randomizer needed for the AddressManager.
type AddressRandomizer interface {
	RandomAddress(addresses []*appmessage.NetAddress) *appmessage.NetAddress
	RandomAddresses(addresses []*appmessage.NetAddress, count int) []*appmessage.NetAddress
}

// AddressKey represents a "string" key of the ip addresses
// for use as keys in maps.
type AddressKey string

// ErrAddressNotFound is an error returned from some functions when a
// given address is not found in the address manager
var ErrAddressNotFound = errors.New("address not found")

// NetAddressKey returns a key of the ip address to use it in maps.
func netAddressKey(netAddress *appmessage.NetAddress) AddressKey {
	port := make([]byte, 2, 2)
	binary.LittleEndian.PutUint16(port, netAddress.Port)

	key := make([]byte, len(netAddress.IP), len(netAddress.IP)+len(port))
	copy(key, netAddress.IP)

	return AddressKey(append(key, port...))
}

// netAddressKeys returns a key of the ip address to use it in maps.
func netAddressesKeys(netAddresses []*appmessage.NetAddress) map[AddressKey]bool {
	result := make(map[AddressKey]bool, len(netAddresses))
	for _, netAddress := range netAddresses {
		key := netAddressKey(netAddress)
		result[key] = true
	}

	return result
}

// AddressManager provides a concurrency safe address manager for caching potential
// peers on the Kaspa network.
type AddressManager struct {
	addresses       map[AddressKey]*appmessage.NetAddress
	bannedAddresses map[AddressKey]*appmessage.NetAddress
	localAddresses  *localAddressManager
	mutex           sync.Mutex
	cfg             *Config
	random          AddressRandomizer
}

// New returns a new Kaspa address manager.
func New(cfg *Config) (*AddressManager, error) {
	localAddresses, err := newLocalAddressManager(cfg)
	if err != nil {
		return nil, err
	}

	return &AddressManager{
		addresses:       map[AddressKey]*appmessage.NetAddress{},
		bannedAddresses: map[AddressKey]*appmessage.NetAddress{},
		localAddresses:  localAddresses,
		random:          NewAddressRandomize(),
		cfg:             cfg,
	}, nil
}

func (am *AddressManager) addAddressNoLock(address *appmessage.NetAddress) {
	if !IsRoutable(address, am.cfg.AcceptUnroutable) {
		return
	}

	key := netAddressKey(address)
	_, ok := am.addresses[key]
	if !ok {
		am.addresses[key] = address
	}
}

// AddAddress adds address to the address manager
func (am *AddressManager) AddAddress(address *appmessage.NetAddress) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.addAddressNoLock(address)
}

// AddAddresses adds addresses to the address manager
func (am *AddressManager) AddAddresses(addresses ...*appmessage.NetAddress) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, address := range addresses {
		am.addAddressNoLock(address)
	}
}

// RemoveAddress removes addresses from the address manager
func (am *AddressManager) RemoveAddress(address *appmessage.NetAddress) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	delete(am.addresses, key)
	delete(am.bannedAddresses, key)
}

// Addresses returns all addresses
func (am *AddressManager) Addresses() []*appmessage.NetAddress {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	result := make([]*appmessage.NetAddress, 0, len(am.addresses))
	for _, address := range am.addresses {
		result = append(result, address)
	}

	return result
}

// BannedAddresses returns all banned addresses
func (am *AddressManager) BannedAddresses() []*appmessage.NetAddress {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	result := make([]*appmessage.NetAddress, 0, len(am.bannedAddresses))
	for _, address := range am.bannedAddresses {
		result = append(result, address)
	}

	return result
}

// notBannedAddressesWithException returns all not banned addresses with excpetion
func (am *AddressManager) notBannedAddressesWithException(exceptions []*appmessage.NetAddress) []*appmessage.NetAddress {
	exceptionsKeys := netAddressesKeys(exceptions)
	am.mutex.Lock()
	defer am.mutex.Unlock()

	result := make([]*appmessage.NetAddress, 0, len(am.addresses))
	for key, address := range am.addresses {
		if !exceptionsKeys[key] {
			result = append(result, address)
		}
	}

	return result
}

// RandomAddress returns a random address that isn't banned and isn't in exceptions
func (am *AddressManager) RandomAddress(exceptions []*appmessage.NetAddress) *appmessage.NetAddress {
	validAddresses := am.notBannedAddressesWithException(exceptions)
	return am.random.RandomAddress(validAddresses)
}

// RandomAddresses returns count addresses at random that aren't banned and aren't in exceptions
func (am *AddressManager) RandomAddresses(count int, exceptions []*appmessage.NetAddress) []*appmessage.NetAddress {
	validAddresses := am.notBannedAddressesWithException(exceptions)
	return am.random.RandomAddresses(validAddresses, count)
}

// BestLocalAddress returns the most appropriate local address to use
// for the given remote address.
func (am *AddressManager) BestLocalAddress(remoteAddress *appmessage.NetAddress) *appmessage.NetAddress {
	return am.localAddresses.bestLocalAddress(remoteAddress)
}

// Ban marks the given address as banned
func (am *AddressManager) Ban(address *appmessage.NetAddress) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	addressToBan, ok := am.addresses[key]
	if !ok {
		return errors.Wrapf(ErrAddressNotFound, "address %s "+
			"is not registered with the address manager", address.TCPAddress())
	}

	delete(am.addresses, key)
	am.bannedAddresses[key] = addressToBan
	return nil

}

// Unban unmarks the given address as banned
func (am *AddressManager) Unban(address *appmessage.NetAddress) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	bannedAddress, ok := am.bannedAddresses[key]
	if !ok {
		return errors.Wrapf(ErrAddressNotFound, "address %s "+
			"is not registered with the address manager as banned", address.TCPAddress())
	}

	delete(am.bannedAddresses, key)
	am.addresses[key] = bannedAddress
	return nil
}

// IsBanned returns true if the given address is marked as banned
func (am *AddressManager) IsBanned(address *appmessage.NetAddress) (bool, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	if _, ok := am.bannedAddresses[key]; !ok {
		if _, ok = am.addresses[key]; !ok {
			return false, errors.Wrapf(ErrAddressNotFound, "address %s "+
				"is not registered with the address manager", address.TCPAddress())
		}
		return false, nil
	}

	return true, nil

}
