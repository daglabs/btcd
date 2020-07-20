package serverutils

import (
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/kaspanet/kaspad/config"

	"github.com/kaspanet/kaspad/util"
)

// GenCertPair generates a key/cert pair to the paths provided.
func GenCertPair(certFile, keyFile string) error {
	log.Infof("Generating TLS certificates...")

	org := "kaspad autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := util.NewTLSCertPair(org, validUntil, nil)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	log.Infof("Done generating TLS certificates")
	return nil
}

// KaspadDial connects to the address on the named network using the appropriate
// dial function depending on the address and configuration options.
func KaspadDial(cfg *config.Config, addr net.Addr) (net.Conn, error) {
	return cfg.Dial(addr.Network(), addr.String(), config.DefaultConnectTimeout)
}
