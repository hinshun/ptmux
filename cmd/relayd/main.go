package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/routing"
	gdiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	privk, err := loadIdentity("identity")
	if err != nil {
		return err
	}

	var idht *dht.IpfsDHT
	host, err := libp2p.New(
		libp2p.Identity(privk),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/4001"),
		libp2p.DisableRelay(),
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableRelayService(),
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			idht, err = dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
			return idht, err
		}),
	)
	if err != nil {
		return err
	}

	host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			fmt.Printf("Connected to %s [%s]\n", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			fmt.Printf("Disconnected %s [%s]\n", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
	})

	err = idht.Bootstrap(ctx)
	if err != nil {
		return err
	}

	for _, maddr := range host.Addrs() {
		p2pAddr := fmt.Sprintf("%s/p2p/%s", maddr.String(), host.ID())
		fmt.Printf("Libp2p swarm listening on %s\n", p2pAddr)
	}

	rd := gdiscovery.NewRoutingDiscovery(idht)
	go Advertise(ctx, rd)

	select {}
	return nil
}

// Advertise advertises this node as a libp2p relay.
func Advertise(ctx context.Context, advertise discovery.Advertiser) {
	for {
		ttl, err := advertise.Advertise(ctx, autorelay.RelayRendezvous, discovery.TTL(autorelay.AdvertiseTTL))
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			select {
			case <-time.After(2 * time.Minute):
				continue
			case <-ctx.Done():
				return
			}
		}

		wait := 7 * ttl / 8
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return
		}
	}
}

func loadIdentity(idPath string) (crypto.PrivKey, error) {
	if _, err := os.Stat(idPath); err == nil {
		return readIdentity(idPath)
	} else if os.IsNotExist(err) {
		fmt.Printf("Generating peer identity in %s\n", idPath)
		return generateIdentity(idPath)
	} else {
		return nil, err
	}
}

func readIdentity(path string) (crypto.PrivKey, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(bytes)
}

func generateIdentity(path string) (crypto.PrivKey, error) {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, err
	}

	bytes, err := crypto.MarshalPrivateKey(privk)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(path, bytes, 0400)

	return privk, err
}
