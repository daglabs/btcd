package appmessage

// IBDBlocksMessage represents a kaspa IBDBlocks message
type IBDBlocksMessage struct {
	baseMessage
	Blocks []*MsgBlock
}

// Command returns the protocol command string for the message
func (msg *IBDBlocksMessage) Command() MessageCommand {
	return CmdIBDBlocks
}

// NewIBDBlocksMessage returns a new kaspa IBDBlocks message
func NewIBDBlocksMessage(blocks []*MsgBlock) *IBDBlocksMessage {
	return &IBDBlocksMessage{
		Blocks: blocks,
	}
}