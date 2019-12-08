// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC commands that are supported by
// a dag server.

package btcjson

import (
	"encoding/json"
	"fmt"

	"github.com/kaspanet/kaspad/wire"
)

// AddManualNodeCmd defines the addManualNode JSON-RPC command.
type AddManualNodeCmd struct {
	Addr   string
	OneTry *bool `jsonrpcdefault:"false"`
}

// NewAddManualNodeCmd returns a new instance which can be used to issue an addManualNode
// JSON-RPC command.
func NewAddManualNodeCmd(addr string, oneTry *bool) *AddManualNodeCmd {
	return &AddManualNodeCmd{
		Addr:   addr,
		OneTry: oneTry,
	}
}

// RemoveManualNodeCmd defines the removeManualNode JSON-RPC command.
type RemoveManualNodeCmd struct {
	Addr string
}

// NewRemoveManualNodeCmd returns a new instance which can be used to issue an removeManualNode
// JSON-RPC command.
func NewRemoveManualNodeCmd(addr string) *RemoveManualNodeCmd {
	return &RemoveManualNodeCmd{
		Addr: addr,
	}
}

// TransactionInput represents the inputs to a transaction.  Specifically a
// transaction hash and output number pair.
type TransactionInput struct {
	TxID string `json:"txId"`
	Vout uint32 `json:"vout"`
}

// CreateRawTransactionCmd defines the createRawTransaction JSON-RPC command.
type CreateRawTransactionCmd struct {
	Inputs   []TransactionInput
	Amounts  map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"` // In BTC
	LockTime *uint64
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createRawTransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewCreateRawTransactionCmd(inputs []TransactionInput, amounts map[string]float64,
	lockTime *uint64) *CreateRawTransactionCmd {

	return &CreateRawTransactionCmd{
		Inputs:   inputs,
		Amounts:  amounts,
		LockTime: lockTime,
	}
}

// DecodeRawTransactionCmd defines the decodeRawTransaction JSON-RPC command.
type DecodeRawTransactionCmd struct {
	HexTx string
}

// NewDecodeRawTransactionCmd returns a new instance which can be used to issue
// a decodeRawTransaction JSON-RPC command.
func NewDecodeRawTransactionCmd(hexTx string) *DecodeRawTransactionCmd {
	return &DecodeRawTransactionCmd{
		HexTx: hexTx,
	}
}

// DecodeScriptCmd defines the decodeScript JSON-RPC command.
type DecodeScriptCmd struct {
	HexScript string
}

// NewDecodeScriptCmd returns a new instance which can be used to issue a
// decodeScript JSON-RPC command.
func NewDecodeScriptCmd(hexScript string) *DecodeScriptCmd {
	return &DecodeScriptCmd{
		HexScript: hexScript,
	}
}

// GetManualNodeInfoCmd defines the getManualNodeInfo JSON-RPC command.
type GetManualNodeInfoCmd struct {
	Node    string
	Details *bool `jsonrpcdefault:"true"`
}

// NewGetManualNodeInfoCmd returns a new instance which can be used to issue a
// getManualNodeInfo JSON-RPC command.
func NewGetManualNodeInfoCmd(node string, details *bool) *GetManualNodeInfoCmd {
	return &GetManualNodeInfoCmd{
		Details: details,
		Node:    node,
	}
}

// GetAllManualNodesInfoCmd defines the getAllManualNodesInfo JSON-RPC command.
type GetAllManualNodesInfoCmd struct {
	Details *bool `jsonrpcdefault:"true"`
}

// NewGetAllManualNodesInfoCmd returns a new instance which can be used to issue a
// getAllManualNodesInfo JSON-RPC command.
func NewGetAllManualNodesInfoCmd(details *bool) *GetAllManualNodesInfoCmd {
	return &GetAllManualNodesInfoCmd{
		Details: details,
	}
}

// GetSelectedTipHashCmd defines the getSelectedTipHash JSON-RPC command.
type GetSelectedTipHashCmd struct{}

// NewGetSelectedTipHashCmd returns a new instance which can be used to issue a
// getSelectedTipHash JSON-RPC command.
func NewGetSelectedTipHashCmd() *GetSelectedTipHashCmd {
	return &GetSelectedTipHashCmd{}
}

// GetBlockCmd defines the getBlock JSON-RPC command.
type GetBlockCmd struct {
	Hash       string
	Verbose    *bool `jsonrpcdefault:"true"`
	VerboseTx  *bool `jsonrpcdefault:"false"`
	Subnetwork *string
}

// NewGetBlockCmd returns a new instance which can be used to issue a getBlock
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetBlockCmd(hash string, verbose, verboseTx *bool, subnetworkID *string) *GetBlockCmd {
	return &GetBlockCmd{
		Hash:       hash,
		Verbose:    verbose,
		VerboseTx:  verboseTx,
		Subnetwork: subnetworkID,
	}
}

// GetBlocksCmd defines the getBlocks JSON-RPC command.
type GetBlocksCmd struct {
	IncludeRawBlockData     bool    `json:"includeRawBlockData"`
	IncludeVerboseBlockData bool    `json:"includeVerboseBlockData"`
	StartHash               *string `json:"startHash"`
}

// NewGetBlocksCmd returns a new instance which can be used to issue a
// GetGetBlocks JSON-RPC command.
func NewGetBlocksCmd(includeRawBlockData bool, includeVerboseBlockData bool, startHash *string) *GetBlocksCmd {
	return &GetBlocksCmd{
		IncludeRawBlockData:     includeRawBlockData,
		IncludeVerboseBlockData: includeVerboseBlockData,
		StartHash:               startHash,
	}
}

// GetBlockDAGInfoCmd defines the getBlockDagInfo JSON-RPC command.
type GetBlockDAGInfoCmd struct{}

// NewGetBlockDAGInfoCmd returns a new instance which can be used to issue a
// getBlockDagInfo JSON-RPC command.
func NewGetBlockDAGInfoCmd() *GetBlockDAGInfoCmd {
	return &GetBlockDAGInfoCmd{}
}

// GetBlockCountCmd defines the getBlockCount JSON-RPC command.
type GetBlockCountCmd struct{}

// NewGetBlockCountCmd returns a new instance which can be used to issue a
// getBlockCount JSON-RPC command.
func NewGetBlockCountCmd() *GetBlockCountCmd {
	return &GetBlockCountCmd{}
}

// GetBlockHeaderCmd defines the getBlockHeader JSON-RPC command.
type GetBlockHeaderCmd struct {
	Hash    string
	Verbose *bool `jsonrpcdefault:"true"`
}

// NewGetBlockHeaderCmd returns a new instance which can be used to issue a
// getBlockHeader JSON-RPC command.
func NewGetBlockHeaderCmd(hash string, verbose *bool) *GetBlockHeaderCmd {
	return &GetBlockHeaderCmd{
		Hash:    hash,
		Verbose: verbose,
	}
}

// TemplateRequest is a request object as defined in BIP22
// (https://en.bitcoin.it/wiki/BIP_0022), it is optionally provided as an
// pointer argument to GetBlockTemplateCmd.
type TemplateRequest struct {
	Mode         string   `json:"mode,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`

	// Optional long polling.
	LongPollID string `json:"longPollId,omitempty"`

	// Optional template tweaking.  SigOpLimit and MassLimit can be int64
	// or bool.
	SigOpLimit interface{} `json:"sigOpLimit,omitempty"`
	MassLimit  interface{} `json:"massLimit,omitempty"`
	MaxVersion uint32      `json:"maxVersion,omitempty"`

	// Basic pool extension from BIP 0023.
	Target string `json:"target,omitempty"`

	// Block proposal from BIP 0023.  Data is only provided when Mode is
	// "proposal".
	Data   string `json:"data,omitempty"`
	WorkID string `json:"workId,omitempty"`
}

// convertTemplateRequestField potentially converts the provided value as
// needed.
func convertTemplateRequestField(fieldName string, iface interface{}) (interface{}, error) {
	switch val := iface.(type) {
	case nil:
		return nil, nil
	case bool:
		return val, nil
	case float64:
		if val == float64(int64(val)) {
			return int64(val), nil
		}
	}

	str := fmt.Sprintf("the %s field must be unspecified, a boolean, or "+
		"a 64-bit integer", fieldName)
	return nil, makeError(ErrInvalidType, str)
}

// UnmarshalJSON provides a custom Unmarshal method for TemplateRequest.  This
// is necessary because the SigOpLimit and MassLimit fields can only be specific
// types.
func (t *TemplateRequest) UnmarshalJSON(data []byte) error {
	type templateRequest TemplateRequest

	request := (*templateRequest)(t)
	if err := json.Unmarshal(data, &request); err != nil {
		return err
	}

	// The SigOpLimit field can only be nil, bool, or int64.
	val, err := convertTemplateRequestField("sigOpLimit", request.SigOpLimit)
	if err != nil {
		return err
	}
	request.SigOpLimit = val

	// The MassLimit field can only be nil, bool, or int64.
	val, err = convertTemplateRequestField("massLimit", request.MassLimit)
	if err != nil {
		return err
	}
	request.MassLimit = val

	return nil
}

// GetBlockTemplateCmd defines the getBlockTemplate JSON-RPC command.
type GetBlockTemplateCmd struct {
	Request *TemplateRequest
}

// NewGetBlockTemplateCmd returns a new instance which can be used to issue a
// getBlockTemplate JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetBlockTemplateCmd(request *TemplateRequest) *GetBlockTemplateCmd {
	return &GetBlockTemplateCmd{
		Request: request,
	}
}

// GetCFilterCmd defines the getCFilter JSON-RPC command.
type GetCFilterCmd struct {
	Hash       string
	FilterType wire.FilterType
}

// NewGetCFilterCmd returns a new instance which can be used to issue a
// getCFilter JSON-RPC command.
func NewGetCFilterCmd(hash string, filterType wire.FilterType) *GetCFilterCmd {
	return &GetCFilterCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

// GetCFilterHeaderCmd defines the getCFilterHeader JSON-RPC command.
type GetCFilterHeaderCmd struct {
	Hash       string
	FilterType wire.FilterType
}

// NewGetCFilterHeaderCmd returns a new instance which can be used to issue a
// getCFilterHeader JSON-RPC command.
func NewGetCFilterHeaderCmd(hash string,
	filterType wire.FilterType) *GetCFilterHeaderCmd {
	return &GetCFilterHeaderCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

// GetChainFromBlockCmd defines the getChainFromBlock JSON-RPC command.
type GetChainFromBlockCmd struct {
	IncludeBlocks bool    `json:"includeBlocks"`
	StartHash     *string `json:"startHash"`
}

// NewGetChainFromBlockCmd returns a new instance which can be used to issue a
// GetChainFromBlock JSON-RPC command.
func NewGetChainFromBlockCmd(includeBlocks bool, startHash *string) *GetChainFromBlockCmd {
	return &GetChainFromBlockCmd{
		IncludeBlocks: includeBlocks,
		StartHash:     startHash,
	}
}

// GetDAGTipsCmd defines the getDagTips JSON-RPC command.
type GetDAGTipsCmd struct{}

// NewGetDAGTipsCmd returns a new instance which can be used to issue a
// getDagTips JSON-RPC command.
func NewGetDAGTipsCmd() *GetDAGTipsCmd {
	return &GetDAGTipsCmd{}
}

// GetConnectionCountCmd defines the getConnectionCount JSON-RPC command.
type GetConnectionCountCmd struct{}

// NewGetConnectionCountCmd returns a new instance which can be used to issue a
// getConnectionCount JSON-RPC command.
func NewGetConnectionCountCmd() *GetConnectionCountCmd {
	return &GetConnectionCountCmd{}
}

// GetDifficultyCmd defines the getDifficulty JSON-RPC command.
type GetDifficultyCmd struct{}

// NewGetDifficultyCmd returns a new instance which can be used to issue a
// getDifficulty JSON-RPC command.
func NewGetDifficultyCmd() *GetDifficultyCmd {
	return &GetDifficultyCmd{}
}

// GetGenerateCmd defines the getGenerate JSON-RPC command.
type GetGenerateCmd struct{}

// NewGetGenerateCmd returns a new instance which can be used to issue a
// getGenerate JSON-RPC command.
func NewGetGenerateCmd() *GetGenerateCmd {
	return &GetGenerateCmd{}
}

// GetHashesPerSecCmd defines the getHashesPerSec JSON-RPC command.
type GetHashesPerSecCmd struct{}

// NewGetHashesPerSecCmd returns a new instance which can be used to issue a
// getHashesPerSec JSON-RPC command.
func NewGetHashesPerSecCmd() *GetHashesPerSecCmd {
	return &GetHashesPerSecCmd{}
}

// GetInfoCmd defines the getInfo JSON-RPC command.
type GetInfoCmd struct{}

// NewGetInfoCmd returns a new instance which can be used to issue a
// getInfo JSON-RPC command.
func NewGetInfoCmd() *GetInfoCmd {
	return &GetInfoCmd{}
}

// GetMempoolEntryCmd defines the getMempoolEntry JSON-RPC command.
type GetMempoolEntryCmd struct {
	TxID string
}

// NewGetMempoolEntryCmd returns a new instance which can be used to issue a
// getMempoolEntry JSON-RPC command.
func NewGetMempoolEntryCmd(txHash string) *GetMempoolEntryCmd {
	return &GetMempoolEntryCmd{
		TxID: txHash,
	}
}

// GetMempoolInfoCmd defines the getMempoolInfo JSON-RPC command.
type GetMempoolInfoCmd struct{}

// NewGetMempoolInfoCmd returns a new instance which can be used to issue a
// getmempool JSON-RPC command.
func NewGetMempoolInfoCmd() *GetMempoolInfoCmd {
	return &GetMempoolInfoCmd{}
}

// GetMiningInfoCmd defines the getMiningInfo JSON-RPC command.
type GetMiningInfoCmd struct{}

// NewGetMiningInfoCmd returns a new instance which can be used to issue a
// getMiningInfo JSON-RPC command.
func NewGetMiningInfoCmd() *GetMiningInfoCmd {
	return &GetMiningInfoCmd{}
}

// GetNetworkInfoCmd defines the getNetworkInfo JSON-RPC command.
type GetNetworkInfoCmd struct{}

// NewGetNetworkInfoCmd returns a new instance which can be used to issue a
// getNetworkInfo JSON-RPC command.
func NewGetNetworkInfoCmd() *GetNetworkInfoCmd {
	return &GetNetworkInfoCmd{}
}

// GetNetTotalsCmd defines the getNetTotals JSON-RPC command.
type GetNetTotalsCmd struct{}

// NewGetNetTotalsCmd returns a new instance which can be used to issue a
// getNetTotals JSON-RPC command.
func NewGetNetTotalsCmd() *GetNetTotalsCmd {
	return &GetNetTotalsCmd{}
}

// GetNetworkHashPSCmd defines the getNetworkHashPs JSON-RPC command.
type GetNetworkHashPSCmd struct {
	Blocks *int `jsonrpcdefault:"120"`
	Height *int `jsonrpcdefault:"-1"`
}

// NewGetNetworkHashPSCmd returns a new instance which can be used to issue a
// getNetworkHashPs JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetNetworkHashPSCmd(numBlocks, height *int) *GetNetworkHashPSCmd {
	return &GetNetworkHashPSCmd{
		Blocks: numBlocks,
		Height: height,
	}
}

// GetPeerInfoCmd defines the getPeerInfo JSON-RPC command.
type GetPeerInfoCmd struct{}

// NewGetPeerInfoCmd returns a new instance which can be used to issue a getpeer
// JSON-RPC command.
func NewGetPeerInfoCmd() *GetPeerInfoCmd {
	return &GetPeerInfoCmd{}
}

// GetRawMempoolCmd defines the getmempool JSON-RPC command.
type GetRawMempoolCmd struct {
	Verbose *bool `jsonrpcdefault:"false"`
}

// NewGetRawMempoolCmd returns a new instance which can be used to issue a
// getRawMempool JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetRawMempoolCmd(verbose *bool) *GetRawMempoolCmd {
	return &GetRawMempoolCmd{
		Verbose: verbose,
	}
}

// GetRawTransactionCmd defines the getRawTransaction JSON-RPC command.
//
// NOTE: This field is an int versus a bool to remain compatible with Bitcoin
// Core even though it really should be a bool.
type GetRawTransactionCmd struct {
	TxID    string
	Verbose *int `jsonrpcdefault:"0"`
}

// NewGetRawTransactionCmd returns a new instance which can be used to issue a
// getRawTransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetRawTransactionCmd(txID string, verbose *int) *GetRawTransactionCmd {
	return &GetRawTransactionCmd{
		TxID:    txID,
		Verbose: verbose,
	}
}

// GetSubnetworkCmd defines the getSubnetwork JSON-RPC command.
type GetSubnetworkCmd struct {
	SubnetworkID string
}

// NewGetSubnetworkCmd returns a new instance which can be used to issue a
// getSubnetworkCmd command.
func NewGetSubnetworkCmd(subnetworkID string) *GetSubnetworkCmd {
	return &GetSubnetworkCmd{
		SubnetworkID: subnetworkID,
	}
}

// GetTxOutCmd defines the getTxOut JSON-RPC command.
type GetTxOutCmd struct {
	TxID           string
	Vout           uint32
	IncludeMempool *bool `jsonrpcdefault:"true"`
}

// NewGetTxOutCmd returns a new instance which can be used to issue a getTxOut
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetTxOutCmd(txHash string, vout uint32, includeMempool *bool) *GetTxOutCmd {
	return &GetTxOutCmd{
		TxID:           txHash,
		Vout:           vout,
		IncludeMempool: includeMempool,
	}
}

// GetTxOutSetInfoCmd defines the getTxOutSetInfo JSON-RPC command.
type GetTxOutSetInfoCmd struct{}

// NewGetTxOutSetInfoCmd returns a new instance which can be used to issue a
// getTxOutSetInfo JSON-RPC command.
func NewGetTxOutSetInfoCmd() *GetTxOutSetInfoCmd {
	return &GetTxOutSetInfoCmd{}
}

// HelpCmd defines the help JSON-RPC command.
type HelpCmd struct {
	Command *string
}

// NewHelpCmd returns a new instance which can be used to issue a help JSON-RPC
// command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewHelpCmd(command *string) *HelpCmd {
	return &HelpCmd{
		Command: command,
	}
}

// InvalidateBlockCmd defines the invalidateBlock JSON-RPC command.
type InvalidateBlockCmd struct {
	BlockHash string
}

// NewInvalidateBlockCmd returns a new instance which can be used to issue a
// invalidateBlock JSON-RPC command.
func NewInvalidateBlockCmd(blockHash string) *InvalidateBlockCmd {
	return &InvalidateBlockCmd{
		BlockHash: blockHash,
	}
}

// PingCmd defines the ping JSON-RPC command.
type PingCmd struct{}

// NewPingCmd returns a new instance which can be used to issue a ping JSON-RPC
// command.
func NewPingCmd() *PingCmd {
	return &PingCmd{}
}

// PreciousBlockCmd defines the preciousBlock JSON-RPC command.
type PreciousBlockCmd struct {
	BlockHash string
}

// NewPreciousBlockCmd returns a new instance which can be used to issue a
// preciousBlock JSON-RPC command.
func NewPreciousBlockCmd(blockHash string) *PreciousBlockCmd {
	return &PreciousBlockCmd{
		BlockHash: blockHash,
	}
}

// ReconsiderBlockCmd defines the reconsiderBlock JSON-RPC command.
type ReconsiderBlockCmd struct {
	BlockHash string
}

// NewReconsiderBlockCmd returns a new instance which can be used to issue a
// reconsiderBlock JSON-RPC command.
func NewReconsiderBlockCmd(blockHash string) *ReconsiderBlockCmd {
	return &ReconsiderBlockCmd{
		BlockHash: blockHash,
	}
}

// SearchRawTransactionsCmd defines the searchRawTransactions JSON-RPC command.
type SearchRawTransactionsCmd struct {
	Address     string
	Verbose     *bool `jsonrpcdefault:"true"`
	Skip        *int  `jsonrpcdefault:"0"`
	Count       *int  `jsonrpcdefault:"100"`
	VinExtra    *bool `jsonrpcdefault:"false"`
	Reverse     *bool `jsonrpcdefault:"false"`
	FilterAddrs *[]string
}

// NewSearchRawTransactionsCmd returns a new instance which can be used to issue a
// sendRawTransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSearchRawTransactionsCmd(address string, verbose *bool, skip, count *int, vinExtra, reverse *bool, filterAddrs *[]string) *SearchRawTransactionsCmd {
	return &SearchRawTransactionsCmd{
		Address:     address,
		Verbose:     verbose,
		Skip:        skip,
		Count:       count,
		VinExtra:    vinExtra,
		Reverse:     reverse,
		FilterAddrs: filterAddrs,
	}
}

// SendRawTransactionCmd defines the sendRawTransaction JSON-RPC command.
type SendRawTransactionCmd struct {
	HexTx         string
	AllowHighFees *bool `jsonrpcdefault:"false"`
}

// NewSendRawTransactionCmd returns a new instance which can be used to issue a
// sendRawTransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSendRawTransactionCmd(hexTx string, allowHighFees *bool) *SendRawTransactionCmd {
	return &SendRawTransactionCmd{
		HexTx:         hexTx,
		AllowHighFees: allowHighFees,
	}
}

// SetGenerateCmd defines the setGenerate JSON-RPC command.
type SetGenerateCmd struct {
	Generate     bool
	GenProcLimit *int `jsonrpcdefault:"-1"`
}

// NewSetGenerateCmd returns a new instance which can be used to issue a
// setGenerate JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSetGenerateCmd(generate bool, genProcLimit *int) *SetGenerateCmd {
	return &SetGenerateCmd{
		Generate:     generate,
		GenProcLimit: genProcLimit,
	}
}

// StopCmd defines the stop JSON-RPC command.
type StopCmd struct{}

// NewStopCmd returns a new instance which can be used to issue a stop JSON-RPC
// command.
func NewStopCmd() *StopCmd {
	return &StopCmd{}
}

// SubmitBlockOptions represents the optional options struct provided with a
// SubmitBlockCmd command.
type SubmitBlockOptions struct {
	// must be provided if server provided a workid with template.
	WorkID string `json:"workId,omitempty"`
}

// SubmitBlockCmd defines the submitBlock JSON-RPC command.
type SubmitBlockCmd struct {
	HexBlock string
	Options  *SubmitBlockOptions
}

// NewSubmitBlockCmd returns a new instance which can be used to issue a
// submitBlock JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSubmitBlockCmd(hexBlock string, options *SubmitBlockOptions) *SubmitBlockCmd {
	return &SubmitBlockCmd{
		HexBlock: hexBlock,
		Options:  options,
	}
}

// UptimeCmd defines the uptime JSON-RPC command.
type UptimeCmd struct{}

// NewUptimeCmd returns a new instance which can be used to issue an uptime JSON-RPC command.
func NewUptimeCmd() *UptimeCmd {
	return &UptimeCmd{}
}

// ValidateAddressCmd defines the validateAddress JSON-RPC command.
type ValidateAddressCmd struct {
	Address string
}

// NewValidateAddressCmd returns a new instance which can be used to issue a
// validateAddress JSON-RPC command.
func NewValidateAddressCmd(address string) *ValidateAddressCmd {
	return &ValidateAddressCmd{
		Address: address,
	}
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	MustRegisterCmd("addManualNode", (*AddManualNodeCmd)(nil), flags)
	MustRegisterCmd("createRawTransaction", (*CreateRawTransactionCmd)(nil), flags)
	MustRegisterCmd("decodeRawTransaction", (*DecodeRawTransactionCmd)(nil), flags)
	MustRegisterCmd("decodeScript", (*DecodeScriptCmd)(nil), flags)
	MustRegisterCmd("getAllManualNodesInfo", (*GetAllManualNodesInfoCmd)(nil), flags)
	MustRegisterCmd("getSelectedTipHash", (*GetSelectedTipHashCmd)(nil), flags)
	MustRegisterCmd("getBlock", (*GetBlockCmd)(nil), flags)
	MustRegisterCmd("getBlocks", (*GetBlocksCmd)(nil), flags)
	MustRegisterCmd("getBlockDagInfo", (*GetBlockDAGInfoCmd)(nil), flags)
	MustRegisterCmd("getBlockCount", (*GetBlockCountCmd)(nil), flags)
	MustRegisterCmd("getBlockHeader", (*GetBlockHeaderCmd)(nil), flags)
	MustRegisterCmd("getBlockTemplate", (*GetBlockTemplateCmd)(nil), flags)
	MustRegisterCmd("getCFilter", (*GetCFilterCmd)(nil), flags)
	MustRegisterCmd("getCFilterHeader", (*GetCFilterHeaderCmd)(nil), flags)
	MustRegisterCmd("getChainFromBlock", (*GetChainFromBlockCmd)(nil), flags)
	MustRegisterCmd("getDagTips", (*GetDAGTipsCmd)(nil), flags)
	MustRegisterCmd("getConnectionCount", (*GetConnectionCountCmd)(nil), flags)
	MustRegisterCmd("getDifficulty", (*GetDifficultyCmd)(nil), flags)
	MustRegisterCmd("getGenerate", (*GetGenerateCmd)(nil), flags)
	MustRegisterCmd("getHashesPerSec", (*GetHashesPerSecCmd)(nil), flags)
	MustRegisterCmd("getInfo", (*GetInfoCmd)(nil), flags)
	MustRegisterCmd("getManualNodeInfo", (*GetManualNodeInfoCmd)(nil), flags)
	MustRegisterCmd("getMempoolEntry", (*GetMempoolEntryCmd)(nil), flags)
	MustRegisterCmd("getMempoolInfo", (*GetMempoolInfoCmd)(nil), flags)
	MustRegisterCmd("getMiningInfo", (*GetMiningInfoCmd)(nil), flags)
	MustRegisterCmd("getNetworkInfo", (*GetNetworkInfoCmd)(nil), flags)
	MustRegisterCmd("getNetTotals", (*GetNetTotalsCmd)(nil), flags)
	MustRegisterCmd("getNetworkHashPs", (*GetNetworkHashPSCmd)(nil), flags)
	MustRegisterCmd("getPeerInfo", (*GetPeerInfoCmd)(nil), flags)
	MustRegisterCmd("getRawMempool", (*GetRawMempoolCmd)(nil), flags)
	MustRegisterCmd("getRawTransaction", (*GetRawTransactionCmd)(nil), flags)
	MustRegisterCmd("getSubnetwork", (*GetSubnetworkCmd)(nil), flags)
	MustRegisterCmd("getTxOut", (*GetTxOutCmd)(nil), flags)
	MustRegisterCmd("getTxOutSetInfo", (*GetTxOutSetInfoCmd)(nil), flags)
	MustRegisterCmd("help", (*HelpCmd)(nil), flags)
	MustRegisterCmd("invalidateBlock", (*InvalidateBlockCmd)(nil), flags)
	MustRegisterCmd("ping", (*PingCmd)(nil), flags)
	MustRegisterCmd("preciousBlock", (*PreciousBlockCmd)(nil), flags)
	MustRegisterCmd("reconsiderBlock", (*ReconsiderBlockCmd)(nil), flags)
	MustRegisterCmd("removeManualNode", (*RemoveManualNodeCmd)(nil), flags)
	MustRegisterCmd("searchRawTransactions", (*SearchRawTransactionsCmd)(nil), flags)
	MustRegisterCmd("sendRawTransaction", (*SendRawTransactionCmd)(nil), flags)
	MustRegisterCmd("setGenerate", (*SetGenerateCmd)(nil), flags)
	MustRegisterCmd("stop", (*StopCmd)(nil), flags)
	MustRegisterCmd("submitBlock", (*SubmitBlockCmd)(nil), flags)
	MustRegisterCmd("uptime", (*UptimeCmd)(nil), flags)
	MustRegisterCmd("validateAddress", (*ValidateAddressCmd)(nil), flags)
}
