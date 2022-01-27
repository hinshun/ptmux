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

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "ptmux"
	app.Usage = "p2p terminal multiplexer"
	app.Action = StartSession
	app.Commands = []*cli.Command{
		attachCommand,
	}
	return app
}

func StartSession(c *cli.Context) error {
	logs, err := os.Create("server.log")
	if err != nil {
		return err
	}
	defer logs.Close()

	ctx := c.Context
	logger := zerolog.Ctx(ctx).Output(zerolog.ConsoleWriter{Out: logs})
	ctx = logger.WithContext(ctx)

	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		p, err := p2p.New(ctx)
		if err != nil {
			return err
		}
		defer p.Close()

		ui, err := ui.New(p.ID().String())
		if err != nil {
			return err
		}

		eg.Go(func() error {
			defer cancel()
			ui.Loop()
			return nil
		})

		screenSrv := rvt.NewServer(ctx, ui.Screen(), p.ID().String())
		defer screenSrv.Close()

		opts := []grpc.ServerOption{}
		grpcSrv := grpc.NewServer(opts...)

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
}
