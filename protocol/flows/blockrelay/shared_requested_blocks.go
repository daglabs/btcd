package blockrelay

import (
	"sync"

	"github.com/kaspanet/kaspad/util/daghash"
)

type sharedRequestedBlocks struct {
	blocks map[daghash.Hash]struct{}
	sync.Mutex
}

func (s *sharedRequestedBlocks) remove(hash *daghash.Hash) {
	s.Lock()
	defer s.Unlock()
	delete(s.blocks, *hash)
}

func (s *sharedRequestedBlocks) removeSet(blockHashes map[daghash.Hash]struct{}) {
	for hash := range blockHashes {
		delete(s.blocks, hash)
	}
}

func (s *sharedRequestedBlocks) addIfNotExists(hash *daghash.Hash) (exists bool) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.blocks[*hash]
	if ok {
		return true
	}
	s.blocks[*hash] = struct{}{}
	return false
}

var requestedBlocks = &sharedRequestedBlocks{
	blocks: make(map[daghash.Hash]struct{}),
}
