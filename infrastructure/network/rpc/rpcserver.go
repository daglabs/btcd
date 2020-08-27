// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaspanet/kaspad/app/protocol"
	"github.com/kaspanet/kaspad/infrastructure/network/addressmanager"
	"github.com/kaspanet/kaspad/infrastructure/network/connmanager"
	"github.com/kaspanet/kaspad/util/mstime"

	"github.com/pkg/errors"

	"github.com/btcsuite/websocket"
	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/blockdag/indexers"
	"github.com/kaspanet/kaspad/domain/mempool"
	"github.com/kaspanet/kaspad/domain/mining"
	"github.com/kaspanet/kaspad/infrastructure/config"
	"github.com/kaspanet/kaspad/infrastructure/network/rpc/model"
	"github.com/kaspanet/kaspad/util/fs"
	"github.com/kaspanet/kaspad/util/network"
)

const (
	// rpcAuthTimeoutSeconds is the number of seconds a connection to the
	// RPC server is allowed to stay open without authenticating before it
	// is closed.
	rpcAuthTimeoutSeconds = 10

	// maxProtocolVersion is the max protocol version the server supports.
	maxProtocolVersion = 70002
)

type commandHandler func(*Server, interface{}, <-chan struct{}) (interface{}, error)

// rpcHandlers maps RPC command strings to appropriate handler functions.
// This is set by init because help references rpcHandlers and thus causes
// a dependency loop.
var rpcHandlers map[string]commandHandler
var rpcHandlersBeforeInit = map[string]commandHandler{
	"connect":                  handleConnect,
	"debugLevel":               handleDebugLevel,
	"getSelectedTip":           handleGetSelectedTip,
	"getSelectedTipHash":       handleGetSelectedTipHash,
	"getBlock":                 handleGetBlock,
	"getBlocks":                handleGetBlocks,
	"getBlockDagInfo":          handleGetBlockDAGInfo,
	"getBlockCount":            handleGetBlockCount,
	"getBlockHeader":           handleGetBlockHeader,
	"getBlockTemplate":         handleGetBlockTemplate,
	"getChainFromBlock":        handleGetChainFromBlock,
	"getConnectionCount":       handleGetConnectionCount,
	"getCurrentNet":            handleGetCurrentNet,
	"getDifficulty":            handleGetDifficulty,
	"getFinalityConflicts":     handleGetFinalityConflicts,
	"resolveFinalityConflicts": handleResolveFinalityConflict,
	"getHeaders":               handleGetHeaders,
	"getTopHeaders":            handleGetTopHeaders,
	"getInfo":                  handleGetInfo,
	"getMempoolInfo":           handleGetMempoolInfo,
	"getMempoolEntry":          handleGetMempoolEntry,
	"getConnectedPeerInfo":     handleGetConnectedPeerInfo,
	"getPeerAddresses":         handleGetPeerAddresses,
	"getRawMempool":            handleGetRawMempool,
	"getSubnetwork":            handleGetSubnetwork,
	"getTxOut":                 handleGetTxOut,
	"help":                     handleHelp,
	"disconnect":               handleDisconnect,
	"sendRawTransaction":       handleSendRawTransaction,
	"stop":                     handleStop,
	"submitBlock":              handleSubmitBlock,
	"uptime":                   handleUptime,
	"version":                  handleVersion,
}

// Commands that are currently unimplemented, but should ultimately be.
var rpcUnimplemented = map[string]struct{}{
	"getMempoolEntry": {},
	"getNetworkInfo":  {},
	"getNetTotals":    {},
}

// Commands that are available to a limited user
var rpcLimited = map[string]struct{}{
	// Websockets commands
	"loadTxFilter":          {},
	"notifyBlocks":          {},
	"notifyChainChanges":    {},
	"notifyNewTransactions": {},
	"notifyReceived":        {},
	"notifySpent":           {},
	"session":               {},

	// Websockets AND HTTP/S commands
	"help": {},

	// HTTP/S-only commands
	"createRawTransaction": {},
	"decodeRawTransaction": {},
	"decodeScript":         {},
	"getSelectedTip":       {},
	"getSelectedTipHash":   {},
	"getBlock":             {},
	"getBlocks":            {},
	"getBlockCount":        {},
	"getBlockHash":         {},
	"getBlockHeader":       {},
	"getChainFromBlock":    {},
	"getCurrentNet":        {},
	"getDifficulty":        {},
	"getHeaders":           {},
	"getInfo":              {},
	"getNetTotals":         {},
	"getRawMempool":        {},
	"getTxOut":             {},
	"sendRawTransaction":   {},
	"submitBlock":          {},
	"uptime":               {},
	"validateAddress":      {},
	"version":              {},
}

// handleUnimplemented is the handler for commands that should ultimately be
// supported but are not yet implemented.
func handleUnimplemented(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, ErrRPCUnimplemented
}

// Server provides a concurrent safe RPC server to a kaspa node.
type Server struct {
	listeners              []net.Listener
	started                int32
	shutdown               int32
	cfg                    *config.Config
	startupTime            mstime.Time
	authsha                [sha256.Size]byte
	limitauthsha           [sha256.Size]byte
	ntfnMgr                *wsNotificationManager
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	gbtWorkState           *gbtWorkState
	helpCacher             *helpCacher
	requestProcessShutdown chan struct{}
	quit                   chan int

	dag                    *blockdag.BlockDAG
	txMempool              *mempool.TxPool
	acceptanceIndex        *indexers.AcceptanceIndex
	blockTemplateGenerator *mining.BlkTmplGenerator
	connectionManager      *connmanager.ConnectionManager
	addressManager         *addressmanager.AddressManager
	protocolManager        *protocol.Manager
}

// httpStatusLine returns a response Status-Line (RFC 2616 Section 6.1)
// for the given request and response status code. This function was lifted and
// adapted from the standard library HTTP server code since it's not exported.
func (s *Server) httpStatusLine(req *http.Request, code int) string {
	// Fast path:
	key := code
	proto11 := req.ProtoAtLeast(1, 1)
	if !proto11 {
		key = -key
	}
	line, ok := func() (string, bool) {
		s.statusLock.RLock()
		defer s.statusLock.RUnlock()
		line, ok := s.statusLines[key]
		return line, ok
	}()
	if ok {
		return line
	}

	// Slow path:
	proto := "HTTP/1.0"
	if proto11 {
		proto = "HTTP/1.1"
	}
	codeStr := strconv.Itoa(code)
	text := http.StatusText(code)
	if text != "" {
		line = proto + " " + codeStr + " " + text + "\r\n"
		s.statusLock.Lock()
		defer s.statusLock.Unlock()
		s.statusLines[key] = line
	} else {
		text = "status code " + codeStr
		line = proto + " " + codeStr + " " + text + "\r\n"
	}

	return line
}

// writeHTTPResponseHeaders writes the necessary response headers prior to
// writing an HTTP body given a request to use for protocol negotiation, headers
// to write, a status code, and a writer.
func (s *Server) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
	_, err := io.WriteString(w, s.httpStatusLine(req, code))
	if err != nil {
		return err
	}

	err = headers.Write(w)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "\r\n")
	return err
}

// Stop is used by server.go to stop the rpc listener.
func (s *Server) Stop() error {
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		log.Infof("RPC server is already in the process of shutting down")
		return nil
	}
	log.Warnf("RPC server shutting down")
	for _, listener := range s.listeners {
		err := listener.Close()
		if err != nil {
			log.Errorf("Problem shutting down rpc: %s", err)
			return err
		}
	}
	s.ntfnMgr.Shutdown()
	s.ntfnMgr.WaitForShutdown()
	close(s.quit)
	s.wg.Wait()
	log.Infof("RPC server shutdown complete")
	return nil
}

// RequestedProcessShutdown returns a channel that is sent to when an authorized
// RPC client requests the process to shutdown. If the request can not be read
// immediately, it is dropped.
func (s *Server) RequestedProcessShutdown() <-chan struct{} {
	return s.requestProcessShutdown
}

// NotifyNewTransactions notifies both websocket and getBlockTemplate long
// poll clients of the passed transactions. This function should be called
// whenever new transactions are added to the mempool.
func (s *Server) NotifyNewTransactions(txns []*mempool.TxDesc) {
	for _, txD := range txns {
		// Notify websocket clients about mempool transactions.
		s.ntfnMgr.NotifyMempoolTx(txD.Tx, true)

		// Potentially notify any getBlockTemplate long poll clients
		// about stale block templates due to the new transaction.
		s.gbtWorkState.NotifyMempoolTx(s.txMempool.LastUpdated())
	}
}

// limitConnections responds with a 503 service unavailable and returns true if
// adding another client would exceed the maximum allow RPC clients.
//
// This function is safe for concurrent access.
func (s *Server) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&s.numClients)+1) > s.cfg.RPCMaxClients {
		log.Infof("Max RPC clients exceeded [%d] - "+
			"disconnecting client %s", s.cfg.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy. Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

// incrementClients adds one to the number of connected RPC clients. Note
// this only applies to standard clients. Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (s *Server) incrementClients() {
	atomic.AddInt32(&s.numClients, 1)
}

// decrementClients subtracts one from the number of connected RPC clients.
// Note this only applies to standard clients. Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (s *Server) decrementClients() {
	atomic.AddInt32(&s.numClients, -1)
}

// checkAuth checks the HTTP Basic authentication supplied by a wallet
// or RPC client in the HTTP request r. If the supplied authentication
// does not match the username and password expected, a non-nil error is
// returned.
//
// This check is time-constant.
//
// The first bool return value signifies auth success (true if successful) and
// the second bool return value specifies whether the user can change the state
// of the server (true) or whether the user is limited (false). The second is
// always false if the first is.
func (s *Server) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			log.Warnf("RPC authentication failure from %s",
				r.RemoteAddr)
			return false, false, errors.New("auth failure")
		}

		return false, false, nil
	}

	authsha := sha256.Sum256([]byte(authhdr[0]))

	// Check for limited auth first as in environments with limited users, those
	// are probably expected to have a higher volume of calls
	limitcmp := subtle.ConstantTimeCompare(authsha[:], s.limitauthsha[:])
	if limitcmp == 1 {
		return true, false, nil
	}

	// Check for admin-level auth
	cmp := subtle.ConstantTimeCompare(authsha[:], s.authsha[:])
	if cmp == 1 {
		return true, true, nil
	}

	// Request's auth doesn't match either user
	log.Warnf("RPC authentication failure from %s", r.RemoteAddr)
	return false, false, errors.New("auth failure")
}

// parsedRPCCmd represents a JSON-RPC request object that has been parsed into
// a known concrete command along with any error that might have happened while
// parsing it.
type parsedRPCCmd struct {
	id     interface{}
	method string
	cmd    interface{}
	err    *model.RPCError
}

// standardCmdResult checks that a parsed command is a standard kaspa JSON-RPC
// command and runs the appropriate handler to reply to the command. Any
// commands which are not recognized or not implemented will return an error
// suitable for use in replies.
func (s *Server) standardCmdResult(cmd *parsedRPCCmd, closeChan <-chan struct{}) (interface{}, error) {
	handler, ok := rpcHandlers[cmd.method]
	if ok {
		goto handled
	}
	_, ok = rpcUnimplemented[cmd.method]
	if ok {
		handler = handleUnimplemented
		goto handled
	}
	return nil, model.ErrRPCMethodNotFound
handled:

	return handler(s, cmd.cmd, closeChan)
}

// parseCmd parses a JSON-RPC request object into known concrete command. The
// err field of the returned parsedRPCCmd struct will contain an RPC error that
// is suitable for use in replies if the command is invalid in some way such as
// an unregistered command or invalid parameters.
func parseCmd(request *model.Request) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	cmd, err := model.UnmarshalCommand(request)
	if err != nil {
		// When the error is because the method is not registered,
		// produce a method not found RPC error.
		var rpcModelErr model.Error
		if ok := errors.As(err, &rpcModelErr); ok &&
			rpcModelErr.ErrorCode == model.ErrUnregisteredMethod {

			parsedCmd.err = model.ErrRPCMethodNotFound
			return &parsedCmd
		}

		// Otherwise, some type of invalid parameters is the
		// cause, so produce the equivalent RPC error.
		parsedCmd.err = model.NewRPCError(
			model.ErrRPCInvalidParams.Code, err.Error())
		return &parsedCmd
	}

	parsedCmd.cmd = cmd
	return &parsedCmd
}

// createMarshalledReply returns a new marshalled JSON-RPC response given the
// passed parameters. It will automatically convert errors that are not of
// the type *model.RPCError to the appropriate type as needed.
func createMarshalledReply(id, result interface{}, replyErr error) ([]byte, error) {
	var jsonErr *model.RPCError
	if replyErr != nil {
		if jErr, ok := replyErr.(*model.RPCError); ok {
			jsonErr = jErr
		} else {
			jsonErr = internalRPCError(replyErr.Error(), "")
		}
	}

	return model.MarshalResponse(id, result, jsonErr)
}

// jsonRPCRead handles reading and responding to RPC messages.
func (s *Server) jsonRPCRead(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	// Read and close the JSON-RPC request body from the caller.
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		errCode := http.StatusBadRequest
		http.Error(w, fmt.Sprintf("%d error reading JSON message: %s",
			errCode, err), errCode)
		return
	}

	// Unfortunately, the http server doesn't provide the ability to
	// change the read deadline for the new connection and having one breaks
	// long polling. However, not having a read deadline on the initial
	// connection would mean clients can connect and idle forever. Thus,
	// hijack the connecton from the HTTP server, clear the read deadline,
	// and handle writing the response manually.
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		log.Warnf(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		log.Warnf("Failed to hijack HTTP connection: %s", err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	conn.SetReadDeadline(timeZeroVal)

	// Attempt to parse the raw body into a JSON-RPC request.
	var responseID interface{}
	var jsonErr error
	var result interface{}
	var request model.Request
	if err := json.Unmarshal(body, &request); err != nil {
		jsonErr = &model.RPCError{
			Code:    model.ErrRPCParse.Code,
			Message: "Failed to parse request: " + err.Error(),
		}
	}
	if jsonErr == nil {
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
			return
		}

		// The parse was at least successful enough to have an ID so
		// set it for the response.
		responseID = request.ID

		// Setup a close notifier. Since the connection is hijacked,
		// the CloseNotifer on the ResponseWriter is not available.
		closeChan := make(chan struct{}, 1)
		spawn("Server.jsonRPCRead-conn.Read", func() {
			_, err := conn.Read(make([]byte, 1))
			if err != nil {
				close(closeChan)
			}
		})

		// Check if the user is limited and set error if method unauthorized
		if !isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr = &model.RPCError{
					Code:    model.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
			}
		}

		if jsonErr == nil {
			// Attempt to parse the JSON-RPC request into a known concrete
			// command.
			parsedCmd := parseCmd(&request)
			if parsedCmd.err != nil {
				jsonErr = parsedCmd.err
			} else {
				log.Debugf("HTTP server received command <%s> from %s", parsedCmd.method, r.RemoteAddr)
				result, jsonErr = s.standardCmdResult(parsedCmd, closeChan)
			}
		}
	}

	// Marshal the response.
	msg, err := createMarshalledReply(responseID, result, jsonErr)
	if err != nil {
		log.Errorf("Failed to marshal reply: %s", err)
		return
	}

	// Write the response.
	err = s.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		log.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		log.Errorf("Failed to write marshalled reply: %s", err)
	}

	// Terminate with newline for historical reasons.
	if err := buf.WriteByte('\n'); err != nil {
		log.Errorf("Failed to append terminating newline to reply: %s", err)
	}
}

// jsonAuthFail sends a message back to the client if the http auth is rejected.
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="kaspad RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

// Start is used by server.go to start the rpc listener.
func (s *Server) Start() {
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	log.Trace("Starting RPC server")
	rpcServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler: rpcServeMux,

		// Timeout connections which don't complete the initial
		// handshake within the allowed timeframe.
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}
	rpcServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		// Limit the number of connections to max allowed.
		if s.limitConnections(w, r.RemoteAddr) {
			return
		}

		// Keep track of the number of connected clients.
		s.incrementClients()
		defer s.decrementClients()
		_, isAdmin, err := s.checkAuth(r, true)
		if err != nil {
			jsonAuthFail(w)
			return
		}

		// Read and respond to the request.
		s.jsonRPCRead(w, r, isAdmin)
	})

	// Websocket endpoint.
	rpcServeMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		authenticated, isAdmin, err := s.checkAuth(r, false)
		if err != nil {
			jsonAuthFail(w)
			return
		}

		// Attempt to upgrade the connection to a websocket connection
		// using the default size for read/write buffers.
		ws, err := websocket.Upgrade(w, r, nil, 0, 0)
		if err != nil {
			if !errors.As(err, &websocket.HandshakeError{}) {
				log.Errorf("Unexpected websocket error: %s",
					err)
			}
			http.Error(w, "400 Bad Request.", http.StatusBadRequest)
			return
		}
		s.WebsocketHandler(ws, r.RemoteAddr, authenticated, isAdmin)
	})

	for _, listener := range s.listeners {
		s.wg.Add(1)
		// Declaring this variable is necessary as it needs be declared in the same
		// scope of the anonymous function below it.
		listenerCopy := listener
		spawn("Server.Start-httpServer.Serve", func() {
			log.Infof("RPC server listening on %s", listenerCopy.Addr())
			httpServer.Serve(listenerCopy)
			log.Tracef("RPC listener done for %s", listenerCopy.Addr())
			s.wg.Done()
		})
	}

	s.ntfnMgr.Start()
}

// setupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func setupRPCListeners(appCfg *config.Config) ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	if !appCfg.DisableTLS {
		// Generate the TLS cert and key file if both don't already
		// exist.
		if !fs.FileExists(appCfg.RPCKey) && !fs.FileExists(appCfg.RPCCert) {
			err := GenCertPair(appCfg.RPCCert, appCfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keypair, err := tls.LoadX509KeyPair(appCfg.RPCCert, appCfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	}

	netAddrs, err := network.ParseListeners(appCfg.RPCListeners)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Warnf("Can't listen on %s: %s", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// NewRPCServer returns a new instance of the rpcServer struct.
func NewRPCServer(
	cfg *config.Config,
	dag *blockdag.BlockDAG,
	txMempool *mempool.TxPool,
	acceptanceIndex *indexers.AcceptanceIndex,
	blockTemplateGenerator *mining.BlkTmplGenerator,
	connectionManager *connmanager.ConnectionManager,
	addressManager *addressmanager.AddressManager,
	protocolManager *protocol.Manager,
) (*Server, error) {
	// Setup listeners for the configured RPC listen addresses and
	// TLS settings.
	rpcListeners, err := setupRPCListeners(cfg)
	if err != nil {
		return nil, err
	}
	if len(rpcListeners) == 0 {
		return nil, errors.New("RPCS: No valid listen address")
	}
	rpc := Server{
		cfg: cfg,

		listeners:              rpcListeners,
		startupTime:            mstime.Now(),
		statusLines:            make(map[int]string),
		gbtWorkState:           newGbtWorkState(),
		helpCacher:             newHelpCacher(),
		requestProcessShutdown: make(chan struct{}),
		quit:                   make(chan int),

		dag:                    dag,
		txMempool:              txMempool,
		acceptanceIndex:        acceptanceIndex,
		blockTemplateGenerator: blockTemplateGenerator,
		connectionManager:      connectionManager,
		addressManager:         addressManager,
		protocolManager:        protocolManager,
	}
	if cfg.RPCUser != "" && cfg.RPCPass != "" {
		login := cfg.RPCUser + ":" + cfg.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.authsha = sha256.Sum256([]byte(auth))
	}
	if cfg.RPCLimitUser != "" && cfg.RPCLimitPass != "" {
		login := cfg.RPCLimitUser + ":" + cfg.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.limitauthsha = sha256.Sum256([]byte(auth))
	}
	rpc.ntfnMgr = newWsNotificationManager(&rpc)
	rpc.dag.Subscribe(rpc.handleBlockDAGNotification)

	return &rpc, nil
}

// Callback for notifications from blockdag. It notifies clients that are
// long polling for changes or subscribed to websockets notifications.
func (s *Server) handleBlockDAGNotification(notification *blockdag.Notification) {
	switch notification.Type {
	case blockdag.NTBlockAdded:
		data, ok := notification.Data.(*blockdag.BlockAddedNotificationData)
		if !ok {
			log.Warnf("Block added notification data is of wrong type.")
			break
		}
		block := data.Block

		tipHashes := s.dag.TipHashes()

		// Allow any clients performing long polling via the
		// getBlockTemplate RPC to be notified when the new block causes
		// their old block template to become stale.
		s.gbtWorkState.NotifyBlockAdded(tipHashes)

		// Notify registered websocket clients of incoming block.
		s.ntfnMgr.NotifyBlockAdded(block)

	case blockdag.NTChainChanged:
		data, ok := notification.Data.(*blockdag.ChainChangedNotificationData)
		if !ok {
			log.Warnf("Chain changed notification data is of wrong type.")
			break
		}

		// If the acceptance index is off we aren't capable of serving
		// ChainChanged notifications.
		if s.acceptanceIndex == nil {
			break
		}

		// Notify registered websocket clients of chain changes.
		s.ntfnMgr.NotifyChainChanged(data.RemovedChainBlockHashes,
			data.AddedChainBlockHashes)

	case blockdag.NTFinalityConflict:
		data, ok := notification.Data.(*blockdag.FinalityConflictNotificationData)
		if !ok {
			log.Warnf("Finality conflict notification data is of wrong type.")
			break
		}

		// Notify registered websocket clients of finality conflict.
		s.ntfnMgr.NotifyFinalityConflict(data.FinalityConflict)

	case blockdag.NTFinalityConflictResolved:
		data, ok := notification.Data.(*blockdag.FinalityConflictResolvedNotificationData)
		if !ok {
			log.Warnf("Finality conflict notification data is of wrong type.")
			break
		}

		// Notify registered websocket clients of finality conflict resolution.
		s.ntfnMgr.NotifyFinalityConflictResolved(
			data.FinalityConflictID, data.ResolutionTime, data.AreAllFinalityConflictsResolved)
	}
}

func init() {
	rpcHandlers = rpcHandlersBeforeInit
	rand.Seed(time.Now().UnixNano())
}
