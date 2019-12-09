// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/kaspanet/kaspad/logger"
	"github.com/kaspanet/kaspad/util/panics"
)

var btcdLog, _ = logger.Get(logger.SubsystemTags.BTCD)
var spawn = panics.GoroutineWrapperFunc(btcdLog)
var srvrLog, _ = logger.Get(logger.SubsystemTags.SRVR)
