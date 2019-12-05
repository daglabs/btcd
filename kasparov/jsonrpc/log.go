package jsonrpc

import (
	"github.com/daglabs/btcd/kasparov/logger"
	"github.com/daglabs/btcd/rpcclient"
	"github.com/daglabs/btcd/util/panics"
)

var (
	log   = logger.BackendLog.Logger("RPCC")
	spawn = panics.GoroutineWrapperFunc(log)
)

func init() {
	rpcclient.UseLogger(log)
}
