package main

import (
	"context"
	"fmt"
	"net"
	"strconv"

	flag "github.com/spf13/pflag"
	log "github.com/sirupsen/logrus"
)

const (
	// Default token file name, located in the user's home directory
	defaultTokenFile = ".hcloud_token"

	// See --help to override these as needed
	defaultAliasIP = "10.0.0.3"
	defaultFloatingIPName = "pfsense"
	defaultPrimaryServerName = "pfsense-01"
	defaultSecondaryServerName = "pfsense-02"
	defaultNetworkName = "lan"
)

var (
	tokenPath string
	floatingIPName string
	primaryServerName string
	secondaryServerName string
	networkName string

	primaryServerAvailable bool
	aliasIP net.IP
)

func parseArgs() {
	var aliasIPArg string

	flag.StringVar(&aliasIPArg, "alias-ip", defaultAliasIP, "alias ip address")
	flag.StringVar(&tokenPath, "token-path", "", fmt.Sprintf(`hcloud token file path (default "~/%s")`, defaultTokenFile))
	flag.StringVar(&floatingIPName, "floating-ip-name", defaultFloatingIPName, "floating ip address name")
	flag.StringVar(&primaryServerName, "primary-server-name", defaultPrimaryServerName, "primary server name")
	flag.StringVar(&secondaryServerName, "secondary-server-name", defaultSecondaryServerName, "secondary server name")
	flag.StringVar(&networkName, "network-name", defaultNetworkName, "network name")
	flag.Parse()

	positionalArgs := flag.Args()
	if len(positionalArgs) < 2 {
		log.Fatal("missing required positional args, see --help")
	}

	if aliasIP = net.ParseIP(aliasIPArg); aliasIP == nil {
		log.WithFields(log.Fields{
			"aliasIP": aliasIPArg,
		}).Fatal("failed to parse alias ip")
	}

	// the alert_cmd is invoked as "alert_cmd dest_addr alarm_flag latency_avg loss_avg"
	alarm := positionalArgs[1]
	alarmInt, err := strconv.ParseInt(alarm, 10, 0)
	if err != nil {
		log.WithFields(log.Fields{
			"alarm": alarm,
		}).Fatal("failed to parse alarm")
	}

	switch alarmInt {
	case 0:
		primaryServerAvailable = true
	case 1:
		primaryServerAvailable = false
	default:
		log.WithFields(log.Fields{
			"alarmInt": alarmInt,
		}).Fatal("invalid alarm, expected 0 or 1")
	}
}

func main() {
	parseArgs()
	ctx := context.Background()

	if err := Execute(ctx); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("failover unsuccessful")
	}
}
