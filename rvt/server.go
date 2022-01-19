package rvt

import (
	"errors"
	"io"
	"sync"

	"github.com/hinshun/ptmux/pkg/terminal"
	"github.com/hinshun/vt10x"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	term *terminal.Terminal
	id   string
	done chan struct{}
	wg   sync.WaitGroup
}

func NewServer(term *terminal.Terminal, id string) *Server {
	return &Server{
		term: term,
		id:   id,
		done: make(chan struct{}),
	}
}

func (s *Server) Cancel() {
	close(s.done)
}

func (s *Server) Close() error {
	s.wg.Wait()
	return nil
}

func (s *Server) Share(srv Terminal_ShareServer) error {
	ctx := srv.Context()

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
	updateCh := make(chan string, 16)
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

			switch shareMsg.GetEvent().(type) {
			case *ShareMessage_Init:
				subscribeOnce.Do(func() {
					zerolog.Ctx(ctx).Info().Str("id", shareMsg.Id).Msg("New terminal subscriber")
					s.term.Subscribe(shareMsg.Id, updateCh)
					updateCh <- "init"
				})
			case *ShareMessage_Tcell:
			}
		}
	})

	eg.Go(func() error {
		defer close(sendMsgs)
		for {
			select {
			case <-s.done:
				return nil
			case <-updateCh:
				sendMsgs <- terminalShareState(s.id, s.term)
			}
		}
	})

	return eg.Wait()
}

func terminalShareState(id string, term *terminal.Terminal) *ShareMessage {
	term.Lock()
	cols, rows := term.Size()
	mode := term.Mode()
	title := term.Title()
	cursor := term.Cursor()

	var lines []*Line
	for y := 0; y < rows; y++ {
		line := &Line{}
		for x := 0; x < cols; x++ {
			glyph := term.Cell(x, y)
			line.Glyphs = append(line.Glyphs, vtGlyphToProto(glyph))
		}
		lines = append(lines, line)
	}
	term.Unlock()

	return &ShareMessage{
		Id: id,
		Event: &ShareMessage_State{
			State: &StateMessage{
				Cols:  int64(cols),
				Rows:  int64(rows),
				Mode:  uint32(mode),
				Title: title,
				Cursor: &Cursor{
					X:     int64(cursor.X),
					Y:     int64(cursor.Y),
					State: uint32(cursor.State),
					Attr:  vtGlyphToProto(cursor.Attr),
				},
				Lines: lines,
			},
		},
	}
}

func vtGlyphToProto(glyph vt10x.Glyph) *Glyph {
	return &Glyph{
		Rune: int32(glyph.Char),
		Mode: int32(glyph.Mode),
		Fg:   uint32(glyph.FG),
		Bg:   uint32(glyph.BG),
	}
}

func protoGlyphToVT(glyph *Glyph) vt10x.Glyph {
	return vt10x.Glyph{
		Char: rune(glyph.Rune),
		Mode: int16(glyph.Mode),
		FG:   vt10x.Color(glyph.Fg),
		BG:   vt10x.Color(glyph.Bg),
	}
}
