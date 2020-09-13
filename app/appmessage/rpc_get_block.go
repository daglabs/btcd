package appmessage

// GetBlockRequestMessage is an appmessage corresponding to
// its respective RPC message
type GetBlockRequestMessage struct {
	baseMessage
	Hash                          string
	SubnetworkID                  string
	IncludeBlockHex               bool
	IncludeBlockVerboseData       bool
	IncludeTransactionVerboseData bool
}

// Command returns the protocol command string for the message
func (msg *GetBlockRequestMessage) Command() MessageCommand {
	return CmdGetBlockRequestMessage
}

// NewGetBlockRequestMessage returns a instance of the message
func NewGetBlockRequestMessage(hash string, subnetworkID string, includeBlockHex bool,
	includeBlockVerboseData bool, includeTransactionVerboseData bool) *GetBlockRequestMessage {
	return &GetBlockRequestMessage{
		Hash:                          hash,
		SubnetworkID:                  subnetworkID,
		IncludeBlockHex:               includeBlockHex,
		IncludeBlockVerboseData:       includeBlockVerboseData,
		IncludeTransactionVerboseData: includeTransactionVerboseData,
	}
}

// GetBlockResponseMessage is an appmessage corresponding to
// its respective RPC message
type GetBlockResponseMessage struct {
	baseMessage
	BlockHex         string
	BlockVerboseData *BlockVerboseData

	Error *RPCError
}

// Command returns the protocol command string for the message
func (msg *GetBlockResponseMessage) Command() MessageCommand {
	return CmdGetBlockResponseMessage
}

// NewGetBlockResponseMessage returns a instance of the message
func NewGetBlockResponseMessage() *GetBlockResponseMessage {
	return &GetBlockResponseMessage{}
}

// BlockVerboseData holds verbose data about a block
type BlockVerboseData struct {
	Hash                   string
	Confirmations          uint64
	Size                   int32
	BlueScore              uint64
	IsChainBlock           bool
	Version                int32
	VersionHex             string
	HashMerkleRoot         string
	AcceptedIDMerkleRoot   string
	UTXOCommitment         string
	TxIDs                  []string
	TransactionVerboseData []*TransactionVerboseData
	Time                   int64
	Nonce                  uint64
	Bits                   string
	Difficulty             float64
	ParentHashes           []string
	SelectedParentHash     string
	ChildHashes            []string
	AcceptedBlockHashes    []string
}

// TransactionVerboseData holds verbose data about a transaction
type TransactionVerboseData struct {
	Hex                       string
	TxID                      string
	Hash                      string
	Size                      int32
	Version                   int32
	LockTime                  uint64
	SubnetworkID              string
	Gas                       uint64
	PayloadHash               string
	Payload                   string
	TransactionVerboseInputs  []*TransactionVerboseInput
	TransactionVerboseOutputs []*TransactionVerboseOutput
	BlockHash                 string
	AcceptedBy                string
	IsInMempool               bool
	Time                      uint64
	BlockTime                 uint64
}

// TransactionVerboseInput holds data about a transaction input
type TransactionVerboseInput struct {
	TxID        string
	OutputIndex uint32
	ScriptSig   *ScriptSig
	Sequence    uint64
}

// ScriptSig holds data about a script signature
type ScriptSig struct {
	Asm string
	Hex string
}

// TransactionVerboseOutput holds data about a transaction output
type TransactionVerboseOutput struct {
	Value        uint64
	Index        uint32
	ScriptPubKey *ScriptPubKeyResult
}

// ScriptPubKeyResult holds data about a script public key
type ScriptPubKeyResult struct {
	Asm     string
	Hex     string
	Type    string
	Address string
}
