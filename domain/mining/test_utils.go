package mining

// This file functions are not considered safe for regular use, and should be used for test purposes only.

import (
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/domain/blockdag"
	"github.com/kaspanet/kaspad/domain/txscript"
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
)

// fakeTxSource is a simple implementation of TxSource interface
type fakeTxSource struct {
	txDescs []*TxDesc
}

func (txs *fakeTxSource) LastUpdated() mstime.Time {
	return mstime.UnixMilliseconds(0)
}

func (txs *fakeTxSource) MiningDescs() []*TxDesc {
	return txs.txDescs
}

func (txs *fakeTxSource) HaveTransaction(txID *daghash.TxID) bool {
	for _, desc := range txs.txDescs {
		if *desc.Tx.ID() == *txID {
			return true
		}
	}
	return false
}

// PrepareBlockForTest generates a block with the proper merkle roots, coinbase transaction etc. This function is used for test purposes only
func PrepareBlockForTest(dag *blockdag.BlockDAG, parentHashes []*daghash.Hash, transactions []*domainmessage.MsgTx, forceTransactions bool,
) (*domainmessage.MsgBlock, error) {

	newVirtual, err := blockdag.GetVirtualFromParentsForTest(dag, parentHashes)
	if err != nil {
		return nil, err
	}
	oldVirtual := blockdag.SetVirtualForTest(dag, newVirtual)
	defer blockdag.SetVirtualForTest(dag, oldVirtual)
	policy := Policy{
		BlockMaxMass: 50000,
	}

	txSource := &fakeTxSource{
		txDescs: make([]*TxDesc, len(transactions)),
	}

	for i, tx := range transactions {
		txSource.txDescs[i] = &TxDesc{
			Tx:  util.NewTx(tx),
			Fee: 1,
		}
	}

	blockTemplateGenerator := NewBlkTmplGenerator(&policy, txSource, dag, txscript.NewSigCache(100000))

	OpTrueAddr, err := OpTrueAddress(dag.Params.Prefix)
	if err != nil {
		return nil, err
	}

	// We create a deterministic extra nonce in order of
	// creating deterministic coinbase tx ids.
	extraNonce := GenerateDeterministicExtraNonceForTest()

	template, err := blockTemplateGenerator.NewBlockTemplate(OpTrueAddr, extraNonce)
	if err != nil {
		return nil, err
	}

	txsToAdd := make([]*domainmessage.MsgTx, 0)
	for _, tx := range transactions {
		found := false
		for _, blockTx := range template.Block.Transactions {
			if blockTx.TxHash().IsEqual(tx.TxHash()) {
				found = true
				break
			}
		}
		if !found {
			if !forceTransactions {
				return nil, errors.Errorf("tx %s wasn't found in the block", tx.TxHash())
			}
			txsToAdd = append(txsToAdd, tx)
		}
	}
	if forceTransactions && len(txsToAdd) > 0 {
		template.Block.Transactions = append(template.Block.Transactions, txsToAdd...)
	}
	updateHeaderFields := forceTransactions && len(txsToAdd) > 0
	if updateHeaderFields {
		utilTxs := make([]*util.Tx, len(template.Block.Transactions))
		for i, tx := range template.Block.Transactions {
			utilTxs[i] = util.NewTx(tx)
		}
		template.Block.Header.HashMerkleRoot = blockdag.BuildHashMerkleTreeStore(utilTxs).Root()

		ms, err := dag.NextBlockMultiset()
		if err != nil {
			return nil, err
		}

		template.Block.Header.UTXOCommitment = (*daghash.Hash)(ms.Finalize())
	}
	return template.Block, nil
}

// GenerateDeterministicExtraNonceForTest returns a unique deterministic extra nonce for coinbase data, in order to create unique coinbase transactions.
func GenerateDeterministicExtraNonceForTest() uint64 {
	extraNonceForTest++
	return extraNonceForTest
}

// OpTrueAddress returns an address pointing to a P2SH anyone-can-spend script
func OpTrueAddress(prefix util.Bech32Prefix) (util.Address, error) {
	return util.NewAddressScriptHash(blockdag.OpTrueScript, prefix)
}

var extraNonceForTest = uint64(0)
