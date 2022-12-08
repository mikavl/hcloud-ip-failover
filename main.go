package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

const (
	// Default token file name, located in the user's home directory
	defaultTokenFile = ".hcloud_token"

	// See --help to override these as needed
	defaultAliasIP             = "10.0.0.3"
	defaultFloatingIPName      = "pfsense"
	defaultPrimaryServerName   = "pfsense-01"
	defaultSecondaryServerName = "pfsense-02"
	defaultNetworkName         = "lan"
)

func main() {
	args, err := ParseArgs()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("failed to parse command line arguments")
	}

	ctx := context.Background()

	if err := Execute(ctx, args); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("failover unsuccessful")
	}
}
