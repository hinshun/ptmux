package command

import (
	"context"
	"os"

	"github.com/hinshun/ptmux/pkg/p2p"
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

		ui, err := ui.New()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		eg, ctx := errgroup.WithContext(ctx)

		eg.Go(func() error {
			defer cancel()
			ui.Loop()
			return nil
		})

		opts := []grpc.ServerOption{}
		grpcSrv := grpc.NewServer(opts...)

		eg.Go(func() error {
			p, err := p2p.New(ctx)
			if err != nil {
				return err
			}
			defer p.Close()

			screenSrv := rvt.NewServer(ctx, ui.Screen(), p.ID().String())
			defer screenSrv.Close()

			rvt.RegisterScreenServer(grpcSrv, screenSrv)

			go func() {
				<-ctx.Done()
				screenSrv.Cancel()
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
