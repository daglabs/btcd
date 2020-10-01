// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package addressmanager

import (
	"sync"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/infrastructure/network/randomaddress"
	"github.com/pkg/errors"
)

// AddressRandomizer is the interface for the randomizer needed for the AddressManager.
type AddressRandomizer interface {
	RandomAddress(addresses []*appmessage.NetAddress) *appmessage.NetAddress
	RandomAddresses(addresses []*appmessage.NetAddress, count int) []*appmessage.NetAddress
}

type addressEntry struct {
	netAddress *appmessage.NetAddress
}

// AddressKey represents a "string" key of the ip addresses
// for use as keys in maps.
type AddressKey string

// ErrAddressNotFound is an error returned from some functions when a
// given address is not found in the address manager
var ErrAddressNotFound = errors.New("address not found")

// NetAddressKey returns a key of the ip address to use it in maps.
func netAddressKey(netAddress *appmessage.NetAddress) AddressKey {
	key := make([]byte, len(netAddress.IP), len(netAddress.IP)+2)
	copy(key, netAddress.IP)
	return AddressKey(append(key, byte(netAddress.Port), byte(netAddress.Port>>8)))
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
	addresses       map[AddressKey]*addressEntry
	bannedAddresses map[AddressKey]*addressEntry
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
		addresses:       map[AddressKey]*addressEntry{},
		bannedAddresses: map[AddressKey]*addressEntry{},
		localAddresses:  localAddresses,
		random:          randomaddress.NewAddressRandomize(),
		cfg:             cfg,
	}, nil
}

// AddAddresses adds addresses to the address manager
func (am *AddressManager) AddAddresses(addresses ...*appmessage.NetAddress) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, address := range addresses {
		if !IsRoutable(address, am.cfg.AcceptUnroutable) {
			continue
		}

		key := netAddressKey(address)
		_, ok := am.addresses[key]

		if !ok {
			am.addresses[key] = &addressEntry{
				netAddress: address,
			}
		}
	}
}

// RemoveAddresses removes addresses from the address manager
func (am *AddressManager) RemoveAddresses(addresses ...*appmessage.NetAddress) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, address := range addresses {
		key := netAddressKey(address)
		delete(am.addresses, key)
	}
}

// Addresses returns all addresses
func (am *AddressManager) Addresses() []*appmessage.NetAddress {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	result := make([]*appmessage.NetAddress, 0, len(am.addresses))
	for _, address := range am.addresses {
		result = append(result, address.netAddress)
	}
	return result
}

// NotBannedAddressesWithException returns all not banned addresses with excpetion
func (am *AddressManager) NotBannedAddressesWithException(exceptions []*appmessage.NetAddress) []*appmessage.NetAddress {
	exceptionsKeys := netAddressesKeys(exceptions)
	am.mutex.Lock()
	defer am.mutex.Unlock()

	result := make([]*appmessage.NetAddress, 0, len(am.addresses))
	for key, address := range am.addresses {
		if !exceptionsKeys[key] {
			result = append(result, address.netAddress)
		}
	}

	return result
}

// RandomAddress returns a random address that isn't banned and isn't in exceptions
func (am *AddressManager) RandomAddress(exceptions []*appmessage.NetAddress) *appmessage.NetAddress {
	validAddresses := am.NotBannedAddressesWithException(exceptions)
	return am.random.RandomAddress(validAddresses)
}

// RandomAddresses returns count addresses at random that aren't banned and aren't in exceptions
func (am *AddressManager) RandomAddresses(count int, exceptions []*appmessage.NetAddress) []*appmessage.NetAddress {
	validAddresses := am.NotBannedAddressesWithException(exceptions)
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
	addressEntry, ok := am.addresses[key]

	if ok {
		delete(am.addresses, key)
		am.bannedAddresses[key] = addressEntry
		return nil
	}

	return errors.Wrapf(ErrAddressNotFound, "address %s "+
		"is not registered with the address manager", address.TCPAddress())
}

// Unban unmarks the given address as banned
func (am *AddressManager) Unban(address *appmessage.NetAddress) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	addressEntry, ok := am.bannedAddresses[key]

	if ok {
		delete(am.bannedAddresses, key)
		am.addresses[key] = addressEntry
		return nil
	}

	return errors.Wrapf(ErrAddressNotFound, "address %s "+
		"is not registered with the address manager", address.TCPAddress())
}

// IsBanned returns true if the given address is marked as banned
func (am *AddressManager) IsBanned(address *appmessage.NetAddress) (bool, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	key := netAddressKey(address)
	_, ok := am.bannedAddresses[key]
	if ok {
		return true, nil
	}

	_, ok = am.addresses[key]
	if ok {
		return false, nil
	}

	return false, errors.Wrapf(ErrAddressNotFound, "address %s "+
		"is not registered with the address manager", address.TCPAddress())
}
