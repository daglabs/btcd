package main

import (
	"github.com/daglabs/kaspad/kasparov/logger"
	"github.com/daglabs/kaspad/util/panics"
)

var (
	log   = logger.Logger("KVSV")
	spawn = panics.GoroutineWrapperFunc(log)
)
