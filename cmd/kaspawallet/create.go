package main

import (
	"bufio"
	"fmt"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"os"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/keys"
)

func create(conf *createConfig) error {
	var encryptedMnemonics []*keys.EncryptedMnemonic
	var signerExtendedPublicKeys []string
	var err error
	isMultisig := conf.NumPublicKeys > 1
	if !conf.Import {
		encryptedMnemonics, signerExtendedPublicKeys, err = keys.CreateKeyPairs(conf.NetParams(), conf.NumPrivateKeys, isMultisig)
	} else {
		encryptedMnemonics, signerExtendedPublicKeys, err = keys.ImportKeyPairs(conf.NetParams(), conf.NumPrivateKeys, isMultisig)
	}
	if err != nil {
		return err
	}

	for i, extendedPublicKey := range signerExtendedPublicKeys {
		fmt.Printf("Extended public key of mnemonic #%d:\n%s\n\n", i+1, extendedPublicKey)
	}

	extendedPublicKeys := make([]string, conf.NumPrivateKeys, conf.NumPublicKeys)
	copy(extendedPublicKeys, signerExtendedPublicKeys)
	reader := bufio.NewReader(os.Stdin)
	for i := conf.NumPrivateKeys; i < conf.NumPublicKeys; i++ {
		fmt.Printf("Enter public key #%d here:\n", i+1)
		extendedPublicKey, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		fmt.Println()

		extendedPublicKeys = append(extendedPublicKeys, string(extendedPublicKey))
	}

	cosignerIndex, err := libkaspawallet.MinimumCosignerIndex(signerExtendedPublicKeys, extendedPublicKeys)
	if err != nil {
		return err
	}

	file := keys.File{
		EncryptedMnemonics:    encryptedMnemonics,
		ExtendedPublicKeys:    extendedPublicKeys,
		MinimumSignatures:     conf.MinimumSignatures,
		CosignerIndex:         cosignerIndex,
		LastUsedExternalIndex: 0,
		LastUsedInternalIndex: 0,
		ECDSA:                 conf.ECDSA,
	}
	file.SetPath(conf.NetParams(), conf.KeysFile)
	err = file.Sync(false)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote the keys into %s\n", file.Path())
	return nil
}
