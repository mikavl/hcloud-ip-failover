package main

import (
	"context"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

type Resources struct {
	floatingIP *hcloud.FloatingIP
	network *hcloud.Network
	primaryServer *hcloud.Server
	secondaryServer *hcloud.Server

	target *hcloud.Server
	other *hcloud.Server
}

func readResources(ctx context.Context, client *hcloud.Client) (Resources, error) {
	var res Resources

	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var err error
		res.floatingIP, _, err = client.FloatingIP.GetByName(ctx, floatingIPName)
		return err
	})

	eg.Go(func() error {
		var err error
		res.network, _, err = client.Network.GetByName(ctx, networkName)
		return err
	})

	eg.Go(func() error {
		var err error
		res.primaryServer, _, err = client.Server.GetByName(ectx, primaryServerName)
		return err
	})

	eg.Go(func() error {
		var err error
		res.secondaryServer, _, err = client.Server.GetByName(ectx, secondaryServerName)
		return err
	})

	err := eg.Wait()

	if primaryServerAvailable {
		res.target = res.primaryServer
		res.other = res.secondaryServer
	} else {
		res.target = res.secondaryServer
		res.other = res.primaryServer
	}

	return res, err
}

func WaitAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) error {
	_, errc := client.Action.WatchProgress(ctx, action)
	err := <-errc
	return err
}

func assignAliasIPs(ctx context.Context, client *hcloud.Client, network *hcloud.Network, server *hcloud.Server, aliasIPs []net.IP) error {
	opts := hcloud.ServerChangeAliasIPsOpts{
		Network:  network,
		AliasIPs: aliasIPs,
	}

	action, _, err := client.Server.ChangeAliasIPs(ctx, server, opts)
	if err != nil {
		return err
	}

	return WaitAction(ctx, client, action)
}

func readToken(tokenPath string) (string, error) {
	if tokenPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		tokenPath = path.Join(homeDir, defaultTokenFile)
	}

	token, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(token)), nil
}

func execute(ctx context.Context) error {
	var otherIPs []net.IP
	targetIPs := []net.IP{
		aliasIP,
	}

	token, err := readToken(tokenPath)
	if err != nil {
		return err
	}

	client := hcloud.NewClient(hcloud.WithToken(token))

	res, err := readResources(ctx, client)
	if err != nil {
		return err
	}

	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		action, _, err := client.FloatingIP.Assign(ectx, res.floatingIP, res.target)
		if err != nil {
			return err
		}
		return WaitAction(ectx, client, action)
	})

	eg.Go(func() error {
		if err := assignAliasIPs(ectx, client, res.network, res.other, otherIPs); err != nil {
			return err
		}

		return assignAliasIPs(ectx, client, res.network, res.target, targetIPs)
	})

	return eg.Wait()
}

