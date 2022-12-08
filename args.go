package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	flag "github.com/spf13/pflag"
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

	// the alert_cmd is invoked as "alert_cmd dest_addr alarm_flag latency_avg loss_avg"
	positionalArgs := flag.Args()

	if len(positionalArgs) < 2 {
		return nil, errors.New("missing required positional args")
	}

	if args.aliasIP = net.ParseIP(aliasIPArg); args.aliasIP == nil {
		return nil, errors.New("failed to parse alias ip")
	}

	alarm, err := strconv.ParseInt(positionalArgs[1], 10, 0)
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
