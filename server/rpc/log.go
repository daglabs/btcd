// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/daglabs/btcd/logs"
	"github.com/daglabs/btcd/logger"
	"github.com/daglabs/btcd/util/panics"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log logs.Logger
var spawn func(func())

func init() {
	log, _ = logger.Get(logger.SubsystemTags.RPCS)
	spawn = panics.GoroutineWrapperFunc(log)
}
