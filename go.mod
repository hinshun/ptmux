module github.com/hinshun/ptmux

go 1.14

require (
	github.com/creack/pty v1.1.17
	github.com/gcla/gowid v1.3.0
	github.com/gdamore/tcell/v2 v2.4.0
	github.com/gogo/protobuf v1.3.2
	github.com/hinshun/vt10x v0.0.0-20220115061537-4526f461cf66
	github.com/libp2p/go-libp2p v0.17.0
	github.com/libp2p/go-libp2p-core v0.13.0
	github.com/libp2p/go-libp2p-discovery v0.6.0
	github.com/libp2p/go-libp2p-gostream v0.3.1
	github.com/libp2p/go-libp2p-kad-dht v0.15.0
	github.com/multiformats/go-multiaddr v0.5.0 // indirect
	github.com/rs/zerolog v1.26.1
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.43.0
)

replace github.com/gcla/gowid => github.com/hinshun/gowid v1.3.1-0.20220119030521-02904aead290
