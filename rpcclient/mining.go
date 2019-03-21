// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/daglabs/btcd/btcjson"
	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/util"
)

// FutureGenerateResult is a future promise to deliver the result of a
// GenerateAsync RPC invocation (or an applicable error).
type FutureGenerateResult chan *response

// Receive waits for the response promised by the future and returns a list of
// block hashes generated by the call.
func (r FutureGenerateResult) Receive() ([]*daghash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a list of strings.
	var result []string
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	// Convert each block hash to a daghash.Hash and store a pointer to
	// each.
	convertedResult := make([]*daghash.Hash, len(result))
	for i, hashString := range result {
		convertedResult[i], err = daghash.NewHashFromStr(hashString)
		if err != nil {
			return nil, err
		}
	}

	return convertedResult, nil
}

// GenerateAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See Generate for the blocking version and more details.
func (c *Client) GenerateAsync(numBlocks uint32) FutureGenerateResult {
	cmd := btcjson.NewGenerateCmd(numBlocks)
	return c.sendCmd(cmd)
}

// Generate generates numBlocks blocks and returns their hashes.
func (c *Client) Generate(numBlocks uint32) ([]*daghash.Hash, error) {
	return c.GenerateAsync(numBlocks).Receive()
}

// FutureGetGenerateResult is a future promise to deliver the result of a
// GetGenerateAsync RPC invocation (or an applicable error).
type FutureGetGenerateResult chan *response

// Receive waits for the response promised by the future and returns true if the
// server is set to mine, otherwise false.
func (r FutureGetGenerateResult) Receive() (bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return false, err
	}

	// Unmarshal result as a boolean.
	var result bool
	err = json.Unmarshal(res, &result)
	if err != nil {
		return false, err
	}

	return result, nil
}

// GetGenerateAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetGenerate for the blocking version and more details.
func (c *Client) GetGenerateAsync() FutureGetGenerateResult {
	cmd := btcjson.NewGetGenerateCmd()
	return c.sendCmd(cmd)
}

// GetGenerate returns true if the server is set to mine, otherwise false.
func (c *Client) GetGenerate() (bool, error) {
	return c.GetGenerateAsync().Receive()
}

// FutureSetGenerateResult is a future promise to deliver the result of a
// SetGenerateAsync RPC invocation (or an applicable error).
type FutureSetGenerateResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when setting the server to generate coins (mine) or not.
func (r FutureSetGenerateResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

// SetGenerateAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See SetGenerate for the blocking version and more details.
func (c *Client) SetGenerateAsync(enable bool, numCPUs int) FutureSetGenerateResult {
	cmd := btcjson.NewSetGenerateCmd(enable, &numCPUs)
	return c.sendCmd(cmd)
}

// SetGenerate sets the server to generate coins (mine) or not.
func (c *Client) SetGenerate(enable bool, numCPUs int) error {
	return c.SetGenerateAsync(enable, numCPUs).Receive()
}

// FutureGetHashesPerSecResult is a future promise to deliver the result of a
// GetHashesPerSecAsync RPC invocation (or an applicable error).
type FutureGetHashesPerSecResult chan *response

// Receive waits for the response promised by the future and returns a recent
// hashes per second performance measurement while generating coins (mining).
// Zero is returned if the server is not mining.
func (r FutureGetHashesPerSecResult) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

	// Unmarshal result as an int64.
	var result int64
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// GetHashesPerSecAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetHashesPerSec for the blocking version and more details.
func (c *Client) GetHashesPerSecAsync() FutureGetHashesPerSecResult {
	cmd := btcjson.NewGetHashesPerSecCmd()
	return c.sendCmd(cmd)
}

// GetHashesPerSec returns a recent hashes per second performance measurement
// while generating coins (mining).  Zero is returned if the server is not
// mining.
func (c *Client) GetHashesPerSec() (int64, error) {
	return c.GetHashesPerSecAsync().Receive()
}

// FutureGetMiningInfoResult is a future promise to deliver the result of a
// GetMiningInfoAsync RPC invocation (or an applicable error).
type FutureGetMiningInfoResult chan *response

// Receive waits for the response promised by the future and returns the mining
// information.
func (r FutureGetMiningInfoResult) Receive() (*btcjson.GetMiningInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a getmininginfo result object.
	var infoResult btcjson.GetMiningInfoResult
	err = json.Unmarshal(res, &infoResult)
	if err != nil {
		return nil, err
	}

	return &infoResult, nil
}

// GetMiningInfoAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetMiningInfo for the blocking version and more details.
func (c *Client) GetMiningInfoAsync() FutureGetMiningInfoResult {
	cmd := btcjson.NewGetMiningInfoCmd()
	return c.sendCmd(cmd)
}

// GetMiningInfo returns mining information.
func (c *Client) GetMiningInfo() (*btcjson.GetMiningInfoResult, error) {
	return c.GetMiningInfoAsync().Receive()
}

// FutureGetNetworkHashPS is a future promise to deliver the result of a
// GetNetworkHashPSAsync RPC invocation (or an applicable error).
type FutureGetNetworkHashPS chan *response

// Receive waits for the response promised by the future and returns the
// estimated network hashes per second for the block heights provided by the
// parameters.
func (r FutureGetNetworkHashPS) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

	// Unmarshal result as an int64.
	var result int64
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// GetNetworkHashPSAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetNetworkHashPS for the blocking version and more details.
func (c *Client) GetNetworkHashPSAsync() FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(nil, nil)
	return c.sendCmd(cmd)
}

// GetNetworkHashPS returns the estimated network hashes per second using the
// default number of blocks and the most recent block height.
//
// See GetNetworkHashPS2 to override the number of blocks to use and
// GetNetworkHashPS3 to override the height at which to calculate the estimate.
func (c *Client) GetNetworkHashPS() (int64, error) {
	return c.GetNetworkHashPSAsync().Receive()
}

// GetNetworkHashPS2Async returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetNetworkHashPS2 for the blocking version and more details.
func (c *Client) GetNetworkHashPS2Async(blocks int) FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(&blocks, nil)
	return c.sendCmd(cmd)
}

// GetNetworkHashPS2 returns the estimated network hashes per second for the
// specified previous number of blocks working backwards from the most recent
// block height.  The blocks parameter can also be -1 in which case the number
// of blocks since the last difficulty change will be used.
//
// See GetNetworkHashPS to use defaults and GetNetworkHashPS3 to override the
// height at which to calculate the estimate.
func (c *Client) GetNetworkHashPS2(blocks int) (int64, error) {
	return c.GetNetworkHashPS2Async(blocks).Receive()
}

// GetNetworkHashPS3Async returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetNetworkHashPS3 for the blocking version and more details.
func (c *Client) GetNetworkHashPS3Async(blocks, height int) FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(&blocks, &height)
	return c.sendCmd(cmd)
}

// GetNetworkHashPS3 returns the estimated network hashes per second for the
// specified previous number of blocks working backwards from the specified
// block height.  The blocks parameter can also be -1 in which case the number
// of blocks since the last difficulty change will be used.
//
// See GetNetworkHashPS and GetNetworkHashPS2 to use defaults.
func (c *Client) GetNetworkHashPS3(blocks, height int) (int64, error) {
	return c.GetNetworkHashPS3Async(blocks, height).Receive()
}

// FutureSubmitBlockResult is a future promise to deliver the result of a
// SubmitBlockAsync RPC invocation (or an applicable error).
type FutureSubmitBlockResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when submitting the block.
func (r FutureSubmitBlockResult) Receive() error {
	res, err := receiveFuture(r)
	if err != nil {
		return err
	}

	if string(res) != "null" {
		var result string
		err = json.Unmarshal(res, &result)
		if err != nil {
			return err
		}

		return errors.New(result)
	}

	return nil
}

// SubmitBlockAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See SubmitBlock for the blocking version and more details.
func (c *Client) SubmitBlockAsync(block *util.Block, options *btcjson.SubmitBlockOptions) FutureSubmitBlockResult {
	blockHex := ""
	if block != nil {
		blockBytes, err := block.Bytes()
		if err != nil {
			return newFutureError(err)
		}

		blockHex = hex.EncodeToString(blockBytes)
	}

	cmd := btcjson.NewSubmitBlockCmd(blockHex, options)
	return c.sendCmd(cmd)
}

// SubmitBlock attempts to submit a new block into the bitcoin network.
func (c *Client) SubmitBlock(block *util.Block, options *btcjson.SubmitBlockOptions) error {
	return c.SubmitBlockAsync(block, options).Receive()
}

// FutureGetBlockTemplateResult is a future promise to deliver the result of a
// GetBlockTemplate RPC invocation (or an applicable error).
type FutureGetBlockTemplateResult chan *response

// GetBlockTemplateAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetBlockTemplate for the blocking version and more details
func (c *Client) GetBlockTemplateAsync() FutureGetBlockTemplateResult {
	request := &btcjson.TemplateRequest{
		Mode: "template",
	}
	cmd := btcjson.NewGetBlockTemplateCmd(request)
	return c.sendCmd(cmd)
}

// Receive waits for the response promised by the future and returns an error if
// any occurred when submitting the block.
func (r FutureGetBlockTemplateResult) Receive() (*btcjson.GetBlockTemplateResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var result btcjson.GetBlockTemplateResult
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBlockTemplate request a block template from the server, to mine upon
func (c *Client) GetBlockTemplate() (*btcjson.GetBlockTemplateResult, error) {
	return c.GetBlockTemplateAsync().Receive()
}
