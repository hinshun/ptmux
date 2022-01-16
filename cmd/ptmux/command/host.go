package command

import (
	"context"
	"os"

	"github.com/hinshun/ptmux/pkg/p2p"
	"github.com/hinshun/ptmux/pkg/terminal"
	"github.com/hinshun/ptmux/rvt"
	"github.com/hinshun/ptmux/ui"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/rs/zerolog"
	cli "github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var hostCommand = &cli.Command{
	Name:  "host",
	Usage: "host a terminal sharing session",
	Action: func(c *cli.Context) error {
		logs, err := os.Create("server.log")
		if err != nil {
			return err
		}
		defer logs.Close()

		ctx := c.Context
		logger := zerolog.Ctx(ctx).Output(zerolog.ConsoleWriter{Out: logs})
		ctx = logger.WithContext(ctx)

		t, err := terminal.New(ctx)
		if err != nil {
			return err
		}

		u, err := ui.New(t)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		eg, ctx := errgroup.WithContext(ctx)

		renderCh := make(chan string, 1)
		t.Subscribe("host", renderCh)

		eg.Go(func() error {
			return u.Loop(ctx)
		})

		eg.Go(func() error {
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-t.Done():
					return nil
				case <-renderCh:
				}
				t.Lock()
				cols, rows := t.Size()
				u.Render(t.Title(), cols, rows)
				t.Unlock()
			}
		})

		opts := []grpc.ServerOption{}
		grpcSrv := grpc.NewServer(opts...)

		eg.Go(func() error {
			p, err := p2p.New(ctx)
			if err != nil {
				return err
			}
			defer p.Close()

			termSrv := rvt.NewServer(t, p.ID().String())
			defer termSrv.Close()

			rvt.RegisterTerminalServer(grpcSrv, termSrv)

			go func() {
				<-ctx.Done()
				termSrv.Cancel()
				grpcSrv.GracefulStop()
			}()

			_, err = p.Discovery.Advertise(ctx, "apple banana")
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("unable to advertise")
			}

			l, err := gostream.Listen(p, "/ptmux/1.0.0")
			if err != nil {
				return err
			}

			return grpcSrv.Serve(l)
		})

		return eg.Wait()
	},
}
