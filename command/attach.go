package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/hinshun/ptmux/pkg/p2p"
	"github.com/hinshun/ptmux/rvt"
	"github.com/libp2p/go-libp2p-core/network"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/rs/zerolog"
	cli "github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var attachCommand = &cli.Command{
	Name:      "attach",
	Usage:     "attach to an existing ptmux session",
	ArgsUsage: "<session-name>",
	Action:    Attach,
}

func Attach(c *cli.Context) error {
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
				ctx = network.WithUseTransient(ctx, "hole-punch")
				return gostream.Dial(ctx, p, peer.ID, "/ptmux/1.0.0")
			})
			conn, err = grpc.DialContext(ctx, peer.ID.String(), dialerOpt, grpc.WithInsecure())
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("unable to dial peer")
				continue
			}
		}
	}

	screenClient := rvt.NewScreenClient(conn)
	shareClient, err := screenClient.Share(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)

	recvMsgs := make(chan *rvt.ShareMessage)
	eg.Go(func() error {
		defer close(recvMsgs)
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
			case recvMsgs <- shareMsg:
			}
		}
	})

	sendMsgs := make(chan *rvt.ShareMessage)
	eg.Go(func() error {
		defer shareClient.CloseSend()
		for msg := range sendMsgs {
			err := shareClient.Send(msg)
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("failed to send share message")
				continue
			}
		}
		return nil
	})

	eg.Go(func() error {
		sendMsgs <- &rvt.ShareMessage{
			Id: p.ID().String(),
			Message: &rvt.ShareMessage_Init{
				Init: &rvt.InitMessage{},
			},
		}
		zerolog.Ctx(ctx).Info().Msg("Sent init message")
		return nil
	})

	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}

	err = s.Init()
	if err != nil {
		return err
	}
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	eg.Go(func() error {
		for {
			var shareMsg *rvt.ShareMessage
			select {
			case <-ctx.Done():
				return nil
			case shareMsg = <-recvMsgs:
			}
			if shareMsg == nil {
				return nil
			}

			switch evt := shareMsg.Message.(type) {
			case *rvt.ShareMessage_Render:
				rvt.RenderToScreen(evt.Render, s)
			}
		}
	})

	eventMsgs := make(chan *rvt.EventMessage, 1)
	eg.Go(func() error {
		defer close(sendMsgs)
		for {
			var eventMsg *rvt.EventMessage
			select {
			case <-ctx.Done():
				return nil
			case eventMsg = <-eventMsgs:
			}
			if eventMsg == nil {
				return nil
			}
			sendMsgs <- &rvt.ShareMessage{
				Id: p.ID().String(),
				Message: &rvt.ShareMessage_Event{
					Event: eventMsg,
				},
			}
		}
	})

	eg.Go(func() error {
		eventCh := make(chan tcell.Event, 4)
		go func() {
			defer close(eventCh)
			for {
				event := s.PollEvent()
				if event == nil {
					return
				}
				eventCh <- event
			}
		}()

		defer func() {
			cancel()
			s.Fini()
			// Drain remaining events.
			for _ = range eventCh {
			}
		}()

		prevWasMouseMove := false
		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-eventCh:
				switch evt := ev.(type) {
				case *tcell.EventKey:
					if evt.Key() == tcell.KeyCtrlQ {
						return nil
					}
				case *tcell.EventMouse:
					if evt.Modifiers() == 0 && evt.Buttons() == 0 {
						if prevWasMouseMove {
							continue
						}
						prevWasMouseMove = true
					} else {
						prevWasMouseMove = false
					}
				}
				msg := rvt.EventToProto(ev)
				if msg != nil {
					eventMsgs <- msg
				}
			}

		}
	})

	return eg.Wait()
}
