package command

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hinshun/ptmux/pkg/p2p"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/rs/zerolog"
	cli "github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var joinCommand = &cli.Command{
	Name:      "join",
	Usage:     "join a terminal sharing session",
	ArgsUsage: "<multiaddr>",
	Action: func(c *cli.Context) error {
		logs, err := os.Create("client.log")
		if err != nil {
			return err
		}
		defer logs.Close()

		ctx := c.Context
		logger := zerolog.Ctx(ctx).Output(zerolog.ConsoleWriter{Out: logs})
		ctx = logger.WithContext(ctx)

		p, err := p2p.New(ctx)
		if err != nil {
			return err
		}
		defer p.Close()

		var conn *grpc.ClientConn
		for conn == nil {
			peerChan, err := p.Discovery.FindPeers(ctx, "apple banana")
			if err != nil {
				return fmt.Errorf("unable to find peers: %w", err)
			}

			for peer := range peerChan {
				if peer.ID == p.ID() {
					continue
				}

				zerolog.Ctx(ctx).Info().Msgf("Discovered peer %s", peer.ID)
				dialerOpt := grpc.WithDialer(func(id string, ttl time.Duration) (net.Conn, error) {
					return gostream.Dial(ctx, p, peer.ID, "/ptmux/1.0.0")
				})
				conn, err = grpc.DialContext(ctx, peer.ID.String(), dialerOpt, grpc.WithInsecure())
				if err != nil {
					zerolog.Ctx(ctx).Error().Err(err).Msg("unable to dial peer")
					continue
				}
			}
		}

		termClient := rvt.NewTerminalClient(conn)
		shareClient, err := termClient.Share(ctx)
		if err != nil {
			return err
		}

		eg, ctx := errgroup.WithContext(ctx)

		msgs := make(chan *rvt.ShareMessage)

		eg.Go(func() error {
			return shareClient.Send(&rvt.ShareMessage{
				Id: p.ID().String(),
				Event: &rvt.ShareMessage_Init{
					Init: &rvt.InitMessage{},
				},
			})
		})

		eg.Go(func() error {
			for {
				shareMsg, err := shareClient.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						return nil
					}
					return err
				}

				select {
				case <-ctx.Done():
				case msgs <- shareMsg:
				}
			}
		})

		view := rvt.NewView()
		r, err := ui.New(view)
		if err != nil {
			return err
		}

		var readyOnce sync.Once
		ready := make(chan struct{})
		renderCh := make(chan struct{}, 1)
		eg.Go(func() error {
			for {
				var shareMsg *rvt.ShareMessage
				select {
				case <-ctx.Done():
					return nil
				case shareMsg = <-msgs:
				}
				if shareMsg == nil {
					return nil
				}

				switch evt := shareMsg.GetEvent().(type) {
				case *rvt.ShareMessage_State:
					view.Lock()
					view.Update(evt.State)
					view.Unlock()
					renderCh <- struct{}{}

					readyOnce.Do(func() {
						close(ready)
					})
				}
			}
		})

		eg.Go(func() error {
			<-ready
			return r.Loop(ctx)
		})

		eg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-renderCh:
				}
				view.Lock()
				cols, rows := view.Size()
				r.Render(view.Title(), cols, rows)
				view.Unlock()
			}
		})

		return eg.Wait()
	},
}
