// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package addressmanager

import (
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/kaspanet/kaspad/wire"
)

func TstKnownAddressIsBad(ka *KnownAddress) bool {
	return ka.isBad()
}

func TstKnownAddressChance(ka *KnownAddress) float64 {
	return ka.chance()
}

func TstNewKnownAddress(na *wire.NetAddress, attempts int,
	lastattempt, lastsuccess mstime.Time, tried bool, refs int) *KnownAddress {
	return &KnownAddress{netAddress: na, attempts: attempts, lastAttempt: lastattempt,
		lastSuccess: lastsuccess, tried: tried, referenceCount: refs}
}
