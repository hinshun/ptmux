package p2p

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	gdiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/rs/zerolog"
)

const (
	RelayAddr = "/ip4/128.199.6.92/tcp/4001/p2p/12D3KooWGDynerXsf3KeXAQAp4RUpQKkusYYEjMq918czPxUXCRX"
)

var (
	DefaultBootstrapPeers []peer.AddrInfo
)

func init() {
	relay, err := peer.AddrInfoFromString(RelayAddr)
	if err != nil {
		panic(err)
	}
	DefaultBootstrapPeers = append(DefaultBootstrapPeers, *relay)
}

type Peer struct {
	host.Host
	DHT       *dht.IpfsDHT
	Discovery discovery.Discovery
}

func New(ctx context.Context) (*Peer, error) {
	var idht *dht.IpfsDHT
	host, err := libp2p.New(
		libp2p.Defaults,
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT.
		libp2p.EnableAutoRelay(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		libp2p.EnableNATService(),
		// EnableHolePunching enables NAT traversal by enabling NATT'd peers to both
		// initiate and respond to hole punching attempts to create direct /
		// NAT-traversed connections with other peers.
		libp2p.EnableHolePunching(holepunch.WithTracer(&holepunchTracer{zerolog.Ctx(ctx)})),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			idht, err = dht.New(ctx, h, dht.BootstrapPeers(DefaultBootstrapPeers...))
			return idht, err
		}),
	)
	if err != nil {
		return nil, err
	}

	host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			zerolog.Ctx(ctx).Info().Msgf("Connected to %s [%s]", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			zerolog.Ctx(ctx).Info().Msgf("Disconnected from %s [%s]", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
	})

	idht, err = dht.New(ctx, host)
	if err != nil {
		return nil, err
	}

	err = idht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	for _, maddr := range host.Addrs() {
		p2pAddr := fmt.Sprintf("%s/p2p/%s", maddr.String(), host.ID())
		zerolog.Ctx(ctx).Info().Msgf("Libp2p swarm listening on %s", p2pAddr)
	}

	for _, peerAddr := range DefaultBootstrapPeers {
		err = host.Connect(ctx, peerAddr)
		if err != nil {
			return nil, err
		}
	}

	return &Peer{
		Host:      host,
		DHT:       idht,
		Discovery: gdiscovery.NewRoutingDiscovery(idht),
	}, nil
}

type holepunchTracer struct {
	log *zerolog.Logger
}

func (ht *holepunchTracer) Trace(evt *holepunch.Event) {
	switch v := evt.Evt.(type) {
	case *holepunch.EndHolePunchEvt:
		if v.Success {
			ht.log.Info().Msgf("Hole punched %s->%s", evt.Peer, evt.Remote)
		} else {
			ht.log.Info().Msgf("Unable to holepunch %s->%s: %s", evt.Peer, evt.Remote, v.Error)
		}
	case *holepunch.DirectDialEvt:
		if v.Success {
			ht.log.Info().Msgf("Direct dial %s->%s", evt.Peer, evt.Remote)
		} else {
			ht.log.Info().Msgf("Unable to direct dial %s->%s: %s", evt.Peer, evt.Remote, v.Error)
		}
	}
}
