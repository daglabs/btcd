// Copyright (c) 2015-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/pkg/errors"
	"sort"
	"strings"
	"sync"

	"github.com/kaspanet/kaspad/jsonrpc"
)

// helpDescsEnUS defines the English descriptions used for the help strings.
var helpDescsEnUS = map[string]string{
	// DebugLevelCmd help.
	"debugLevel--synopsis": "Dynamically changes the debug logging level.\n" +
		"The levelspec can either a debug level or of the form:\n" +
		"<subsystem>=<level>,<subsystem2>=<level2>,...\n" +
		"The valid debug levels are trace, debug, info, warn, error, and critical.\n" +
		"The valid subsystems are AMGR, ADXR, BCDB, BMGR, KSPD, BDAG, DISC, PEER, RPCS, SCRP, SRVR, and TXMP.\n" +
		"Finally the keyword 'show' will return a list of the available subsystems.",
	"debugLevel-levelSpec":   "The debug level(s) to use or the keyword 'show'",
	"debugLevel--condition0": "levelspec!=show",
	"debugLevel--condition1": "levelspec=show",
	"debugLevel--result0":    "The string 'Done.'",
	"debugLevel--result1":    "The list of subsystems",

	// AddManualNodeCmd help.
	"addManualNode--synopsis": "Attempts to add or remove a persistent peer.",
	"addManualNode-addr":      "IP address and port of the peer to operate on",
	"addManualNode-oneTry":    "When enabled, will try a single connection to a peer",

	// NodeCmd help.
	"node--synopsis":     "Attempts to add or remove a peer.",
	"node-subCmd":        "'disconnect' to remove all matching non-persistent peers, 'remove' to remove a persistent peer, or 'connect' to connect to a peer",
	"node-target":        "Either the IP address and port of the peer to operate on, or a valid peer ID.",
	"node-connectSubCmd": "'perm' to make the connected peer a permanent one, 'temp' to try a single connect to a peer",

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
	"createRawTransaction-amounts--desc":  "The destination address as the key and the amount in BTC as the value",
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
	"scriptPubKeyResult-address": "The bitcoin address (if any) associated with this script",

	// Vout help.
	"vout-value":        "The amount in BTC",
	"vout-n":            "The index of this transaction output",
	"vout-scriptPubKey": "The public key script used to pay coins as a JSON object",

	// ChainBlock help.
	"chainBlock-hash":           "The hash of the chain block",
	"chainBlock-acceptedBlocks": "The blocks accepted by this chain block",

	// AcceptedBlock help.
	"acceptedBlock-hash":          "The hash of the accepted block",
	"acceptedBlock-acceptedTxIds": "The transactions in this block accepted by the chain block",

	// TxRawDecodeResult help.
	"txRawDecodeResult-txId":     "The hash of the transaction",
	"txRawDecodeResult-version":  "The transaction version",
	"txRawDecodeResult-lockTime": "The transaction lock time",
	"txRawDecodeResult-vin":      "The transaction inputs as JSON objects",
	"txRawDecodeResult-vout":     "The transaction outputs as JSON objects",

	// DecodeRawTransactionCmd help.
	"decodeRawTransaction--synopsis": "Returns a JSON object representing the provided serialized, hex-encoded transaction.",
	"decodeRawTransaction-hexTx":     "Serialized, hex-encoded transaction",

	// DecodeScriptResult help.
	"decodeScriptResult-asm":     "Disassembly of the script",
	"decodeScriptResult-type":    "The type of the script (e.g. 'pubkeyhash')",
	"decodeScriptResult-reqSigs": "The number of required signatures",
	"decodeScriptResult-address": "The bitcoin address (if any) associated with this script",
	"decodeScriptResult-p2sh":    "The script hash for use in pay-to-script-hash transactions (only present if the provided redeem script is not already a pay-to-script-hash script)",

	// DecodeScriptCmd help.
	"decodeScript--synopsis": "Returns a JSON object with information about the provided hex-encoded script.",
	"decodeScript-hexScript": "Hex-encoded script",

	// GenerateCmd help
	"generate--synopsis": "Generates a set number of blocks (simnet or regtest only) and returns a JSON\n" +
		" array of their hashes.",
	"generate-numBlocks": "Number of blocks to generate",
	"generate--result0":  "The hashes, in order, of blocks generated by the call",

	// GetAllManualNodesInfoCmd help.
	"getAllManualNodesInfo--synopsis":   "Returns information about manually added (persistent) peers.",
	"getAllManualNodesInfo-details":     "Specifies whether the returned data is a JSON object including DNS and connection information, or just a list of added peers",
	"getAllManualNodesInfo--condition0": "details=false",
	"getAllManualNodesInfo--condition1": "details=true",
	"getAllManualNodesInfo--result0":    "List of added peers",

	// GetManualNodeInfoResultAddr help.
	"getManualNodeInfoResultAddr-address":   "The ip address for this DNS entry",
	"getManualNodeInfoResultAddr-connected": "The connection 'direction' (inbound/outbound/false)",

	// GetManualNodeInfoResult help.
	"getManualNodeInfoResult-manualNode": "The ip address or domain of the manually added peer",
	"getManualNodeInfoResult-connected":  "Whether or not the peer is currently connected",
	"getManualNodeInfoResult-addresses":  "DNS lookup and connection information about the peer",

	// GetManualNodeInfoCmd help.
	"getManualNodeInfo--synopsis":   "Returns information about manually added (persistent) peers.",
	"getManualNodeInfo-details":     "Specifies whether the returned data is a JSON object including DNS and connection information, or just a list of added peers",
	"getManualNodeInfo-node":        "Only return information about this specific peer instead of all added peers",
	"getManualNodeInfo--condition0": "details=false",
	"getManualNodeInfo--condition1": "details=true",
	"getManualNodeInfo--result0":    "List of added peers",

	// GetSelectedTipResult help.
	"getSelectedTipResult-hash":   "Hex-encoded bytes of the best block hash",
	"getSelectedTipResult-height": "Height of the best block",

	// GetSelectedTipCmd help.
	"getSelectedTip--synopsis":   "Returns information about the selected tip of the blockDAG.",
	"getSelectedTip-verbose":     "Specifies the block is returned as a JSON object instead of hex-encoded string",
	"getSelectedTip-verboseTx":   "Specifies that each transaction is returned as a JSON object and only applies if the verbose flag is true (btcd extension)",
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
	"getBlock-verboseTx":   "Specifies that each transaction is returned as a JSON object and only applies if the verbose flag is true (btcd extension)",
	"getBlock-subnetwork":  "If passed, the returned block will be a partial block of the specified subnetwork",
	"getBlock--condition0": "verbose=false",
	"getBlock--condition1": "verbose=true",
	"getBlock-acceptedTx":  "Specifies if the transaction got accepted",
	"getBlock--result0":    "Hex-encoded bytes of the serialized block",

	// GetBlocksCmd help.
	"getBlocks--synopsis":               "Return the blocks starting from startHash up to the virtual ordered by blue score.",
	"getBlocks-includeRawBlockData":     "If set to true - the raw block data would be also included.",
	"getBlocks-includeVerboseBlockData": "If set to true - the verbose block data would also be included.",
	"getBlocks-startHash":               "Hash of the block with the bottom blue score. If this hash is unknown - returns an error.",
	"getBlocks--result0":                "Blocks starting from startHash. The result may contains up to 1000 blocks. For the remainder, call the command again with the bluest block's hash.",

	// GetChainFromBlockResult help.
	"getBlocksResult-hashes":        "List of hashes from StartHash (excluding StartHash) ordered by smallest blue score to greatest.",
	"getBlocksResult-rawBlocks":     "If includeBlocks=true - contains the block contents. Otherwise - omitted.",
	"getBlocksResult-verboseBlocks": "If includeBlocks=true and verboseBlocks=true - each block is returned as a JSON object. Otherwise - hex encoded string.",

	// GetBlockChainInfoCmd help.
	"getBlockDagInfo--synopsis": "Returns information about the current blockDAG state and the status of any active soft-fork deployments.",

	// GetBlockDAGInfoResult help.
	"getBlockDagInfoResult-dag":                  "The name of the DAG the daemon is on (testnet, mainnet, etc)",
	"getBlockDagInfoResult-blocks":               "The number of blocks in the best known chain",
	"getBlockDagInfoResult-headers":              "The number of headers that we've gathered for in the best known chain",
	"getBlockDagInfoResult-tipHashes":            "The block hashes for the tips in the DAG",
	"getBlockDagInfoResult-difficulty":           "The current chain difficulty",
	"getBlockDagInfoResult-medianTime":           "The median time from the PoV of the best block in the chain",
	"getBlockDagInfoResult-utxoCommitment":       "Commitment to the dag's UTXOSet",
	"getBlockDagInfoResult-verificationProgress": "An estimate for how much of the best chain we've verified",
	"getBlockDagInfoResult-pruned":               "A bool that indicates if the node is pruned or not",
	"getBlockDagInfoResult-pruneHeight":          "The lowest block retained in the current pruned chain",
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
	"txRawResult-hex":           "Hex-encoded transaction",
	"txRawResult-txId":          "The hash of the transaction",
	"txRawResult-version":       "The transaction version",
	"txRawResult-lockTime":      "The transaction lock time",
	"txRawResult-subnetwork":    "The transaction subnetwork",
	"txRawResult-gas":           "The transaction gas",
	"txRawResult-mass":          "The transaction mass",
	"txRawResult-payloadHash":   "The transaction payload hash",
	"txRawResult-payload":       "The transaction payload",
	"txRawResult-vin":           "The transaction inputs as JSON objects",
	"txRawResult-vout":          "The transaction outputs as JSON objects",
	"txRawResult-blockHash":     "Hash of the block the transaction is part of",
	"txRawResult-confirmations": "Number of confirmations of the block (Will be 'null' if txindex is not disabled)",
	"txRawResult-isInMempool":   "Whether the transaction is in the mempool",
	"txRawResult-time":          "Transaction time in seconds since 1 Jan 1970 GMT",
	"txRawResult-blockTime":     "Block time in seconds since the 1 Jan 1970 GMT",
	"txRawResult-size":          "The size of the transaction in bytes",
	"txRawResult-hash":          "The wtxid of the transaction",
	"txRawResult-acceptedBy":    "The block in which the transaction got accepted in (Will be 'null' if txindex is not disabled)",

	// SearchRawTransactionsResult help.
	"searchRawTransactionsResult-hex":           "Hex-encoded transaction",
	"searchRawTransactionsResult-txId":          "The hash of the transaction",
	"searchRawTransactionsResult-hash":          "The wxtid of the transaction",
	"searchRawTransactionsResult-version":       "The transaction version",
	"searchRawTransactionsResult-lockTime":      "The transaction lock time",
	"searchRawTransactionsResult-vin":           "The transaction inputs as JSON objects",
	"searchRawTransactionsResult-vout":          "The transaction outputs as JSON objects",
	"searchRawTransactionsResult-blockHash":     "Hash of the block the transaction is part of",
	"searchRawTransactionsResult-confirmations": "Number of confirmations of the block (Will be 'null' if txindex is not disabled)",
	"searchRawTransactionsResult-isInMempool":   "Whether the transaction is in the mempool",
	"searchRawTransactionsResult-time":          "Transaction time in seconds since 1 Jan 1970 GMT",
	"searchRawTransactionsResult-blockTime":     "Block time in seconds since the 1 Jan 1970 GMT",
	"searchRawTransactionsResult-size":          "The size of the transaction in bytes",

	// GetBlockVerboseResult help.
	"getBlockVerboseResult-hash":                 "The hash of the block (same as provided)",
	"getBlockVerboseResult-confirmations":        "The number of confirmations",
	"getBlockVerboseResult-size":                 "The size of the block",
	"getBlockVerboseResult-mass":                 "The mass of the block",
	"getBlockVerboseResult-height":               "The height of the block in the block chain",
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
	"getBlockVerboseResult-nextHashes":           "The hashes of the next blocks (only if there are any)",

	// GetBlockCountCmd help.
	"getBlockCount--synopsis": "Returns the number of blocks in the longest block chain.",
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
	"getBlockHeaderVerboseResult-height":               "The height of the block in the block chain",
	"getBlockHeaderVerboseResult-version":              "The block version",
	"getBlockHeaderVerboseResult-versionHex":           "The block version in hexadecimal",
	"getBlockHeaderVerboseResult-hashMerkleRoot":       "Merkle tree reference to hash of all transactions for the block",
	"getBlockHeaderVerboseResult-acceptedIdMerkleRoot": "Merkle tree reference to hash all transactions accepted form the block blues",
	"getBlockHeaderVerboseResult-time":                 "The block time in seconds since 1 Jan 1970 GMT",
	"getBlockHeaderVerboseResult-nonce":                "The block nonce",
	"getBlockHeaderVerboseResult-bits":                 "The bits which represent the block difficulty",
	"getBlockHeaderVerboseResult-difficulty":           "The proof-of-work difficulty as a multiple of the minimum difficulty",
	"getBlockHeaderVerboseResult-parentHashes":         "The hashes of the parent blocks",
	"getBlockHeaderVerboseResult-nextHashes":           "The hashes of the next blocks (only if there are any)",

	// TemplateRequest help.
	"templateRequest-mode":         "This is 'template', 'proposal', or omitted",
	"templateRequest-capabilities": "List of capabilities",
	"templateRequest-longPollId":   "The long poll ID of a job to monitor for expiration; required and valid only for long poll requests ",
	"templateRequest-sigOpLimit":   "Number of signature operations allowed in blocks (this parameter is ignored)",
	"templateRequest-massLimit":    "Max transaction mass allowed in blocks (this parameter is ignored)",
	"templateRequest-maxVersion":   "Highest supported block version number (this parameter is ignored)",
	"templateRequest-target":       "The desired target for the block template (this parameter is ignored)",
	"templateRequest-data":         "Hex-encoded block data (only for mode=proposal)",
	"templateRequest-workId":       "The server provided workid if provided in block template (not applicable)",

	// GetBlockTemplateResultTx help.
	"getBlockTemplateResultTx-data":    "Hex-encoded transaction data (byte-for-byte)",
	"getBlockTemplateResultTx-hash":    "Hex-encoded transaction hash (little endian if treated as a 256-bit number)",
	"getBlockTemplateResultTx-id":      "Hex-encoded transaction ID (little endian if treated as a 256-bit number)",
	"getBlockTemplateResultTx-depends": "Other transactions before this one (by 1-based index in the 'transactions'  list) that must be present in the final block if this one is",
	"getBlockTemplateResultTx-mass":    "Total mass of all transactions in the block",
	"getBlockTemplateResultTx-fee":     "Difference in value between transaction inputs and outputs (in Satoshi)",
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
	"getBlockTemplateResult-acceptedIdMerkleRoot": "The root of the merkle tree of transaction IDs accepted by this block",
	"getBlockTemplateResult-utxoCommitment":       "An ECMH UTXO commitment of this block",
	"getBlockTemplateResult-version":              "The block version",
	"getBlockTemplateResult-coinbaseAux":          "Data that should be included in the coinbase signature script",
	"getBlockTemplateResult-coinbaseTxn":          "Information about the coinbase transaction",
	"getBlockTemplateResult-coinbaseValue":        "Total amount available for the coinbase in Satoshi",
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

	// GetBlockTemplateCmd help.
	"getBlockTemplate--synopsis": "Returns a JSON object with information necessary to construct a block to mine or accepts a proposal to validate.\n" +
		"See BIP0022 and BIP0023 for the full specification.",
	"getBlockTemplate-request":     "Request object which controls the mode and several parameters",
	"getBlockTemplate--condition0": "mode=template",
	"getBlockTemplate--condition1": "mode=proposal, rejected",
	"getBlockTemplate--condition2": "mode=proposal, accepted",
	"getBlockTemplate--result1":    "An error string which represents why the proposal was rejected or nothing if accepted",

	// GetCFilterCmd help.
	"getCFilter--synopsis":  "Returns a block's committed filter given its hash.",
	"getCFilter-filterType": "The type of filter to return (0=regular, 1=extended)",
	"getCFilter-hash":       "The hash of the block",
	"getCFilter--result0":   "The block's committed filter",

	// GetChainFromBlockCmd help.
	"getChainFromBlock--synopsis":     "Return the selected parent chain starting from startHash up to the virtual. If startHash is not in the selected parent chain, it goes down the DAG until it does reach a hash in the selected parent chain while collecting hashes into removedChainBlockHashes.",
	"getChainFromBlock-startHash":     "Hash of the bottom of the requested chain. If this hash is unknown or is not a chain block - returns an error.",
	"getChainFromBlock-includeBlocks": "If set to true - the block contents would be also included.",
	"getChainFromBlock--result0":      "The selected parent chain. The result may contains up to 1000 blocks. For the remainder, call the command again with the bluest block's hash.",

	// GetChainFromBlockResult help.
	"getChainFromBlockResult-removedChainBlockHashes": "List chain-block hashes that were removed from the selected parent chain in top-to-bottom order",
	"getChainFromBlockResult-addedChainBlocks":        "List of ChainBlocks from Virtual.SelectedTip to StartHash (excluding StartHash) ordered bottom-to-top.",
	"getChainFromBlockResult-blocks":                  "If includeBlocks=true - contains the contents of all chain and accepted blocks in the AddedChainBlocks. Otherwise - omitted.",

	// GetCFilterHeaderCmd help.
	"getCFilterHeader--synopsis":  "Returns a block's compact filter header given its hash.",
	"getCFilterHeader-filterType": "The type of filter header to return (0=regular, 1=extended)",
	"getCFilterHeader-hash":       "The hash of the block",
	"getCFilterHeader--result0":   "The block's gcs filter header",

	// GetConnectionCountCmd help.
	"getConnectionCount--synopsis": "Returns the number of active connections to other peers.",
	"getConnectionCount--result0":  "The number of connections",

	// GetCurrentNetCmd help.
	"getCurrentNet--synopsis": "Get bitcoin network the server is running on.",
	"getCurrentNet--result0":  "The network identifer",

	// GetDifficultyCmd help.
	"getDifficulty--synopsis": "Returns the proof-of-work difficulty as a multiple of the minimum difficulty.",
	"getDifficulty--result0":  "The difficulty",

	// GetGenerateCmd help.
	"getGenerate--synopsis": "Returns if the server is set to generate coins (mine) or not.",
	"getGenerate--result0":  "True if mining, false if not",

	// GetHashesPerSecCmd help.
	"getHashesPerSec--synopsis": "Returns a recent hashes per second performance measurement while generating coins (mining).",
	"getHashesPerSec--result0":  "The number of hashes per second",

	// InfoDAGResult help.
	"infoDagResult-version":         "The version of the server",
	"infoDagResult-protocolVersion": "The latest supported protocol version",
	"infoDagResult-blocks":          "The number of blocks processed",
	"infoDagResult-timeOffset":      "The time offset",
	"infoDagResult-connections":     "The number of connected peers",
	"infoDagResult-proxy":           "The proxy used by the server",
	"infoDagResult-difficulty":      "The current target difficulty",
	"infoDagResult-testNet":         "Whether or not server is using testnet",
	"infoDagResult-devNet":          "Whether or not server is using devnet",
	"infoDagResult-relayFee":        "The minimum relay fee for non-free transactions in BTC/KB",
	"infoDagResult-errors":          "Any current errors",

	// GetTopHeadersCmd help.
	"getTopHeaders--synopsis": "Returns the top block headers starting with the provided start hash (not inclusive)",
	"getTopHeaders-startHash": "Block hash to start including block headers from; if not found, it'll start from the virtual.",
	"getTopHeaders--result0":  "Serialized block headers of all located blocks, limited to some arbitrary maximum number of hashes (currently 2000, which matches the wire protocol headers message, but this is not guaranteed)",

	// GetHeadersCmd help.
	"getHeaders--synopsis": "Returns block headers starting with the first known block hash from the request",
	"getHeaders-startHash": "Block hash to start including headers from; if not found, it'll start from the genesis block.",
	"getHeaders-stopHash":  "Block hash to stop including block headers for; if not found, all headers to the latest known block are returned.",
	"getHeaders--result0":  "Serialized block headers of all located blocks, limited to some arbitrary maximum number of hashes (currently 2000, which matches the wire protocol headers message, but this is not guaranteed)",

	// GetInfoCmd help.
	"getInfo--synopsis": "Returns a JSON object containing various state info.",

	// GetMempoolInfoCmd help.
	"getMempoolInfo--synopsis": "Returns memory pool information",

	// GetMempoolInfoResult help.
	"getMempoolInfoResult-bytes": "Size in bytes of the mempool",
	"getMempoolInfoResult-size":  "Number of transactions in the mempool",

	// GetMiningInfoResult help.
	"getMiningInfoResult-blocks":           "Height of the latest best block",
	"getMiningInfoResult-currentBlockSize": "Size of the latest best block",
	"getMiningInfoResult-currentBlockTx":   "Number of transactions in the latest best block",
	"getMiningInfoResult-difficulty":       "Current target difficulty",
	"getMiningInfoResult-errors":           "Any current errors",
	"getMiningInfoResult-generate":         "Whether or not server is set to generate coins",
	"getMiningInfoResult-genProcLimit":     "Number of processors to use for coin generation (-1 when disabled)",
	"getMiningInfoResult-hashesPerSec":     "Recent hashes per second performance measurement while generating coins",
	"getMiningInfoResult-networkHashPs":    "Estimated network hashes per second for the most recent blocks",
	"getMiningInfoResult-pooledTx":         "Number of transactions in the memory pool",
	"getMiningInfoResult-testNet":          "Whether or not server is using testnet",
	"getMiningInfoResult-devNet":           "Whether or not server is using devnet",

	// GetMiningInfoCmd help.
	"getMiningInfo--synopsis": "Returns a JSON object containing mining-related information.",

	// GetNetworkHashPSCmd help.
	"getNetworkHashPs--synopsis": "Returns the estimated network hashes per second for the block heights provided by the parameters.",
	"getNetworkHashPs-blocks":    "The number of blocks, or -1 for blocks since last difficulty change",
	"getNetworkHashPs-height":    "Perform estimate ending with this height or -1 for current best chain block height",
	"getNetworkHashPs--result0":  "Estimated hashes per second",

	// GetNetTotalsCmd help.
	"getNetTotals--synopsis": "Returns a JSON object containing network traffic statistics.",

	// GetNetTotalsResult help.
	"getNetTotalsResult-totalBytesRecv": "Total bytes received",
	"getNetTotalsResult-totalBytesSent": "Total bytes sent",
	"getNetTotalsResult-timeMillis":     "Number of milliseconds since 1 Jan 1970 GMT",

	// GetPeerInfoResult help.
	"getPeerInfoResult-id":          "A unique node ID",
	"getPeerInfoResult-addr":        "The ip address and port of the peer",
	"getPeerInfoResult-services":    "Services bitmask which represents the services supported by the peer",
	"getPeerInfoResult-relayTxes":   "Peer has requested transactions be relayed to it",
	"getPeerInfoResult-lastSend":    "Time the last message was received in seconds since 1 Jan 1970 GMT",
	"getPeerInfoResult-lastRecv":    "Time the last message was sent in seconds since 1 Jan 1970 GMT",
	"getPeerInfoResult-bytesSent":   "Total bytes sent",
	"getPeerInfoResult-bytesRecv":   "Total bytes received",
	"getPeerInfoResult-connTime":    "Time the connection was made in seconds since 1 Jan 1970 GMT",
	"getPeerInfoResult-timeOffset":  "The time offset of the peer",
	"getPeerInfoResult-pingTime":    "Number of microseconds the last ping took",
	"getPeerInfoResult-pingWait":    "Number of microseconds a queued ping has been waiting for a response",
	"getPeerInfoResult-version":     "The protocol version of the peer",
	"getPeerInfoResult-subVer":      "The user agent of the peer",
	"getPeerInfoResult-inbound":     "Whether or not the peer is an inbound connection",
	"getPeerInfoResult-selectedTip": "The selected tip of the peer",
	"getPeerInfoResult-banScore":    "The ban score",
	"getPeerInfoResult-feeFilter":   "The requested minimum fee a transaction must have to be announced to the peer",
	"getPeerInfoResult-syncNode":    "Whether or not the peer is the sync peer",

	// GetPeerInfoCmd help.
	"getPeerInfo--synopsis": "Returns data about each connected network peer as an array of json objects.",

	// GetRawMempoolVerboseResult help.
	"getRawMempoolVerboseResult-size":             "Transaction size in bytes",
	"getRawMempoolVerboseResult-fee":              "Transaction fee in bitcoins",
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

	// GetRawTransactionCmd help.
	"getRawTransaction--synopsis":   "Returns information about a transaction given its hash.",
	"getRawTransaction-txId":        "The hash of the transaction",
	"getRawTransaction-verbose":     "Specifies the transaction is returned as a JSON object instead of a hex-encoded string",
	"getRawTransaction--condition0": "verbose=false",
	"getRawTransaction--condition1": "verbose=true",
	"getRawTransaction--result0":    "Hex-encoded bytes of the serialized transaction",

	// GetSubnetworkCmd help.
	"getSubnetwork--synopsis":    "Returns information about a subnetwork given its ID.",
	"getSubnetwork-subnetworkId": "The ID of the subnetwork",

	// GetSubnetworkResult help.
	"getSubnetworkResult-gasLimit": "The gas limit of the subnetwork",

	// GetTxOutResult help.
	"getTxOutResult-selectedTip":   "The block hash that contains the transaction output",
	"getTxOutResult-confirmations": "The number of confirmations (Will be 'null' if txindex is not disabled)",
	"getTxOutResult-isInMempool":   "Whether the transaction is in the mempool",
	"getTxOutResult-value":         "The transaction amount in BTC",
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
		"Ping times are provided by getPeerInfo via the pingtime and pingwait fields.",

	// RemoveManualNodeCmd help.
	"removeManualNode--synopsis": "Removes a peer from the manual nodes list",
	"removeManualNode-addr":      "IP address and port of the peer to remove",

	// SearchRawTransactionsCmd help.
	"searchRawTransactions--synopsis": "Returns raw data for transactions involving the passed address.\n" +
		"Returned transactions are pulled from both the database, and transactions currently in the mempool.\n" +
		"Transactions pulled from the mempool will have the 'confirmations' field set to 0.\n" +
		"Usage of this RPC requires the optional --addrindex flag to be activated, otherwise all responses will simply return with an error stating the address index has not yet been built.\n" +
		"Similarly, until the address index has caught up with the current best height, all requests will return an error response in order to avoid serving stale data.",
	"searchRawTransactions-address":     "The Bitcoin address to search for",
	"searchRawTransactions-verbose":     "Specifies the transaction is returned as a JSON object instead of hex-encoded string",
	"searchRawTransactions--condition0": "verbose=0",
	"searchRawTransactions--condition1": "verbose=1",
	"searchRawTransactions-skip":        "The number of leading transactions to leave out of the final response",
	"searchRawTransactions-count":       "The maximum number of transactions to return",
	"searchRawTransactions-vinExtra":    "Specify that extra data from previous output will be returned in vin",
	"searchRawTransactions-reverse":     "Specifies that the transactions should be returned in reverse chronological order",
	"searchRawTransactions-filterAddrs": "Address list. Only inputs or outputs with matching address will be returned",
	"searchRawTransactions--result0":    "Hex-encoded serialized transaction",

	// SendRawTransactionCmd help.
	"sendRawTransaction--synopsis":     "Submits the serialized, hex-encoded transaction to the local peer and relays it to the network.",
	"sendRawTransaction-hexTx":         "Serialized, hex-encoded signed transaction",
	"sendRawTransaction-allowHighFees": "Whether or not to allow insanely high fees (btcd does not yet implement this parameter, so it has no effect)",
	"sendRawTransaction--result0":      "The hash of the transaction",

	// SetGenerateCmd help.
	"setGenerate--synopsis":    "Set the server to generate coins (mine) or not.",
	"setGenerate-generate":     "Use true to enable generation, false to disable it",
	"setGenerate-genProcLimit": "The number of processors (cores) to limit generation to or -1 for default",

	// StopCmd help.
	"stop--synopsis": "Shutdown btcd.",
	"stop--result0":  "The string 'btcd stopping.'",

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
	"validateAddressResult-address": "The bitcoin address (only when isvalid is true)",

	// ValidateAddressCmd help.
	"validateAddress--synopsis": "Verify an address is valid.",
	"validateAddress-address":   "Bitcoin address to validate",

	// -------- Websocket-specific help --------

	// Session help.
	"session--synopsis":       "Return details regarding a websocket client's current connection session.",
	"sessionResult-sessionId": "The unique session ID for a client's websocket connection.",

	// NotifyBlocksCmd help.
	"notifyBlocks--synopsis": "Request notifications for whenever a block is connected or disconnected from the main (best) chain.",

	// StopNotifyBlocksCmd help.
	"stopNotifyBlocks--synopsis": "Cancel registered notifications for whenever a block is connected or disconnected from the main (best) chain.",

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
	"loadTxFilter--synopsis": "Load, add to, or reload a websocket client's transaction filter for mempool transactions, new blocks and rescanBlocks.",
	"loadTxFilter-reload":    "Load a new filter instead of adding data to an existing one",
	"loadTxFilter-addresses": "Array of addresses to add to the transaction filter",
	"loadTxFilter-outpoints": "Array of outpoints to add to the transaction filter",

	// RescanBlocks help.
	"rescanBlocks--synopsis":   "Rescan blocks for transactions matching the loaded transaction filter.",
	"rescanBlocks-blockHashes": "List of hashes to rescan. Each next block must be a child of the previous.",
	"rescanBlocks--result0":    "List of matching blocks.",

	// RescannedBlock help.
	"rescannedBlock-hash":         "Hash of the matching block.",
	"rescannedBlock-transactions": "List of matching transactions, serialized and hex-encoded.",

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
}

// rpcResultTypes specifies the result types that each RPC command can return.
// This information is used to generate the help. Each result type must be a
// pointer to the type (or nil to indicate no return value).
var rpcResultTypes = map[string][]interface{}{
	"addManualNode":         nil,
	"createRawTransaction":  {(*string)(nil)},
	"debugLevel":            {(*string)(nil), (*string)(nil)},
	"decodeRawTransaction":  {(*jsonrpc.TxRawDecodeResult)(nil)},
	"decodeScript":          {(*jsonrpc.DecodeScriptResult)(nil)},
	"generate":              {(*[]string)(nil)},
	"getAllManualNodesInfo": {(*[]string)(nil), (*[]jsonrpc.GetManualNodeInfoResult)(nil)},
	"getSelectedTip":        {(*jsonrpc.GetBlockVerboseResult)(nil)},
	"getSelectedTipHash":    {(*string)(nil)},
	"getBlock":              {(*string)(nil), (*jsonrpc.GetBlockVerboseResult)(nil)},
	"getBlocks":             {(*jsonrpc.GetBlocksResult)(nil)},
	"getBlockCount":         {(*int64)(nil)},
	"getBlockHeader":        {(*string)(nil), (*jsonrpc.GetBlockHeaderVerboseResult)(nil)},
	"getBlockTemplate":      {(*jsonrpc.GetBlockTemplateResult)(nil), (*string)(nil), nil},
	"getBlockDagInfo":       {(*jsonrpc.GetBlockDAGInfoResult)(nil)},
	"getCFilter":            {(*string)(nil)},
	"getCFilterHeader":      {(*string)(nil)},
	"getChainFromBlock":     {(*jsonrpc.GetChainFromBlockResult)(nil)},
	"getConnectionCount":    {(*int32)(nil)},
	"getCurrentNet":         {(*uint32)(nil)},
	"getDifficulty":         {(*float64)(nil)},
	"getGenerate":           {(*bool)(nil)},
	"getHashesPerSec":       {(*float64)(nil)},
	"getTopHeaders":         {(*[]string)(nil)},
	"getHeaders":            {(*[]string)(nil)},
	"getInfo":               {(*jsonrpc.InfoDAGResult)(nil)},
	"getManualNodeInfo":     {(*string)(nil), (*jsonrpc.GetManualNodeInfoResult)(nil)},
	"getMempoolInfo":        {(*jsonrpc.GetMempoolInfoResult)(nil)},
	"getMiningInfo":         {(*jsonrpc.GetMiningInfoResult)(nil)},
	"getNetTotals":          {(*jsonrpc.GetNetTotalsResult)(nil)},
	"getNetworkHashPs":      {(*int64)(nil)},
	"getPeerInfo":           {(*[]jsonrpc.GetPeerInfoResult)(nil)},
	"getRawMempool":         {(*[]string)(nil), (*jsonrpc.GetRawMempoolVerboseResult)(nil)},
	"getRawTransaction":     {(*string)(nil), (*jsonrpc.TxRawResult)(nil)},
	"getSubnetwork":         {(*jsonrpc.GetSubnetworkResult)(nil)},
	"getTxOut":              {(*jsonrpc.GetTxOutResult)(nil)},
	"node":                  nil,
	"help":                  {(*string)(nil), (*string)(nil)},
	"ping":                  nil,
	"removeManualNode":      nil,
	"searchRawTransactions": {(*string)(nil), (*[]jsonrpc.SearchRawTransactionsResult)(nil)},
	"sendRawTransaction":    {(*string)(nil)},
	"setGenerate":           nil,
	"stop":                  {(*string)(nil)},
	"submitBlock":           {nil, (*string)(nil)},
	"uptime":                {(*int64)(nil)},
	"validateAddress":       {(*jsonrpc.ValidateAddressResult)(nil)},
	"version":               {(*map[string]jsonrpc.VersionResult)(nil)},

	// Websocket commands.
	"loadTxFilter":              nil,
	"session":                   {(*jsonrpc.SessionResult)(nil)},
	"notifyBlocks":              nil,
	"stopNotifyBlocks":          nil,
	"notifyChainChanges":        nil,
	"stopNotifyChainChanges":    nil,
	"notifyNewTransactions":     nil,
	"stopNotifyNewTransactions": nil,
	"rescanBlocks":              {(*[]jsonrpc.RescannedBlock)(nil)},
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
	help, err := jsonrpc.GenerateHelp(method, helpDescsEnUS, resultTypes...)
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
		usage, err := jsonrpc.MethodUsageText(k)
		if err != nil {
			return "", err
		}
		usageTexts = append(usageTexts, usage)
	}

	// Include websockets commands if requested.
	if includeWebsockets {
		for k := range wsHandlers {
			usage, err := jsonrpc.MethodUsageText(k)
			if err != nil {
				return "", err
			}
			usageTexts = append(usageTexts, usage)
		}
	}

	sort.Sort(sort.StringSlice(usageTexts))
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
