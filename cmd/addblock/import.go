// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"github.com/kaspanet/kaspad/blockdag/indexers"
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/pkg/errors"
	"io"
	"sync"
	"time"

	"github.com/kaspanet/kaspad/blockdag"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/wire"
)

// importResults houses the stats and result as an import operation.
type importResults struct {
	blocksProcessed int64
	blocksImported  int64
	err             error
}

// blockImporter houses information about an ongoing import from a block data
// file to the block database.
type blockImporter struct {
	dag               *blockdag.BlockDAG
	r                 io.ReadSeeker
	processQueue      chan []byte
	doneChan          chan bool
	errChan           chan error
	quit              chan struct{}
	wg                sync.WaitGroup
	blocksProcessed   int64
	blocksImported    int64
	receivedLogBlocks int64
	receivedLogTx     int64
	lastHeight        int64
	lastBlockTime     mstime.Time
	lastLogTime       mstime.Time
}

// readBlock reads the next block from the input file.
func (bi *blockImporter) readBlock() ([]byte, error) {
	// The block file format is:
	//  <network> <block length> <serialized block>
	var net uint32
	err := binary.Read(bi.r, binary.LittleEndian, &net)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}

		// No block and no error means there are no more blocks to read.
		return nil, nil
	}
	if net != uint32(ActiveConfig().NetParams().Net) {
		return nil, errors.Errorf("network mismatch -- got %x, want %x",
			net, uint32(ActiveConfig().NetParams().Net))
	}

	// Read the block length and ensure it is sane.
	var blockLen uint32
	if err := binary.Read(bi.r, binary.LittleEndian, &blockLen); err != nil {
		return nil, err
	}
	if blockLen > wire.MaxMessagePayload {
		return nil, errors.Errorf("block payload of %d bytes is larger "+
			"than the max allowed %d bytes", blockLen,
			wire.MaxMessagePayload)
	}

	serializedBlock := make([]byte, blockLen)
	if _, err := io.ReadFull(bi.r, serializedBlock); err != nil {
		return nil, err
	}

	return serializedBlock, nil
}

// processBlock potentially imports the block into the database. It first
// deserializes the raw block while checking for errors. Already known blocks
// are skipped and orphan blocks are considered errors. Finally, it runs the
// block through the DAG rules to ensure it follows all rules.
// Returns whether the block was imported along with any potential errors.
func (bi *blockImporter) processBlock(serializedBlock []byte) (bool, error) {
	// Deserialize the block which includes checks for malformed blocks.
	block, err := util.NewBlockFromBytes(serializedBlock)
	if err != nil {
		return false, err
	}

	// update progress statistics
	bi.lastBlockTime = block.MsgBlock().Header.Timestamp
	bi.receivedLogTx += int64(len(block.MsgBlock().Transactions))

	// Skip blocks that already exist.
	blockHash := block.Hash()
	if bi.dag.IsKnownBlock(blockHash) {
		return false, nil
	}

	// Don't bother trying to process orphans.
	parentHashes := block.MsgBlock().Header.ParentHashes
	if len(parentHashes) > 0 {
		if !bi.dag.AreKnownBlocks(parentHashes) {
			return false, errors.Errorf("import file contains block "+
				"%v which does not link to the available "+
				"block DAG", parentHashes)
		}
	}

	// Ensure the blocks follows all of the DAG rules.
	isOrphan, isDelayed, err := bi.dag.ProcessBlock(block,
		blockdag.BFFastAdd)
	if err != nil {
		return false, err
	}
	if isDelayed {
		return false, errors.Errorf("import file contains a block that is too far in the future")
	}
	if isOrphan {
		return false, errors.Errorf("import file contains an orphan "+
			"block: %s", blockHash)
	}

	return true, nil
}

// readHandler is the main handler for reading blocks from the import file.
// This allows block processing to take place in parallel with block reads.
// It must be run as a goroutine.
func (bi *blockImporter) readHandler() {
out:
	for {
		// Read the next block from the file and if anything goes wrong
		// notify the status handler with the error and bail.
		serializedBlock, err := bi.readBlock()
		if err != nil {
			bi.errChan <- errors.Errorf("Error reading from input "+
				"file: %s", err.Error())
			break out
		}

		// A nil block with no error means we're done.
		if serializedBlock == nil {
			break out
		}

		// Send the block or quit if we've been signalled to exit by
		// the status handler due to an error elsewhere.
		select {
		case bi.processQueue <- serializedBlock:
		case <-bi.quit:
			break out
		}
	}

	// Close the processing channel to signal no more blocks are coming.
	close(bi.processQueue)
	bi.wg.Done()
}

// logProgress logs block progress as an information message. In order to
// prevent spam, it limits logging to one message every cfg.Progress seconds
// with duration and totals included.
func (bi *blockImporter) logProgress() {
	bi.receivedLogBlocks++

	now := mstime.Now()
	duration := now.Sub(bi.lastLogTime)
	if duration < time.Second*time.Duration(cfg.Progress) {
		return
	}

	// Truncate the duration to 10s of milliseconds.
	durationMillis := int64(duration / time.Millisecond)
	tDuration := 10 * time.Millisecond * time.Duration(durationMillis/10)

	// Log information about new block height.
	blockStr := "blocks"
	if bi.receivedLogBlocks == 1 {
		blockStr = "block"
	}
	txStr := "transactions"
	if bi.receivedLogTx == 1 {
		txStr = "transaction"
	}
	log.Infof("Processed %d %s in the last %s (%d %s, height %d, %s)",
		bi.receivedLogBlocks, blockStr, tDuration, bi.receivedLogTx,
		txStr, bi.lastHeight, bi.lastBlockTime)

	bi.receivedLogBlocks = 0
	bi.receivedLogTx = 0
	bi.lastLogTime = now
}

// processHandler is the main handler for processing blocks. This allows block
// processing to take place in parallel with block reads from the import file.
// It must be run as a goroutine.
func (bi *blockImporter) processHandler() {
out:
	for {
		select {
		case serializedBlock, ok := <-bi.processQueue:
			// We're done when the channel is closed.
			if !ok {
				break out
			}

			bi.blocksProcessed++
			bi.lastHeight++
			imported, err := bi.processBlock(serializedBlock)
			if err != nil {
				bi.errChan <- err
				break out
			}

			if imported {
				bi.blocksImported++
			}

			bi.logProgress()

		case <-bi.quit:
			break out
		}
	}
	bi.wg.Done()
}

// statusHandler waits for updates from the import operation and notifies
// the passed doneChan with the results of the import. It also causes all
// goroutines to exit if an error is reported from any of them.
func (bi *blockImporter) statusHandler(resultsChan chan *importResults) {
	select {
	// An error from either of the goroutines means we're done so signal
	// caller with the error and signal all goroutines to quit.
	case err := <-bi.errChan:
		resultsChan <- &importResults{
			blocksProcessed: bi.blocksProcessed,
			blocksImported:  bi.blocksImported,
			err:             err,
		}
		close(bi.quit)

	// The import finished normally.
	case <-bi.doneChan:
		resultsChan <- &importResults{
			blocksProcessed: bi.blocksProcessed,
			blocksImported:  bi.blocksImported,
			err:             nil,
		}
	}
}

// Import is the core function which handles importing the blocks from the file
// associated with the block importer to the database. It returns a channel
// on which the results will be returned when the operation has completed.
func (bi *blockImporter) Import() chan *importResults {
	// Start up the read and process handling goroutines. This setup allows
	// blocks to be read from disk in parallel while being processed.
	bi.wg.Add(2)
	spawn("blockImporter.readHandler", bi.readHandler)
	spawn("blockImporter.processHandler", bi.processHandler)

	// Wait for the import to finish in a separate goroutine and signal
	// the status handler when done.
	spawn("blockImporter.sendToDoneChan", func() {
		bi.wg.Wait()
		bi.doneChan <- true
	})

	// Start the status handler and return the result channel that it will
	// send the results on when the import is done.
	resultChan := make(chan *importResults)
	spawn("blockImporter.statusHandler", func() {
		bi.statusHandler(resultChan)
	})
	return resultChan
}

// newBlockImporter returns a new importer for the provided file reader seeker
// and database.
func newBlockImporter(r io.ReadSeeker) (*blockImporter, error) {
	// Create the acceptance index if needed.
	var indexes []indexers.Indexer
	if cfg.AcceptanceIndex {
		log.Info("Acceptance index is enabled")
		indexes = append(indexes, indexers.NewAcceptanceIndex())
	}

	// Create an index manager if any of the optional indexes are enabled.
	var indexManager blockdag.IndexManager
	if len(indexes) > 0 {
		indexManager = indexers.NewManager(indexes)
	}

	dag, err := blockdag.New(&blockdag.Config{
		DAGParams:    ActiveConfig().NetParams(),
		TimeSource:   blockdag.NewTimeSource(),
		IndexManager: indexManager,
	})
	if err != nil {
		return nil, err
	}

	return &blockImporter{
		r:            r,
		processQueue: make(chan []byte, 2),
		doneChan:     make(chan bool),
		errChan:      make(chan error),
		quit:         make(chan struct{}),
		dag:          dag,
		lastLogTime:  mstime.Now(),
	}, nil
}
