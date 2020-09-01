// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/kaspanet/kaspad/util/mstime"

	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/util/random"
	"github.com/kaspanet/kaspad/util/subnetworkid"

	"golang.org/x/crypto/ripemd160"

	"github.com/btcsuite/websocket"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/infrastructure/network/rpc/model"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
)

const (
	// websocketSendBufferSize is the number of elements the send channel
	// can queue before blocking. Note that this only applies to requests
	// handled directly in the websocket client input handler or the async
	// handler since notifications have their own queuing mechanism
	// independent of the send channel buffer.
	websocketSendBufferSize = 50
)

type semaphore chan struct{}

func makeSemaphore(n int) semaphore {
	return make(chan struct{}, n)
}

func (s semaphore) acquire() { s <- struct{}{} }
func (s semaphore) release() { <-s }

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

// wsCommandHandler describes a callback function used to handle a specific
// command.
type wsCommandHandler func(*wsClient, interface{}) (interface{}, error)

// wsHandlers maps RPC command strings to appropriate websocket handler
// functions. This is set by init because help references wsHandlers and thus
// causes a dependency loop.
var wsHandlers map[string]wsCommandHandler
var wsHandlersBeforeInit = map[string]wsCommandHandler{
	"loadTxFilter":              handleLoadTxFilter,
	"help":                      handleWebsocketHelp,
	"notifyBlocks":              handleNotifyBlocks,
	"notifyChainChanges":        handleNotifyChainChanges,
	"notifyNewTransactions":     handleNotifyNewTransactions,
	"session":                   handleSession,
	"stopNotifyBlocks":          handleStopNotifyBlocks,
	"stopNotifyChainChanges":    handleStopNotifyChainChanges,
	"stopNotifyNewTransactions": handleStopNotifyNewTransactions,
}

// WebsocketHandler handles a new websocket client by creating a new wsClient,
// starting it, and blocking until the connection closes. Since it blocks, it
// must be run in a separate goroutine. It should be invoked from the websocket
// server handler which runs each new connection in a new goroutine thereby
// satisfying the requirement.
func (s *Server) WebsocketHandler(conn *websocket.Conn, remoteAddr string,
	authenticated bool, isAdmin bool) {

	// Clear the read deadline that was set before the websocket hijacked
	// the connection.
	conn.SetReadDeadline(timeZeroVal)

	// Limit max number of websocket clients.
	log.Infof("New websocket client %s", remoteAddr)
	if s.ntfnMgr.NumClients()+1 > s.cfg.RPCMaxWebsockets {
		log.Infof("Max websocket clients exceeded [%d] - "+
			"disconnecting client %s", s.cfg.RPCMaxWebsockets,
			remoteAddr)
		conn.Close()
		return
	}

	// Create a new websocket client to handle the new websocket connection
	// and wait for it to shutdown. Once it has shutdown (and hence
	// disconnected), remove it and any notifications it registered for.
	client, err := newWebsocketClient(s, conn, remoteAddr, authenticated, isAdmin)
	if err != nil {
		log.Errorf("Failed to serve client %s: %s", remoteAddr, err)
		conn.Close()
		return
	}
	s.ntfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	s.ntfnMgr.RemoveClient(client)
	log.Infof("Disconnected websocket client %s", remoteAddr)
}

// wsNotificationManager is a connection and notification manager used for
// websockets. It allows websocket clients to register for notifications they
// are interested in. When an event happens elsewhere in the code such as
// transactions being added to the memory pool or block connects/disconnects,
// the notification manager is provided with the relevant details needed to
// figure out which websocket clients need to be notified based on what they
// have registered for and notifies them accordingly. It is also used to keep
// track of all connected websocket clients.
type wsNotificationManager struct {
	// server is the RPC server the notification manager is associated with.
	server *Server

	// queueNotification queues a notification for handling.
	queueNotification chan interface{}

	// notificationMsgs feeds notificationHandler with notifications
	// and client (un)registeration requests from a queue as well as
	// registeration and unregisteration requests from clients.
	notificationMsgs chan interface{}

	// Access channel for current number of connected clients.
	numClients chan int

	// Shutdown handling
	wg   sync.WaitGroup
	quit chan struct{}
}

// queueHandler manages a queue of empty interfaces, reading from in and
// sending the oldest unsent to out. This handler stops when either of the
// in or quit channels are closed, and closes out before returning, without
// waiting to send any variables still remaining in the queue.
func queueHandler(in <-chan interface{}, out chan<- interface{}, quit <-chan struct{}) {
	var q []interface{}
	var dequeue chan<- interface{}
	skipQueue := out
	var next interface{}
out:
	for {
		select {
		case n, ok := <-in:
			if !ok {
				// Sender closed input channel.
				break out
			}

			// Either send to out immediately if skipQueue is
			// non-nil (queue is empty) and reader is ready,
			// or append to the queue and send later.
			select {
			case skipQueue <- n:
			default:
				q = append(q, n)
				dequeue = out
				skipQueue = nil
				next = q[0]
			}

		case dequeue <- next:
			copy(q, q[1:])
			q[len(q)-1] = nil // avoid leak
			q = q[:len(q)-1]
			if len(q) == 0 {
				dequeue = nil
				skipQueue = out
			} else {
				next = q[0]
			}

		case <-quit:
			break out
		}
	}
	close(out)
}

// queueHandler maintains a queue of notifications and notification handler
// control messages.
func (m *wsNotificationManager) queueHandler() {
	queueHandler(m.queueNotification, m.notificationMsgs, m.quit)
	m.wg.Done()
}

// NotifyBlockAdded passes a block newly-added to the blockDAG
// to the notification manager for block and transaction notification
// processing.
func (m *wsNotificationManager) NotifyBlockAdded(block *util.Block) {
	// As NotifyBlockAdded will be called by the block manager
	// and the RPC server may no longer be running, use a select
	// statement to unblock enqueuing the notification once the RPC
	// server has begun shutting down.
	select {
	case m.queueNotification <- (*notificationBlockAdded)(block):
	case <-m.quit:
	}
}

// NotifyChainChanged passes changes to the selected parent chain of
// the blockDAG to the notification manager for processing.
func (m *wsNotificationManager) NotifyChainChanged(removedChainBlockHashes []*daghash.Hash,
	addedChainBlockHashes []*daghash.Hash) {
	n := &notificationChainChanged{
		removedChainBlockHashes: removedChainBlockHashes,
		addedChainBlocksHashes:  addedChainBlockHashes,
	}
	// As NotifyChainChanged will be called by the DAG manager
	// and the RPC server may no longer be running, use a select
	// statement to unblock enqueuing the notification once the RPC
	// server has begun shutting down.
	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

func (m *wsNotificationManager) NotifyFinalityConflict(violatingBlockHash *daghash.Hash, conflictTime mstime.Time) {
	n := notificationFinalityConflict{
		violatingBlockHash: violatingBlockHash,
		conflictTime:       conflictTime,
	}
	// As NotifyFinalityConflict will be called by the DAG manager
	// and the RPC server may no longer be running, use a select
	// statement to unblock enqueuing the notification once the RPC
	// server has begun shutting down.
	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

func (m *wsNotificationManager) NotifyFinalityConflictResolved(
	finalityBlockHash *daghash.Hash, resolutionTime mstime.Time) {

	n := notificationFinalityConflictResolved{
		finalityBlockHash: finalityBlockHash,
		resolutionTime:    resolutionTime,
	}
	// As NotifyFinalityConflictResolved will be called by the DAG manager
	// and the RPC server may no longer be running, use a select
	// statement to unblock enqueuing the notification once the RPC
	// server has begun shutting down.
	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

// NotifyMempoolTx passes a transaction accepted by mempool to the
// notification manager for transaction notification processing. If
// isNew is true, the tx is is a new transaction, rather than one
// added to the mempool during a reorg.
func (m *wsNotificationManager) NotifyMempoolTx(tx *util.Tx, isNew bool) {
	n := &notificationTxAcceptedByMempool{
		isNew: isNew,
		tx:    tx,
	}

	// As NotifyMempoolTx will be called by mempool and the RPC server
	// may no longer be running, use a select statement to unblock
	// enqueuing the notification once the RPC server has begun
	// shutting down.
	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

// wsClientFilter tracks relevant addresses for each websocket client for
// the `rescanBlocks` extension. It is modified by the `loadTxFilter` command.
//
// NOTE: This extension was ported from github.com/decred/dcrd
type wsClientFilter struct {
	mu sync.Mutex

	// Implemented fast paths for address lookup.
	pubKeyHashes        map[[ripemd160.Size]byte]struct{}
	scriptHashes        map[[ripemd160.Size]byte]struct{}
	compressedPubKeys   map[[33]byte]struct{}
	uncompressedPubKeys map[[65]byte]struct{}

	// A fallback address lookup map in case a fast path doesn't exist.
	// Only exists for completeness. If using this shows up in a profile,
	// there's a good chance a fast path should be added.
	otherAddresses map[string]struct{}

	// Outpoints of unspent outputs.
	unspent map[appmessage.Outpoint]struct{}
}

// newWSClientFilter creates a new, empty wsClientFilter struct to be used
// for a websocket client.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func newWSClientFilter(addresses []string, unspentOutpoints []appmessage.Outpoint, params *dagconfig.Params) *wsClientFilter {
	filter := &wsClientFilter{
		pubKeyHashes:        map[[ripemd160.Size]byte]struct{}{},
		scriptHashes:        map[[ripemd160.Size]byte]struct{}{},
		compressedPubKeys:   map[[33]byte]struct{}{},
		uncompressedPubKeys: map[[65]byte]struct{}{},
		otherAddresses:      map[string]struct{}{},
		unspent:             make(map[appmessage.Outpoint]struct{}, len(unspentOutpoints)),
	}

	for _, s := range addresses {
		filter.addAddressStr(s, params)
	}
	for i := range unspentOutpoints {
		filter.addUnspentOutpoint(&unspentOutpoints[i])
	}

	return filter
}

// addAddress adds an address to a wsClientFilter, treating it correctly based
// on the type of address passed as an argument.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func (f *wsClientFilter) addAddress(a util.Address) {
	switch a := a.(type) {
	case *util.AddressPubKeyHash:
		f.pubKeyHashes[*a.Hash160()] = struct{}{}
		return
	case *util.AddressScriptHash:
		f.scriptHashes[*a.Hash160()] = struct{}{}
		return
	}

	f.otherAddresses[a.EncodeAddress()] = struct{}{}
}

// addAddressStr parses an address from a string and then adds it to the
// wsClientFilter using addAddress.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func (f *wsClientFilter) addAddressStr(s string, params *dagconfig.Params) {
	// If address can't be decoded, no point in saving it since it should also
	// impossible to create the address from an inspected transaction output
	// script.
	a, err := util.DecodeAddress(s, params.Prefix)
	if err != nil {
		return
	}
	f.addAddress(a)
}

// existsAddress returns true if the passed address has been added to the
// wsClientFilter.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func (f *wsClientFilter) existsAddress(a util.Address) bool {
	switch a := a.(type) {
	case *util.AddressPubKeyHash:
		_, ok := f.pubKeyHashes[*a.Hash160()]
		return ok
	case *util.AddressScriptHash:
		_, ok := f.scriptHashes[*a.Hash160()]
		return ok
	}

	_, ok := f.otherAddresses[a.EncodeAddress()]
	return ok
}

// addUnspentOutpoint adds an outpoint to the wsClientFilter.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func (f *wsClientFilter) addUnspentOutpoint(op *appmessage.Outpoint) {
	f.unspent[*op] = struct{}{}
}

// existsUnspentOutpointNoLock returns true if the passed outpoint has been added to
// the wsClientFilter.
//
// NOTE: This extension was ported from github.com/decred/dcrd
func (f *wsClientFilter) existsUnspentOutpointNoLock(op *appmessage.Outpoint) bool {
	_, ok := f.unspent[*op]
	return ok
}

func (f *wsClientFilter) existsUnspentOutpoint(op *appmessage.Outpoint) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.existsUnspentOutpointNoLock(op)
}

// Notification types
type notificationBlockAdded util.Block
type notificationChainChanged struct {
	removedChainBlockHashes []*daghash.Hash
	addedChainBlocksHashes  []*daghash.Hash
}

type notificationTxAcceptedByMempool struct {
	isNew bool
	tx    *util.Tx
}

type notificationFinalityConflict struct {
	violatingBlockHash *daghash.Hash
	conflictTime       mstime.Time
}

type notificationFinalityConflictResolved struct {
	finalityBlockHash *daghash.Hash
	resolutionTime    mstime.Time
}

// Notification control requests
type notificationRegisterClient wsClient
type notificationUnregisterClient wsClient
type notificationRegisterBlocks wsClient
type notificationUnregisterBlocks wsClient
type notificationRegisterChainChanges wsClient
type notificationUnregisterChainChanges wsClient
type notificationRegisterFinalityConflicts wsClient
type notificationUnregisterFinalityConflicts wsClient
type notificationRegisterNewMempoolTxs wsClient
type notificationUnregisterNewMempoolTxs wsClient

// notificationHandler reads notifications and control messages from the queue
// handler and processes one at a time.
func (m *wsNotificationManager) notificationHandler() {
	// clients is a map of all currently connected websocket clients.
	clients := make(map[chan struct{}]*wsClient)

	// Maps used to hold lists of websocket clients to be notified on
	// certain events. Each websocket client also keeps maps for the events
	// which have multiple triggers to make removal from these lists on
	// connection close less horrendously expensive.
	//
	// Where possible, the quit channel is used as the unique id for a client
	// since it is quite a bit more efficient than using the entire struct.
	blockNotifications := make(map[chan struct{}]*wsClient)
	chainChangeNotifications := make(map[chan struct{}]*wsClient)
	txNotifications := make(map[chan struct{}]*wsClient)

out:
	for {
		select {
		case n, ok := <-m.notificationMsgs:
			if !ok {
				// queueHandler quit.
				break out
			}
			switch n := n.(type) {
			case *notificationBlockAdded:
				block := (*util.Block)(n)

				if len(blockNotifications) != 0 {
					m.notifyFilteredBlockAdded(blockNotifications,
						block)
				}

			case *notificationChainChanged:
				m.notifyChainChanged(chainChangeNotifications,
					n.removedChainBlockHashes, n.addedChainBlocksHashes)

			case *notificationTxAcceptedByMempool:
				if n.isNew && len(txNotifications) != 0 {
					m.notifyForNewTx(txNotifications, n.tx)
				}
				m.notifyRelevantTxAccepted(n.tx, clients)

			case *notificationRegisterBlocks:
				wsc := (*wsClient)(n)
				blockNotifications[wsc.quit] = wsc

			case *notificationUnregisterBlocks:
				wsc := (*wsClient)(n)
				delete(blockNotifications, wsc.quit)

			case *notificationRegisterChainChanges:
				wsc := (*wsClient)(n)
				chainChangeNotifications[wsc.quit] = wsc

			case *notificationUnregisterChainChanges:
				wsc := (*wsClient)(n)
				delete(chainChangeNotifications, wsc.quit)

			case *notificationRegisterFinalityConflicts:
				wsc := (*wsClient)(n)
				chainChangeNotifications[wsc.quit] = wsc

			case *notificationUnregisterFinalityConflicts:
				wsc := (*wsClient)(n)
				delete(chainChangeNotifications, wsc.quit)

			case *notificationRegisterClient:
				wsc := (*wsClient)(n)
				clients[wsc.quit] = wsc

			case *notificationUnregisterClient:
				wsc := (*wsClient)(n)
				// Remove any requests made by the client as well as
				// the client itself.
				delete(blockNotifications, wsc.quit)
				delete(chainChangeNotifications, wsc.quit)
				delete(txNotifications, wsc.quit)
				delete(clients, wsc.quit)

			case *notificationRegisterNewMempoolTxs:
				wsc := (*wsClient)(n)
				txNotifications[wsc.quit] = wsc

			case *notificationUnregisterNewMempoolTxs:
				wsc := (*wsClient)(n)
				delete(txNotifications, wsc.quit)

			default:
				log.Warn("Unhandled notification type")
			}

		case m.numClients <- len(clients):

		case <-m.quit:
			// RPC server shutting down.
			break out
		}
	}

	for _, c := range clients {
		c.Disconnect()
	}
	m.wg.Done()
}

// NumClients returns the number of clients actively being served.
func (m *wsNotificationManager) NumClients() (n int) {
	select {
	case n = <-m.numClients:
	case <-m.quit: // Use default n (0) if server has shut down.
	}
	return
}

// RegisterBlockUpdates requests block update notifications to the passed
// websocket client.
func (m *wsNotificationManager) RegisterBlockUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterBlocks)(wsc)
}

// UnregisterBlockUpdates removes block update notifications for the passed
// websocket client.
func (m *wsNotificationManager) UnregisterBlockUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterBlocks)(wsc)
}

// RegisterChainChanges requests chain change notifications to the passed
// websocket client.
func (m *wsNotificationManager) RegisterChainChanges(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterChainChanges)(wsc)
}

// UnregisterChainChanges removes chain change notifications for the passed
// websocket client.
func (m *wsNotificationManager) UnregisterChainChanges(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterChainChanges)(wsc)
}

// RegisterFinalityConflicts requests chain change notifications to the passed
// websocket client.
func (m *wsNotificationManager) RegisterFinalityConflicts(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterFinalityConflicts)(wsc)
}

// UnregisterFinalityConflicts removes chain change notifications for the passed
// websocket client.
func (m *wsNotificationManager) UnregisterFinalityConflicts(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterFinalityConflicts)(wsc)
}

// notifyChainChanged notifies websocket clients that have registered for
// chain changes.
func (m *wsNotificationManager) notifyChainChanged(clients map[chan struct{}]*wsClient,
	removedChainBlockHashes []*daghash.Hash, addedChainBlockHashes []*daghash.Hash) {

	// Collect removed chain hashes.
	removedChainHashesStrs := make([]string, len(removedChainBlockHashes))
	for i, hash := range removedChainBlockHashes {
		removedChainHashesStrs[i] = hash.String()
	}

	// Collect added chain blocks.
	addedChainBlocks, err := collectChainBlocks(m.server, addedChainBlockHashes)
	if err != nil {
		log.Errorf("Failed to collect chain blocks: %s", err)
		return
	}

	// Create the notification.
	ntfn := model.NewChainChangedNtfn(removedChainHashesStrs, addedChainBlocks)

	var marshalledJSON []byte
	if len(clients) != 0 {
		// Marshal notification
		var err error
		marshalledJSON, err = model.MarshalCommand(nil, ntfn)
		if err != nil {
			log.Errorf("Failed to marshal chain changed "+
				"notification: %s", err)
			return
		}
	}

	for _, wsc := range clients {
		// Queue notification.
		wsc.QueueNotification(marshalledJSON)
	}
}

// subscribedClients returns the set of all websocket client quit channels that
// are registered to receive notifications regarding tx, either due to tx
// spending a watched output or outputting to a watched address. Matching
// client's filters are updated based on this transaction's outputs and output
// addresses that may be relevant for a client.
func (m *wsNotificationManager) subscribedClients(tx *util.Tx,
	clients map[chan struct{}]*wsClient) map[chan struct{}]struct{} {

	// Use a map of client quit channels as keys to prevent duplicates when
	// multiple inputs and/or outputs are relevant to the client.
	subscribed := make(map[chan struct{}]struct{})

	msgTx := tx.MsgTx()
	for _, input := range msgTx.TxIn {
		for quitChan, wsc := range clients {
			filter := wsc.FilterData()
			if filter == nil {
				continue
			}
			if filter.existsUnspentOutpoint(&input.PreviousOutpoint) {
				subscribed[quitChan] = struct{}{}
			}
		}
	}

	for i, output := range msgTx.TxOut {
		_, addr, err := txscript.ExtractScriptPubKeyAddress(
			output.ScriptPubKey, m.server.dag.Params)
		if err != nil || addr == nil {
			// Clients are not able to subscribe to
			// nonstandard or non-address outputs.
			continue
		}
		for quitChan, wsc := range clients {
			filter := wsc.FilterData()
			if filter == nil {
				continue
			}
			func() {
				filter.mu.Lock()
				defer filter.mu.Unlock()
				if filter.existsAddress(addr) {
					subscribed[quitChan] = struct{}{}
					op := appmessage.Outpoint{
						TxID:  *tx.ID(),
						Index: uint32(i),
					}
					filter.addUnspentOutpoint(&op)
				}
			}()
		}
	}

	return subscribed
}

// notifyFilteredBlockAdded notifies websocket clients that have registered for
// block updates when a block is added to the blockDAG.
func (m *wsNotificationManager) notifyFilteredBlockAdded(clients map[chan struct{}]*wsClient,
	block *util.Block) {

	// Create the common portion of the notification that is the same for
	// every client.
	var w bytes.Buffer
	err := block.MsgBlock().Header.Serialize(&w)
	if err != nil {
		log.Errorf("Failed to serialize header for filtered block "+
			"added notification: %s", err)
		return
	}
	blueScore, err := block.BlueScore()
	if err != nil {
		log.Errorf("Failed to deserialize blue score for filtered block "+
			"added notification: %s", err)
		return
	}
	ntfn := model.NewFilteredBlockAddedNtfn(blueScore, hex.EncodeToString(w.Bytes()), nil)

	// Search for relevant transactions for each client and save them
	// serialized in hex encoding for the notification.
	subscribedTxs := make(map[chan struct{}][]string)
	for _, tx := range block.Transactions() {
		var txHex string
		for quitChan := range m.subscribedClients(tx, clients) {
			if txHex == "" {
				txHex = txHexString(tx.MsgTx())
			}
			subscribedTxs[quitChan] = append(subscribedTxs[quitChan], txHex)
		}
	}
	for quitChan, wsc := range clients {
		// Add all discovered transactions for this client. For clients
		// that have no new-style filter, add the empty string slice.
		ntfn.SubscribedTxs = subscribedTxs[quitChan]

		// Marshal and queue notification.
		marshalledJSON, err := model.MarshalCommand(nil, ntfn)
		if err != nil {
			log.Errorf("Failed to marshal filtered block "+
				"connected notification: %s", err)
			return
		}
		wsc.QueueNotification(marshalledJSON)
	}
}

// RegisterNewMempoolTxsUpdates requests notifications to the passed websocket
// client when new transactions are added to the memory pool.
func (m *wsNotificationManager) RegisterNewMempoolTxsUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterNewMempoolTxs)(wsc)
}

// UnregisterNewMempoolTxsUpdates removes notifications to the passed websocket
// client when new transaction are added to the memory pool.
func (m *wsNotificationManager) UnregisterNewMempoolTxsUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterNewMempoolTxs)(wsc)
}

// notifyForNewTx notifies websocket clients that have registered for updates
// when a new transaction is added to the memory pool.
func (m *wsNotificationManager) notifyForNewTx(clients map[chan struct{}]*wsClient, tx *util.Tx) {
	txIDStr := tx.ID().String()
	mtx := tx.MsgTx()

	var amount uint64
	for _, txOut := range mtx.TxOut {
		amount += txOut.Value
	}

	ntfn := model.NewTxAcceptedNtfn(txIDStr, util.Amount(amount).ToKAS())
	marshalledJSON, err := model.MarshalCommand(nil, ntfn)
	if err != nil {
		log.Errorf("Failed to marshal tx notification: %s", err.Error())
		return
	}

	// To avoid unnecessary marshalling of verbose transactions, only initialize
	// marshalledJSONVerboseFull and marshalledJSONVerbosePartial if required.
	// Note: both are initialized at the same time
	// Note: for simplicity's sake, this operation modifies mtx in place
	var marshalledJSONVerboseFull []byte
	var marshalledJSONVerbosePartial []byte
	initializeMarshalledJSONVerbose := func() bool {
		net := m.server.dag.Params
		build := func() ([]byte, bool) {
			rawTx, err := createTxRawResult(net, mtx, txIDStr, nil, "", nil, true)
			if err != nil {
				return nil, false
			}
			verboseNtfn := model.NewTxAcceptedVerboseNtfn(*rawTx)
			marshalledJSONVerbose, err := model.MarshalCommand(nil, verboseNtfn)
			if err != nil {
				log.Errorf("Failed to marshal verbose tx notification: %s", err.Error())
				return nil, false
			}

			return marshalledJSONVerbose, true
		}

		// First, build the given mtx for a Full version of the transaction
		var ok bool
		marshalledJSONVerboseFull, ok = build()
		if !ok {
			return false
		}

		// Second, modify the given mtx to make it partial
		mtx.Payload = []byte{}

		// Third, build again, now with the modified mtx, for a Partial version
		marshalledJSONVerbosePartial, ok = build()
		return ok
	}

	for _, wsc := range clients {
		if wsc.verboseTxUpdates {
			if marshalledJSONVerboseFull == nil {
				ok := initializeMarshalledJSONVerbose()
				if !ok {
					return
				}
			}

			nodeSubnetworkID := m.server.dag.SubnetworkID()
			if wsc.subnetworkIDForTxUpdates == nil || wsc.subnetworkIDForTxUpdates.IsEqual(nodeSubnetworkID) {
				wsc.QueueNotification(marshalledJSONVerboseFull)
			} else {
				wsc.QueueNotification(marshalledJSONVerbosePartial)
			}
		} else {
			wsc.QueueNotification(marshalledJSON)
		}
	}
}

// txHexString returns the serialized transaction encoded in hexadecimal.
func txHexString(tx *appmessage.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	// Ignore Serialize's error, as writing to a bytes.buffer cannot fail.
	tx.Serialize(buf)
	return hex.EncodeToString(buf.Bytes())
}

// notifyRelevantTxAccepted examines the inputs and outputs of the passed
// transaction, notifying websocket clients of outputs spending to a watched
// address and inputs spending a watched outpoint. Any outputs paying to a
// watched address result in the output being watched as well for future
// notifications.
func (m *wsNotificationManager) notifyRelevantTxAccepted(tx *util.Tx,
	clients map[chan struct{}]*wsClient) {

	clientsToNotify := m.subscribedClients(tx, clients)

	if len(clientsToNotify) != 0 {
		n := model.NewRelevantTxAcceptedNtfn(txHexString(tx.MsgTx()))
		marshalled, err := model.MarshalCommand(nil, n)
		if err != nil {
			log.Errorf("Failed to marshal notification: %s", err)
			return
		}
		for quitChan := range clientsToNotify {
			clients[quitChan].QueueNotification(marshalled)
		}
	}
}

// AddClient adds the passed websocket client to the notification manager.
func (m *wsNotificationManager) AddClient(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterClient)(wsc)
}

// RemoveClient removes the passed websocket client and all notifications
// registered for it.
func (m *wsNotificationManager) RemoveClient(wsc *wsClient) {
	select {
	case m.queueNotification <- (*notificationUnregisterClient)(wsc):
	case <-m.quit:
	}
}

// Start starts the goroutines required for the manager to queue and process
// websocket client notifications.
func (m *wsNotificationManager) Start() {
	m.wg.Add(2)
	spawn("wsNotificationManager.queueHandler", m.queueHandler)
	spawn("wsNotificationManager.notificationHandler", m.notificationHandler)
}

// WaitForShutdown blocks until all notification manager goroutines have
// finished.
func (m *wsNotificationManager) WaitForShutdown() {
	m.wg.Wait()
}

// Shutdown shuts down the manager, stopping the notification queue and
// notification handler goroutines.
func (m *wsNotificationManager) Shutdown() {
	close(m.quit)
}

// newWsNotificationManager returns a new notification manager ready for use.
// See wsNotificationManager for more details.
func newWsNotificationManager(server *Server) *wsNotificationManager {
	return &wsNotificationManager{
		server:            server,
		queueNotification: make(chan interface{}),
		notificationMsgs:  make(chan interface{}),
		numClients:        make(chan int),
		quit:              make(chan struct{}),
	}
}

// wsResponse houses a message to send to a connected websocket client as
// well as a channel to reply on when the message is sent.
type wsResponse struct {
	msg      []byte
	doneChan chan bool
}

// wsClient provides an abstraction for handling a websocket client. The
// overall data flow is split into 3 main goroutines, a possible 4th goroutine
// for long-running operations (only started if request is made), and a
// websocket manager which is used to allow things such as broadcasting
// requested notifications to all connected websocket clients. Inbound
// messages are read via the inHandler goroutine and generally dispatched to
// their own handler. However, certain potentially long-running operations such
// as rescans, are sent to the asyncHander goroutine and are limited to one at a
// time. There are two outbound message types - one for responding to client
// requests and another for async notifications. Responses to client requests
// use SendMessage which employs a buffered channel thereby limiting the number
// of outstanding requests that can be made. Notifications are sent via
// QueueNotification which implements a queue via notificationQueueHandler to
// ensure sending notifications from other subsystems can't block. Ultimately,
// all messages are sent via the outHandler.
type wsClient struct {
	sync.Mutex

	// server is the RPC server that is servicing the client.
	server *Server

	// conn is the underlying websocket connection.
	conn *websocket.Conn

	// disconnected indicated whether or not the websocket client is
	// disconnected.
	disconnected bool

	// addr is the remote address of the client.
	addr string

	// authenticated specifies whether a client has been authenticated
	// and therefore is allowed to communicated over the websocket.
	authenticated bool

	// isAdmin specifies whether a client may change the state of the server;
	// false means its access is only to the limited set of RPC calls.
	isAdmin bool

	// sessionID is a random ID generated for each client when connected.
	// These IDs may be queried by a client using the session RPC. A change
	// to the session ID indicates that the client reconnected.
	sessionID uint64

	// verboseTxUpdates specifies whether a client has requested verbose
	// information about all new transactions.
	verboseTxUpdates bool

	// subnetworkIDForTxUpdates specifies whether a client has requested to receive
	// new transaction information from a specific subnetwork.
	subnetworkIDForTxUpdates *subnetworkid.SubnetworkID

	// filterData is the new generation transaction filter backported from
	// github.com/decred/dcrd for the new backported `loadTxFilter` and
	// `rescanBlocks` methods.
	filterData *wsClientFilter

	// Networking infrastructure.
	serviceRequestSem semaphore
	ntfnChan          chan []byte
	sendChan          chan wsResponse
	quit              chan struct{}
	wg                sync.WaitGroup
}

// inHandler handles all incoming messages for the websocket connection. It
// must be run as a goroutine.
func (c *wsClient) inHandler() {
out:
	for {
		// Break out of the loop once the quit channel has been closed.
		// Use a non-blocking select here so we fall through otherwise.
		select {
		case <-c.quit:
			break out
		default:
		}

		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			// Log the error if it's not due to disconnecting.
			if err != io.EOF {
				log.Errorf("Websocket receive error from "+
					"%s: %s", c.addr, err)
			}
			break out
		}

		var request model.Request
		err = json.Unmarshal(msg, &request)
		if err != nil {
			if !c.authenticated {
				break out
			}

			jsonErr := &model.RPCError{
				Code:    model.ErrRPCParse.Code,
				Message: "Failed to parse request: " + err.Error(),
			}
			reply, err := createMarshalledReply(nil, nil, jsonErr)
			if err != nil {
				log.Errorf("Failed to marshal parse failure "+
					"reply: %s", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

		// The JSON-RPC 1.0 spec defines that notifications must have their "id"
		// set to null and states that notifications do not have a response.
		//
		// A JSON-RPC 2.0 notification is a request with "json-rpc":"2.0", and
		// without an "id" member. The specification states that notifications
		// must not be responded to. JSON-RPC 2.0 permits the null value as a
		// valid request id, therefore such requests are not notifications.
		//
		// Kaspad does not respond to any request without an "id" or "id":null,
		// regardless the indicated JSON-RPC protocol version.
		if request.ID == nil {
			if !c.authenticated {
				break out
			}
			continue
		}

		cmd := parseCmd(&request)
		if cmd.err != nil {
			if !c.authenticated {
				break out
			}

			reply, err := createMarshalledReply(cmd.id, nil, cmd.err)
			if err != nil {
				log.Errorf("Failed to marshal parse failure "+
					"reply: %s", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}
		log.Debugf("Websocket server received command <%s> from %s", cmd.method, c.addr)

		// Check auth. The client is immediately disconnected if the
		// first request of an unauthentiated websocket client is not
		// the authenticate request, an authenticate request is received
		// when the client is already authenticated, or incorrect
		// authentication credentials are provided in the request.
		switch authCmd, ok := cmd.cmd.(*model.AuthenticateCmd); {
		case c.authenticated && ok:
			log.Warnf("Websocket client %s is already authenticated",
				c.addr)
			break out
		case !c.authenticated && !ok:
			log.Warnf("Unauthenticated websocket message " +
				"received")
			break out
		case !c.authenticated:
			// Check credentials.
			login := authCmd.Username + ":" + authCmd.Passphrase
			auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
			authSha := sha256.Sum256([]byte(auth))
			cmp := subtle.ConstantTimeCompare(authSha[:], c.server.authsha[:])
			limitcmp := subtle.ConstantTimeCompare(authSha[:], c.server.limitauthsha[:])
			if cmp != 1 && limitcmp != 1 {
				log.Warnf("Auth failure.")
				break out
			}
			c.authenticated = true
			c.isAdmin = cmp == 1

			// Marshal and send response.
			reply, err := createMarshalledReply(cmd.id, nil, nil)
			if err != nil {
				log.Errorf("Failed to marshal authenticate reply: "+
					"%s", err.Error())
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

		// Check if the client is using limited RPC credentials and
		// error when not authorized to call this RPC.
		if !c.isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr := &model.RPCError{
					Code:    model.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
				// Marshal and send response.
				reply, err := createMarshalledReply(request.ID, nil, jsonErr)
				if err != nil {
					log.Errorf("Failed to marshal parse failure "+
						"reply: %s", err)
					continue
				}
				c.SendMessage(reply, nil)
				continue
			}
		}

		// Asynchronously handle the request. A semaphore is used to
		// limit the number of concurrent requests currently being
		// serviced. If the semaphore can not be acquired, simply wait
		// until a request finished before reading the next RPC request
		// from the websocket client.
		//
		// This could be a little fancier by timing out and erroring
		// when it takes too long to service the request, but if that is
		// done, the read of the next request should not be blocked by
		// this semaphore, otherwise the next request will be read and
		// will probably sit here for another few seconds before timing
		// out as well. This will cause the total timeout duration for
		// later requests to be much longer than the check here would
		// imply.
		//
		// If a timeout is added, the semaphore acquiring should be
		// moved inside of the new goroutine with a select statement
		// that also reads a time.After channel. This will unblock the
		// read of the next request from the websocket client and allow
		// many requests to be waited on concurrently.
		c.serviceRequestSem.acquire()
		spawn("wsClient.inHandler-serviceRequest", func() {
			c.serviceRequest(cmd)
			c.serviceRequestSem.release()
		})
	}

	// Ensure the connection is closed.
	c.Disconnect()
	c.wg.Done()
	log.Tracef("Websocket client input handler done for %s", c.addr)
}

// serviceRequest services a parsed RPC request by looking up and executing the
// appropriate RPC handler. The response is marshalled and sent to the
// websocket client.
func (c *wsClient) serviceRequest(r *parsedRPCCmd) {
	var (
		result interface{}
		err    error
	)

	// Lookup the websocket extension for the command and if it doesn't
	// exist fallback to handling the command as a standard command.
	wsHandler, ok := wsHandlers[r.method]
	if ok {
		result, err = wsHandler(c, r.cmd)
	} else {
		result, err = c.server.standardCmdResult(r, nil)
	}
	reply, err := createMarshalledReply(r.id, result, err)
	if err != nil {
		log.Errorf("Failed to marshal reply for <%s> "+
			"command: %s", r.method, err)
		return
	}
	c.SendMessage(reply, nil)
}

// notificationQueueHandler handles the queuing of outgoing notifications for
// the websocket client. This runs as a muxer for various sources of input to
// ensure that queuing up notifications to be sent will not block. Otherwise,
// slow clients could bog down the other systems (such as the mempool or block
// manager) which are queuing the data. The data is passed on to outHandler to
// actually be written. It must be run as a goroutine.
func (c *wsClient) notificationQueueHandler() {
	ntfnSentChan := make(chan bool, 1) // nonblocking sync

	// pendingNtfns is used as a queue for notifications that are ready to
	// be sent once there are no outstanding notifications currently being
	// sent. The waiting flag is used over simply checking for items in the
	// pending list to ensure cleanup knows what has and hasn't been sent
	// to the outHandler. Currently no special cleanup is needed, however
	// if something like a done channel is added to notifications in the
	// future, not knowing what has and hasn't been sent to the outHandler
	// (and thus who should respond to the done channel) would be
	// problematic without using this approach.
	pendingNtfns := list.New()
	waiting := false
out:
	for {
		select {
		// This channel is notified when a message is being queued to
		// be sent across the network socket. It will either send the
		// message immediately if a send is not already in progress, or
		// queue the message to be sent once the other pending messages
		// are sent.
		case msg := <-c.ntfnChan:
			if !waiting {
				c.SendMessage(msg, ntfnSentChan)
			} else {
				pendingNtfns.PushBack(msg)
			}
			waiting = true

			// This channel is notified when a notification has been sent
			// across the network socket.
		case <-ntfnSentChan:
			// No longer waiting if there are no more messages in
			// the pending messages queue.
			next := pendingNtfns.Front()
			if next == nil {
				waiting = false
				continue
			}

			// Notify the outHandler about the next item to
			// asynchronously send.
			msg := pendingNtfns.Remove(next).([]byte)
			c.SendMessage(msg, ntfnSentChan)

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case <-c.ntfnChan:
		case <-ntfnSentChan:
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("Websocket client notification queue handler done "+
		"for %s", c.addr)
}

// outHandler handles all outgoing messages for the websocket connection. It
// must be run as a goroutine. It uses a buffered channel to serialize output
// messages while allowing the sender to continue running asynchronously. It
// must be run as a goroutine.
func (c *wsClient) outHandler() {
out:
	for {
		// Send any messages ready for send until the quit channel is
		// closed.
		select {
		case r := <-c.sendChan:
			err := c.conn.WriteMessage(websocket.TextMessage, r.msg)
			if err != nil {
				c.Disconnect()
				break out
			}
			if r.doneChan != nil {
				r.doneChan <- true
			}

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case r := <-c.sendChan:
			if r.doneChan != nil {
				r.doneChan <- false
			}
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("Websocket client output handler done for %s", c.addr)
}

// SendMessage sends the passed json to the websocket client. It is backed
// by a buffered channel, so it will not block until the send channel is full.
// Note however that QueueNotification must be used for sending async
// notifications instead of the this function. This approach allows a limit to
// the number of outstanding requests a client can make without preventing or
// blocking on async notifications.
func (c *wsClient) SendMessage(marshalledJSON []byte, doneChan chan bool) {
	// Don't send the message if disconnected.
	if c.Disconnected() {
		if doneChan != nil {
			doneChan <- false
		}
		return
	}

	c.sendChan <- wsResponse{msg: marshalledJSON, doneChan: doneChan}
}

// ErrClientQuit describes the error where a client send is not processed due
// to the client having already been disconnected or dropped.
var ErrClientQuit = errors.New("client quit")

// QueueNotification queues the passed notification to be sent to the websocket
// client. This function, as the name implies, is only intended for
// notifications since it has additional logic to prevent other subsystems, such
// as the memory pool and block manager, from blocking even when the send
// channel is full.
//
// If the client is in the process of shutting down, this function returns
// ErrClientQuit. This is intended to be checked by long-running notification
// handlers to stop processing if there is no more work needed to be done.
func (c *wsClient) QueueNotification(marshalledJSON []byte) error {
	// Don't queue the message if disconnected.
	if c.Disconnected() {
		return ErrClientQuit
	}

	c.ntfnChan <- marshalledJSON
	return nil
}

// Disconnected returns whether or not the websocket client is disconnected.
func (c *wsClient) Disconnected() bool {
	c.Lock()
	defer c.Unlock()
	return c.disconnected
}

// Disconnect disconnects the websocket client.
func (c *wsClient) Disconnect() {
	c.Lock()
	defer c.Unlock()

	// Nothing to do if already disconnected.
	if c.disconnected {
		return
	}

	log.Tracef("Disconnecting websocket client %s", c.addr)
	close(c.quit)
	c.conn.Close()
	c.disconnected = true
}

// Start begins processing input and output messages.
func (c *wsClient) Start() {
	log.Tracef("Starting websocket client %s", c.addr)

	// Start processing input and output.
	c.wg.Add(3)
	spawn("wsClient.inHandler", c.inHandler)
	spawn("wsClient.notificationQueueHandler", c.notificationQueueHandler)
	spawn("wsClient.outHandler", c.outHandler)
}

// WaitForShutdown blocks until the websocket client goroutines are stopped
// and the connection is closed.
func (c *wsClient) WaitForShutdown() {
	c.wg.Wait()
}

// FilterData returns the websocket client filter data.
func (c *wsClient) FilterData() *wsClientFilter {
	c.Lock()
	defer c.Unlock()
	return c.filterData
}

// newWebsocketClient returns a new websocket client given the notification
// manager, websocket connection, remote address, and whether or not the client
// has already been authenticated (via HTTP Basic access authentication). The
// returned client is ready to start. Once started, the client will process
// incoming and outgoing messages in separate goroutines complete with queuing
// and asynchrous handling for long-running operations.
func newWebsocketClient(server *Server, conn *websocket.Conn,
	remoteAddr string, authenticated bool, isAdmin bool) (*wsClient, error) {

	sessionID, err := random.Uint64()
	if err != nil {
		return nil, err
	}

	client := &wsClient{
		conn:              conn,
		addr:              remoteAddr,
		authenticated:     authenticated,
		isAdmin:           isAdmin,
		sessionID:         sessionID,
		server:            server,
		serviceRequestSem: makeSemaphore(server.cfg.RPCMaxConcurrentReqs),
		ntfnChan:          make(chan []byte, 1), // nonblocking sync
		sendChan:          make(chan wsResponse, websocketSendBufferSize),
		quit:              make(chan struct{}),
	}
	return client, nil
}

func init() {
	wsHandlers = wsHandlersBeforeInit
}
