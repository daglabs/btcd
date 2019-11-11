package main

import (
	"github.com/daglabs/btcd/btcec"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/util/base58"
)

func decodeKeys(cfg *ConfigFlags) (*btcec.PrivateKey, *util.AddressPubKeyHash, error) {
	privateKeyBytes := base58.Decode(cfg.PrivateKey)
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privateKeyBytes)
	serializedPrivateKey := privateKey.PubKey().SerializeCompressed()

	addr, err := util.NewAddressPubKeyHashFromPublicKey(serializedPrivateKey, ActiveConfig().NetParams().Prefix)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, addr, nil
}
