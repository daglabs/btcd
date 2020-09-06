// Copyright (c) 2015-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"sort"
	"strings"
	"sync"

	"github.com/kaspanet/kaspad/infrastructure/network/rpc/model"
	"github.com/pkg/errors"
)

// helpDescsEnUS defines the English descriptions used for the help strings.
var helpDescsEnUS = map[string]string{
	// DebugLevelCmd help.
	"debugLevel--synopsis": "Dynamically changes the debug logging level.\n" +
		"The levelspec can either a debug level or of the form:\n" +
		"<subsystem>=<level>,<subsystem2>=<level2>,...\n" +
		"The valid debug levels are trace, debug, info, warn, error, and critical.\n" +
		"The valid subsystems are AMGR, ADXR, KSDB, BMGR, KASD, BDAG, DISC, PEER, RPCS, SCRP, SRVR, and TXMP.\n" +
		"Finally the keyword 'show' will return a list of the available subsystems.",
	"debugLevel-levelSpec":   "The debug level(s) to use or the keyword 'show'",
	"debugLevel--condition0": "levelspec!=show",
	"debugLevel--condition1": "levelspec=show",
	"debugLevel--result0":    "The string 'Done.'",
	"debugLevel--result1":    "The list of subsystems",

	// ConnectCmd help.
	"connect--synopsis":   "Attempts to connect a peer.",
	"connect-address":     "IP address and port of the peer to connect",
	"connect-isPermanent": "Whether the connection for this address should be permanent",

	// TransactionInput help.
	"transactionInput-txId": "The hash of the input transaction",
	"transactionInput-vout": "The specific output of the input transaction to redeem",

	// CreateRawTransactionCmd help.
	"createRawTransaction--synopsis": "Returns a new transaction spending the provided inputs and sending to the provided addresses.\n" +
		"The transaction inputs are not signed in the created transaction, and as such must be signed separately.",
	"createRawTransaction-inputs":         "The inputs to the transaction",
	"createRawTransaction-amounts":        "JSON object with the destination addresses as keys and amounts as values",
	"createRawTransaction-amounts--key":   "address",
	"createRawTransaction-amounts--value": "n.nnn",
	"createRawTransaction-amounts--desc":  "The destination address as the key and the amount in KAS as the value",
	"createRawTransaction-lockTime":       "Locktime value; a non-zero value will also locktime-activate the inputs",
	"createRawTransaction--result0":       "Hex-encoded bytes of the serialized transaction",

	// ScriptSig help.
	"scriptSig-asm": "Disassembly of the script",
	"scriptSig-hex": "Hex-encoded bytes of the script",

	// PrevOut help.
	"prevOut-address": "previous output address (if any)",
	"prevOut-value":   "previous output value",

	// VinPrevOut help.
	"vinPrevOut-coinbase":  "The hex-encoded bytes of the signature script (coinbase txns only)",
	"vinPrevOut-txId":      "The hash of the origin transaction (non-coinbase txns only)",
	"vinPrevOut-vout":      "The index of the output being redeemed from the origin transaction (non-coinbase txns only)",
	"vinPrevOut-scriptSig": "The signature script used to redeem the origin transaction as a JSON object (non-coinbase txns only)",
	"vinPrevOut-prevOut":   "Data from the origin transaction output with index vout.",
	"vinPrevOut-sequence":  "The script sequence number",

	// Vin help.
	"vin-coinbase":  "The hex-encoded bytes of the signature script (coinbase txns only)",
	"vin-txId":      "The hash of the origin transaction (non-coinbase txns only)",
	"vin-vout":      "The index of the output being redeemed from the origin transaction (non-coinbase txns only)",
	"vin-scriptSig": "The signature script used to redeem the origin transaction as a JSON object (non-coinbase txns only)",
	"vin-sequence":  "The script sequence number",

	// ScriptPubKeyResult help.
	"scriptPubKeyResult-asm":     "Disassembly of the script",
	"scriptPubKeyResult-hex":     "Hex-encoded bytes of the script",
	"scriptPubKeyResult-type":    "The type of the script (e.g. 'pubkeyhash')",
	"scriptPubKeyResult-address": "The kaspa address (if any) associated with this script",

	// Vout help.
	"vout-value":        "The amount in KAS",
	"vout-n":            "The index of this transaction output",
	"vout-scriptPubKey": "The public key script used to pay coins as a JSON object",

	// ChainBlock help.
	"chainBlock-hash":           "The hash of the chain block",
	"chainBlock-acceptedBlocks": "The blocks accepted by this chain block",

	// AcceptedBlock help.
	"acceptedBlock-hash":          "The hash of the accepted block",
	"acceptedBlock-acceptedTxIds": "The transactions in this block accepted by the chain block",

	// GetSelectedTipResult help.
	"getSelectedTipResult-hash":   "Hex-encoded bytes of the best block hash",
	"getSelectedTipResult-height": "Height of the best block",

	// GetSelectedTipCmd help.
	"getSelectedTip--synopsis":   "Returns information about the selected tip of the blockDAG.",
	"getSelectedTip-verbose":     "Specifies the block is returned as a JSON object instead of hex-encoded string",
	"getSelectedTip-verboseTx":   "Specifies that each transaction is returned as a JSON object and only applies if the verbose flag is true",
	"getSelectedTip--condition0": "verbose=false",
	"getSelectedTip--condition1": "verbose=true",
	"getSelectedTip-acceptedTx":  "Specifies if the transaction got accepted",
	"getSelectedTip--result0":    "Hex-encoded bytes of the serialized block",

	// GetSelectedTipHashCmd help.
	"getSelectedTipHash--synopsis": "Returns the hash of the of the selected tip of the blockDAG.",
	"getSelectedTipHash--result0":  "The hex-encoded block hash",

	// GetBlockCmd help.
	"getBlock--synopsis":   "Returns information about a block given its hash.",
	"getBlock-hash":        "The hash of the block",
	"getBlock-verbose":     "Specifies the block is returned as a JSON object instead of hex-encoded string",
	"getBlock-verboseTx":   "Specifies that each transaction is returned as a JSON object and only applies if the verbose flag is true",
	"getBlock-subnetwork":  "If passed, the returned block will be a partial block of the specified subnetwork",
	"getBlock--condition0": "verbose=false",
	"getBlock--condition1": "verbose=true",
	"getBlock-acceptedTx":  "Specifies if the transaction got accepted",
	"getBlock--result0":    "Hex-encoded bytes of the serialized block",

	// GetBlocksCmd help.
	"getBlocks--synopsis":               "Return the blocks starting from lowHash up to the virtual ordered by blue score.",
	"getBlocks-includeRawBlockData":     "If set to true - the raw block data would be also included.",
	"getBlocks-includeVerboseBlockData": "If set to true - the verbose block data would also be included.",
	"getBlocks-lowHash":                 "Hash of the block with the bottom blue score. If this hash is unknown - returns an error.",
	"getBlocks--result0":                "Blocks starting from lowHash. The result may contains up to 1000 blocks. For the remainder, call the command again with the bluest block's hash.",

	// GetChainFromBlockResult help.
	"getBlocksResult-hashes":        "List of hashes from StartHash (excluding StartHash) ordered by smallest blue score to greatest.",
	"getBlocksResult-rawBlocks":     "If includeBlocks=true - contains the block contents. Otherwise - omitted.",
	"getBlocksResult-verboseBlocks": "If includeBlocks=true and verboseBlocks=true - each block is returned as a JSON object. Otherwise - hex encoded string.",

	// GetBlockChainInfoCmd help.
	"getBlockDagInfo--synopsis": "Returns information about the current blockDAG state and the status of any active soft-fork deployments.",

	// GetBlockDAGInfoResult help.
	"getBlockDagInfoResult-dag":                  "The name of the DAG the daemon is on (testnet, mainnet, etc)",
	"getBlockDagInfoResult-blocks":               "The number of blocks in the DAG",
	"getBlockDagInfoResult-headers":              "The number of headers that we've gathered for in the DAG",
	"getBlockDagInfoResult-tipHashes":            "The block hashes for the tips in the DAG",
	"getBlockDagInfoResult-virtualParentsHashes": "The block hashes for the parents of the virtual block",
	"getBlockDagInfoResult-difficulty":           "The current DAG difficulty",
	"getBlockDagInfoResult-medianTime":           "The median time from the PoV of the selected tip in the DAG",
	"getBlockDagInfoResult-utxoCommitment":       "Commitment to the dag's UTXOSet",
	"getBlockDagInfoResult-verificationProgress": "An estimate for how much of the DAG we've verified",
	"getBlockDagInfoResult-pruned":               "A bool that indicates if the node is pruned or not",
	"getBlockDagInfoResult-pruneHeight":          "The lowest block retained in the current pruned DAG",
	"getBlockDagInfoResult-dagWork":              "The total cumulative work in the DAG",
	"getBlockDagInfoResult-softForks":            "The status of the super-majority soft-forks",
	"getBlockDagInfoResult-bip9SoftForks":        "JSON object describing active BIP0009 deployments",
	"getBlockDagInfoResult-bip9SoftForks--key":   "bip9_softforks",
	"getBlockDagInfoResult-bip9SoftForks--value": "An object describing a particular BIP009 deployment",
	"getBlockDagInfoResult-bip9SoftForks--desc":  "The status of any defined BIP0009 soft-fork deployments",

	// SoftForkDescription help.
	"softForkDescription-reject":  "The current activation status of the softfork",
	"softForkDescription-version": "The block version that signals enforcement of this softfork",
	"softForkDescription-id":      "The string identifier for the soft fork",
	"-status":                     "A bool which indicates if the soft fork is active",

	// TxRawResult help.
	"txRawResult-hex":         "Hex-encoded transaction",
	"txRawResult-txId":        "The hash of the transaction",
	"txRawResult-version":     "The transaction version",
	"txRawResult-lockTime":    "The transaction lock time",
	"txRawResult-subnetwork":  "The transaction subnetwork",
	"txRawResult-gas":         "The transaction gas",
	"txRawResult-mass":        "The transaction mass",
	"txRawResult-payloadHash": "The transaction payload hash",
	"txRawResult-payload":     "The transaction payload",
	"txRawResult-vin":         "The transaction inputs as JSON objects",
	"txRawResult-vout":        "The transaction outputs as JSON objects",
	"txRawResult-blockHash":   "Hash of the block the transaction is part of",
	"txRawResult-isInMempool": "Whether the transaction is in the mempool",
	"txRawResult-time":        "Transaction time in seconds since 1 Jan 1970 GMT",
	"txRawResult-blockTime":   "Block time in seconds since the 1 Jan 1970 GMT",
	"txRawResult-size":        "The size of the transaction in bytes",
	"txRawResult-hash":        "The hash of the transaction",
	"txRawResult-acceptedBy":  "The block in which the transaction got accepted in",

	// GetBlockVerboseResult help.
	"getBlockVerboseResult-hash":                 "The hash of the block (same as provided)",
	"getBlockVerboseResult-confirmations":        "The number of confirmations",
	"getBlockVerboseResult-size":                 "The size of the block",
	"getBlockVerboseResult-mass":                 "The mass of the block",
	"getBlockVerboseResult-height":               "The height of the block in the block DAG",
	"getBlockVerboseResult-version":              "The block version",
	"getBlockVerboseResult-versionHex":           "The block version in hexadecimal",
	"getBlockVerboseResult-hashMerkleRoot":       "Merkle tree reference to hash of all transactions for the block",
	"getBlockVerboseResult-acceptedIdMerkleRoot": "Merkle tree reference to hash all transactions accepted form the block blues",
	"getBlockVerboseResult-utxoCommitment":       "An ECMH UTXO commitment of this block",
	"getBlockVerboseResult-blueScore":            "The block blue score",
	"getBlockVerboseResult-isChainBlock":         "Whether the block is in the selected parent chain",
	"getBlockVerboseResult-tx":                   "The transaction hashes (only when verbosetx=false)",
	"getBlockVerboseResult-rawRx":                "The transactions as JSON objects (only when verbosetx=true)",
	"getBlockVerboseResult-time":                 "The block time in seconds since 1 Jan 1970 GMT",
	"getBlockVerboseResult-nonce":                "The block nonce",
	"getBlockVerboseResult-bits":                 "The bits which represent the block difficulty",
	"getBlockVerboseResult-difficulty":           "The proof-of-work difficulty as a multiple of the minimum difficulty",
	"getBlockVerboseResult-parentHashes":         "The hashes of the parent blocks",
	"getBlockVerboseResult-selectedParentHash":   "The selected parent hash",
	"getBlockVerboseResult-childHashes":          "The hashes of the child blocks (only if there are any)",
	"getBlockVerboseResult-acceptedBlockHashes":  "The hashes of the blocks accepted by this block",

	// GetBlockCountCmd help.
	"getBlockCount--synopsis": "Returns the number of blocks in the block DAG.",
	"getBlockCount--result0":  "The current block count",

	// GetBlockHeaderCmd help.
	"getBlockHeader--synopsis":   "Returns information about a block header given its hash.",
	"getBlockHeader-hash":        "The hash of the block",
	"getBlockHeader-verbose":     "Specifies the block header is returned as a JSON object instead of hex-encoded string",
	"getBlockHeader--condition0": "verbose=false",
	"getBlockHeader--condition1": "verbose=true",
	"getBlockHeader--result0":    "The block header hash",

	// GetBlockHeaderVerboseResult help.
	"getBlockHeaderVerboseResult-hash":                 "The hash of the block (same as provided)",
	"getBlockHeaderVerboseResult-confirmations":        "The number of confirmations",
	"getBlockHeaderVerboseResult-height":               "The height of the block in the block DAG",
	"getBlockHeaderVerboseResult-version":              "The block version",
	"getBlockHeaderVerboseResult-versionHex":           "The block version in hexadecimal",
	"getBlockHeaderVerboseResult-hashMerkleRoot":       "Merkle tree reference to hash of all transactions for the block",
	"getBlockHeaderVerboseResult-acceptedIdMerkleRoot": "Merkle tree reference to hash all transactions accepted form the block blues",
	"getBlockHeaderVerboseResult-time":                 "The block time in seconds since 1 Jan 1970 GMT",
	"getBlockHeaderVerboseResult-nonce":                "The block nonce",
	"getBlockHeaderVerboseResult-bits":                 "The bits which represent the block difficulty",
	"getBlockHeaderVerboseResult-difficulty":           "The proof-of-work difficulty as a multiple of the minimum difficulty",
	"getBlockHeaderVerboseResult-parentHashes":         "The hashes of the parent blocks",
	"getBlockHeaderVerboseResult-selectedParentHash":   "The selected parent hash",
	"getBlockHeaderVerboseResult-childHashes":          "The hashes of the child blocks (only if there are any)",

	// TemplateRequest help.
	"templateRequest-mode":       "This is 'template', 'proposal', or omitted",
	"templateRequest-payAddress": "The address the coinbase pays to",
	"templateRequest-longPollId": "The long poll ID of a job to monitor for expiration; required and valid only for long poll requests ",
	"templateRequest-sigOpLimit": "Number of signature operations allowed in blocks (this parameter is ignored)",
	"templateRequest-massLimit":  "Max transaction mass allowed in blocks (this parameter is ignored)",
	"templateRequest-maxVersion": "Highest supported block version number (this parameter is ignored)",
	"templateRequest-target":     "The desired target for the block template (this parameter is ignored)",
	"templateRequest-data":       "Hex-encoded block data (only for mode=proposal)",
	"templateRequest-workId":     "The server provided workid if provided in block template (not applicable)",

	// GetBlockTemplateResultTx help.
	"getBlockTemplateResultTx-data":    "Hex-encoded transaction data (byte-for-byte)",
	"getBlockTemplateResultTx-hash":    "Hex-encoded transaction hash (little endian if treated as a 256-bit number)",
	"getBlockTemplateResultTx-id":      "Hex-encoded transaction ID (little endian if treated as a 256-bit number)",
	"getBlockTemplateResultTx-depends": "Other transactions before this one (by 1-based index in the 'transactions'  list) that must be present in the final block if this one is",
	"getBlockTemplateResultTx-mass":    "Total mass of all transactions in the block",
	"getBlockTemplateResultTx-fee":     "Difference in value between transaction inputs and outputs (in sompi)",
	"getBlockTemplateResultTx-sigOps":  "Total number of signature operations as counted for purposes of block limits",

	// GetBlockTemplateResultAux help.
	"getBlockTemplateResultAux-flags": "Hex-encoded byte-for-byte data to include in the coinbase signature script",

	// GetBlockTemplateResult help.
	"getBlockTemplateResult-bits":                 "Hex-encoded compressed difficulty",
	"getBlockTemplateResult-curTime":              "Current time as seen by the server (recommended for block time); must fall within mintime/maxtime rules",
	"getBlockTemplateResult-height":               "Height of the block to be solved",
	"getBlockTemplateResult-parentHashes":         "Hex-encoded big-endian hashes of the parent blocks",
	"getBlockTemplateResult-sigOpLimit":           "Number of sigops allowed in blocks",
	"getBlockTemplateResult-massLimit":            "Max transaction mass allowed in blocks",
	"getBlockTemplateResult-transactions":         "Array of transactions as JSON objects",
	"getBlockTemplateResult-hashMerkleRoot":       "The root of the merkle tree of all transaction IDs in this block",
	"getBlockTemplateResult-acceptedIdMerkleRoot": "The root of the merkle tree of transaction IDs accepted by this block",
	"getBlockTemplateResult-utxoCommitment":       "An ECMH UTXO commitment of this block",
	"getBlockTemplateResult-version":              "The block version",
	"getBlockTemplateResult-coinbaseAux":          "Data that should be included in the coinbase signature script",
	"getBlockTemplateResult-coinbaseTxn":          "Information about the coinbase transaction",
	"getBlockTemplateResult-coinbaseValue":        "Total amount available for the coinbase in sompi",
	"getBlockTemplateResult-workId":               "This value must be returned with result if provided (not provided)",
	"getBlockTemplateResult-longPollId":           "Identifier for long poll request which allows monitoring for expiration",
	"getBlockTemplateResult-longPollUri":          "An alternate URI to use for long poll requests if provided (not provided)",
	"getBlockTemplateResult-submitOld":            "Not applicable",
	"getBlockTemplateResult-target":               "Hex-encoded big-endian number which valid results must be less than",
	"getBlockTemplateResult-expires":              "Maximum number of seconds (starting from when the server sent the response) this work is valid for",
	"getBlockTemplateResult-maxTime":              "Maximum allowed time",
	"getBlockTemplateResult-minTime":              "Minimum allowed time",
	"getBlockTemplateResult-mutable":              "List of mutations the server explicitly allows",
	"getBlockTemplateResult-nonceRange":           "Two concatenated hex-encoded big-endian 64-bit integers which represent the valid ranges of nonces the miner may scan",
	"getBlockTemplateResult-capabilities":         "List of server capabilities including 'proposal' to indicate support for block proposals",
	"getBlockTemplateResult-rejectReason":         "Reason the proposal was invalid as-is (only applies to proposal responses)",
	"getBlockTemplateResult-isSynced":             "Whether this node is synced with the rest of of the network. Miners are generally expected not to mine when isSynced is false",

	// GetBlockTemplateCmd help.
	"getBlockTemplate--synopsis": "Returns a JSON object with information necessary to construct a block to mine or accepts a proposal to validate.\n" +
		"See BIP0022 and BIP0023 for the full specification.",
	"getBlockTemplate-request":     "Request object which controls the mode and several parameters",
	"getBlockTemplate--condition0": "mode=template",
	"getBlockTemplate--condition1": "mode=proposal, rejected",
	"getBlockTemplate--condition2": "mode=proposal, accepted",
	"getBlockTemplate--result1":    "An error string which represents why the proposal was rejected or nothing if accepted",

	// GetChainFromBlockCmd help.
	"getChainFromBlock--synopsis":     "Return the selected parent chain starting from startHash up to the virtual. If startHash is not in the selected parent chain, it goes down the DAG until it does reach a hash in the selected parent chain while collecting hashes into removedChainBlockHashes.",
	"getChainFromBlock-startHash":     "Hash of the bottom of the requested chain. If this hash is unknown or is not a chain block - returns an error.",
	"getChainFromBlock-includeBlocks": "If set to true - the block contents would be also included.",
	"getChainFromBlock--result0":      "The selected parent chain. The result may contains up to 1000 blocks. For the remainder, call the command again with the bluest block's hash.",

	// GetChainFromBlockResult help.
	"getChainFromBlockResult-removedChainBlockHashes": "List chain-block hashes that were removed from the selected parent chain in top-to-bottom order",
	"getChainFromBlockResult-addedChainBlocks":        "List of ChainBlocks from Virtual.SelectedTipHashAndBlueScore to StartHash (excluding StartHash) ordered bottom-to-top.",
	"getChainFromBlockResult-blocks":                  "If includeBlocks=true - contains the contents of all chain and accepted blocks in the AddedChainBlocks. Otherwise - omitted.",

	// GetConnectionCountCmd help.
	"getConnectionCount--synopsis": "Returns the number of active connections to other peers.",
	"getConnectionCount--result0":  "The number of connections",

	// GetCurrentNetCmd help.
	"getCurrentNet--synopsis": "Get kaspa network the server is running on.",
	"getCurrentNet--result0":  "The network identifer",

	// GetDifficultyCmd help.
	"getDifficulty--synopsis": "Returns the proof-of-work difficulty as a multiple of the minimum difficulty.",
	"getDifficulty--result0":  "The difficulty",

	// InfoDAGResult help.
	"infoDagResult-version":         "The version of the server",
	"infoDagResult-protocolVersion": "The latest supported protocol version",
	"infoDagResult-blocks":          "The number of blocks processed",
	"infoDagResult-timeOffset":      "The time offset",
	"infoDagResult-connections":     "The number of connected peers",
	"infoDagResult-proxy":           "The proxy used by the server",
	"infoDagResult-difficulty":      "The current target difficulty",
	"infoDagResult-testnet":         "Whether or not server is using testnet",
	"infoDagResult-devnet":          "Whether or not server is using devnet",
	"infoDagResult-relayFee":        "The minimum relay fee for non-free transactions in KAS/KB",
	"infoDagResult-errors":          "Any current errors",

	// GetTopHeadersCmd help.
	"getTopHeaders--synopsis": "Returns the top block headers starting with the provided high hash (not inclusive)",
	"getTopHeaders-highHash":  "Block hash to start including block headers from; if not found, it'll start from the virtual.",
	"getTopHeaders--result0":  "Serialized block headers of all located blocks, limited to some arbitrary maximum number of hashes (currently 2000, which matches the appmessage protocol headers message, but this is not guaranteed)",

	// GetHeadersCmd help.
	"getHeaders--synopsis": "Returns block headers starting with the first known block hash from the request",
	"getHeaders-lowHash":   "Block hash to start including headers from; if not found, it'll start from the genesis block.",
	"getHeaders-highHash":  "Block hash to stop including block headers for; if not found, all headers to the latest known block are returned.",
	"getHeaders--result0":  "Serialized block headers of all located blocks, limited to some arbitrary maximum number of hashes (currently 2000, which matches the appmessage protocol headers message, but this is not guaranteed)",

	// GetInfoCmd help.
	"getInfo--synopsis": "Returns a JSON object containing various state info.",

	// getMempoolEntry help.
	"getMempoolEntry--synopsis": "Returns mempool data for given transaction",
	"getMempoolEntry-txId":      "The transaction ID",

	// getMempoolEntryResult help.
	"getMempoolEntryResult-fee":   "Transaction fee in sompis",
	"getMempoolEntryResult-time":  "Local time transaction entered pool in seconds since 1 Jan 1970 GMT",
	"getMempoolEntryResult-rawTx": "The transaction as a JSON object",

	// GetMempoolInfoCmd help.
	"getMempoolInfo--synopsis": "Returns memory pool information",

	// GetMempoolInfoResult help.
	"getMempoolInfoResult-bytes": "Size in bytes of the mempool",
	"getMempoolInfoResult-size":  "Number of transactions in the mempool",

	// GetNetTotalsCmd help.
	"getNetTotals--synopsis": "Returns a JSON object containing network traffic statistics.",

	// GetNetTotalsResult help.
	"getNetTotalsResult-totalBytesRecv": "Total bytes received",
	"getNetTotalsResult-totalBytesSent": "Total bytes sent",
	"getNetTotalsResult-timeMillis":     "Number of milliseconds since 1 Jan 1970 GMT",

	// GetConnectedPeerInfoResult help.
	"getConnectedPeerInfoResult-id":                        "A unique node ID",
	"getConnectedPeerInfoResult-address":                   "The ip address and port of the peer",
	"getConnectedPeerInfoResult-selectedTipHash":           "The hash of the selected tip of the peer",
	"getConnectedPeerInfoResult-lastPingDuration":          "The duration of the last ping to the peer in milliseconds",
	"getConnectedPeerInfoResult-isSyncNode":                "Whether or not the peer is the sync peer",
	"getConnectedPeerInfoResult-isOutbound":                "Whether the peer is inbound or outbound",
	"getConnectedPeerInfoResult-timeOffset":                "The time difference between this node and the peer",
	"getConnectedPeerInfoResult-userAgent":                 "The user agent of the peer",
	"getConnectedPeerInfoResult-advertisedProtocolVersion": "The advertised p2p protocol version of the peer",
	"getConnectedPeerInfoResult-timeConnected":             "The timestamp of when the peer connected to this node",

	// GetConnectedPeerInfoCmd help.
	"getConnectedPeerInfo--synopsis": "Returns data about each connected network peer as an array of json objects.",

	// GetPeerAddressesResult help.
	"getPeerAddressesResult-version":              "Peers state serialization version",
	"getPeerAddressesResult-key":                  "Address manager's key for randomness purposes.",
	"getPeerAddressesResult-addresses":            "The node's known addresses",
	"getPeerAddressesResult-newBuckets":           "Peers state subnetwork new buckets",
	"getPeerAddressesResult-newBuckets--desc":     "New buckets keyed by subnetwork ID",
	"getPeerAddressesResult-newBuckets--key":      "subnetworkId",
	"getPeerAddressesResult-newBuckets--value":    "New bucket",
	"getPeerAddressesResult-newBucketFullNodes":   "Peers state full nodes new bucket",
	"getPeerAddressesResult-triedBuckets":         "Peers state subnetwork tried buckets",
	"getPeerAddressesResult-triedBuckets--desc":   "Tried buckets keyed by subnetwork ID",
	"getPeerAddressesResult-triedBuckets--key":    "subnetworkId",
	"getPeerAddressesResult-triedBuckets--value":  "Tried bucket",
	"getPeerAddressesResult-triedBucketFullNodes": "Peers state tried full nodes bucket",

	"getPeerAddressesKnownAddressResult-addr":         "Address",
	"getPeerAddressesKnownAddressResult-src":          "Address of the peer that handed the address",
	"getPeerAddressesKnownAddressResult-subnetworkId": "Address subnetwork ID",
	"getPeerAddressesKnownAddressResult-attempts":     "Number of attempts to connect to the address",
	"getPeerAddressesKnownAddressResult-timeStamp":    "Time the address was added",
	"getPeerAddressesKnownAddressResult-lastAttempt":  "Last attempt to connect to the address",
	"getPeerAddressesKnownAddressResult-lastSuccess":  "Last successful attempt to connect to the address",
	"getPeerAddressesKnownAddressResult-isBanned":     "Whether the address was banned",
	"getPeerAddressesKnownAddressResult-bannedTime":   "Time the address was banned",

	// GetPeerAddressesCmd help.
	"getPeerAddresses--synopsis": "Returns the peers state.",

	// GetRawMempoolVerboseResult help.
	"getRawMempoolVerboseResult-size":             "Transaction size in bytes",
	"getRawMempoolVerboseResult-fee":              "Transaction fee in kaspa",
	"getRawMempoolVerboseResult-time":             "Local time transaction entered pool in seconds since 1 Jan 1970 GMT",
	"getRawMempoolVerboseResult-height":           "Block height when transaction entered the pool",
	"getRawMempoolVerboseResult-startingPriority": "Priority when transaction entered the pool",
	"getRawMempoolVerboseResult-currentPriority":  "Current priority",
	"getRawMempoolVerboseResult-depends":          "Unconfirmed transactions used as inputs for this transaction",

	// GetRawMempoolCmd help.
	"getRawMempool--synopsis":   "Returns information about all of the transactions currently in the memory pool.",
	"getRawMempool-verbose":     "Returns JSON object when true or an array of transaction hashes when false",
	"getRawMempool--condition0": "verbose=false",
	"getRawMempool--condition1": "verbose=true",
	"getRawMempool--result0":    "Array of transaction hashes",

	// GetSubnetworkCmd help.
	"getSubnetwork--synopsis":    "Returns information about a subnetwork given its ID.",
	"getSubnetwork-subnetworkId": "The ID of the subnetwork",

	// GetSubnetworkResult help.
	"getSubnetworkResult-gasLimit": "The gas limit of the subnetwork",

	// GetTxOutResult help.
	"getTxOutResult-selectedTip":   "The block hash that contains the transaction output",
	"getTxOutResult-confirmations": "The number of confirmations",
	"getTxOutResult-isInMempool":   "Whether the transaction is in the mempool",
	"getTxOutResult-value":         "The transaction amount in KAS",
	"getTxOutResult-scriptPubKey":  "The public key script used to pay coins as a JSON object",
	"getTxOutResult-version":       "The transaction version",
	"getTxOutResult-coinbase":      "Whether or not the transaction is a coinbase",

	// GetTxOutCmd help.
	"getTxOut--synopsis":      "Returns information about an unspent transaction output..",
	"getTxOut-txId":           "The hash of the transaction",
	"getTxOut-vout":           "The index of the output",
	"getTxOut-includeMempool": "Include the mempool when true",

	// HelpCmd help.
	"help--synopsis":   "Returns a list of all commands or help for a specified command.",
	"help-command":     "The command to retrieve help for",
	"help--condition0": "no command provided",
	"help--condition1": "command specified",
	"help--result0":    "List of commands",
	"help--result1":    "Help for specified command",

	// PingCmd help.
	"ping--synopsis": "Queues a ping to be sent to each connected peer.\n" +
		"Ping times are provided by getConnectedPeerInfo via the pingtime and pingwait fields.",

	// DisconnectCmd help.
	"disconnect--synopsis": "Disconnects a peer",
	"disconnect-address":   "IP address and port of the peer to disconnect",

	// SendRawTransactionCmd help.
	"sendRawTransaction--synopsis":     "Submits the serialized, hex-encoded transaction to the local peer and relays it to the network.",
	"sendRawTransaction-hexTx":         "Serialized, hex-encoded signed transaction",
	"sendRawTransaction-allowHighFees": "Whether or not to allow insanely high fees (kaspad does not yet implement this parameter, so it has no effect)",
	"sendRawTransaction--result0":      "The hash of the transaction",

	// StopCmd help.
	"stop--synopsis": "Shutdown kaspad.",
	"stop--result0":  "The string 'kaspad stopping.'",

	// SubmitBlockOptions help.
	"submitBlockOptions-workId": "This parameter is currently ignored",

	// SubmitBlockCmd help.
	"submitBlock--synopsis":   "Attempts to submit a new serialized, hex-encoded block to the network.",
	"submitBlock-hexBlock":    "Serialized, hex-encoded block",
	"submitBlock-options":     "This parameter is currently ignored",
	"submitBlock--condition0": "Block successfully submitted",
	"submitBlock--condition1": "Block rejected",
	"submitBlock--result1":    "The reason the block was rejected",

	// ValidateAddressResult help.
	"validateAddressResult-isValid": "Whether or not the address is valid",
	"validateAddressResult-address": "The kaspa address (only when isvalid is true)",

	// ValidateAddressCmd help.
	"validateAddress--synopsis": "Verify an address is valid.",
	"validateAddress-address":   "Kaspa address to validate",

	// -------- Websocket-specific help --------

	// Session help.
	"session--synopsis":       "Return details regarding a websocket client's current connection session.",
	"sessionResult-sessionId": "The unique session ID for a client's websocket connection.",

	// NotifyBlocksCmd help.
	"notifyBlocks--synopsis": "Request notifications for whenever a block is connected to the DAG.",

	// StopNotifyBlocksCmd help.
	"stopNotifyBlocks--synopsis": "Cancel registered notifications for whenever a block is connected to the DAG.",

	// NotifyChainChangesCmd help.
	"notifyChainChanges--synopsis": "Request notifications for whenever the selected parent chain changes.",

	// StopNotifyChainChangesCmd help.
	"stopNotifyChainChanges--synopsis": "Cancel registered notifications for whenever the selected parent chain changes.",

	// NotifyNewTransactionsCmd help.
	"notifyNewTransactions--synopsis":  "Send either a txaccepted or a txacceptedverbose notification when a new transaction is accepted into the mempool.",
	"notifyNewTransactions-verbose":    "Specifies which type of notification to receive. If verbose is true, then the caller receives txacceptedverbose, otherwise the caller receives txaccepted",
	"notifyNewTransactions-subnetwork": "Specifies which subnetwork to receive full transactions of. Requires verbose=true. Not allowed when node subnetwork is Native. Must be equal to node subnetwork when node is partial.",

	// StopNotifyNewTransactionsCmd help.
	"stopNotifyNewTransactions--synopsis": "Stop sending either a txaccepted or a txacceptedverbose notification when a new transaction is accepted into the mempool.",

	// Outpoint help.
	"outpoint-txid":  "The hex-encoded bytes of the outpoint transaction ID",
	"outpoint-index": "The index of the outpoint",

	// LoadTxFilterCmd help.
	"loadTxFilter--synopsis": "Load, add to, or reload a websocket client's transaction filter for mempool transactions and new blocks.",
	"loadTxFilter-reload":    "Load a new filter instead of adding data to an existing one",
	"loadTxFilter-addresses": "Array of addresses to add to the transaction filter",
	"loadTxFilter-outpoints": "Array of outpoints to add to the transaction filter",

	// Uptime help.
	"uptime--synopsis": "Returns the total uptime of the server.",
	"uptime--result0":  "The number of seconds that the server has been running",

	// Version help.
	"version--synopsis":       "Returns the JSON-RPC API version (semver)",
	"version--result0--desc":  "Version objects keyed by the program or API name",
	"version--result0--key":   "Program or API name",
	"version--result0--value": "Object containing the semantic version",

	// VersionResult help.
	"versionResult-versionString": "The JSON-RPC API version (semver)",
	"versionResult-major":         "The major component of the JSON-RPC API version",
	"versionResult-minor":         "The minor component of the JSON-RPC API version",
	"versionResult-patch":         "The patch component of the JSON-RPC API version",
	"versionResult-prerelease":    "Prerelease info about the current build",
	"versionResult-buildMetadata": "Metadata about the current build",

	"resolveFinalityConflicts--synopsis":         "Tells kaspad how to resolve a finality conflict that occured",
	"resolveFinalityConflicts-finalityBlockHash": "A block that should be in the selected-parent-chain of all valid blocks",
}

// rpcResultTypes specifies the result types that each RPC command can return.
// This information is used to generate the help. Each result type must be a
// pointer to the type (or nil to indicate no return value).
var rpcResultTypes = map[string][]interface{}{
	"connect":                  nil,
	"debugLevel":               {(*string)(nil), (*string)(nil)},
	"getSelectedTip":           {(*model.GetBlockVerboseResult)(nil)},
	"getSelectedTipHash":       {(*string)(nil)},
	"getBlock":                 {(*string)(nil), (*model.GetBlockVerboseResult)(nil)},
	"getBlocks":                {(*model.GetBlocksResult)(nil)},
	"getBlockCount":            {(*int64)(nil)},
	"getBlockHeader":           {(*string)(nil), (*model.GetBlockHeaderVerboseResult)(nil)},
	"getBlockTemplate":         {(*model.GetBlockTemplateResult)(nil), (*string)(nil), nil},
	"getBlockDagInfo":          {(*model.GetBlockDAGInfoResult)(nil)},
	"getChainFromBlock":        {(*model.GetChainFromBlockResult)(nil)},
	"getConnectionCount":       {(*int32)(nil)},
	"getCurrentNet":            {(*uint32)(nil)},
	"getDifficulty":            {(*float64)(nil)},
	"getTopHeaders":            {(*[]string)(nil)},
	"getHeaders":               {(*[]string)(nil)},
	"getInfo":                  {(*model.InfoDAGResult)(nil)},
	"getMempoolInfo":           {(*model.GetMempoolInfoResult)(nil)},
	"getMempoolEntry":          {(*model.GetMempoolEntryResult)(nil)},
	"getNetTotals":             {(*model.GetNetTotalsResult)(nil)},
	"getConnectedPeerInfo":     {(*[]model.GetConnectedPeerInfoResult)(nil)},
	"getPeerAddresses":         {(*[]model.GetPeerAddressesResult)(nil)},
	"getRawMempool":            {(*[]string)(nil), (*model.GetRawMempoolVerboseResult)(nil)},
	"getSubnetwork":            {(*model.GetSubnetworkResult)(nil)},
	"getTxOut":                 {(*model.GetTxOutResult)(nil)},
	"node":                     nil,
	"help":                     {(*string)(nil), (*string)(nil)},
	"ping":                     nil,
	"disconnect":               nil,
	"sendRawTransaction":       {(*string)(nil)},
	"stop":                     {(*string)(nil)},
	"submitBlock":              {nil, (*string)(nil)},
	"uptime":                   {(*int64)(nil)},
	"validateAddress":          {(*model.ValidateAddressResult)(nil)},
	"version":                  {(*map[string]model.VersionResult)(nil)},
	"resolveFinalityConflicts": nil,

	// Websocket commands.
	"loadTxFilter":              nil,
	"session":                   {(*model.SessionResult)(nil)},
	"notifyBlocks":              nil,
	"stopNotifyBlocks":          nil,
	"notifyChainChanges":        nil,
	"stopNotifyChainChanges":    nil,
	"notifyNewTransactions":     nil,
	"stopNotifyNewTransactions": nil,
}

// helpCacher provides a concurrent safe type that provides help and usage for
// the RPC server commands and caches the results for future calls.
type helpCacher struct {
	sync.Mutex
	usage      string
	methodHelp map[string]string
}

// rpcMethodHelp returns an RPC help string for the provided method.
//
// This function is safe for concurrent access.
func (c *helpCacher) rpcMethodHelp(method string) (string, error) {
	c.Lock()
	defer c.Unlock()

	// Return the cached method help if it exists.
	if help, exists := c.methodHelp[method]; exists {
		return help, nil
	}

	// Look up the result types for the method.
	resultTypes, ok := rpcResultTypes[method]
	if !ok {
		return "", errors.New("no result types specified for method " +
			method)
	}

	// Generate, cache, and return the help.
	help, err := model.GenerateHelp(method, helpDescsEnUS, resultTypes...)
	if err != nil {
		return "", err
	}
	c.methodHelp[method] = help
	return help, nil
}

// rpcUsage returns one-line usage for all support RPC commands.
//
// This function is safe for concurrent access.
func (c *helpCacher) rpcUsage(includeWebsockets bool) (string, error) {
	c.Lock()
	defer c.Unlock()

	// Return the cached usage if it is available.
	if c.usage != "" {
		return c.usage, nil
	}

	// Generate a list of one-line usage for every command.
	usageTexts := make([]string, 0, len(rpcHandlers))
	for k := range rpcHandlers {
		usage, err := model.MethodUsageText(k)
		if err != nil {
			return "", err
		}
		usageTexts = append(usageTexts, usage)
	}

	// Include websockets commands if requested.
	if includeWebsockets {
		for k := range wsHandlers {
			usage, err := model.MethodUsageText(k)
			if err != nil {
				return "", err
			}
			usageTexts = append(usageTexts, usage)
		}
	}

	sort.Strings(usageTexts)
	c.usage = strings.Join(usageTexts, "\n")
	return c.usage, nil
}

// newHelpCacher returns a new instance of a help cacher which provides help and
// usage for the RPC server commands and caches the results for future calls.
func newHelpCacher() *helpCacher {
	return &helpCacher{
		methodHelp: make(map[string]string),
	}
}
