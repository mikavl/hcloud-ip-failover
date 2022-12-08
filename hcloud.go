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
	floatingIP      *hcloud.FloatingIP
	network         *hcloud.Network
	primaryServer   *hcloud.Server
	secondaryServer *hcloud.Server

	target *hcloud.Server
	other  *hcloud.Server
}

func NewResources() *Resources {
	r := new(Resources)
	return r
}

func (r *Resources) ReadFloatingIP(ctx context.Context, client *hcloud.FloatingIPClient, name string) error {
	var err error
	r.floatingIP, _, err = client.GetByName(ctx, name)
	return err
}

func (r *Resources) ReadNetwork(ctx context.Context, client *hcloud.NetworkClient, name string) error {
	var err error
	r.network, _, err = client.GetByName(ctx, name)
	return err
}

func ReadServer(ctx context.Context, client *hcloud.ServerClient, name string) (*hcloud.Server, error) {
	server, _, err := client.GetByName(ctx, name)
	return server, err
}

func (r *Resources) ReadPrimaryServer(ctx context.Context, client *hcloud.ServerClient, name string) error {
	var err error
	r.primaryServer, err = ReadServer(ctx, client, name)
	return err
}

func (r *Resources) ReadSecondaryServer(ctx context.Context, client *hcloud.ServerClient, name string) error {
	var err error
	r.secondaryServer, err = ReadServer(ctx, client, name)
	return err
}

func (r *Resources) Read(ctx context.Context, client *hcloud.Client, args *Args) error {
	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error { return r.ReadFloatingIP(ectx, &client.FloatingIP, args.floatingIPName) })
	eg.Go(func() error { return r.ReadNetwork(ectx, &client.Network, args.networkName) })
	eg.Go(func() error { return r.ReadPrimaryServer(ectx, &client.Server, args.primaryServerName) })
	eg.Go(func() error { return r.ReadSecondaryServer(ectx, &client.Server, args.secondaryServerName) })

	err := eg.Wait()

	if args.primaryServerAvailable {
		r.target = r.primaryServer
		r.other = r.secondaryServer
	} else {
		r.target = r.secondaryServer
		r.other = r.primaryServer
	}

	return err
}

func WaitAction(ctx context.Context, client *hcloud.ActionClient, action *hcloud.Action) error {
	_, errc := client.WatchProgress(ctx, action)
	err := <-errc
	return err
}

func AssignAliasIP(ctx context.Context, client *hcloud.Client, network *hcloud.Network, server *hcloud.Server, aliasIP net.IP) error {
	var aliasIPs []net.IP

	if aliasIP != nil {
		aliasIPs = []net.IP{
			aliasIP,
		}
	}

	opts := hcloud.ServerChangeAliasIPsOpts{
		Network:  network,
		AliasIPs: aliasIPs,
	}

	action, _, err := client.Server.ChangeAliasIPs(ctx, server, opts)
	if err != nil {
		return err
	}

	return WaitAction(ctx, &client.Action, action)
}

func AssignFloatingIP(ctx context.Context, client *hcloud.Client, server *hcloud.Server, floatingIP *hcloud.FloatingIP) error {
	action, _, err := client.FloatingIP.Assign(ctx, floatingIP, server)
	if err != nil {
		return err
	}

	return WaitAction(ctx, &client.Action, action)
}

func TokenPath(tokenPath string) (string, error) {
	if tokenPath != "" {
		return tokenPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, defaultTokenFile), nil
}

func ReadToken(tokenPath string) (string, error) {
	p, err := TokenPath(tokenPath)
	if err != nil {
		return "", err
	}

	token, err := ioutil.ReadFile(p)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(token)), nil
}

func Execute(ctx context.Context, args *Args) error {
	token, err := ReadToken(args.tokenPath)
	if err != nil {
		return err
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	res := NewResources()

	if err := res.Read(ctx, client, args); err != nil {
		return err
	}

	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		// Assign floating IP to target
		return AssignFloatingIP(ectx, client, res.target, res.floatingIP)
	})

	eg.Go(func() error {
		// Remove alias IP from other
		if err := AssignAliasIP(ectx, client, res.network, res.other, nil); err != nil {
			return err
		}

		// Assign alias IP to target
		if err := AssignAliasIP(ectx, client, res.network, res.target, args.aliasIP); err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}
