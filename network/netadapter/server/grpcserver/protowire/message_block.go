package protowire

import (
	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/util/mstime"
	"github.com/pkg/errors"
)

func (x *KaspadMessage_Block) toDomainMessage() (domainmessage.Message, error) {
	return x.Block.toDomainMessage()
}

func (x *KaspadMessage_Block) fromDomainMessage(msgBlock *domainmessage.MsgBlock) error {
	x.Block = new(BlockMessage)
	return x.Block.fromDomainMessage(msgBlock)
}

func (x *BlockMessage) toDomainMessage() (domainmessage.Message, error) {
	if len(x.Transactions) > domainmessage.MaxTxPerBlock {
		return nil, errors.Errorf("too many transactions to fit into a block "+
			"[count %d, max %d]", len(x.Transactions), domainmessage.MaxTxPerBlock)
	}

	protoBlockHeader := x.Header
	if protoBlockHeader == nil {
		return nil, errors.New("block header field cannot be nil")
	}

	if len(protoBlockHeader.ParentHashes) > domainmessage.MaxBlockParents {
		return nil, errors.Errorf("block header has %d parents, but the maximum allowed amount "+
			"is %d", len(protoBlockHeader.ParentHashes), domainmessage.MaxBlockParents)
	}

	parentHashes, err := protoHashesToWire(protoBlockHeader.ParentHashes)
	if err != nil {
		return nil, err
	}

	hashMerkleRoot, err := protoBlockHeader.HashMerkleRoot.toWire()
	if err != nil {
		return nil, err
	}

	acceptedIDMerkleRoot, err := protoBlockHeader.AcceptedIDMerkleRoot.toWire()
	if err != nil {
		return nil, err
	}

	utxoCommitment, err := protoBlockHeader.UtxoCommitment.toWire()
	if err != nil {
		return nil, err
	}

	header := domainmessage.BlockHeader{
		Version:              protoBlockHeader.Version,
		ParentHashes:         parentHashes,
		HashMerkleRoot:       hashMerkleRoot,
		AcceptedIDMerkleRoot: acceptedIDMerkleRoot,
		UTXOCommitment:       utxoCommitment,
		Timestamp:            mstime.UnixMilliseconds(protoBlockHeader.Timestamp),
		Bits:                 protoBlockHeader.Bits,
		Nonce:                protoBlockHeader.Nonce,
	}

	transactions := make([]*domainmessage.MsgTx, len(x.Transactions))
	for i, protoTx := range x.Transactions {
		msgTx, err := protoTx.toDomainMessage()
		if err != nil {
			return nil, err
		}
		transactions[i] = msgTx.(*domainmessage.MsgTx)
	}

	return &domainmessage.MsgBlock{
		Header:       header,
		Transactions: transactions,
	}, nil
}

func (x *BlockMessage) fromDomainMessage(msgBlock *domainmessage.MsgBlock) error {
	if len(msgBlock.Transactions) > domainmessage.MaxTxPerBlock {
		return errors.Errorf("too many transactions to fit into a block "+
			"[count %d, max %d]", len(msgBlock.Transactions), domainmessage.MaxTxPerBlock)
	}

	if len(msgBlock.Header.ParentHashes) > domainmessage.MaxBlockParents {
		return errors.Errorf("block header has %d parents, but the maximum allowed amount "+
			"is %d", len(msgBlock.Header.ParentHashes), domainmessage.MaxBlockParents)
	}

	header := msgBlock.Header
	protoHeader := &BlockHeader{
		Version:              header.Version,
		ParentHashes:         wireHashesToProto(header.ParentHashes),
		HashMerkleRoot:       wireHashToProto(header.HashMerkleRoot),
		AcceptedIDMerkleRoot: wireHashToProto(header.AcceptedIDMerkleRoot),
		UtxoCommitment:       wireHashToProto(header.UTXOCommitment),
		Timestamp:            header.Timestamp.UnixMilliseconds(),
		Bits:                 header.Bits,
		Nonce:                header.Nonce,
	}
	protoTransactions := make([]*TransactionMessage, len(msgBlock.Transactions))
	for i, tx := range msgBlock.Transactions {
		protoTx := new(TransactionMessage)
		protoTx.fromDomainMessage(tx)
		protoTransactions[i] = protoTx
	}
	*x = BlockMessage{
		Header:       protoHeader,
		Transactions: protoTransactions,
	}
	return nil
}
