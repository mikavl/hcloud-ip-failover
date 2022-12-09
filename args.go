package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	flag "github.com/spf13/pflag"
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

type Args struct {
	tokenPath           string
	floatingIPName      string
	primaryServerName   string
	secondaryServerName string
	networkName         string

	primaryServerAvailable bool
	aliasIP                net.IP
}

func NewArgs() *Args {
	args := new(Args)
	return args
}

func ParseArgs() (*Args, error) {
	var aliasIPArg string

	args := NewArgs()

	// Save this to its own var and parse later
	flag.StringVar(&aliasIPArg, "alias-ip", defaultAliasIP, "alias ip address")

	flag.StringVar(&args.tokenPath, "token-path", "", fmt.Sprintf(`hcloud token file path (default "~/%s")`, defaultTokenFile))
	flag.StringVar(&args.floatingIPName, "floating-ip-name", defaultFloatingIPName, "floating ip address name")
	flag.StringVar(&args.primaryServerName, "primary-server-name", defaultPrimaryServerName, "primary server name")
	flag.StringVar(&args.secondaryServerName, "secondary-server-name", defaultSecondaryServerName, "secondary server name")
	flag.StringVar(&args.networkName, "network-name", defaultNetworkName, "network name")
	flag.Parse()

	positionalArgs := flag.Args()

	if len(positionalArgs) != 1 {
		return nil, errors.New("invalid number of positional args")
	}

	if args.aliasIP = net.ParseIP(aliasIPArg); args.aliasIP == nil {
		return nil, errors.New("failed to parse alias ip")
	}

	alarm, err := strconv.ParseInt(positionalArgs[0], 10, 0)
	if err != nil {
		return nil, errors.New("failed to parse alarm, expected 0 or 1")
	}

	if alarm == 0 {
		args.primaryServerAvailable = true
	} else if alarm == 1 {
		args.primaryServerAvailable = false
	} else {
		return nil, errors.New("invalid alarm, expected 0 or 1")
	}

	return args, nil
}
