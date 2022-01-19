package rvt

import (
	"context"
	"errors"
	"io"
	"sync"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Screen interface {
	tcell.Screen

	Subscribe(id string, ch chan string)
	Unsubscribe(id string)
}

type Server struct {
	ctx    context.Context
	screen Screen
	id     string
	done   chan struct{}
	wg     sync.WaitGroup
}

func NewServer(ctx context.Context, screen Screen, id string) *Server {
	return &Server{
		ctx:    ctx,
		screen: screen,
		id:     id,
		done:   make(chan struct{}),
	}
}

func (s *Server) Cancel() {
	close(s.done)
}

func (s *Server) Close() error {
	s.wg.Wait()
	return nil
}

func (s *Server) Share(srv Screen_ShareServer) error {
	ctx := s.ctx

	s.wg.Add(1)
	recvMsgs := make(chan *ShareMessage)
	go func() {
		defer s.wg.Done()
		defer close(recvMsgs)
		for {
			shareMsg, err := srv.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					zerolog.Ctx(ctx).Error().Err(err).Msg("failed to receive share message")
				}
				return
			}

			select {
			case <-s.done:
				return
			case recvMsgs <- shareMsg:
			}
		}
	}()

	s.wg.Add(1)
	sendMsgs := make(chan *ShareMessage)
	go func() {
		defer s.wg.Done()
		for msg := range sendMsgs {
			err := srv.Send(msg)
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("failed to send share message")
				continue
			}
		}
	}()

	eg := new(errgroup.Group)

	var subscribeOnce sync.Once
	renderCh := make(chan string, 16)
	eg.Go(func() error {
		for {
			var shareMsg *ShareMessage
			select {
			case <-s.done:
				return nil
			case shareMsg = <-recvMsgs:
			}
			if shareMsg == nil {
				return nil
			}

			switch msg := shareMsg.Message.(type) {
			case *ShareMessage_Init:
				subscribeOnce.Do(func() {
					zerolog.Ctx(ctx).Info().Str("id", shareMsg.Id).Msg("New screen subscriber")
					s.screen.Subscribe(shareMsg.Id, renderCh)
					renderCh <- "init"
				})
			case *ShareMessage_Event:
				ev := ProtoToEvent(msg.Event)
				s.screen.PostEvent(&RemoteEvent{
					ID:    shareMsg.Id,
					Event: ev,
				})
			}
		}
	})

	eg.Go(func() error {
		defer close(sendMsgs)
		for {
			select {
			case <-s.done:
				return nil
			case <-renderCh:
				sendMsgs <- &ShareMessage{
					Id: s.id,
					Message: &ShareMessage_Render{
						Render: ScreenToRender(s.screen),
					},
				}
			}
		}
	})

	return eg.Wait()
}
