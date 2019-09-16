package main

import (
	"fmt"

	"github.com/daglabs/btcd/apiserver/config"
	"github.com/daglabs/btcd/apiserver/database"
	"github.com/daglabs/btcd/apiserver/jsonrpc"
	"github.com/daglabs/btcd/apiserver/server"
	"github.com/daglabs/btcd/logger"
	"github.com/daglabs/btcd/signal"
	"github.com/daglabs/btcd/util/panics"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func main() {
	defer panics.HandlePanic(log, logger.BackendLog)

	cfg, err := config.Parse()
	if err != nil {
		panic(fmt.Errorf("Error parsing command-line arguments: %s", err))
	}

	err = database.Connect(cfg)
	if err != nil {
		panic(fmt.Errorf("Error connecting to database: %s", err))
	}
	defer func() {
		err := database.Close()
		if err != nil {
			panic(fmt.Errorf("Error closing the database: %s", err))
		}
	}()

	err = jsonrpc.Connect(cfg)
	if err != nil {
		panic(fmt.Errorf("Error connecting to servers: %s", err))
	}
	defer jsonrpc.Close()

	shutdownServer := server.Start(cfg.HTTPListen)
	defer shutdownServer()

	doneChan := make(chan struct{}, 1)
	spawn(func() {
		err := startSync(doneChan)
		if err != nil {
			panic(err)
		}
	})

	interrupt := signal.InterruptListener()
	<-interrupt

	// Gracefully stop syncing
	doneChan <- struct{}{}
}
