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

type Client struct {
	client *hcloud.Client

	floatingIP      *hcloud.FloatingIP
	network         *hcloud.Network
	primaryServer   *hcloud.Server
	secondaryServer *hcloud.Server

	target *hcloud.Server
	other  *hcloud.Server
}

func NewClient(token string) *Client {
	c := new(Client)
	c.client = hcloud.NewClient(hcloud.WithToken(token))
	return c
}

func (c *Client) ReadFloatingIP(ctx context.Context, name string) error {
	var err error
	c.floatingIP, _, err = c.client.FloatingIP.GetByName(ctx, name)
	return err
}

func (c *Client) ReadNetwork(ctx context.Context, name string) error {
	var err error
	c.network, _, err = c.client.Network.GetByName(ctx, name)
	return err
}

func (c *Client) ReadPrimaryServer(ctx context.Context, name string) error {
	var err error
	c.primaryServer, _, err = c.client.Server.GetByName(ctx, name)
	return err
}

func (c *Client) ReadSecondaryServer(ctx context.Context, name string) error {
	var err error
	c.secondaryServer, _, err = c.client.Server.GetByName(ctx, name)
	return err
}

func (c *Client) Read(ctx context.Context, args *Args) error {
	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error { return c.ReadFloatingIP(ectx, args.floatingIPName) })
	eg.Go(func() error { return c.ReadNetwork(ectx, args.networkName) })
	eg.Go(func() error { return c.ReadPrimaryServer(ectx, args.primaryServerName) })
	eg.Go(func() error { return c.ReadSecondaryServer(ectx, args.secondaryServerName) })

	err := eg.Wait()

	if args.primaryServerAvailable {
		c.target = c.primaryServer
		c.other = c.secondaryServer
	} else {
		c.target = c.secondaryServer
		c.other = c.primaryServer
	}

	return err
}

func (c *Client) WaitAction(ctx context.Context, action *hcloud.Action) error {
	_, errc := c.client.Action.WatchProgress(ctx, action)
	err := <-errc
	return err
}

func (c *Client) AssignAliasIP(ctx context.Context, network *hcloud.Network, server *hcloud.Server, aliasIP net.IP) error {
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

	action, _, err := c.client.Server.ChangeAliasIPs(ctx, server, opts)
	if err != nil {
		return err
	}

	return c.WaitAction(ctx, action)
}

func (c *Client) AssignFloatingIP(ctx context.Context, server *hcloud.Server, floatingIP *hcloud.FloatingIP) error {
	action, _, err := c.client.FloatingIP.Assign(ctx, floatingIP, server)
	if err != nil {
		return err
	}

	return c.WaitAction(ctx, action)
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

	c := NewClient(token)

	if err := c.Read(ctx, args); err != nil {
		return err
	}

	eg, ectx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		// Assign floating IP to target
		return c.AssignFloatingIP(ectx, c.target, c.floatingIP)
	})

	eg.Go(func() error {
		// Remove alias IP from other
		if err := c.AssignAliasIP(ectx, c.network, c.other, nil); err != nil {
			return err
		}

		// Assign alias IP to target
		if err := c.AssignAliasIP(ectx, c.network, c.target, args.aliasIP); err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}
