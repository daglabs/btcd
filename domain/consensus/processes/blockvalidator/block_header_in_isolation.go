package blockvalidator

import (
	"sort"

	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/ruleerrors"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensusserialization"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/consensus/utils/hashes"
	"github.com/pkg/errors"
)

// ValidateHeaderInIsolation validates block headers in isolation from the current
// consensus state
func (v *blockValidator) ValidateHeaderInIsolation(blockHash *externalapi.DomainHash) error {
	header, err := v.blockHeaderStore.BlockHeader(v.databaseContext, blockHash)
	if err != nil {
		return err
	}

	err = v.checkParentsLimit(header)
	if err != nil {
		return err
	}

	err = checkBlockParentsOrder(header)
	if err != nil {
		return err
	}

	return nil
}

func (v *blockValidator) checkParentsLimit(header *externalapi.DomainBlockHeader) error {
	hash := consensusserialization.HeaderHash(header)
	if len(header.ParentHashes) == 0 && *hash != *v.genesisHash {
		return errors.Wrapf(ruleerrors.ErrNoParents, "block has no parents")
	}

	if len(header.ParentHashes) > constants.MaxBlockParents {
		return errors.Wrapf(ruleerrors.ErrTooManyParents, "block header has %d parents, but the maximum allowed amount "+
			"is %d", len(header.ParentHashes), constants.MaxBlockParents)
	}
	return nil
}

//checkBlockParentsOrder ensures that the block's parents are ordered by hash
func checkBlockParentsOrder(header *externalapi.DomainBlockHeader) error {
	sortedHashes := make([]*externalapi.DomainHash, len(header.ParentHashes))
	for i, hash := range header.ParentHashes {
		sortedHashes[i] = hash
	}

	isSorted := sort.SliceIsSorted(sortedHashes, func(i, j int) bool {
		return hashes.Less(sortedHashes[i], sortedHashes[j])
	})

	if !isSorted {
		return errors.Wrapf(ruleerrors.ErrWrongParentsOrder, "block parents are not ordered by hash")
	}

	return nil
}
